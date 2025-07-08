package repositories

import (
	"context"
	"walletapp/internal/db"
	"walletapp/internal/models"

	"github.com/jackc/pgx/v5"
)

func GetWalletByUserID(ctx context.Context, userID string) (*models.Wallet, error) {
	var w models.Wallet
	err := db.DB.QueryRow(ctx, "SELECT id, user_id, balance, created_at, updated_at FROM wallets WHERE user_id = $1", userID).
		Scan(&w.ID, &w.UserID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func GetWalletByUserIDTx(ctx context.Context, tx pgx.Tx, userID string) (*models.Wallet, error) {
	var w models.Wallet
	err := tx.QueryRow(ctx, "SELECT id, user_id, balance, created_at, updated_at FROM wallets WHERE user_id = $1", userID).
		Scan(&w.ID, &w.UserID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func CreateWallet(ctx context.Context, userID string) (*models.Wallet, error) {
	var w models.Wallet
	err := db.DB.QueryRow(ctx, `
        INSERT INTO wallets (user_id, balance, created_at, updated_at)
        VALUES ($1, 0, NOW(), NOW())
        RETURNING id, user_id, balance, created_at, updated_at
    `, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func UpdateWalletBalanceTx(ctx context.Context, tx pgx.Tx, userID string, newBalance float64) error {
	_, err := tx.Exec(ctx, "UPDATE wallets SET balance = $1, updated_at = NOW() WHERE user_id = $2", newBalance, userID)
	return err
}
