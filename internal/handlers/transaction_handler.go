package handlers

import (
	"context"
	"net/http"
	"walletapp/internal/models"
	"walletapp/internal/repositories"
	"walletapp/internal/services"

	"github.com/gin-gonic/gin"
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
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	ctx := context.Background()
	err := services.Transfer(ctx, req.FromUserID, req.ToUserID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "transfer successful"})
}

// GetTransactionHistory godoc
// @Summary      Get transaction history
// @Description  Get user's wallet transaction history
// @Tags         wallet
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {array} models.Transaction
// @Failure      404 {object} models.ErrorResponse
// @Router       /wallets/{user_id}/transactions [get]
func GetTransactionHistory(c *gin.Context) {
	userID := c.Param("user_id")
	ctx := context.Background()
	wallet, err := services.GetWallet(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	txs, err := repositories.GetTransactionsByWalletID(ctx, wallet.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}
