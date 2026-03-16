package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
)

func createTestUser(t *testing.T, repo *PostgresUserRepository) uuid.UUID {
	t.Helper()
	user := &model.User{
		ID:           uuid.New(),
		Login:        "entryuser_" + uuid.New().String()[:8],
		PasswordHash: "$2a$10$examplehash",
	}
	err := repo.Create(context.Background(), user)
	require.NoError(t, err)
	return user.ID
}

func TestPostgresEntryRepository_Create_Credential(t *testing.T) {
	db := setupTestDB(t)
	userRepo := newPostgresUserRepository(db)
	entryRepo := newPostgresEntryRepository(db)

	userID := createTestUser(t, userRepo)

	entry := &model.Entry{
		ID:        uuid.New(),
		UserID:    userID,
		EntryType: model.EntryTypeCredential,
		Name:      "Test Credential",
		Credential: &model.CredentialData{
			EncryptedLogin:    []byte("encrypted-login-data"),
			EncryptedPassword: []byte("encrypted-password-data"),
		},
	}

	err := entryRepo.Create(context.Background(), entry)
	require.NoError(t, err)
	assert.NotZero(t, entry.CreatedAt)
	assert.NotZero(t, entry.UpdatedAt)
}

func TestPostgresEntryRepository_Create_Credential_WithMetadata(t *testing.T) {
	db := setupTestDB(t)
	userRepo := newPostgresUserRepository(db)
	entryRepo := newPostgresEntryRepository(db)

	userID := createTestUser(t, userRepo)

	meta := json.RawMessage(`{"url":"https://example.com"}`)
	entry := &model.Entry{
		ID:        uuid.New(),
		UserID:    userID,
		EntryType: model.EntryTypeCredential,
		Name:      "Credential With Metadata",
		Metadata:  &meta,
		Credential: &model.CredentialData{
			EncryptedLogin:    []byte("encrypted-login"),
			EncryptedPassword: []byte("encrypted-password"),
		},
	}

	err := entryRepo.Create(context.Background(), entry)
	require.NoError(t, err)
}

func TestPostgresEntryRepository_Create_Credential_NoData(t *testing.T) {
	db := setupTestDB(t)
	userRepo := newPostgresUserRepository(db)
	entryRepo := newPostgresEntryRepository(db)

	userID := createTestUser(t, userRepo)

	entry := &model.Entry{
		ID:        uuid.New(),
		UserID:    userID,
		EntryType: model.EntryTypeCredential,
		Name:      "Missing Credential Data",
	}

	err := entryRepo.Create(context.Background(), entry)
	assert.Error(t, err)
}

func TestPostgresEntryRepository_Create_InvalidUserFK(t *testing.T) {
	db := setupTestDB(t)
	entryRepo := newPostgresEntryRepository(db)

	entry := &model.Entry{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "Invalid FK",
		Credential: &model.CredentialData{
			EncryptedLogin:    []byte("login"),
			EncryptedPassword: []byte("password"),
		},
	}

	err := entryRepo.Create(context.Background(), entry)
	assert.Error(t, err)
}

func TestPostgresEntryRepository_GetByID_Credential(t *testing.T) {
	db := setupTestDB(t)
	userRepo := newPostgresUserRepository(db)
	entryRepo := newPostgresEntryRepository(db)

	userID := createTestUser(t, userRepo)

	meta := json.RawMessage(`{"site":"example.com"}`)
	entry := &model.Entry{
		ID:        uuid.New(),
		UserID:    userID,
		EntryType: model.EntryTypeCredential,
		Name:      "Get Test",
		Metadata:  &meta,
		Credential: &model.CredentialData{
			EncryptedLogin:    []byte("enc-login"),
			EncryptedPassword: []byte("enc-pass"),
		},
	}

	err := entryRepo.Create(context.Background(), entry)
	require.NoError(t, err)

	found, err := entryRepo.GetByID(context.Background(), entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, found.ID)
	assert.Equal(t, userID, found.UserID)
	assert.Equal(t, model.EntryTypeCredential, found.EntryType)
	assert.Equal(t, "Get Test", found.Name)
	assert.NotNil(t, found.Metadata)
	require.NotNil(t, found.Credential)
	assert.Equal(t, []byte("enc-login"), found.Credential.EncryptedLogin)
	assert.Equal(t, []byte("enc-pass"), found.Credential.EncryptedPassword)
}

func TestPostgresEntryRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	entryRepo := newPostgresEntryRepository(db)

	_, err := entryRepo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresEntryRepository_ListByUserID(t *testing.T) {
	db := setupTestDB(t)
	userRepo := newPostgresUserRepository(db)
	entryRepo := newPostgresEntryRepository(db)

	userID := createTestUser(t, userRepo)

	for i := 0; i < 3; i++ {
		entry := &model.Entry{
			ID:        uuid.New(),
			UserID:    userID,
			EntryType: model.EntryTypeCredential,
			Name:      "Entry " + uuid.New().String()[:4],
			Credential: &model.CredentialData{
				EncryptedLogin:    []byte("login"),
				EncryptedPassword: []byte("pass"),
			},
		}
		err := entryRepo.Create(context.Background(), entry)
		require.NoError(t, err)
	}

	entries, err := entryRepo.ListByUserID(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestPostgresEntryRepository_ListByUserID_Empty(t *testing.T) {
	db := setupTestDB(t)
	entryRepo := newPostgresEntryRepository(db)

	entries, err := entryRepo.ListByUserID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, entries)
}
