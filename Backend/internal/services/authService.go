package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.FindByLogin(ctx, req.Login)
	if err != nil || user == nil {
		return nil, errors.New("invalid username/email")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid password")
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
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("encryption error")
	}
	newUser := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedBytes),
		CreatedAt:    time.Now(),
	}
	if err = s.userRepo.CreateUser(ctx, newUser); err != nil {
		return nil, errors.New("username or email already taken")
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
