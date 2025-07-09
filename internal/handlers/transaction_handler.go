package handlers

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"walletapp/internal/logger"
	"walletapp/internal/models"
	"walletapp/internal/repositories"
	"walletapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type TransferRequest struct {
	FromUserID string  `json:"from_user_id"`
	ToUserID   string  `json:"to_user_id"`
	Amount     float64 `json:"amount"`
}

// Transfer godoc
// @Summary      Transfer money
// @Description  Transfer money from one user to another
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Param        transfer body TransferRequest true "Transfer details"
// @Success      200 {object} models.SuccessResponse
// @Failure      400 {object} models.ErrorResponse
// @Router       /wallets/transfer [post]
func Transfer(c *gin.Context) {
	log := logger.WithField("operation", "api_transfer")

	log.Info("Transfer request received")

	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithField("error", err.Error()).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	log.WithFields(logrus.Fields{
		"from_user_id": req.FromUserID,
		"to_user_id":   req.ToUserID,
		"amount":       req.Amount,
	}).Debug("Processing transfer request")

	// Validate user IDs
	if _, err := uuid.Parse(req.FromUserID); err != nil {
		log.WithField("from_user_id", req.FromUserID).Warn("Invalid from_user_id format")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid from_user_id format"})
		return
	}
	if _, err := uuid.Parse(req.ToUserID); err != nil {
		log.WithField("to_user_id", req.ToUserID).Warn("Invalid to_user_id format")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid to_user_id format"})
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		log.WithField("amount", req.Amount).Warn("Invalid amount: must be positive")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "amount must be positive"})
		return
	}
	if req.Amount > 1000000 { // $1M limit
		log.WithField("amount", req.Amount).Warn("Amount exceeds maximum limit")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "amount exceeds maximum limit"})
		return
	}
	if math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) {
		log.WithField("amount", req.Amount).Warn("Invalid amount: NaN or infinity")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid amount"})
		return
	}

	// Check if users exist
	ctx := context.Background()
	_, err := repositories.GetUserByID(ctx, req.FromUserID)
	if err != nil {
		log.WithField("from_user_id", req.FromUserID).Warn("From user not found")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "from_user_id not found"})
		return
	}
	_, err = repositories.GetUserByID(ctx, req.ToUserID)
	if err != nil {
		log.WithField("to_user_id", req.ToUserID).Warn("To user not found")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "to_user_id not found"})
		return
	}

	err = services.Transfer(ctx, req.FromUserID, req.ToUserID, req.Amount)
	if err != nil {
		log.WithField("error", err.Error()).Error("Transfer operation failed")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	log.Info("Transfer completed successfully")
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "transfer successful"})
}

// GetTransactionHistory godoc
// @Summary      Get transaction history
// @Description  Get user's wallet transaction history with pagination
// @Tags         wallet
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        limit query int false "Number of transactions to return (default: 50, max: 100)"
// @Param        offset query int false "Number of transactions to skip (default: 0)"
// @Success      200 {array} models.Transaction
// @Failure      404 {object} models.ErrorResponse
// @Router       /wallets/{user_id}/transactions [get]
func GetTransactionHistory(c *gin.Context) {
	userID := c.Param("user_id")
	log := logger.WithUser(userID).WithField("operation", "api_get_transaction_history")

	log.Info("Transaction history request received")

	// Validate user ID format
	if _, err := uuid.Parse(userID); err != nil {
		log.WithField("user_id", userID).Warn("Invalid user_id format")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid user_id format"})
		return
	}

	// Parse pagination parameters
	limit := 50 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		} else {
			log.WithField("limit", limitStr).Warn("Invalid limit parameter")
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "limit must be between 1 and 100"})
			return
		}
	}

	offset := 0 // default
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		} else {
			log.WithField("offset", offsetStr).Warn("Invalid offset parameter")
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "offset must be non-negative"})
			return
		}
	}

	log.WithFields(logrus.Fields{
		"limit":  limit,
		"offset": offset,
	}).Debug("Pagination parameters")

	ctx := context.Background()
	wallet, err := services.GetWallet(ctx, userID)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get wallet")
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}

	txs, err := repositories.GetTransactionsByWalletID(ctx, wallet.ID.String())
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get transactions")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start >= len(txs) {
		txs = []models.Transaction{}
	} else if end > len(txs) {
		txs = txs[start:]
	} else {
		txs = txs[start:end]
	}

	log.WithField("transaction_count", len(txs)).Info("Transaction history retrieved successfully")
	c.JSON(http.StatusOK, txs)
}
