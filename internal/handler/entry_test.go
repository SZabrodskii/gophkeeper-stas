package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (m *mockEntryRepo) Update(_ context.Context, entry *model.Entry) error {
	if _, ok := m.entries[entry.ID]; !ok {
		return repository.ErrNotFound
	}
	entry.UpdatedAt = time.Now()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockEntryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.entries[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.entries, id)
	return nil
}
func (m *mockEntryRepo) ListUpdatedAfter(_ context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	var result []model.Entry
	for _, e := range m.entries {
		if e.UserID == userID && e.UpdatedAt.After(since) {
			result = append(result, *e)
		}
	}
	return result, nil
}

func setupEntryRouter() (*gin.Engine, *service.EntryService, *service.AuthService) {
	gin.SetMode(gin.TestMode)

	userRepo := newMockUserRepo()
	authSvc := service.NewAuthServiceFromRaw(userRepo, testJWTSecret)
	authHandler := &AuthHandler{authService: authSvc}

	entryRepo := newMockEntryRepo()
	entrySvc := service.NewEntryServiceFromRaw(entryRepo, testEntryEncryptionKey, 10*1024*1024)
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

func setupEntryRouterWithMaxBinary(maxSize int64) (*gin.Engine, *service.EntryService, *service.AuthService) {
	gin.SetMode(gin.TestMode)

	userRepo := newMockUserRepo()
	authSvc := service.NewAuthServiceFromRaw(userRepo, testJWTSecret)
	authHandler := &AuthHandler{authService: authSvc}

	entryRepo := newMockEntryRepo()
	entrySvc := service.NewEntryServiceFromRaw(entryRepo, testEntryEncryptionKey, maxSize)
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

func TestCreateEntry_Binary_Success_201(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	rawData := []byte("hello binary world")
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	reqBody := map[string]interface{}{
		"entry_type": "binary",
		"name":       "My File",
		"data": map[string]string{
			"data":              b64Data,
			"original_filename": "test.bin",
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

func TestGetEntry_Binary_Success_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	rawData := []byte("round trip binary via HTTP")
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	reqBody := map[string]interface{}{
		"entry_type": "binary",
		"name":       "My File",
		"data": map[string]string{
			"data":              b64Data,
			"original_filename": "roundtrip.bin",
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

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+createResp.ID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)

	assert.Equal(t, http.StatusOK, gw.Code)

	var resp entryResponse
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &resp))
	assert.Equal(t, createResp.ID, resp.ID)
	assert.Equal(t, model.EntryTypeBinary, resp.EntryType)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, b64Data, data["data"])
	assert.Equal(t, "roundtrip.bin", data["original_filename"])
}

func TestCreateEntry_Binary_TooLarge_413(t *testing.T) {
	r, _, _ := setupEntryRouterWithMaxBinary(10) // 10 bytes max
	token := getTestToken(t, r)

	rawData := []byte(strings.Repeat("x", 11))
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	reqBody := map[string]interface{}{
		"entry_type": "binary",
		"name":       "Too Big",
		"data": map[string]string{
			"data":              b64Data,
			"original_filename": "big.bin",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestUpdateEntry_Credential_Success_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	entryID := createTestEntry(t, r, token)

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "Updated Website",
		"data": map[string]string{
			"login":    "newlogin",
			"password": "newpassword",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+entryID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp updateEntryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, entryID, resp.ID)
	assert.NotEmpty(t, resp.UpdatedAt)
}

func TestUpdateEntry_TypeMismatch_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	entryID := createTestEntry(t, r, token)

	reqBody := map[string]interface{}{
		"entry_type": "text",
		"name":       "Changed Type",
		"data": map[string]string{
			"content": "some text",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+entryID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEntry_NotFound_404(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "Nonexistent",
		"data": map[string]string{
			"login":    "user",
			"password": "pass",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateEntry_Unauthorized_401(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateEntry_VerifyDataChanged_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	entryID := createTestEntry(t, r, token)

	reqBody := map[string]interface{}{
		"entry_type": "credential",
		"name":       "Updated Entry",
		"data": map[string]string{
			"login":    "changedlogin",
			"password": "changedpass",
		},
	}
	body, _ := json.Marshal(reqBody)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+entryID.String(), bytes.NewReader(body))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)
	uw := httptest.NewRecorder()
	r.ServeHTTP(uw, updateReq)
	require.Equal(t, http.StatusOK, uw.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+entryID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)
	require.Equal(t, http.StatusOK, gw.Code)

	var resp entryResponse
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &resp))
	assert.Equal(t, "Updated Entry", resp.Name)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "changedlogin", data["login"])
	assert.Equal(t, "changedpass", data["password"])
}

func TestDeleteEntry_Success_204(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	entryID := createTestEntry(t, r, token)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/"+entryID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteEntry_NotFound_404(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteEntry_Unauthorized_401(t *testing.T) {
	r, _, _ := setupEntryRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteEntry_ThenGet_404(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	entryID := createTestEntry(t, r, token)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/"+entryID.String(), nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	dw := httptest.NewRecorder()
	r.ServeHTTP(dw, delReq)
	require.Equal(t, http.StatusNoContent, dw.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+entryID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)

	assert.Equal(t, http.StatusNotFound, gw.Code)
}

func TestSyncEntries_Success_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	since := time.Now().Add(-time.Hour).Format(time.RFC3339)

	createTestEntry(t, r, token)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since="+since, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp syncResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Entries, 1)
	assert.NotEmpty(t, resp.ServerTime)

	assert.Equal(t, "Test Entry", resp.Entries[0].Name)
	assert.Equal(t, model.EntryTypeCredential, resp.Entries[0].EntryType)
	assert.NotNil(t, resp.Entries[0].Data)
}

func TestSyncEntries_Empty_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	since := time.Now().Add(time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since="+since, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp syncResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Entries)
	assert.NotEmpty(t, resp.ServerTime)
}

func TestSyncEntries_MissingSince_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSyncEntries_InvalidSince_400(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since=not-a-date", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSyncEntries_Unauthorized_401(t *testing.T) {
	r, _, _ := setupEntryRouter()

	since := time.Now().Add(-time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since="+since, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSyncEntries_VerifyServerTime_200(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	before := time.Now()
	since := time.Now().Add(-time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since="+since, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	after := time.Now()

	assert.Equal(t, http.StatusOK, w.Code)

	var resp syncResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	serverTime, err := time.Parse(time.RFC3339, resp.ServerTime)
	require.NoError(t, err)
	assert.True(t, !serverTime.Before(before.Truncate(time.Second)))
	assert.True(t, !serverTime.After(after.Add(time.Second)))
}

func TestErrorResponseFormat_Consistent(t *testing.T) {
	r, _, _ := setupEntryRouter()
	token := getTestToken(t, r)

	errorCases := []struct {
		name   string
		method string
		path   string
		body   string
		token  string
		code   int
	}{
		{"no auth on entries", http.MethodGet, "/api/v1/entries", "", "", http.StatusUnauthorized},
		{"no auth on entry", http.MethodGet, "/api/v1/entries/" + uuid.New().String(), "", "", http.StatusUnauthorized},
		{"invalid entry id", http.MethodGet, "/api/v1/entries/not-uuid", "", token, http.StatusBadRequest},
		{"not found entry", http.MethodGet, "/api/v1/entries/" + uuid.New().String(), "", token, http.StatusNotFound},
		{"invalid json create", http.MethodPost, "/api/v1/entries", "bad json", token, http.StatusBadRequest},
		{"invalid type create", http.MethodPost, "/api/v1/entries", `{"entry_type":"invalid","name":"x","data":{}}`, token, http.StatusBadRequest},
		{"missing since sync", http.MethodGet, "/api/v1/sync", "", token, http.StatusBadRequest},
		{"bad since sync", http.MethodGet, "/api/v1/sync?since=nope", "", token, http.StatusBadRequest},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(tc.body)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.code, w.Code)

			var errResp map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp), "response must be valid JSON")
			assert.NotEmpty(t, errResp["error"], "response must have 'error' field")
		})
	}
}
