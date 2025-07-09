package handlers

import (
	"net/http"
	"walletapp/internal/logger"
	"walletapp/internal/models"
	"walletapp/internal/services"

	"github.com/gin-gonic/gin"
)

// Deposit handles wallet deposit requests
func Deposit(c *gin.Context) {
	userID := c.Param("user_id")
	log := logger.WithUser(userID).WithField("operation", "api_deposit")

	log.Info("Deposit request received")

	var req models.AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithField("error", err.Error()).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid request body",
		})
		return
	}

	log.WithField("amount", req.Amount).Debug("Processing deposit request")

	wallet, err := services.Deposit(c.Request.Context(), userID, req.Amount)
	if err != nil {
		log.WithField("error", err.Error()).Error("Deposit operation failed")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	log.WithField("new_balance", wallet.Balance).Info("Deposit completed successfully")
	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Deposit successful",
	})
}

// Withdraw handles wallet withdrawal requests
func Withdraw(c *gin.Context) {
	userID := c.Param("user_id")
	log := logger.WithUser(userID).WithField("operation", "api_withdraw")

	log.Info("Withdrawal request received")

	var req models.AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithField("error", err.Error()).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid request body",
		})
		return
	}

	log.WithField("amount", req.Amount).Debug("Processing withdrawal request")

	wallet, err := services.Withdraw(c.Request.Context(), userID, req.Amount)
	if err != nil {
		log.WithField("error", err.Error()).Error("Withdrawal operation failed")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	log.WithField("new_balance", wallet.Balance).Info("Withdrawal completed successfully")
	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Withdrawal successful",
	})
}

// GetBalance handles balance inquiry requests
func GetBalance(c *gin.Context) {
	userID := c.Param("user_id")
	log := logger.WithUser(userID).WithField("operation", "api_get_balance")

	log.Info("Balance inquiry request received")

	wallet, err := services.GetWallet(c.Request.Context(), userID)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get wallet balance")
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Wallet not found",
		})
		return
	}

	log.WithField("balance", wallet.Balance).Info("Balance retrieved successfully")
	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Balance retrieved successfully",
	})
}
