package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered application user.
type User struct {
	ID           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
