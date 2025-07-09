package services

import (
	"context"
	"errors"
	"math"
	"walletapp/internal/models"

	"github.com/jackc/pgx/v5"
)

// Maximum amount of money that can be transferred or deposited/withdrawn
const MAX_AMOUNT = 1000000

// Minimum amount of money that can be transferred or deposited/withdrawn
const MIN_AMOUNT = 0.01

// Interfaces for dependency injection
type WalletRepo interface {
	GetWalletByUserID(ctx context.Context, userID string) (*models.Wallet, error)
	GetWalletByUserIDTx(ctx context.Context, tx pgx.Tx, userID string) (*models.Wallet, error)
	UpdateWalletBalanceTx(ctx context.Context, tx pgx.Tx, userID string, newBalance float64) error
}

type TransactionRepo interface {
	CreateTransactionTx(ctx context.Context, tx pgx.Tx, t *models.Transaction) error
}

type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// WalletService holds the business logic for wallet operations
type WalletService struct {
	walletRepo      WalletRepo
	transactionRepo TransactionRepo
	db              DB
}

// NewWalletService creates a new WalletService with the given dependencies
func NewWalletService(walletRepo WalletRepo, transactionRepo TransactionRepo, db DB) *WalletService {
	return &WalletService{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		db:              db,
	}
}

// GetWallet retrieves a wallet by user ID
func (s *WalletService) GetWallet(ctx context.Context, userID string) (*models.Wallet, error) {
	return s.walletRepo.GetWalletByUserID(ctx, userID)
}

// Transfer transfers money from one user to another
func (s *WalletService) Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) (err error) {
	if err := ValidateAmount(amount); err != nil {
		return err
	}

	if fromUserID == toUserID {
		return errors.New("cannot self transfer")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	fromWallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, fromUserID)
	if err != nil {
		return err
	}
	toWallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, toUserID)
	if err != nil {
		return err
	}
	if fromWallet.Balance < amount {
		return errors.New("insufficient balance")
	}

	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, fromUserID, fromWallet.Balance-amount)
	if err != nil {
		return err
	}
	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, toUserID, toWallet.Balance+amount)
	if err != nil {
		return err
	}

	// Record transactions
	_ = s.transactionRepo.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID:      fromWallet.ID,
		Type:          models.TransactionTypeTransferOut,
		Amount:        amount,
		RelatedUserID: &toUserID,
	})
	_ = s.transactionRepo.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID:      toWallet.ID,
		Type:          models.TransactionTypeTransferIn,
		Amount:        amount,
		RelatedUserID: &fromUserID,
	})

	return nil
}

// Deposit adds money to a user's wallet
func (s *WalletService) Deposit(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	if err := ValidateAmount(amount); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	wallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	newBalance := wallet.Balance + amount
	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, userID, newBalance)
	if err != nil {
		return nil, err
	}
	wallet.Balance = newBalance
	_ = s.transactionRepo.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID: wallet.ID,
		Type:     models.TransactionTypeDeposit,
		Amount:   amount,
	})
	return wallet, nil
}

// Withdraw removes money from a user's wallet
func (s *WalletService) Withdraw(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	if err := ValidateAmount(amount); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	wallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if wallet.Balance < amount {
		return nil, errors.New("insufficient balance")
	}
	newBalance := wallet.Balance - amount
	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, userID, newBalance)
	if err != nil {
		return nil, err
	}
	wallet.Balance = newBalance
	_ = s.transactionRepo.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID: wallet.ID,
		Type:     models.TransactionTypeWithdraw,
		Amount:   amount,
	})
	return wallet, nil
}

// ValidateAmount validates that an amount is within acceptable bounds
func ValidateAmount(amount float64) error {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return errors.New("amount cannot be NaN or infinity")
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	if amount < MIN_AMOUNT {
		return errors.New("amount must be at least 0.01")
	}
	if amount > MAX_AMOUNT {
		return errors.New("amount exceeds maximum limit")
	}
	return nil
}

// Legacy functions for backward compatibility (will be removed after refactor)
// These now delegate to a default service instance

var defaultService *WalletService

func init() {
	// This will be set up in main.go with real implementations
}

// SetDefaultService sets the default service instance for legacy functions
func SetDefaultService(service *WalletService) {
	defaultService = service
}

func GetWallet(ctx context.Context, userID string) (*models.Wallet, error) {
	if defaultService == nil {
		panic("default service not initialized - call SetDefaultService first")
	}
	return defaultService.GetWallet(ctx, userID)
}

func Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) error {
	if defaultService == nil {
		panic("default service not initialized - call SetDefaultService first")
	}
	return defaultService.Transfer(ctx, fromUserID, toUserID, amount)
}

func Deposit(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	if defaultService == nil {
		panic("default service not initialized - call SetDefaultService first")
	}
	return defaultService.Deposit(ctx, userID, amount)
}

func Withdraw(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	if defaultService == nil {
		panic("default service not initialized - call SetDefaultService first")
	}
	return defaultService.Withdraw(ctx, userID, amount)
}
