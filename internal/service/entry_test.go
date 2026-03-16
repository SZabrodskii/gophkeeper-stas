package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/crypto"
	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
)

const testEncryptionKey = "01234567890123456789012345678901"

type mockEntryRepo struct {
	entries map[uuid.UUID]*model.Entry
}

func newMockEntryRepo() *mockEntryRepo {
	return &mockEntryRepo{entries: make(map[uuid.UUID]*model.Entry)}
}

func (m *mockEntryRepo) Create(_ context.Context, entry *model.Entry) error {
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockEntryRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Entry, error) {
	e, ok := m.entries[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return e, nil
}

func (m *mockEntryRepo) ListByUserID(_ context.Context, userID uuid.UUID) ([]model.Entry, error) {
	var result []model.Entry
	for _, e := range m.entries {
		if e.UserID == userID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *mockEntryRepo) Update(_ context.Context, entry *model.Entry) error {
	return nil
}

func (m *mockEntryRepo) Delete(_ context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockEntryRepo) ListUpdatedAfter(_ context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	return nil, nil
}

func newTestEntryService(repo *mockEntryRepo) *EntryService {
	return NewEntryServiceFromRaw(repo, testEncryptionKey)
}

func TestEntryService_Create_Credential_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "My Website",
		Credential: &model.CredentialData{
			Login:    "user@example.com",
			Password: "secretpass123",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.NotEmpty(t, entry.Credential.EncryptedLogin)
	assert.NotEmpty(t, entry.Credential.EncryptedPassword)

	stored, ok := repo.entries[entry.ID]
	require.True(t, ok)
	assert.Equal(t, model.EntryTypeCredential, stored.EntryType)
	assert.Equal(t, "My Website", stored.Name)
}

func TestEntryService_Create_Credential_WithMetadata(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	meta := json.RawMessage(`{"url":"https://example.com","notes":"work account"}`)
	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "Work Account",
		Metadata:  &meta,
		Credential: &model.CredentialData{
			Login:    "admin",
			Password: "admin1234",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)
	assert.NotNil(t, repo.entries[entry.ID].Metadata)
}

func TestEntryService_Create_EmptyName(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "",
		Credential: &model.CredentialData{
			Login:    "user",
			Password: "password123",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_InvalidEntryType(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: "invalid",
		Name:      "Test",
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Credential_MissingData(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "Test",
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Credential_EmptyLogin(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "Test",
		Credential: &model.CredentialData{
			Login:    "",
			Password: "password123",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Credential_EmptyPassword(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "Test",
		Credential: &model.CredentialData{
			Login:    "user",
			Password: "",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Credential_EncryptionVerify(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCredential,
		Name:      "Encrypted Test",
		Credential: &model.CredentialData{
			Login:    "mylogin",
			Password: "mypassword",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	stored := repo.entries[entry.ID]
	assert.NotEqual(t, []byte("mylogin"), stored.Credential.EncryptedLogin)
	assert.NotEqual(t, []byte("mypassword"), stored.Credential.EncryptedPassword)
	assert.NotEmpty(t, stored.Credential.EncryptedLogin)
	assert.NotEmpty(t, stored.Credential.EncryptedPassword)
}

func createEncryptedEntry(t *testing.T, repo *mockEntryRepo, svc *EntryService, userID uuid.UUID) *model.Entry {
	t.Helper()
	entry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeCredential,
		Name:      "Test Credential",
		Credential: &model.CredentialData{
			Login:    "mylogin",
			Password: "mypassword",
		},
	}
	require.NoError(t, svc.Create(context.Background(), entry))
	return entry
}

func TestEntryService_GetByID_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	entry := createEncryptedEntry(t, repo, svc, userID)

	result, err := svc.GetByID(context.Background(), entry.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, result.ID)
	assert.Equal(t, "mylogin", result.Credential.Login)
	assert.Equal(t, "mypassword", result.Credential.Password)
}

func TestEntryService_GetByID_NotFound(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestEntryService_GetByID_WrongUser(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	ownerID := uuid.New()
	otherID := uuid.New()

	entry := createEncryptedEntry(t, repo, svc, ownerID)

	_, err := svc.GetByID(context.Background(), entry.ID, otherID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestEntryService_ListByUserID_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	createEncryptedEntry(t, repo, svc, userID)
	createEncryptedEntry(t, repo, svc, userID)
	createEncryptedEntry(t, repo, svc, uuid.New()) // different user

	entries, err := svc.ListByUserID(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestEntryService_ListByUserID_Empty(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	entries, err := svc.ListByUserID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// Verify that decryption uses actual crypto.Decrypt
func TestEntryService_GetByID_DecryptionVerify(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	// Manually create an entry with encrypted data to verify decryption path
	key := []byte(testEncryptionKey)
	encLogin, err := crypto.Encrypt(key, []byte("directlogin"))
	require.NoError(t, err)
	encPass, err := crypto.Encrypt(key, []byte("directpass"))
	require.NoError(t, err)

	entryID := uuid.New()
	repo.entries[entryID] = &model.Entry{
		ID:        entryID,
		UserID:    userID,
		EntryType: model.EntryTypeCredential,
		Name:      "Direct",
		Credential: &model.CredentialData{
			EntryID:           entryID,
			EncryptedLogin:    encLogin,
			EncryptedPassword: encPass,
		},
	}

	result, err := svc.GetByID(context.Background(), entryID, userID)
	require.NoError(t, err)
	assert.Equal(t, "directlogin", result.Credential.Login)
	assert.Equal(t, "directpass", result.Credential.Password)
}
