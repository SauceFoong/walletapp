package services

import (
	"context"
	"errors"
	"walletapp/internal/db"
	"walletapp/internal/models"
	"walletapp/internal/repositories"
)

func GetWallet(ctx context.Context, userID string) (*models.Wallet, error) {
	return repositories.GetWalletByUserID(ctx, userID)
}

func Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	if fromUserID == toUserID {
		return errors.New("cannot transfer to self")
	}

	pool := db.GetPool()
	tx, err := pool.Begin(ctx)
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

	fromWallet, err := repositories.GetWalletByUserIDTx(ctx, tx, fromUserID)
	if err != nil {
		return err
	}
	toWallet, err := repositories.GetWalletByUserIDTx(ctx, tx, toUserID)
	if err != nil {
		return err
	}
	if fromWallet.Balance < amount {
		return errors.New("insufficient balance")
	}

	err = repositories.UpdateWalletBalanceTx(ctx, tx, fromUserID, fromWallet.Balance-amount)
	if err != nil {
		return err
	}
	err = repositories.UpdateWalletBalanceTx(ctx, tx, toUserID, toWallet.Balance+amount)
	if err != nil {
		return err
	}

	// Record transactions
	_ = repositories.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID:      fromWallet.ID,
		Type:          models.TransactionTypeTransferOut,
		Amount:        amount,
		RelatedUserID: &toUserID,
	})
	_ = repositories.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID:      toWallet.ID,
		Type:          models.TransactionTypeTransferIn,
		Amount:        amount,
		RelatedUserID: &fromUserID,
	})

	return nil
}

func Deposit(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	pool := db.GetPool()
	tx, err := pool.Begin(ctx)
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

	wallet, err := repositories.GetWalletByUserIDTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	newBalance := wallet.Balance + amount
	err = repositories.UpdateWalletBalanceTx(ctx, tx, userID, newBalance)
	if err != nil {
		return nil, err
	}
	wallet.Balance = newBalance
	_ = repositories.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID: wallet.ID,
		Type:     models.TransactionTypeDeposit,
		Amount:   amount,
	})
	return wallet, nil
}

func Withdraw(ctx context.Context, userID string, amount float64) (*models.Wallet, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	pool := db.GetPool()
	tx, err := pool.Begin(ctx)
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

	wallet, err := repositories.GetWalletByUserIDTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if wallet.Balance < amount {
		return nil, errors.New("insufficient balance")
	}
	newBalance := wallet.Balance - amount
	err = repositories.UpdateWalletBalanceTx(ctx, tx, userID, newBalance)
	if err != nil {
		return nil, err
	}
	wallet.Balance = newBalance
	_ = repositories.CreateTransactionTx(ctx, tx, &models.Transaction{
		WalletID: wallet.ID,
		Type:     models.TransactionTypeWithdraw,
		Amount:   amount,
	})
	return wallet, nil
}
