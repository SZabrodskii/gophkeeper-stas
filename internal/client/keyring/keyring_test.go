package keyring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryKeyring_SetGet(t *testing.T) {
	kr := NewInMemory()

	require.NoError(t, kr.Set("my-token"))
	tok, err := kr.Get()
	require.NoError(t, err)
	assert.Equal(t, "my-token", tok)
}

func TestInMemoryKeyring_Get_Empty(t *testing.T) {
	kr := NewInMemory()

	_, err := kr.Get()
	require.Error(t, err)
}

func TestInMemoryKeyring_Delete(t *testing.T) {
	kr := NewInMemory()

	require.NoError(t, kr.Set("my-token"))
	require.NoError(t, kr.Delete())
	_, err := kr.Get()
	require.Error(t, err)
}

func TestInMemoryKeyring_SetLastSync(t *testing.T) {
	kr := NewInMemory()

	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(t, kr.SetLastSync(ts))

	got, err := kr.GetLastSync()
	require.NoError(t, err)
	assert.Equal(t, ts.Format(time.RFC3339), got.Format(time.RFC3339))
}

func TestInMemoryKeyring_GetLastSync_Empty(t *testing.T) {
	kr := NewInMemory()

	_, err := kr.GetLastSync()
	require.Error(t, err)
}

func TestInMemoryKeyring_Overwrite(t *testing.T) {
	kr := NewInMemory()

	require.NoError(t, kr.Set("first"))
	require.NoError(t, kr.Set("second"))

	tok, err := kr.Get()
	require.NoError(t, err)
	assert.Equal(t, "second", tok)
}
