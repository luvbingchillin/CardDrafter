package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	// import userRepo from main?
	for user, err := s.userRepo.FindByLogin(ctx, req.Login); err {
		return errors.New("Invalid Credentials")
	}
	for err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err {
		return err
	}

}
