package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gopybara/httpbara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/service"
)

const testEntryEncryptionKey = "01234567890123456789012345678901"

type mockEntryRepo struct {
	entries map[uuid.UUID]*model.Entry
}

func newMockEntryRepo() *mockEntryRepo {
	return &mockEntryRepo{entries: make(map[uuid.UUID]*model.Entry)}
}

func (m *mockEntryRepo) Create(_ context.Context, entry *model.Entry) error {
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockEntryRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Entry, error) {
	e, ok := m.entries[id]
	if !ok {
		return nil, service.ErrNotFound
	}
	return e, nil
}

func (m *mockEntryRepo) ListByUserID(_ context.Context, userID uuid.UUID) ([]model.Entry, error) {
	var result []model.Entry
	for _, e := range m.entries {
		if e.UserID == userID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *mockEntryRepo) Update(_ context.Context, entry *model.Entry) error { return nil }
func (m *mockEntryRepo) Delete(_ context.Context, id uuid.UUID) error       { return nil }
func (m *mockEntryRepo) ListUpdatedAfter(_ context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	return nil, nil
}

func setupEntryRouter() (*gin.Engine, *service.EntryService, *service.AuthService) {
	gin.SetMode(gin.TestMode)

	userRepo := newMockUserRepo()
	authSvc := service.NewAuthServiceFromRaw(userRepo, testJWTSecret)
	authHandler := &AuthHandler{authService: authSvc}

	entryRepo := newMockEntryRepo()
	entrySvc := service.NewEntryServiceFromRaw(entryRepo, testEntryEncryptionKey)
	entryHandler := &EntryHandler{entryService: entrySvc}

	authH, err := httpbara.AsHandler(authHandler)
	if err != nil {
		panic(err)
	}
	entryH, err := httpbara.AsHandler(entryHandler)
	if err != nil {
		panic(err)
	}

	r := gin.New()
	if _, err := httpbara.New([]*httpbara.Handler{authH, entryH}, httpbara.WithGinEngine(r)); err != nil {
		panic(err)
	}
	return r, entrySvc, authSvc
}

func getTestToken(t *testing.T, r *gin.Engine) string {
	t.Helper()
	body, _ := json.Marshal(authRequest{Login: "testuser_" + uuid.New().String()[:8], Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp tokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Token
}

func TestCreateEntry_Credential_Success_201(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "My Website",
		"data": map[string]string{
			"login":    "user@example.com",
			"password": "secret123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp createEntryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.NotEmpty(t, resp.CreatedAt)
}

func TestCreateEntry_Credential_WithMetadata_201(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "Work Account",
		"metadata":   map[string]string{"url": "https://work.example.com"},
		"data": map[string]string{
			"login":    "admin",
			"password": "admin1234",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateEntry_MissingName_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "",
		"data": map[string]string{
			"login":    "user",
			"password": "pass1234",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEntry_InvalidType_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "invalid",
		"name":       "Test",
		"data":       map[string]string{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEntry_Unauthorized_401(t *testing.T) {
	r, _, _ := setupEntryRouter()

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "Test",
		"data": map[string]string{
			"login":    "user",
			"password": "pass",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateEntry_InvalidJSON_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEntry_MissingCredentialLogin_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "Test",
		"data": map[string]string{
			"login":    "",
			"password": "pass1234",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
