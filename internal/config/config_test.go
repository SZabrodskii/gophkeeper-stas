package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientConfig_Defaults(t *testing.T) {
	os.Unsetenv("SERVER_ADDRESS")
	os.Unsetenv("TLS_INSECURE")

	cfg, err := NewClientConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://localhost:8443", cfg.ServerAddress)
	assert.False(t, cfg.TLSInsecure)
}

func TestNewClientConfig_EnvOverride(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "https://custom:9999")
	t.Setenv("TLS_INSECURE", "true")

	cfg, err := NewClientConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://custom:9999", cfg.ServerAddress)
	assert.True(t, cfg.TLSInsecure)
}

func TestNewServerConfig_RequiredFields(t *testing.T) {
	os.Unsetenv("DATABASE_DSN")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("ENCRYPTION_KEY")
	os.Unsetenv("TLS_CERT")
	os.Unsetenv("TLS_KEY")

	_, err := NewServerConfig()
	require.Error(t, err)
}

func TestNewServerConfig_Success(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "secret123")
	t.Setenv("ENCRYPTION_KEY", "01234567890123456789012345678901")
	t.Setenv("TLS_CERT", "/tmp/cert.pem")
	t.Setenv("TLS_KEY", "/tmp/key.pem")
	t.Setenv("ADDRESS", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MAX_BINARY_SIZE", "5242880")

	out, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, ":9090", out.Full.Address)
	assert.Equal(t, "postgres://localhost/test", out.DB.DSN)
	assert.Equal(t, "secret123", out.Auth.JWTSecret)
	assert.Equal(t, "01234567890123456789012345678901", out.Auth.EncryptionKey)
	assert.Equal(t, ":9090", out.Listen.Address)
	assert.Equal(t, "/tmp/cert.pem", out.Listen.TLSCert)
	assert.Equal(t, "/tmp/key.pem", out.Listen.TLSKey)
	assert.Equal(t, "debug", out.Logging.Level)
	assert.Equal(t, int64(5242880), out.Full.MaxBinarySize)
}

func TestNewServerConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("ENCRYPTION_KEY", "key")
	t.Setenv("TLS_CERT", "cert.pem")
	t.Setenv("TLS_KEY", "key.pem")
	os.Unsetenv("ADDRESS")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("MAX_BINARY_SIZE")

	out, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, ":8443", out.Full.Address)
	assert.Equal(t, "info", out.Logging.Level)
	assert.Equal(t, int64(10485760), out.Full.MaxBinarySize)
}
