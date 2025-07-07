package repositories

import (
	"context"
	"walletapp/internal/db"
	"walletapp/internal/models"
)

func GetAllUsers(ctx context.Context) ([]models.User, error) {
	rows, err := db.DB.Query(ctx, "SELECT id, username, first_name, last_name, email, password, created_at, updated_at FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := db.DB.QueryRow(ctx, "SELECT id, username, first_name, last_name, email, password, created_at, updated_at FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	var user models.User
	err := db.DB.QueryRow(ctx, `
        INSERT INTO users (username, first_name, last_name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
        RETURNING id, username, first_name, last_name, email, password, created_at, updated_at
    `,
		req.Username, req.FirstName, req.LastName, req.Email, req.Password,
	).Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
