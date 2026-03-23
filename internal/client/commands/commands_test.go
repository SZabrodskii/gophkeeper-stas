package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientapi "github.com/SZabrodskii/gophkeeper-stas/internal/client/api"
	"github.com/SZabrodskii/gophkeeper-stas/internal/client/keyring"
	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
)

func setupTestApp(t *testing.T, mux *http.ServeMux) *keyring.InMemoryKeyring {
	t.Helper()
	srv := httptest.NewServer(mux)

	kr := keyring.NewInMemory()

	origPreRun := rootCmd.PersistentPreRunE
	origSilenceErr := rootCmd.SilenceErrors
	origSilenceUsage := rootCmd.SilenceUsage

	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cfg := &config.ClientConfig{ServerAddress: srv.URL}
		app.Config = cfg
		app.API = clientapi.NewHTTPClient(cfg)
		app.Keyring = kr
		return nil
	}

	t.Cleanup(func() {
		rootCmd.PersistentPreRunE = origPreRun
		rootCmd.SilenceErrors = origSilenceErr
		rootCmd.SilenceUsage = origSilenceUsage
		srv.Close()
	})

	return kr
}

func silenceStdout(t *testing.T) {
	t.Helper()
	orig := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	os.Stdout = devNull
	t.Cleanup(func() {
		os.Stdout = orig
		devNull.Close()
	})
}

func execCmd(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func TestRegisterCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"token": "reg-token-123"})
	})
	kr := setupTestApp(t, mux)

	err := execCmd("register", "--login", "testuser", "--password", "testpass123")
	require.NoError(t, err)

	tok, err := kr.Get()
	require.NoError(t, err)
	assert.Equal(t, "reg-token-123", tok)
}

func TestRegisterCmd_ServerError(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "login already exists"})
	})
	_ = setupTestApp(t, mux)

	err := execCmd("register", "--login", "existing", "--password", "testpass123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login already exists")
}

func TestLoginCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"token": "login-token-456"})
	})
	kr := setupTestApp(t, mux)

	err := execCmd("login", "--login", "testuser", "--password", "testpass123")
	require.NoError(t, err)

	tok, err := kr.Get()
	require.NoError(t, err)
	assert.Equal(t, "login-token-456", tok)
}

func TestLoginCmd_InvalidCredentials(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
	})
	_ = setupTestApp(t, mux)

	err := execCmd("login", "--login", "user", "--password", "wrong")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestListCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "id1", "entry_type": "credential", "name": "github", "updated_at": "2026-01-01T00:00:00Z"},
		})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("list")
	require.NoError(t, err)
}

func TestListCmd_Empty(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("list")
	require.NoError(t, err)
}

func TestGetCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/test-uuid", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "test-uuid",
			"entry_type": "credential",
			"name":       "github",
			"data":       map[string]string{"login": "user", "password": "pass"},
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-01-01T00:00:00Z",
		})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("get", "test-uuid")
	require.NoError(t, err)
}

func TestDeleteCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/test-uuid", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("delete", "test-uuid")
	require.NoError(t, err)
}

func TestCreateCredentialCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "new-uuid", "created_at": "2026-01-01T00:00:00Z"})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("create", "credential", "--name", "github", "--login", "user", "--password", "pass")
	require.NoError(t, err)
}

func TestCreateTextCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "new-uuid", "created_at": "2026-01-01T00:00:00Z"})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("create", "text", "--name", "note", "--content", "my secret")
	require.NoError(t, err)
}

func TestCreateCardCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "new-uuid", "created_at": "2026-01-01T00:00:00Z"})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("create", "card", "--name", "visa", "--number", "4532015112830366", "--expiry", "12/25", "--holder", "John", "--cvv", "123")
	require.NoError(t, err)
}

func TestCreateBinaryCmd_Success(t *testing.T) {
	silenceStdout(t)

	tmpFile, err := os.CreateTemp("", "test-binary-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write([]byte("test binary data"))
	require.NoError(t, err)
	tmpFile.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "new-uuid", "created_at": "2026-01-01T00:00:00Z"})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err = execCmd("create", "binary", "--name", "myfile", "--file", tmpFile.Name())
	require.NoError(t, err)
}

func TestUpdateCredentialCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/entries/test-uuid", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "test-uuid", "updated_at": "2026-01-02T00:00:00Z"})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("update", "credential", "--name", "github", "--login", "new", "--password", "new", "test-uuid")
	require.NoError(t, err)
}

func TestSyncCmd_Success(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sync", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"entries": []map[string]interface{}{
				{
					"id":         "id1",
					"entry_type": "credential",
					"name":       "github",
					"data":       map[string]string{"login": "user", "password": "pass"},
					"updated_at": "2026-01-01T12:00:00Z",
				},
			},
			"server_time": "2026-01-01T12:00:00Z",
		})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("sync")
	require.NoError(t, err)

	lastSync, err := kr.GetLastSync()
	require.NoError(t, err)
	assert.Equal(t, "2026-01-01T12:00:00Z", lastSync.Format(time.RFC3339))
}

func TestSyncCmd_NoUpdates(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sync", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"entries":     []interface{}{},
			"server_time": "2026-01-01T12:00:00Z",
		})
	})
	kr := setupTestApp(t, mux)
	require.NoError(t, kr.Set("test-token"))

	err := execCmd("sync")
	require.NoError(t, err)
}

func TestRequireAuth_NotLoggedIn(t *testing.T) {
	silenceStdout(t)

	mux := http.NewServeMux()
	_ = setupTestApp(t, mux)

	err := execCmd("list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}
