package services

import (
	"context"
	"errors"
	"math"
	"testing"

	"walletapp/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing
type MockWalletRepo struct {
	mock.Mock
}

func (m *MockWalletRepo) GetWalletByUserID(ctx context.Context, userID string) (*models.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepo) GetWalletByUserIDTx(ctx context.Context, tx pgx.Tx, userID string) (*models.Wallet, error) {
	args := m.Called(ctx, tx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepo) UpdateWalletBalanceTx(ctx context.Context, tx pgx.Tx, userID string, newBalance float64) error {
	args := m.Called(ctx, tx, userID, newBalance)
	return args.Error(0)
}

type MockTransactionRepo struct {
	mock.Mock
}

func (m *MockTransactionRepo) CreateTransactionTx(ctx context.Context, tx pgx.Tx, t *models.Transaction) error {
	args := m.Called(ctx, tx, t)
	return args.Error(0)
}

// Test helper functions
func setupMocks() (*MockWalletRepo, *MockTransactionRepo, pgxmock.PgxPoolIface, error) {
	mockWalletRepo := new(MockWalletRepo)
	mockTxRepo := new(MockTransactionRepo)

	// pgxmock handles all the pgx.Tx interface complexity for us!
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		return nil, nil, nil, err
	}

	return mockWalletRepo, mockTxRepo, mockDB, nil
}

// Table-driven tests using the real WalletService struct
func TestWalletService_Transfer(t *testing.T) {
	tests := []struct {
		name          string
		fromBalance   float64
		toBalance     float64
		amount        float64
		fromUserID    string
		toUserID      string
		setupMocks    func(*MockWalletRepo, *MockTransactionRepo, pgxmock.PgxPoolIface)
		expectedError string
	}{
		{
			name:        "successful transfer",
			fromBalance: 100,
			toBalance:   50,
			amount:      30,
			fromUserID:  "user1",
			toUserID:    "user2",
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				// Set up pgxmock expectations for transaction flow
				db.ExpectBegin()
				db.ExpectCommit()

				// Set up repository mocks
				wr.On("GetWalletByUserIDTx", mock.Anything, mock.Anything, "user1").Return(&models.Wallet{Balance: 100}, nil)
				wr.On("GetWalletByUserIDTx", mock.Anything, mock.Anything, "user2").Return(&models.Wallet{Balance: 50}, nil)
				wr.On("UpdateWalletBalanceTx", mock.Anything, mock.Anything, "user1", 70.0).Return(nil)
				wr.On("UpdateWalletBalanceTx", mock.Anything, mock.Anything, "user2", 80.0).Return(nil)
				tr.On("CreateTransactionTx", mock.Anything, mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil).Twice()
			},
		},
		{
			name:        "insufficient funds",
			fromBalance: 10,
			toBalance:   50,
			amount:      30,
			fromUserID:  "user1",
			toUserID:    "user2",
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				// Set up pgxmock expectations for transaction flow
				db.ExpectBegin()
				// Note: When we return an error from business logic, the defer function will call Rollback
				// But since we're returning an error, the transaction should be rolled back
				db.ExpectRollback()

				// Set up repository mocks
				wr.On("GetWalletByUserIDTx", mock.Anything, mock.Anything, "user1").Return(&models.Wallet{Balance: 10}, nil)
				wr.On("GetWalletByUserIDTx", mock.Anything, mock.Anything, "user2").Return(&models.Wallet{Balance: 50}, nil)
			},
			expectedError: "insufficient balance",
		},
		{
			name:        "self transfer",
			fromBalance: 100,
			toBalance:   50,
			amount:      30,
			fromUserID:  "user1",
			toUserID:    "user1",
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				// No database calls expected for self transfer
			},
			expectedError: "cannot self transfer",
		},
		{
			name:        "zero amount",
			fromBalance: 100,
			toBalance:   50,
			amount:      0,
			fromUserID:  "user1",
			toUserID:    "user2",
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				// No database calls expected for validation error
			},
			expectedError: "amount must be positive",
		},
		{
			name:        "database connection failure",
			fromBalance: 100,
			toBalance:   50,
			amount:      30,
			fromUserID:  "user1",
			toUserID:    "user2",
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				// pgxmock can simulate connection failures
				db.ExpectBegin().WillReturnError(errors.New("connection refused"))
			},
			expectedError: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWalletRepo, mockTxRepo, mockDB, err := setupMocks()
			assert.NoError(t, err)
			defer mockDB.Close()

			tt.setupMocks(mockWalletRepo, mockTxRepo, mockDB)

			// Create the real service with mocked dependencies
			service := NewWalletService(mockWalletRepo, mockTxRepo, mockDB)

			ctx := context.Background()
			err = service.Transfer(ctx, tt.fromUserID, tt.toUserID, tt.amount)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockWalletRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestWalletService_Deposit(t *testing.T) {
	tests := []struct {
		name            string
		initialBalance  float64
		amount          float64
		setupMocks      func(*MockWalletRepo, *MockTransactionRepo, pgxmock.PgxPoolIface)
		expectedError   string
		expectedBalance float64
	}{
		{
			name:           "successful deposit",
			initialBalance: 100,
			amount:         50,
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				db.ExpectBegin()
				db.ExpectCommit()
				wr.On("GetWalletByUserIDTx", mock.Anything, mock.Anything, "user1").Return(&models.Wallet{Balance: 100}, nil)
				wr.On("UpdateWalletBalanceTx", mock.Anything, mock.Anything, "user1", 150.0).Return(nil)
				tr.On("CreateTransactionTx", mock.Anything, mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
			},
			expectedBalance: 150,
		},
		{
			name:           "zero amount",
			initialBalance: 100,
			amount:         0,
			setupMocks:     func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {},
			expectedError:  "amount must be positive",
		},
		{
			name:           "database error",
			initialBalance: 100,
			amount:         50,
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				db.ExpectBegin().WillReturnError(errors.New("connection refused"))
			},
			expectedError: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWalletRepo, mockTxRepo, mockDB, err := setupMocks()
			assert.NoError(t, err)
			defer mockDB.Close()

			tt.setupMocks(mockWalletRepo, mockTxRepo, mockDB)

			// Create the real service with mocked dependencies
			service := NewWalletService(mockWalletRepo, mockTxRepo, mockDB)

			ctx := context.Background()
			userID := "user1"

			wallet, err := service.Deposit(ctx, userID, tt.amount)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Nil(t, wallet)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wallet)
				assert.Equal(t, tt.expectedBalance, wallet.Balance)
			}

			mockWalletRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestWalletService_Withdraw(t *testing.T) {
	tests := []struct {
		name            string
		initialBalance  float64
		amount          float64
		setupMocks      func(*MockWalletRepo, *MockTransactionRepo, pgxmock.PgxPoolIface)
		expectedError   string
		expectedBalance float64
	}{
		{
			name:           "successful withdraw",
			initialBalance: 100,
			amount:         30,
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				db.ExpectBegin()
				db.ExpectCommit()
				wr.On("GetWalletByUserIDTx", mock.Anything, mock.Anything, "user1").Return(&models.Wallet{Balance: 100}, nil)
				wr.On("UpdateWalletBalanceTx", mock.Anything, mock.Anything, "user1", 70.0).Return(nil)
				tr.On("CreateTransactionTx", mock.Anything, mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
			},
			expectedBalance: 70,
		},
		{
			name:           "insufficient funds",
			initialBalance: 10,
			amount:         30,
			setupMocks: func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {
				db.ExpectBegin()
				// Remove db.ExpectRollback() because rollback is only called if the transaction is started and an error occurs after
				wr.On("GetWalletByUserIDTx", mock.Anything, mock.Anything, "user1").Return(&models.Wallet{Balance: 10}, nil)
			},
			expectedError: "insufficient balance",
		},
		{
			name:           "zero amount",
			initialBalance: 100,
			amount:         0,
			setupMocks:     func(wr *MockWalletRepo, tr *MockTransactionRepo, db pgxmock.PgxPoolIface) {},
			expectedError:  "amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWalletRepo, mockTxRepo, mockDB, err := setupMocks()
			assert.NoError(t, err)
			defer mockDB.Close()

			tt.setupMocks(mockWalletRepo, mockTxRepo, mockDB)

			// Create the real service with mocked dependencies
			service := NewWalletService(mockWalletRepo, mockTxRepo, mockDB)

			ctx := context.Background()
			userID := "user1"

			wallet, err := service.Withdraw(ctx, userID, tt.amount)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Nil(t, wallet)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wallet)
				assert.Equal(t, tt.expectedBalance, wallet.Balance)
			}

			mockWalletRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		expectedError string
	}{
		{"zero amount", 0, "amount must be positive"},
		{"negative amount", -10, "amount must be positive"},
		{"extremely small amount", 0.0001, "amount must be at least 0.01"},
		{"small amount below minimum", 0.009, "amount must be at least 0.01"},
		{"exactly minimum amount", 0.01, ""},
		{"slightly above minimum", 0.011, ""},
		{"extremely large amount", 1e20, "amount exceeds maximum limit"},
		{"NaN amount", math.NaN(), "amount cannot be NaN or infinity"},
		{"positive infinity", math.Inf(1), "amount cannot be NaN or infinity"},
		{"negative infinity", math.Inf(-1), "amount cannot be NaN or infinity"},
		{"valid amount", 100, ""},
		{"small valid amount", 0.5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAmount(tt.amount)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
