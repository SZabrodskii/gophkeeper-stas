package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
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
		return nil, ErrNotFound
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
