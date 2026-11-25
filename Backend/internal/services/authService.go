package services

import (
	"backend/internal/models"
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

// UserStore defines the database access contract required by AuthService.
type UserStore interface {
	FindByLogin(ctx context.Context, login string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	FindByGoogleId(ctx context.Context, googleID string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
}

type AuthService struct {
	userRepo UserStore
}

func NewAuthService(userRepo UserStore) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	login := strings.TrimSpace(req.Login)
	if login == "" || req.Password == "" {
		return nil, errors.New("username/email and password are required")
	}

	user, err := s.userRepo.FindByLogin(ctx, login)
	if err != nil || user == nil {
		return nil, errors.New("invalid username/email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid username/email or password")
	}
	return &models.AuthResponse{
		User: models.UserSummary{
			ID:       user.ID.Hex(),
			Username: user.Username,
			Email:    user.Email,
		},
		Message: "Login successful",
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	if username == "" || email == "" {
		return nil, errors.New("username and email are required")
	}
	// bcrypt truncates/errors above 72 bytes and empty passwords must be rejected
	if len(req.Password) < 8 || len(req.Password) > 72 {
		return nil, errors.New("password must be between 8 and 72 characters")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}
	newUser := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedBytes),
		CreatedAt:    time.Now().UTC(),
	}
	if err = s.userRepo.CreateUser(ctx, newUser); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errors.New("username or email already taken")
		}
		return nil, errors.New("failed to create user account")
	}
	return &models.AuthResponse{
		User: models.UserSummary{
			ID:       newUser.ID.Hex(),
			Username: newUser.Username,
			Email:    newUser.Email,
		},
		Message: "Registration successful",
	}, nil
}

func (s *AuthService) ProcessGoogleUser(ctx context.Context, googleID, email, name string) (*models.User, error) {
	user, err := s.userRepo.FindByGoogleId(ctx, googleID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	existingUser, err := s.userRepo.FindByLogin(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return existingUser, nil
	}

	newUser := &models.User{
		Username:  name,
		Email:     email,
		GoogleID:  googleID,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id string) (*models.UserSummary, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	return &models.UserSummary{
		ID:       user.ID.Hex(),
		Username: user.Username,
		Email:    user.Email,
	}, nil
}
