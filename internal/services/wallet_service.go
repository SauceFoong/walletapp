package services

import (
	"context"
	"errors"
	"math"
	"walletapp/internal/logger"
	"walletapp/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
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
	log := logger.WithUser(userID).WithField("operation", "get_wallet")
	log.Info("Getting wallet for user")

	wallet, err := s.walletRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get wallet")
		return nil, err
	}

	log.WithField("balance", wallet.Balance).Info("Successfully retrieved wallet")
	return wallet, nil
}

// Transfer transfers money from one user to another
func (s *WalletService) Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) (err error) {
	log := logger.WithFields(logrus.Fields{
		"from_user_id": fromUserID,
		"to_user_id":   toUserID,
		"amount":       amount,
		"operation":    "transfer",
	})

	log.Info("Starting transfer operation")

	if err := ValidateAmount(amount); err != nil {
		log.WithField("validation_error", err.Error()).Warn("Transfer validation failed")
		return err
	}

	if fromUserID == toUserID {
		log.Warn("Self-transfer attempt blocked")
		return errors.New("cannot self transfer")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to begin transaction")
		return err
	}
	defer func() {
		if err != nil {
			log.WithField("error", err.Error()).Error("Transfer failed, rolling back transaction")
			tx.Rollback(ctx)
		} else {
			log.Info("Transfer successful, committing transaction")
			tx.Commit(ctx)
		}
	}()

	fromWallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, fromUserID)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get from user wallet")
		return err
	}

	toWallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, toUserID)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get to user wallet")
		return err
	}

	if fromWallet.Balance < amount {
		log.WithFields(logrus.Fields{
			"from_balance": fromWallet.Balance,
			"amount":       amount,
		}).Warn("Insufficient balance for transfer")
		return errors.New("insufficient balance")
	}

	log.WithFields(logrus.Fields{
		"from_balance_before": fromWallet.Balance,
		"to_balance_before":   toWallet.Balance,
	}).Debug("Updating wallet balances")

	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, fromUserID, fromWallet.Balance-amount)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to update from user balance")
		return err
	}

	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, toUserID, toWallet.Balance+amount)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to update to user balance")
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

	log.WithFields(logrus.Fields{
		"from_balance_after": fromWallet.Balance - amount,
		"to_balance_after":   toWallet.Balance + amount,
	}).Info("Transfer completed successfully")

	return nil
}

// Deposit adds money to a user's wallet
func (s *WalletService) Deposit(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	log := logger.WithUser(userID).WithFields(logrus.Fields{
		"operation": "deposit",
		"amount":    amount,
	})
	log.Info("Starting deposit operation")

	if err := ValidateAmount(amount); err != nil {
		log.WithField("validation_error", err.Error()).Warn("Deposit validation failed")
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to begin transaction")
		return nil, err
	}
	defer func() {
		if err != nil {
			log.WithField("error", err.Error()).Error("Deposit failed, rolling back transaction")
			tx.Rollback(ctx)
		} else {
			log.Info("Deposit successful, committing transaction")
			tx.Commit(ctx)
		}
	}()

	wallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, userID)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get wallet for deposit")
		return nil, err
	}

	log.WithField("balance_before", wallet.Balance).Debug("Processing deposit")

	newBalance := wallet.Balance + amount
	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, userID, newBalance)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to update wallet balance")
		return nil, err
	}

	wallet.Balance = newBalance
	_ = s.transactionRepo.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID: wallet.ID,
		Type:     models.TransactionTypeDeposit,
		Amount:   amount,
	})

	log.WithFields(logrus.Fields{
		"balance_before": wallet.Balance - amount,
		"balance_after":  wallet.Balance,
		"deposit_amount": amount,
	}).Info("Deposit completed successfully")

	return wallet, nil
}

// Withdraw removes money from a user's wallet
func (s *WalletService) Withdraw(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	log := logger.WithUser(userID).WithFields(logrus.Fields{
		"operation": "withdraw",
		"amount":    amount,
	})
	log.Info("Starting withdrawal operation")

	if err := ValidateAmount(amount); err != nil {
		log.WithField("validation_error", err.Error()).Warn("Withdrawal validation failed")
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to begin transaction")
		return nil, err
	}
	defer func() {
		if err != nil {
			log.WithField("error", err.Error()).Error("Withdrawal failed, rolling back transaction")
			tx.Rollback(ctx)
		} else {
			log.Info("Withdrawal successful, committing transaction")
			tx.Commit(ctx)
		}
	}()

	wallet, err := s.walletRepo.GetWalletByUserIDTx(ctx, tx, userID)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get wallet for withdrawal")
		return nil, err
	}

	log.WithField("balance_before", wallet.Balance).Debug("Processing withdrawal")

	if wallet.Balance < amount {
		log.WithFields(logrus.Fields{
			"balance": wallet.Balance,
			"amount":  amount,
		}).Warn("Insufficient balance for withdrawal")
		return nil, errors.New("insufficient balance")
	}

	newBalance := wallet.Balance - amount
	err = s.walletRepo.UpdateWalletBalanceTx(ctx, tx, userID, newBalance)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to update wallet balance")
		return nil, err
	}

	wallet.Balance = newBalance
	_ = s.transactionRepo.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID: wallet.ID,
		Type:     models.TransactionTypeWithdraw,
		Amount:   amount,
	})

	log.WithFields(logrus.Fields{
		"balance_before":  wallet.Balance + amount,
		"balance_after":   wallet.Balance,
		"withdraw_amount": amount,
	}).Info("Withdrawal completed successfully")

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
