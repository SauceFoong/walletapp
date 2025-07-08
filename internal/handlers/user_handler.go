package handlers

import (
	"context"
	"net/http"
	"walletapp/internal/models"
	"walletapp/internal/repositories"
	"walletapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers godoc
// @Summary      List users
// @Description  get all users
// @Tags         users
// @Produce      json
// @Success      200  {array}   models.UserResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /users [get]
func GetUsers(c *gin.Context) {
	ctx := context.Background()
	users, err := repositories.GetAllUsers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	var resp []models.UserResponse
	for _, u := range users {
		// Get wallet for the user
		wallet, err := repositories.GetWalletByUserID(ctx, u.ID.String())
		if err != nil {
			// Include user with nil wallet
			resp = append(resp, toUserResponse(&u, nil))
			continue
		}

		resp = append(resp, toUserResponse(&u, wallet))
	}
	c.JSON(http.StatusOK, resp)
}

// GetUserByID godoc
// @Summary      Get user by ID
// @Description  get user by ID
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  models.UserResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /users/{id} [get]
func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	// Get user by ID
	user, err := repositories.GetUserByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}
	// Get wallet for the user
	wallet, err := repositories.GetWalletByUserID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Wallet not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user, wallet))
}

// CreateUser godoc
// @Summary      Create user
// @Description  create a new user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user body models.CreateUserRequest true "User to create"
// @Success      201   {object}  models.User
// @Failure      400   {object}  models.ErrorResponse
// @Failure      500   {object}  models.ErrorResponse
// @Router       /users [post]
func CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to hash password"})
		return
	}
	req.Password = string(hashedPassword)

	ctx := context.Background()
	user, err := services.CreateUserWithWallet(ctx, &req)
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok && err.Code == "23505" {
			// 23505 is unique_violation in Postgres
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Email or username already exists"})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
}

// Helper to map User to UserResponse
func toUserResponse(u *models.User, wallet *models.Wallet) models.UserResponse {
	var walletResp *models.WalletResponse
	if wallet != nil {
		walletResp = &models.WalletResponse{
			ID:        wallet.ID,
			Balance:   wallet.Balance,
			CreatedAt: wallet.CreatedAt,
			UpdatedAt: wallet.UpdatedAt,
		}
	}
	return models.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		Wallet:    walletResp,
	}
}

// HashPassword hashes the password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
