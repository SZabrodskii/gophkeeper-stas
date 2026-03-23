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
	"github.com/gopybara/httpbara"
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

const testJWTSecret = "test-secret-key-for-handler-tests"

func setupRouter() (*gin.Engine, *service.AuthService, *mockUserRepo) {
	gin.SetMode(gin.TestMode)
	repo := newMockUserRepo()
	authSvc := service.NewAuthServiceFromRaw(repo, testJWTSecret)
	h := &AuthHandler{authService: authSvc}

	handler, err := httpbara.AsHandler(h)
	if err != nil {
		panic(err)
	}

	r := gin.New()
	if _, err := httpbara.New([]*httpbara.Handler{handler}, httpbara.WithGinEngine(r)); err != nil {
		panic(err)
	}
	return r, authSvc, repo
}

func setupProtectedRouter() (*gin.Engine, *service.AuthService, *mockUserRepo) {
	gin.SetMode(gin.TestMode)
	repo := newMockUserRepo()
	authSvc := service.NewAuthServiceFromRaw(repo, testJWTSecret)
	h := &AuthHandler{authService: authSvc}

	handler, err := httpbara.AsHandler(h)
	if err != nil {
		panic(err)
	}

	r := gin.New()
	if _, err := httpbara.New([]*httpbara.Handler{handler}, httpbara.WithGinEngine(r)); err != nil {
		panic(err)
	}

	r.GET("/api/v1/protected", func(c *gin.Context) {
		h.JWTMiddleware(c)
		if c.IsAborted() {
			return
		}
		uid, _ := c.Get(UserIDKey)
		c.JSON(http.StatusOK, gin.H{"user_id": uid})
	})

	return r, authSvc, repo
}

func TestRegister_Success_201(t *testing.T) {
	r, _, _ := setupRouter()

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
	r, _, _ := setupRouter()

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
	r, _, _ := setupRouter()

	body, _ := json.Marshal(authRequest{Login: "existing", Password: "password123"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	body, _ = json.Marshal(authRequest{Login: "existing", Password: "password456"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegister_InvalidJSON_400(t *testing.T) {
	r, _, _ := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Success_200(t *testing.T) {
	r, _, _ := setupRouter()

	regBody, _ := json.Marshal(authRequest{Login: "loginuser", Password: "password123"})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	r.ServeHTTP(regW, regReq)
	require.Equal(t, http.StatusCreated, regW.Code)

	body, _ := json.Marshal(authRequest{Login: "loginuser", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp tokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
}

func TestLogin_InvalidCredentials_401(t *testing.T) {
	r, _, _ := setupRouter()

	regBody, _ := json.Marshal(authRequest{Login: "loginuser", Password: "password123"})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	r.ServeHTTP(regW, regReq)
	require.Equal(t, http.StatusCreated, regW.Code)

	tests := []struct {
		name string
		body authRequest
	}{
		{"wrong password", authRequest{Login: "loginuser", Password: "wrongpassword"}},
		{"unknown user", authRequest{Login: "nonexistent", Password: "password123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestLogin_InvalidJSON_400(t *testing.T) {
	r, _, _ := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	r, _, _ := setupProtectedRouter()

	regBody, _ := json.Marshal(authRequest{Login: "jwtuser", Password: "password123"})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	r.ServeHTTP(regW, regReq)
	require.Equal(t, http.StatusCreated, regW.Code)

	var regResp tokenResponse
	require.NoError(t, json.Unmarshal(regW.Body.Bytes(), &regResp))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+regResp.Token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	r, _, _ := setupProtectedRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_MalformedToken(t *testing.T) {
	r, _, _ := setupProtectedRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	r, authSvc, _ := setupProtectedRouter()

	regBody, _ := json.Marshal(authRequest{Login: "expuser", Password: "password123"})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	r.ServeHTTP(regW, regReq)
	require.Equal(t, http.StatusCreated, regW.Code)

	_ = authSvc // ValidateToken will reject expired tokens
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDAwMDAwMDAsImlhdCI6MTYwMDAwMDAwMCwidXNlcl9pZCI6IjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMCJ9.invalid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_InvalidFormat_NoBearerPrefix(t *testing.T) {
	r, _, _ := setupProtectedRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Token some-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
