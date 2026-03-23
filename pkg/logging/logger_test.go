package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger_DefaultLevel(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNewLogger_DebugLevel(t *testing.T) {
	logger, err := NewLogger(Config{Level: "debug"})
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNewLogger_InvalidLevel_FallsBackToInfo(t *testing.T) {
	logger, err := NewLogger(Config{Level: "invalid-level"})
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNewLogger_EmptyLevel(t *testing.T) {
	logger, err := NewLogger(Config{Level: ""})
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNewHttpbaraLogger(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)

	hLogger := NewHttpbaraLogger(logger)
	assert.NotNil(t, hLogger)
}
