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
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
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
		return nil, repository.ErrNotFound
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

func createTestEntry(t *testing.T, r *gin.Engine, token string) uuid.UUID {
	t.Helper()
	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "Test Entry",
		"data": map[string]string{
			"login":    "testlogin",
			"password": "testpass",
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp createEntryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.ID
}

func TestListEntries_Success_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	createTestEntry(t, r, token)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []entryListItem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries))
	assert.Len(t, entries, 1)
	assert.Equal(t, "Test Entry", entries[0].Name)
	assert.Equal(t, model.EntryTypeCredential, entries[0].EntryType)
}

func TestListEntries_Empty_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []entryListItem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries))
	assert.Empty(t, entries)
}

func TestListEntries_Unauthorized_401(t *testing.T) {
	r, _, _ := setupEntryRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetEntry_Success_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	entryID := createTestEntry(t, r, token)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+entryID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp entryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, entryID, resp.ID)
	assert.Equal(t, "Test Entry", resp.Name)
	assert.Equal(t, model.EntryTypeCredential, resp.EntryType)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "testlogin", data["login"])
	assert.Equal(t, "testpass", data["password"])
}

func TestGetEntry_NotFound_404(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetEntry_Unauthorized_401(t *testing.T) {
	r, _, _ := setupEntryRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateEntry_Text_Success_201(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "text",
		"name":       "My Note",
		"data": map[string]string{
			"content": "This is my secret note",
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

func TestGetEntry_Text_Success_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	// Create text entry
	reqBody := map[string]interface{}{
		"entry_type": "text",
		"name":       "My Note",
		"data": map[string]string{
			"content": "decrypted text content",
		},
	}
	body, _ := json.Marshal(reqBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, createReq)
	require.Equal(t, http.StatusCreated, cw.Code)

	var createResp createEntryResponse
	require.NoError(t, json.Unmarshal(cw.Body.Bytes(), &createResp))

	// Get text entry
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+createResp.ID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)

	assert.Equal(t, http.StatusOK, gw.Code)

	var resp entryResponse
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &resp))
	assert.Equal(t, createResp.ID, resp.ID)
	assert.Equal(t, model.EntryTypeText, resp.EntryType)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "decrypted text content", data["content"])
}

func TestCreateEntry_Text_EmptyContent_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "text",
		"name":       "Empty Note",
		"data": map[string]string{
			"content": "",
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

func TestCreateEntry_Card_Success_201(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "card",
		"name":       "My Visa",
		"data": map[string]string{
			"number":      "4532015112830366",
			"expiry":      "12/25",
			"holder_name": "John Doe",
			"cvv":         "123",
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

func TestGetEntry_Card_Success_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	// Create card entry
	reqBody := map[string]interface{}{
		"entry_type": "card",
		"name":       "My Visa",
		"data": map[string]string{
			"number":      "4532015112830366",
			"expiry":      "12/25",
			"holder_name": "John Doe",
			"cvv":         "123",
		},
	}
	body, _ := json.Marshal(reqBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, createReq)
	require.Equal(t, http.StatusCreated, cw.Code)

	var createResp createEntryResponse
	require.NoError(t, json.Unmarshal(cw.Body.Bytes(), &createResp))

	// Get card entry
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+createResp.ID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)

	assert.Equal(t, http.StatusOK, gw.Code)

	var resp entryResponse
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &resp))
	assert.Equal(t, createResp.ID, resp.ID)
	assert.Equal(t, model.EntryTypeCard, resp.EntryType)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "4532015112830366", data["number"])
	assert.Equal(t, "12/25", data["expiry"])
	assert.Equal(t, "John Doe", data["holder_name"])
	assert.Equal(t, "123", data["cvv"])
}

func TestCreateEntry_Card_InvalidLuhn_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "card",
		"name":       "Bad Card",
		"data": map[string]string{
			"number":      "1234567890",
			"expiry":      "12/25",
			"holder_name": "John Doe",
			"cvv":         "123",
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

func TestGetEntry_InvalidID_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
