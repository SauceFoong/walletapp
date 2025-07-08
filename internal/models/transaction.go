package models

import "time"

type TransactionType string

const (
	TransactionTypeDeposit     TransactionType = "DEPOSIT"
	TransactionTypeWithdraw    TransactionType = "WITHDRAW"
	TransactionTypeTransferIn  TransactionType = "TRANSFER_IN"
	TransactionTypeTransferOut TransactionType = "TRANSFER_OUT"
)

type Transaction struct {
	ID            string          `json:"id"`
	WalletID      string          `json:"wallet_id"`
	Type          TransactionType `json:"type"`
	Amount        float64         `json:"amount"`
	RelatedUserID *string         `json:"related_user_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
