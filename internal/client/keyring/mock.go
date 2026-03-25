package keyring

import (
	"fmt"
	"time"
)

// InMemoryKeyring is an in-memory implementation of TokenStore for testing.
type InMemoryKeyring struct {
	token    string
	lastSync string
	hasToken bool
}

// NewInMemory creates an in-memory token store for testing.
func NewInMemory() *InMemoryKeyring {
	return &InMemoryKeyring{}
}

func (k *InMemoryKeyring) Set(token string) error {
	k.token = token
	k.hasToken = true
	return nil
}

func (k *InMemoryKeyring) Get() (string, error) {
	if !k.hasToken {
		return "", fmt.Errorf("secret not found in keyring")
	}
	return k.token, nil
}

func (k *InMemoryKeyring) Delete() error {
	k.token = ""
	k.hasToken = false
	return nil
}

func (k *InMemoryKeyring) SetLastSync(t time.Time) error {
	k.lastSync = t.Format(time.RFC3339)
	return nil
}

func (k *InMemoryKeyring) GetLastSync() (time.Time, error) {
	if k.lastSync == "" {
		return time.Time{}, fmt.Errorf("secret not found in keyring")
	}
	return time.Parse(time.RFC3339, k.lastSync)
}
