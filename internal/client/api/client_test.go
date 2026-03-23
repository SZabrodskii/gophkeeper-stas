package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
)

func newTestClient(t *testing.T, handler http.Handler) (*HTTPClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)

	client := NewHTTPClient(&config.ClientConfig{
		ServerAddress: srv.URL,
		TLSInsecure:   true,
	})
	client.httpClient = srv.Client()
	return client, srv
}

func TestRegister_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var req AuthRequest
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "user1", req.Login)
		assert.Equal(t, "pass1", req.Password)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(TokenResponse{Token: "tok123"})
	})

	c, _ := newTestClient(t, mux)
	resp, err := c.Register(context.Background(), "user1", "pass1")

	require.NoError(t, err)
	assert.Equal(t, "tok123", resp.Token)
}

func TestRegister_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "login already exists"})
	})

	c, _ := newTestClient(t, mux)
	_, err := c.Register(context.Background(), "user1", "pass1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "login already exists")
}

func TestLogin_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(TokenResponse{Token: "tok456"})
	})

	c, _ := newTestClient(t, mux)
	resp, err := c.Login(context.Background(), "user1", "pass1")

	require.NoError(t, err)
	assert.Equal(t, "tok456", resp.Token)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
	})

	c, _ := newTestClient(t, mux)
	_, err := c.Login(context.Background(), "user1", "wrong")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestCreateEntry_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))

		var req CreateEntryRequest
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "credential", req.EntryType)
		assert.Equal(t, "github", req.Name)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEntryResponse{
			ID:        "uuid-1",
			CreatedAt: "2026-01-01T00:00:00Z",
		})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("mytoken")

	data, _ := json.Marshal(map[string]string{"login": "l", "password": "p"})
	resp, err := c.CreateEntry(context.Background(), CreateEntryRequest{
		EntryType: "credential",
		Name:      "github",
		Data:      data,
	})

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", resp.ID)
}

func TestCreateEntry_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing authorization header"})
	})

	c, _ := newTestClient(t, mux)
	data, _ := json.Marshal(map[string]string{"login": "l", "password": "p"})
	_, err := c.CreateEntry(context.Background(), CreateEntryRequest{
		EntryType: "credential",
		Name:      "github",
		Data:      data,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing authorization header")
}

func TestListEntries_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]EntryListItem{
			{ID: "id1", EntryType: "credential", Name: "github", UpdatedAt: "2026-01-01T00:00:00Z"},
			{ID: "id2", EntryType: "text", Name: "note", UpdatedAt: "2026-01-02T00:00:00Z"},
		})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")
	entries, err := c.ListEntries(context.Background())

	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "github", entries[0].Name)
}

func TestGetEntry_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/uuid-1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(EntryResponse{
			ID:        "uuid-1",
			EntryType: "credential",
			Name:      "github",
			Data:      json.RawMessage(`{"login":"user","password":"pass"}`),
			CreatedAt: "2026-01-01T00:00:00Z",
			UpdatedAt: "2026-01-01T00:00:00Z",
		})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")
	entry, err := c.GetEntry(context.Background(), "uuid-1")

	require.NoError(t, err)
	assert.Equal(t, "github", entry.Name)
	assert.Equal(t, "credential", entry.EntryType)
}

func TestGetEntry_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/uuid-missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "entry not found"})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")
	_, err := c.GetEntry(context.Background(), "uuid-missing")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry not found")
}

func TestUpdateEntry_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/uuid-1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UpdateEntryResponse{
			ID:        "uuid-1",
			UpdatedAt: "2026-01-02T00:00:00Z",
		})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")

	data, _ := json.Marshal(map[string]string{"login": "new", "password": "new"})
	resp, err := c.UpdateEntry(context.Background(), "uuid-1", CreateEntryRequest{
		EntryType: "credential",
		Name:      "github",
		Data:      data,
	})

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", resp.ID)
}

func TestDeleteEntry_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/uuid-1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")
	err := c.DeleteEntry(context.Background(), "uuid-1")

	require.NoError(t, err)
}

func TestDeleteEntry_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/uuid-missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "entry not found"})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")
	err := c.DeleteEntry(context.Background(), "uuid-missing")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry not found")
}

func TestSync_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sync", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		since := r.URL.Query().Get("since")
		assert.NotEmpty(t, since)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SyncResponse{
			Entries: []EntryResponse{
				{ID: "id1", EntryType: "text", Name: "note", UpdatedAt: "2026-01-01T12:00:00Z"},
			},
			ServerTime: "2026-01-01T12:00:00Z",
		})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")
	resp, err := c.Sync(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	assert.Len(t, resp.Entries, 1)
	assert.Equal(t, "2026-01-01T12:00:00Z", resp.ServerTime)
}

func TestSync_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sync", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid since time"})
	})

	c, _ := newTestClient(t, mux)
	c.SetToken("tok")
	_, err := c.Sync(context.Background(), time.Time{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid since time")
}

func TestSetToken(t *testing.T) {
	c := &HTTPClient{}
	c.SetToken("mytoken")
	assert.Equal(t, "mytoken", c.token)
}

func TestDoRequest_ServerDown(t *testing.T) {
	c := NewHTTPClient(&config.ClientConfig{
		ServerAddress: "https://127.0.0.1:1",
		TLSInsecure:   true,
	})

	_, err := c.Login(context.Background(), "u", "p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do request")
}
