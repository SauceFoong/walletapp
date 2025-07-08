package services

import (
	"context"
	"walletapp/internal/models"
	"walletapp/internal/repositories"
)

func CreateUserWithWallet(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	user, err := repositories.CreateUser(ctx, req)
	if err != nil {
		return nil, err
	}
	// Create wallet for the new user
	_, err = repositories.CreateWallet(ctx, user.ID.String())
	if err != nil {
		return nil, err
	}
	return user, nil
}
