package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
	"github.com/SZabrodskii/gophkeeper-stas/internal/service"
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

func setupRouter() (*gin.Engine, *mockUserRepo) {
	gin.SetMode(gin.TestMode)
	repo := newMockUserRepo()
	authSvc := service.NewAuthServiceFromRaw(repo, "test-secret-key-for-handler-tests")
	h := NewAuthHandler(authSvc)

	r := gin.New()
	RegisterAuthRoutes(r, h)
	return r, repo
}

func TestRegister_Success_201(t *testing.T) {
	r, _ := setupRouter()

	body, _ := json.Marshal(authRequest{Login: "newuser", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp tokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
}

func TestRegister_ValidationError_400(t *testing.T) {
	r, _ := setupRouter()

	tests := []struct {
		name string
		body authRequest
	}{
		{"empty login", authRequest{Login: "", Password: "password123"}},
		{"short password", authRequest{Login: "user", Password: "short"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestRegister_Conflict_409(t *testing.T) {
	r, _ := setupRouter()

	body, _ := json.Marshal(authRequest{Login: "existing", Password: "password123"})

	// First registration — success.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Second registration with same login — conflict.
	body, _ = json.Marshal(authRequest{Login: "existing", Password: "password456"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegister_InvalidJSON_400(t *testing.T) {
	r, _ := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
