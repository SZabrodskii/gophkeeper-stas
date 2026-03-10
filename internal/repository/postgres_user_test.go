package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM users")
		db.Close()
	})

	return db
}

func TestPostgresUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := newPostgresUserRepository(db)

	user := &model.User{
		ID:           uuid.New(),
		Login:        "testuser_" + uuid.New().String()[:8],
		PasswordHash: "$2a$10$examplehash",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}

	err := repo.Create(context.Background(), user)
	require.NoError(t, err)
}

func TestPostgresUserRepository_Create_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	repo := newPostgresUserRepository(db)

	login := "duplicate_" + uuid.New().String()[:8]

	user1 := &model.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: "$2a$10$examplehash",
		CreatedAt:    time.Now(),
	}
	err := repo.Create(context.Background(), user1)
	require.NoError(t, err)

	user2 := &model.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: "$2a$10$examplehash2",
		CreatedAt:    time.Now(),
	}
	err = repo.Create(context.Background(), user2)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestPostgresUserRepository_GetByLogin(t *testing.T) {
	db := setupTestDB(t)
	repo := newPostgresUserRepository(db)

	user := &model.User{
		ID:           uuid.New(),
		Login:        "getbylogin_" + uuid.New().String()[:8],
		PasswordHash: "$2a$10$examplehash",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	err := repo.Create(context.Background(), user)
	require.NoError(t, err)

	found, err := repo.GetByLogin(context.Background(), user.Login)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.Login, found.Login)
	assert.Equal(t, user.PasswordHash, found.PasswordHash)
}

func TestPostgresUserRepository_GetByLogin_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := newPostgresUserRepository(db)

	_, err := repo.GetByLogin(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresUserRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := newPostgresUserRepository(db)

	user := &model.User{
		ID:           uuid.New(),
		Login:        "getbyid_" + uuid.New().String()[:8],
		PasswordHash: "$2a$10$examplehash",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	err := repo.Create(context.Background(), user)
	require.NoError(t, err)

	found, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Login, found.Login)
}
