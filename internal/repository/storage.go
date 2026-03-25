package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// EntryRepository defines persistence operations for secret entries.
type EntryRepository interface {
	Create(ctx context.Context, entry *model.Entry) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Entry, error)
	Update(ctx context.Context, entry *model.Entry) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListUpdatedAfter(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error)
}
