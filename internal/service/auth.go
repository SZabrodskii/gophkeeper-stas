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

// AuthModule provides the AuthService via fx DI.
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

// AuthService handles user registration, login and JWT validation.
type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
}

// NewAuthService creates an AuthService from fx-injected parameters.
func NewAuthService(params authServiceParams) *AuthService {
	return &AuthService{
		userRepo:  params.UserRepo,
		jwtSecret: []byte(params.AuthConfig.JWTSecret),
	}
}

// NewAuthServiceFromRaw creates an AuthService without fx (useful for tests).
func NewAuthServiceFromRaw(userRepo repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates a new user and returns a JWT token.
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

// Login authenticates an existing user and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

// ValidateToken parses and validates a JWT, returning the user ID.
func (s *AuthService) ValidateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return uuid.Nil, ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, ErrInvalidCredentials
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, ErrInvalidCredentials
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, ErrInvalidCredentials
	}

	return userID, nil
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
