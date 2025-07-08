package repositories

import (
	"context"
	"walletapp/internal/db"
	"walletapp/internal/models"

	"github.com/jackc/pgx/v5"
)

func CreateTransactionTx(ctx context.Context, tx pgx.Tx, t *models.Transaction) error {
	return tx.QueryRow(ctx, `
        INSERT INTO transactions (wallet_id, type, amount, related_user_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
        RETURNING id, created_at, updated_at
    `, t.WalletID, t.Type, t.Amount, t.RelatedUserID).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func GetTransactionsByWalletID(ctx context.Context, walletID string) ([]models.Transaction, error) {
	rows, err := db.DB.Query(ctx, `
        SELECT id, wallet_id, type, amount, related_user_id, created_at, updated_at
        FROM transactions
        WHERE wallet_id = $1
        ORDER BY created_at DESC
    `, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(&tx.ID, &tx.WalletID, &tx.Type, &tx.Amount, &tx.RelatedUserID, &tx.CreatedAt, &tx.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}
