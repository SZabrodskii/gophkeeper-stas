package keyring

import (
	"time"

	zkr "github.com/zalando/go-keyring"
)

const (
	serviceName     = "gophkeeper"
	accountToken    = "auth-token"
	accountLastSync = "last-sync-time"
)

// TokenStore abstracts secure storage for authentication tokens and sync state.
type TokenStore interface {
	Set(token string) error
	Get() (string, error)
	Delete() error
	SetLastSync(t time.Time) error
	GetLastSync() (time.Time, error)
}

// OSKeyring implements TokenStore using the OS-level secret store.
type OSKeyring struct{}

// New returns a new OSKeyring instance.
func New() *OSKeyring {
	return &OSKeyring{}
}

func (k *OSKeyring) Set(token string) error {
	return zkr.Set(serviceName, accountToken, token)
}

func (k *OSKeyring) Get() (string, error) {
	return zkr.Get(serviceName, accountToken)
}

func (k *OSKeyring) Delete() error {
	return zkr.Delete(serviceName, accountToken)
}

func (k *OSKeyring) SetLastSync(t time.Time) error {
	return zkr.Set(serviceName, accountLastSync, t.Format(time.RFC3339))
}

func (k *OSKeyring) GetLastSync() (time.Time, error) {
	s, err := zkr.Get(serviceName, accountLastSync)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, s)
}
