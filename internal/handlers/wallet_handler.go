package handlers

import (
	"context"
	"net/http"
	"walletapp/internal/models"
	"walletapp/internal/services"

	"github.com/gin-gonic/gin"
)

// Deposit godoc
// @Summary      Deposit money
// @Description  Deposit money into user's wallet
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        amount body models.AmountRequest true "Amount to deposit"
// @Success      200 {object} models.Wallet
// @Failure      400 {object} models.ErrorResponse
// @Router       /wallets/{user_id}/deposit [post]
func Deposit(c *gin.Context) {
	userID := c.Param("user_id")
	var req models.AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	ctx := context.Background()
	wallet, err := services.Deposit(ctx, userID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

// Withdraw godoc
// @Summary      Withdraw money
// @Description  Withdraw money from user's wallet
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        amount body models.AmountRequest true "Amount to withdraw"
// @Success      200 {object} models.Wallet
// @Failure      400 {object} models.ErrorResponse
// @Router       /wallets/{user_id}/withdraw [post]
func Withdraw(c *gin.Context) {
	userID := c.Param("user_id")
	var req models.AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	ctx := context.Background()
	wallet, err := services.Withdraw(ctx, userID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

// GetBalance godoc
// @Summary      Get wallet balance
// @Description  Get user's wallet balance
// @Tags         wallet
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} models.Wallet
// @Failure      404 {object} models.ErrorResponse
// @Router       /wallets/{user_id}/balance [get]
func GetBalance(c *gin.Context) {
	userID := c.Param("user_id")
	ctx := context.Background()
	wallet, err := services.GetWallet(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, wallet)
}
