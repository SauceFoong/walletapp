package services

import (
	"context"
	"walletapp/internal/logger"
	"walletapp/internal/models"
	"walletapp/internal/repositories"
)

func CreateUserWithWallet(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	log := logger.Get()

	log.WithFields(map[string]interface{}{
		"username": req.Username,
		"email":    req.Email,
	}).Info("Creating new user with wallet")

	user, err := repositories.CreateUser(ctx, req)
	if err != nil {
		log.WithError(err).WithFields(map[string]interface{}{
			"username": req.Username,
			"email":    req.Email,
		}).Error("Failed to create user")
		return nil, err
	}

	log.WithFields(map[string]interface{}{
		"user_id":  user.ID.String(),
		"username": user.Username,
	}).Info("User created successfully, creating wallet")

	// Create wallet for the new user
	_, err = repositories.CreateWallet(ctx, user.ID.String())
	if err != nil {
		log.WithError(err).WithFields(map[string]interface{}{
			"user_id": user.ID.String(),
		}).Error("Failed to create wallet for user")
		return nil, err
	}

	log.WithFields(map[string]interface{}{
		"user_id": user.ID.String(),
	}).Info("User and wallet created successfully")

	return user, nil
}
