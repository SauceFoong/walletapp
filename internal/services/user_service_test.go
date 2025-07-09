package services

import (
	"context"
	"errors"
	"testing"
	"walletapp/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// UserRepo interface for dependency injection
// (needed for UserService and MockUserRepo)
type UserRepo interface {
	IsEmailExists(ctx context.Context, email string) (bool, error)
	IsUsernameExists(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error)
}

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) IsEmailExists(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}
func (m *MockUserRepo) IsUsernameExists(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}
func (m *MockUserRepo) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UserService struct with dependency injection
type UserService struct {
	repo UserRepo
}

func NewUserService(repo UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	if exists, _ := s.repo.IsEmailExists(ctx, req.Email); exists {
		return nil, errors.New("email already in use")
	}
	if exists, _ := s.repo.IsUsernameExists(ctx, req.Username); exists {
		return nil, errors.New("username already in use")
	}
	user, err := s.repo.CreateUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func TestUserService_CreateUser(t *testing.T) {
	cases := []struct {
		name         string
		setupMock    func(*MockUserRepo)
		req          *models.CreateUserRequest
		wantErr      string
		wantUsername string
	}{
		{
			name: "success",
			setupMock: func(m *MockUserRepo) {
				m.On("IsEmailExists", mock.Anything, "test@example.com").Return(false, nil)
				m.On("IsUsernameExists", mock.Anything, "testuser").Return(false, nil)
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.CreateUserRequest")).Return(&models.User{
					ID:        uuid.New(),
					Username:  "testuser",
					FirstName: "Test",
					LastName:  "User",
					Email:     "test@example.com",
					Password:  "password",
				}, nil)
			},
			req:          &models.CreateUserRequest{Username: "testuser", FirstName: "Test", LastName: "User", Email: "test@example.com", Password: "password"},
			wantErr:      "",
			wantUsername: "testuser",
		},
		{
			name: "duplicate email",
			setupMock: func(m *MockUserRepo) {
				m.On("IsEmailExists", mock.Anything, "test@example.com").Return(true, nil)
			},
			req:     &models.CreateUserRequest{Username: "testuser", FirstName: "Test", LastName: "User", Email: "test@example.com", Password: "password"},
			wantErr: "email already in use",
		},
		{
			name: "duplicate username",
			setupMock: func(m *MockUserRepo) {
				m.On("IsEmailExists", mock.Anything, "test@example.com").Return(false, nil)
				m.On("IsUsernameExists", mock.Anything, "testuser").Return(true, nil)
			},
			req:     &models.CreateUserRequest{Username: "testuser", FirstName: "Test", LastName: "User", Email: "test@example.com", Password: "password"},
			wantErr: "username already in use",
		},
		{
			name: "user repo error",
			setupMock: func(m *MockUserRepo) {
				m.On("IsEmailExists", mock.Anything, "test@example.com").Return(false, nil)
				m.On("IsUsernameExists", mock.Anything, "testuser").Return(false, nil)
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.CreateUserRequest")).Return(nil, errors.New("db error"))
			},
			req:     &models.CreateUserRequest{Username: "testuser", FirstName: "Test", LastName: "User", Email: "test@example.com", Password: "password"},
			wantErr: "db error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockUserRepo)
			if tc.setupMock != nil {
				tc.setupMock(mockRepo)
			}
			service := NewUserService(mockRepo)
			user, err := service.CreateUser(context.Background(), tc.req)
			if tc.wantErr == "" && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			if tc.wantUsername != "" && user != nil {
				assert.Equal(t, tc.wantUsername, user.Username)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
