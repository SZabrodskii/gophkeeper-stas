package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
)

type mockUserRepo struct {
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	if _, exists := m.users[user.Login]; exists {
		return repository.ErrAlreadyExists
	}
	m.users[user.Login] = user
	return nil
}

func (m *mockUserRepo) GetByLogin(_ context.Context, login string) (*model.User, error) {
	u, ok := m.users[login]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

const testJWTSecret = "test-secret-key-for-unit-tests!"

func newTestAuthService(repo repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo:  repo,
		jwtSecret: []byte(testJWTSecret),
	}
}

// --- Register tests ---

func TestAuthService_Register_Success(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	token, err := svc.Register(context.Background(), "testuser", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.NotEmpty(t, claims["user_id"])
	assert.NotEmpty(t, claims["exp"])
}

func TestAuthService_Register_DuplicateLogin(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	_, err := svc.Register(context.Background(), "duplicate", "password123")
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), "duplicate", "password456")
	assert.ErrorIs(t, err, ErrUserExists)
}

func TestAuthService_Register_ShortPassword(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	_, err := svc.Register(context.Background(), "testuser", "short")
	assert.ErrorIs(t, err, ErrValidation)
}

func TestAuthService_Register_EmptyLogin(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	_, err := svc.Register(context.Background(), "", "password123")
	assert.ErrorIs(t, err, ErrValidation)
}

// --- Login tests ---

func seedUser(t *testing.T, repo *mockUserRepo, login, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	repo.users[login] = &model.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedUser(t, repo, "testuser", "password123")
	svc := newTestAuthService(repo)

	token, err := svc.Login(context.Background(), "testuser", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, repo.users["testuser"].ID.String(), claims["user_id"])
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	seedUser(t, repo, "testuser", "password123")
	svc := newTestAuthService(repo)

	_, err := svc.Login(context.Background(), "testuser", "wrongpassword")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	_, err := svc.Login(context.Background(), "nonexistent", "password123")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

// --- ValidateToken tests ---

func TestAuthService_ValidateToken_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedUser(t, repo, "testuser", "password123")
	svc := newTestAuthService(repo)

	token, err := svc.Login(context.Background(), "testuser", "password123")
	require.NoError(t, err)

	userID, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, repo.users["testuser"].ID, userID)
}

func TestAuthService_ValidateToken_ExpiredToken(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	claims := jwt.MapClaims{
		"user_id": uuid.New().String(),
		"exp":     time.Now().Add(-1 * time.Hour).Unix(),
		"iat":     time.Now().Add(-25 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	_, err = svc.ValidateToken(tokenStr)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_ValidateToken_MalformedToken(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	_, err := svc.ValidateToken("not-a-valid-token")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_ValidateToken_WrongSigningKey(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": uuid.New().String(),
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("different-secret-key-entirely!!"))
	require.NoError(t, err)

	svc := newTestAuthService(newMockUserRepo())
	_, err = svc.ValidateToken(tokenStr)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
