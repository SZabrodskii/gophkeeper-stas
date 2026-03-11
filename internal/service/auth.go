package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
)

var AuthModule = fx.Module("service.auth",
	fx.Provide(NewAuthService),
)

const (
	jwtExpiry = 24 * time.Hour
)

type authServiceParams struct {
	fx.In

	UserRepo   repository.UserRepository
	AuthConfig config.AuthConfig
}

type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(params authServiceParams) *AuthService {
	return &AuthService{
		userRepo:  params.UserRepo,
		jwtSecret: []byte(params.AuthConfig.JWTSecret),
	}
}

func NewAuthServiceFromRaw(userRepo repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthService) Register(ctx context.Context, login, password string) (string, error) {
	if login == "" {
		return "", fmt.Errorf("%w: login is required", ErrValidation)
	}
	if len(password) < 8 {
		return "", fmt.Errorf("%w: password must be at least 8 characters", ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return "", ErrUserExists
		}
		return "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

func (s *AuthService) generateJWT(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(jwtExpiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
