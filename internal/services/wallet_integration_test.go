package services

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"testing"
	"walletapp/internal/db"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// testDB holds the database connection for our integration tests
var testDB *sql.DB
var walletService *WalletService

// TestMain sets up the test environment before running tests
func TestMain(m *testing.M) {
	var err error

	// Connect to test database using standard sql driver
	testDB, err = sql.Open("postgres", "postgres://walletuser:walletpass@localhost:5432/walletdb?sslmode=disable")
	if err != nil {
		panic(err)
	}

	// Initialize pgxpool for service functions (they use pgx, not database/sql)
	os.Setenv("DATABASE_URL", "postgres://walletuser:walletpass@localhost:5432/walletdb?sslmode=disable")
	db.Connect()

	// Create the real service with real implementations for integration tests
	walletRepo := NewWalletRepoImpl()
	transactionRepo := NewTransactionRepoImpl()
	dbImpl := NewDBImpl()
	walletService = NewWalletService(walletRepo, transactionRepo, dbImpl)

	// Run all tests
	code := m.Run()

	// Clean up connections
	testDB.Close()
	db.DB.Close()
	os.Exit(code)
}

// setupTestUser creates a test user in the database
// We use this to set up test data before running wallet operations
func setupTestUser(t *testing.T, userID uuid.UUID) {
	_, err := testDB.Exec(`INSERT INTO users (id, username, first_name, last_name, email, password, created_at, updated_at)
		VALUES ($1, $2, 'Test', 'User', $3, 'password', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`,
		userID.String(),
		userID.String()+"_testuser",
		userID.String()+"@example.com")
	if err != nil {
		t.Fatalf("setupTestUser: %v", err)
	}
}

// setupTestWallet creates a wallet for a test user with a specific balance
// This ensures we have a known starting state for our tests
func setupTestWallet(t *testing.T, userID uuid.UUID, balance float64) {
	_, err := testDB.Exec(`INSERT INTO wallets (id, user_id, balance, created_at, updated_at) 
		VALUES (gen_random_uuid(), $1, $2, NOW(), NOW()) 
		ON CONFLICT (user_id) DO UPDATE SET balance = $2`,
		userID.String(), balance)
	if err != nil {
		t.Fatalf("setupTestWallet: %v", err)
	}
}

// cleanupTestUser deletes a test user and all related entities
// This ensures we don't leave test data in the database
func cleanupTestUser(t *testing.T, userID uuid.UUID) {
	// Delete in order to respect foreign key constraints
	// 1. Delete transactions related to the user's wallet
	_, err := testDB.Exec(`DELETE FROM transactions WHERE wallet_id IN (SELECT id FROM wallets WHERE user_id = $1)`, userID.String())
	if err != nil {
		t.Logf("cleanupTestUser - delete transactions: %v", err)
	}

	// 2. Delete the wallet
	_, err = testDB.Exec(`DELETE FROM wallets WHERE user_id = $1`, userID.String())
	if err != nil {
		t.Logf("cleanupTestUser - delete wallet: %v", err)
	}

	// 3. Delete the user
	_, err = testDB.Exec(`DELETE FROM users WHERE id = $1`, userID.String())
	if err != nil {
		t.Logf("cleanupTestUser - delete user: %v", err)
	}
}

// getWalletBalance retrieves the current balance of a user's wallet
// We use this to verify that operations worked correctly
func getWalletBalance(t *testing.T, userID uuid.UUID) float64 {
	var balance float64
	err := testDB.QueryRow(`SELECT balance FROM wallets WHERE user_id = $1`, userID.String()).Scan(&balance)
	if err != nil {
		t.Fatalf("getWalletBalance: %v", err)
	}
	return balance
}

// TestTransfer_Atomicity verifies that money transfers are atomic
// This means either the entire transfer succeeds or it fails completely
// No partial state should be possible (e.g., money deducted but not credited)
func TestTransfer_Atomicity(t *testing.T) {
	// Create two test users
	user1ID := uuid.New()
	user2ID := uuid.New()

	// Set up initial state: user1 has $100, user2 has $50
	setupTestUser(t, user1ID)
	setupTestUser(t, user2ID)
	setupTestWallet(t, user1ID, 100)
	setupTestWallet(t, user2ID, 50)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, user1ID)
		cleanupTestUser(t, user2ID)
	}()

	// Perform a transfer of $30 from user1 to user2
	ctx := context.Background()
	err := walletService.Transfer(ctx, user1ID.String(), user2ID.String(), 30)
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
	}

	// Check the final balances
	bal1 := getWalletBalance(t, user1ID)
	bal2 := getWalletBalance(t, user2ID)

	// Verify atomicity: both balances must be updated correctly
	// If transfer succeeded: user1 should have $70, user2 should have $80
	// If transfer failed: both should have original amounts
	if bal1 == 100 && bal2 == 50 {
		t.Error("transfer did not update balances - operation may have failed silently")
	} else if bal1 != 70 || bal2 != 80 {
		t.Errorf("atomicity violated: got balances %v and %v, want 70 and 80", bal1, bal2)
	}
}

// TestWithdraw_RaceCondition tests concurrent withdrawals to ensure
// our locking mechanism prevents race conditions and double-spending
func TestWithdraw_RaceCondition(t *testing.T) {
	// Create a test user with $100
	userID := uuid.New()
	setupTestUser(t, userID)
	setupTestWallet(t, userID, 100)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, userID)
	}()

	// Launch 10 concurrent withdrawal attempts of $15 each
	// With only $100 available, at most 6 should succeed (6 * $15 = $90)
	var wg sync.WaitGroup
	errorsCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := walletService.Withdraw(context.Background(), userID.String(), 15)
			errorsCh <- err
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorsCh)

	// Count how many withdrawals succeeded
	success := 0
	for err := range errorsCh {
		if err == nil {
			success++
		}
	}

	// Verify we don't exceed the maximum possible withdrawals
	// With $100 balance and $15 withdrawals, max 6 should succeed
	if success > 6 {
		t.Errorf("race condition detected: more withdrawals succeeded than should be possible, got %d", success)
	}

	// Check final balance
	bal := getWalletBalance(t, userID)
	if bal < 0 {
		t.Errorf("race condition: balance went negative, got %v", bal)
	}

	// Verify the final balance is mathematically consistent
	// If N withdrawals succeeded, balance should be 100 - (N * 15)
	expectedBalance := 100 - float64(success)*15
	if bal != expectedBalance {
		t.Errorf("inconsistent final balance: got %v, expected %v", bal, expectedBalance)
	}
}

// TestTransfer_InvalidUUID tests that transfers with invalid UUIDs are rejected
func TestTransfer_InvalidUUID(t *testing.T) {
	ctx := context.Background()

	// Test with invalid UUID format
	err := walletService.Transfer(ctx, "invalid-uuid", "also-invalid", 10)
	if err == nil {
		t.Error("expected error for invalid UUID, got nil")
	}

	// Test with malformed UUID
	err = walletService.Transfer(ctx, "12345678-1234-1234-1234-123456789012", "87654321-4321-4321-4321-210987654321", 10)
	if err == nil {
		t.Error("expected error for malformed UUID, got nil")
	}
}

// TestTransfer_NonExistentUser tests that transfers to non-existent users fail
func TestTransfer_NonExistentUser(t *testing.T) {
	// Create one real user
	userID := uuid.New()
	setupTestUser(t, userID)
	setupTestWallet(t, userID, 100)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, userID)
	}()

	// Try to transfer to non-existent user
	nonExistentUserID := uuid.New()
	ctx := context.Background()
	err := walletService.Transfer(ctx, userID.String(), nonExistentUserID.String(), 10)
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}

	// Verify original balance unchanged
	bal := getWalletBalance(t, userID)
	if bal != 100 {
		t.Errorf("balance should remain unchanged, got %v", bal)
	}
}

// TestTransfer_NonExistentFromUser tests that transfers from non-existent users fail
func TestTransfer_NonExistentFromUser(t *testing.T) {
	// Create one real user
	userID := uuid.New()
	setupTestUser(t, userID)
	setupTestWallet(t, userID, 100)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, userID)
	}()

	// Try to transfer from non-existent user
	nonExistentUserID := uuid.New()
	ctx := context.Background()
	err := walletService.Transfer(ctx, nonExistentUserID.String(), userID.String(), 10)
	if err == nil {
		t.Error("expected error for non-existent from user, got nil")
	}

	// Verify original balance unchanged
	bal := getWalletBalance(t, userID)
	if bal != 100 {
		t.Errorf("balance should remain unchanged, got %v", bal)
	}
}

// TestTransfer_SelfTransferIntegration tests that users cannot transfer to themselves
func TestTransfer_SelfTransferIntegration(t *testing.T) {
	userID := uuid.New()
	setupTestUser(t, userID)
	setupTestWallet(t, userID, 100)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, userID)
	}()

	ctx := context.Background()
	err := walletService.Transfer(ctx, userID.String(), userID.String(), 10)
	if err == nil {
		t.Error("expected error for self-transfer, got nil")
	}

	// Verify balance unchanged
	bal := getWalletBalance(t, userID)
	if bal != 100 {
		t.Errorf("balance should remain unchanged for self-transfer, got %v", bal)
	}
}

// TestTransactionRollback tests that failed transactions are properly rolled back
func TestTransactionRollback(t *testing.T) {
	user1ID := uuid.New()
	user2ID := uuid.New()
	setupTestUser(t, user1ID)
	setupTestUser(t, user2ID)
	setupTestWallet(t, user1ID, 100)
	setupTestWallet(t, user2ID, 50)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, user1ID)
		cleanupTestUser(t, user2ID)
	}()

	// Try to transfer more than available balance
	ctx := context.Background()
	err := walletService.Transfer(ctx, user1ID.String(), user2ID.String(), 150) // More than $100
	if err == nil {
		t.Error("expected error for insufficient funds, got nil")
	}

	// Verify both balances unchanged
	bal1 := getWalletBalance(t, user1ID)
	bal2 := getWalletBalance(t, user2ID)
	if bal1 != 100 || bal2 != 50 {
		t.Errorf("transaction rollback failed: got balances %v and %v, want 100 and 50", bal1, bal2)
	}
}

// TestMinimumAmount_Transfer tests that very small amounts are rejected for transfers
func TestMinimumAmount_Transfer(t *testing.T) {
	user1ID := uuid.New()
	user2ID := uuid.New()
	setupTestUser(t, user1ID)
	setupTestUser(t, user2ID)
	setupTestWallet(t, user1ID, 100)
	setupTestWallet(t, user2ID, 50)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, user1ID)
		cleanupTestUser(t, user2ID)
	}()

	ctx := context.Background()

	// Test various small amounts
	smallAmounts := []float64{0.0001, 0.001, 0.009, 0.005}
	for _, amount := range smallAmounts {
		err := walletService.Transfer(ctx, user1ID.String(), user2ID.String(), amount)
		if err == nil {
			t.Errorf("expected error for small amount %v, got nil", amount)
		} else if !strings.Contains(err.Error(), "amount must be at least 0.01") {
			t.Errorf("expected 'amount must be at least 0.01' error for amount %v, got: %v", amount, err)
		}
	}

	// Verify balances unchanged
	bal1 := getWalletBalance(t, user1ID)
	bal2 := getWalletBalance(t, user2ID)
	if bal1 != 100 || bal2 != 50 {
		t.Errorf("balances should remain unchanged, got %v and %v", bal1, bal2)
	}
}

// TestMinimumAmount_Deposit tests that very small amounts are rejected for deposits
func TestMinimumAmount_Deposit(t *testing.T) {
	userID := uuid.New()
	setupTestUser(t, userID)
	setupTestWallet(t, userID, 100)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, userID)
	}()

	ctx := context.Background()

	// Test various small amounts
	smallAmounts := []float64{0.0001, 0.001, 0.009, 0.005}
	for _, amount := range smallAmounts {
		_, err := walletService.Deposit(ctx, userID.String(), amount)
		if err == nil {
			t.Errorf("expected error for small amount %v, got nil", amount)
		} else if !strings.Contains(err.Error(), "amount must be at least 0.01") {
			t.Errorf("expected 'amount must be at least 0.01' error for amount %v, got: %v", amount, err)
		}
	}

	// Verify balance unchanged
	bal := getWalletBalance(t, userID)
	if bal != 100 {
		t.Errorf("balance should remain unchanged, got %v", bal)
	}
}

// TestMinimumAmount_Withdraw tests that very small amounts are rejected for withdrawals
func TestMinimumAmount_Withdraw(t *testing.T) {
	userID := uuid.New()
	setupTestUser(t, userID)
	setupTestWallet(t, userID, 100)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, userID)
	}()

	ctx := context.Background()

	// Test various small amounts
	smallAmounts := []float64{0.0001, 0.001, 0.009, 0.005}
	for _, amount := range smallAmounts {
		_, err := walletService.Withdraw(ctx, userID.String(), amount)
		if err == nil {
			t.Errorf("expected error for small amount %v, got nil", amount)
		} else if !strings.Contains(err.Error(), "amount must be at least 0.01") {
			t.Errorf("expected 'amount must be at least 0.01' error for amount %v, got: %v", amount, err)
		}
	}

	// Verify balance unchanged
	bal := getWalletBalance(t, userID)
	if bal != 100 {
		t.Errorf("balance should remain unchanged, got %v", bal)
	}
}

// TestValidMinimumAmount tests that the minimum valid amount (0.01) works correctly
func TestValidMinimumAmount(t *testing.T) {
	user1ID := uuid.New()
	user2ID := uuid.New()
	setupTestUser(t, user1ID)
	setupTestUser(t, user2ID)
	setupTestWallet(t, user1ID, 100)
	setupTestWallet(t, user2ID, 50)

	// Clean up after test
	defer func() {
		cleanupTestUser(t, user1ID)
		cleanupTestUser(t, user2ID)
	}()

	ctx := context.Background()

	// Test minimum valid amount for transfer
	err := walletService.Transfer(ctx, user1ID.String(), user2ID.String(), 0.01)
	if err != nil {
		t.Errorf("expected no error for minimum valid amount 0.01, got: %v", err)
	}

	// Verify transfer worked
	bal1 := getWalletBalance(t, user1ID)
	bal2 := getWalletBalance(t, user2ID)
	if bal1 != 99.99 || bal2 != 50.01 {
		t.Errorf("transfer failed: got balances %v and %v, want 99.99 and 50.01", bal1, bal2)
	}
}
