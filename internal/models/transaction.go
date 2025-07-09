package models

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeDeposit     TransactionType = "DEPOSIT"
	TransactionTypeWithdraw    TransactionType = "WITHDRAW"
	TransactionTypeTransferIn  TransactionType = "TRANSFER_IN"
	TransactionTypeTransferOut TransactionType = "TRANSFER_OUT"
)

type Transaction struct {
	ID            uuid.UUID       `json:"id"`
	WalletID      uuid.UUID       `json:"wallet_id"`
	Type          TransactionType `json:"type"`
	Amount        float64         `json:"amount"`
	RelatedUserID *string         `json:"related_user_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
