package services

import (
	"context"
	"walletapp/internal/db"
	"walletapp/internal/models"
	"walletapp/internal/repositories"

	"github.com/jackc/pgx/v5"
)

// WalletRepoImpl implements WalletRepo interface
type WalletRepoImpl struct{}

// NewWalletRepoImpl creates a new WalletRepoImpl
func NewWalletRepoImpl() *WalletRepoImpl {
	return &WalletRepoImpl{}
}

// GetWalletByUserID retrieves a wallet by user ID
func (r *WalletRepoImpl) GetWalletByUserID(ctx context.Context, userID string) (*models.Wallet, error) {
	return repositories.GetWalletByUserID(ctx, userID)
}

// GetWalletByUserIDTx retrieves a wallet by user ID within a transaction
func (r *WalletRepoImpl) GetWalletByUserIDTx(ctx context.Context, tx pgx.Tx, userID string) (*models.Wallet, error) {
	return repositories.GetWalletByUserIDTx(ctx, tx, userID)
}

// UpdateWalletBalanceTx updates a wallet balance within a transaction
func (r *WalletRepoImpl) UpdateWalletBalanceTx(ctx context.Context, tx pgx.Tx, userID string, newBalance float64) error {
	return repositories.UpdateWalletBalanceTx(ctx, tx, userID, newBalance)
}

// TransactionRepoImpl implements TransactionRepo interface
type TransactionRepoImpl struct{}

// NewTransactionRepoImpl creates a new TransactionRepoImpl
func NewTransactionRepoImpl() *TransactionRepoImpl {
	return &TransactionRepoImpl{}
}

// CreateTransactionTx creates a transaction record within a transaction
func (r *TransactionRepoImpl) CreateTransactionTx(ctx context.Context, tx pgx.Tx, t *models.Transaction) error {
	return repositories.CreateTransactionTx(ctx, tx, t)
}

// DBImpl implements DB interface
type DBImpl struct{}

// NewDBImpl creates a new DBImpl
func NewDBImpl() *DBImpl {
	return &DBImpl{}
}

// Begin starts a new transaction
func (d *DBImpl) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.GetPool().Begin(ctx)
}
