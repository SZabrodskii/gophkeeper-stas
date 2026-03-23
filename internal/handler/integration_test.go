package handler

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gopybara/httpbara"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
	"github.com/SZabrodskii/gophkeeper-stas/internal/service"
)

const (
	integrationJWTSecret     = "integration-test-jwt-secret-32b!"
	integrationEncryptionKey = "01234567890123456789012345678901"
)

func setupIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			login VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS entries (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			entry_type VARCHAR(20) NOT NULL CHECK (entry_type IN ('credential','text','binary','card')),
			name VARCHAR(255) NOT NULL,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_user_id ON entries(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_user_id_updated_at ON entries(user_id, updated_at)`,
		`CREATE TABLE IF NOT EXISTS credential_data (
			entry_id UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
			encrypted_login BYTEA NOT NULL,
			encrypted_password BYTEA NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS text_data (
			entry_id UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
			encrypted_content BYTEA NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS card_data (
			entry_id UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
			encrypted_number BYTEA NOT NULL,
			encrypted_expiry BYTEA NOT NULL,
			encrypted_holder_name BYTEA NOT NULL,
			encrypted_cvv BYTEA NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS binary_data (
			entry_id UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
			encrypted_data BYTEA NOT NULL,
			original_filename TEXT NOT NULL DEFAULT ''
		)`,
	}

	for _, m := range migrations {
		_, err := db.Exec(m)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM credential_data")
		db.Exec("DELETE FROM text_data")
		db.Exec("DELETE FROM card_data")
		db.Exec("DELETE FROM binary_data")
		db.Exec("DELETE FROM entries")
		db.Exec("DELETE FROM users")
		db.Close()
	})

	return db
}

func setupIntegrationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	db := setupIntegrationDB(t)

	gin.SetMode(gin.TestMode)

	userRepo := repository.NewPostgresUserRepository(db).Repo
	entryRepo := repository.NewPostgresEntryRepository(db).Repo

	authSvc := service.NewAuthServiceFromRaw(userRepo, integrationJWTSecret)
	entrySvc := service.NewEntryServiceFromRaw(entryRepo, integrationEncryptionKey, 10*1024*1024)

	authHandler := &AuthHandler{authService: authSvc}
	entryHandler := &EntryHandler{entryService: entrySvc}

	authH, err := httpbara.AsHandler(authHandler)
	require.NoError(t, err)
	entryH, err := httpbara.AsHandler(entryHandler)
	require.NoError(t, err)

	r := gin.New()
	_, err = httpbara.New([]*httpbara.Handler{authH, entryH}, httpbara.WithGinEngine(r))
	require.NoError(t, err)
	return r
}

func registerIntegrationUser(t *testing.T, r *gin.Engine) string {
	t.Helper()
	login := "intuser_" + uuid.New().String()[:8]
	body, _ := json.Marshal(authRequest{Login: login, Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp tokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Token
}

func TestIntegration_FullCRUDFlow(t *testing.T) {
	r := setupIntegrationRouter(t)
	token := registerIntegrationUser(t, r)

	body, _ := json.Marshal(map[string]interface{}{
		"entry_type": "credential",
		"name":       "Integration Test",
		"data":       map[string]string{"login": "admin", "password": "secret"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp createEntryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	entryID := createResp.ID

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+entryID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)
	require.Equal(t, http.StatusOK, gw.Code)

	var getResp entryResponse
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &getResp))
	assert.Equal(t, "Integration Test", getResp.Name)
	data, ok := getResp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "admin", data["login"])
	assert.Equal(t, "secret", data["password"])

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, listReq)
	require.Equal(t, http.StatusOK, lw.Code)

	var entries []entryListItem
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &entries))
	require.Len(t, entries, 1)
	assert.Equal(t, "Integration Test", entries[0].Name)

	updateBody, _ := json.Marshal(map[string]interface{}{
		"entry_type": "credential",
		"name":       "Updated Integration",
		"data":       map[string]string{"login": "newadmin", "password": "newsecret"},
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+entryID.String(), bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)
	uw := httptest.NewRecorder()
	r.ServeHTTP(uw, updateReq)
	require.Equal(t, http.StatusOK, uw.Code)

	getReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+entryID.String(), nil)
	getReq2.Header.Set("Authorization", "Bearer "+token)
	gw2 := httptest.NewRecorder()
	r.ServeHTTP(gw2, getReq2)
	require.Equal(t, http.StatusOK, gw2.Code)

	var getResp2 entryResponse
	require.NoError(t, json.Unmarshal(gw2.Body.Bytes(), &getResp2))
	assert.Equal(t, "Updated Integration", getResp2.Name)
	data2, _ := getResp2.Data.(map[string]interface{})
	assert.Equal(t, "newadmin", data2["login"])

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/"+entryID.String(), nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	dw := httptest.NewRecorder()
	r.ServeHTTP(dw, delReq)
	require.Equal(t, http.StatusNoContent, dw.Code)

	getReq3 := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+entryID.String(), nil)
	getReq3.Header.Set("Authorization", "Bearer "+token)
	gw3 := httptest.NewRecorder()
	r.ServeHTTP(gw3, getReq3)
	assert.Equal(t, http.StatusNotFound, gw3.Code)
}

func TestIntegration_AllDataTypes(t *testing.T) {
	r := setupIntegrationRouter(t)
	token := registerIntegrationUser(t, r)

	credBody, _ := json.Marshal(map[string]interface{}{
		"entry_type": "credential",
		"name":       "GitHub",
		"metadata":   map[string]string{"url": "https://github.com"},
		"data":       map[string]string{"login": "dev", "password": "gh_token123"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(credBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	textBody, _ := json.Marshal(map[string]interface{}{
		"entry_type": "text",
		"name":       "Secret Note",
		"data":       map[string]string{"content": "my secret text content"},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(textBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	cardBody, _ := json.Marshal(map[string]interface{}{
		"entry_type": "card",
		"name":       "My Visa",
		"data":       map[string]string{"number": "4532015112830366", "expiry": "12/25", "holder_name": "John Doe", "cvv": "123"},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(cardBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	rawBinary := []byte("binary file content here")
	binaryBody, _ := json.Marshal(map[string]interface{}{
		"entry_type": "binary",
		"name":       "My File",
		"data":       map[string]string{"data": base64.StdEncoding.EncodeToString(rawBinary), "original_filename": "test.bin"},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(binaryBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, listReq)
	require.Equal(t, http.StatusOK, lw.Code)

	var entries []entryListItem
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &entries))
	assert.Len(t, entries, 4)

	types := map[model.EntryType]bool{}
	for _, e := range entries {
		types[e.EntryType] = true
	}
	assert.True(t, types[model.EntryTypeCredential])
	assert.True(t, types[model.EntryTypeText])
	assert.True(t, types[model.EntryTypeCard])
	assert.True(t, types[model.EntryTypeBinary])
}

func TestIntegration_AuthFlow(t *testing.T) {
	r := setupIntegrationRouter(t)

	login := "authflow_" + uuid.New().String()[:8]
	regBody, _ := json.Marshal(authRequest{Login: login, Password: "password123"})

	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, regReq)
	require.Equal(t, http.StatusCreated, rw.Code)

	var regResp tokenResponse
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &regResp))
	assert.NotEmpty(t, regResp.Token)

	loginBody, _ := json.Marshal(authRequest{Login: login, Password: "password123"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, loginReq)
	require.Equal(t, http.StatusOK, lw.Code)

	var loginResp tokenResponse
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &loginResp))
	assert.NotEmpty(t, loginResp.Token)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	listReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	ew := httptest.NewRecorder()
	r.ServeHTTP(ew, listReq)
	assert.Equal(t, http.StatusOK, ew.Code)

	dupReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	dupReq.Header.Set("Content-Type", "application/json")
	dw := httptest.NewRecorder()
	r.ServeHTTP(dw, dupReq)
	assert.Equal(t, http.StatusConflict, dw.Code)

	badLogin, _ := json.Marshal(authRequest{Login: login, Password: "wrongpassword"})
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badLogin))
	badReq.Header.Set("Content-Type", "application/json")
	bw := httptest.NewRecorder()
	r.ServeHTTP(bw, badReq)
	assert.Equal(t, http.StatusUnauthorized, bw.Code)
}

func TestIntegration_Sync_LWW(t *testing.T) {
	r := setupIntegrationRouter(t)
	token := registerIntegrationUser(t, r)

	createBody, _ := json.Marshal(map[string]interface{}{
		"entry_type": "credential",
		"name":       "LWW Entry",
		"data":       map[string]string{"login": "original", "password": "original"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, req)
	require.Equal(t, http.StatusCreated, cw.Code)

	var createResp createEntryResponse
	require.NoError(t, json.Unmarshal(cw.Body.Bytes(), &createResp))
	entryID := createResp.ID

	update1, _ := json.Marshal(map[string]interface{}{
		"entry_type": "credential",
		"name":       "First Update",
		"data":       map[string]string{"login": "first", "password": "first"},
	})
	uReq := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+entryID.String(), bytes.NewReader(update1))
	uReq.Header.Set("Content-Type", "application/json")
	uReq.Header.Set("Authorization", "Bearer "+token)
	uw := httptest.NewRecorder()
	r.ServeHTTP(uw, uReq)
	require.Equal(t, http.StatusOK, uw.Code)

	time.Sleep(1100 * time.Millisecond)
	since := time.Now().Format(time.RFC3339)
	time.Sleep(1100 * time.Millisecond)

	update2, _ := json.Marshal(map[string]interface{}{
		"entry_type": "credential",
		"name":       "Second Update",
		"data":       map[string]string{"login": "second", "password": "second"},
	})
	uReq2 := httptest.NewRequest(http.MethodPut, "/api/v1/entries/"+entryID.String(), bytes.NewReader(update2))
	uReq2.Header.Set("Content-Type", "application/json")
	uReq2.Header.Set("Authorization", "Bearer "+token)
	uw2 := httptest.NewRecorder()
	r.ServeHTTP(uw2, uReq2)
	require.Equal(t, http.StatusOK, uw2.Code)

	syncReq := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since="+since, nil)
	syncReq.Header.Set("Authorization", "Bearer "+token)
	sw := httptest.NewRecorder()
	r.ServeHTTP(sw, syncReq)
	require.Equal(t, http.StatusOK, sw.Code)

	var syncResp syncResponse
	require.NoError(t, json.Unmarshal(sw.Body.Bytes(), &syncResp))
	require.Len(t, syncResp.Entries, 1)
	assert.Equal(t, "Second Update", syncResp.Entries[0].Name)
	syncData, ok := syncResp.Entries[0].Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "second", syncData["login"])
	assert.Equal(t, "second", syncData["password"])
	assert.NotEmpty(t, syncResp.ServerTime)
}

func TestIntegration_UserIsolation(t *testing.T) {
	r := setupIntegrationRouter(t)
	token1 := registerIntegrationUser(t, r)
	token2 := registerIntegrationUser(t, r)

	body, _ := json.Marshal(map[string]interface{}{
		"entry_type": "credential",
		"name":       "User1 Secret",
		"data":       map[string]string{"login": "u1", "password": "p1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp createEntryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	listReq.Header.Set("Authorization", "Bearer "+token2)
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, listReq)
	require.Equal(t, http.StatusOK, lw.Code)
	var entries []entryListItem
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &entries))
	assert.Empty(t, entries)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/"+createResp.ID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer "+token2)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)
	assert.Equal(t, http.StatusNotFound, gw.Code)
}
