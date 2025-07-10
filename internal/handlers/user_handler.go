package handlers

import (
	"context"
	"net/http"
	"walletapp/internal/logger"
	"walletapp/internal/models"
	"walletapp/internal/repositories"
	"walletapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers godoc
// @Summary      List all users
// @Description  get all users
// @Tags         users
// @Produce      json
// @Success      200  {object}  models.SuccessResponse{data=[]models.UserResponse}
// @Failure      500  {object}  models.ErrorResponse
// @Router       /v1/users [get]
func GetUsers(c *gin.Context) {
	log := logger.Get()
	log.Info("Getting all users")

	ctx := context.Background()
	users, err := repositories.GetAllUsers(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get all users")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	log.WithField("count", len(users)).Info("Retrieved users successfully")

	var resp []models.UserResponse
	for _, u := range users {
		// Get wallet for the user
		wallet, err := repositories.GetWalletByUserID(ctx, u.ID.String())
		if err != nil {
			log.WithError(err).WithField("user_id", u.ID.String()).Warn("Failed to get wallet for user, including user with nil wallet")
			// Include user with nil wallet
			resp = append(resp, toUserResponse(&u, nil))
			continue
		}

		resp = append(resp, toUserResponse(&u, wallet))
	}

	// Return success response
	c.JSON(http.StatusOK, models.SuccessResponse{
		Code:    200,
		Message: "Users retrieved successfully",
		Data:    resp,
	})
}

// GetUserByID godoc
// @Summary      Get user by ID
// @Description  get user by ID
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  models.UserResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /v1/users/{id} [get]
func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	log := logger.Get().WithField("user_id", id)
	log.Info("Getting user by ID")

	ctx := context.Background()
	// Get user by ID
	user, err := repositories.GetUserByID(ctx, id)
	if err != nil {
		log.WithError(err).Error("User not found")
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}
	// Get wallet for the user
	wallet, err := repositories.GetWalletByUserID(ctx, id)
	if err != nil {
		log.WithError(err).Error("Wallet not found for user")
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Wallet not found"})
		return
	}

	log.Info("User retrieved successfully")
	// Return success response
	c.JSON(http.StatusOK, models.SuccessResponse{
		Code:    200,
		Message: "User retrieved successfully",
		Data:    toUserResponse(user, wallet),
	})
}

// CreateUser godoc
// @Summary      Create user
// @Description  create a new user, wallet will be created automatically after user creation
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user body models.CreateUserRequest true "User to create"
// @Success      201   {object}  models.SuccessResponse{data=models.UserResponse}
// @Failure      400   {object}  models.ErrorResponse
// @Failure      500   {object}  models.ErrorResponse
// @Router       /v1/users [post]
func CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().WithError(err).Error("Invalid request body for user creation")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	log := logger.Get().WithFields(map[string]interface{}{
		"username": req.Username,
		"email":    req.Email,
	})
	log.Info("Creating new user")

	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("Failed to hash password")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to hash password"})
		return
	}
	req.Password = string(hashedPassword)

	ctx := context.Background()
	user, err := services.CreateUserWithWallet(ctx, &req)
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok && err.Code == "23505" {
			// 23505 is unique_violation in Postgres
			log.WithError(err).Warn("User creation failed - email or username already exists")
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Email or username already exists"})
			return
		}

		log.WithError(err).Error("Failed to create user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	log.WithField("user_id", user.ID.String()).Info("User created successfully")
	c.JSON(http.StatusCreated, user)
}

// Helper to map User to UserResponse
func toUserResponse(u *models.User, wallet *models.Wallet) models.UserResponse {
	var walletResp *models.WalletResponse
	if wallet != nil {
		walletResp = &models.WalletResponse{
			ID:        wallet.ID.String(),
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
