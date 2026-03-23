package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
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
	if _, ok := m.entries[entry.ID]; !ok {
		return repository.ErrNotFound
	}
	entry.UpdatedAt = time.Now()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockEntryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.entries[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.entries, id)
	return nil
}

func (m *mockEntryRepo) ListUpdatedAfter(_ context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	var result []model.Entry
	for _, e := range m.entries {
		if e.UserID == userID && e.UpdatedAt.After(since) {
			result = append(result, *e)
		}
	}
	return result, nil
}

func newTestEntryService(repo *mockEntryRepo) *EntryService {
	return NewEntryServiceFromRaw(repo, testEncryptionKey, 10*1024*1024)
}

func newTestEntryServiceWithMaxBinary(repo *mockEntryRepo, maxSize int64) *EntryService {
	return NewEntryServiceFromRaw(repo, testEncryptionKey, maxSize)
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

func TestEntryService_Create_Text_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeText,
		Name:      "My Note",
		Text: &model.TextData{
			Content: "This is my secret note",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.NotEmpty(t, entry.Text.EncryptedContent)

	stored, ok := repo.entries[entry.ID]
	require.True(t, ok)
	assert.Equal(t, model.EntryTypeText, stored.EntryType)
	assert.Equal(t, "My Note", stored.Name)
}

func TestEntryService_Create_Text_EmptyContent(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeText,
		Name:      "Test",
		Text: &model.TextData{
			Content: "",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Text_MissingData(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeText,
		Name:      "Test",
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Text_EncryptionVerify(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeText,
		Name:      "Encrypted Text",
		Text: &model.TextData{
			Content: "my secret content",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	stored := repo.entries[entry.ID]
	assert.NotEqual(t, []byte("my secret content"), stored.Text.EncryptedContent)
	assert.NotEmpty(t, stored.Text.EncryptedContent)
}

func TestEntryService_GetByID_Text_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	entry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeText,
		Name:      "My Note",
		Text: &model.TextData{
			Content: "round trip content",
		},
	}
	require.NoError(t, svc.Create(context.Background(), entry))

	result, err := svc.GetByID(context.Background(), entry.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, result.ID)
	assert.Equal(t, "round trip content", result.Text.Content)
}

func TestEntryService_Create_Card_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCard,
		Name:      "My Card",
		Card: &model.CardData{
			Number:     "4532015112830366",
			Expiry:     "12/25",
			HolderName: "John Doe",
			CVV:        "123",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.NotEmpty(t, entry.Card.EncryptedNumber)
	assert.NotEmpty(t, entry.Card.EncryptedExpiry)
	assert.NotEmpty(t, entry.Card.EncryptedHolderName)
	assert.NotEmpty(t, entry.Card.EncryptedCVV)

	stored, ok := repo.entries[entry.ID]
	require.True(t, ok)
	assert.Equal(t, model.EntryTypeCard, stored.EntryType)
	assert.Equal(t, "My Card", stored.Name)
}

func TestEntryService_Create_Card_InvalidLuhn(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCard,
		Name:      "Bad Card",
		Card: &model.CardData{
			Number:     "1234567890",
			Expiry:     "12/25",
			HolderName: "John Doe",
			CVV:        "123",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Card_InvalidExpiry(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCard,
		Name:      "Bad Card",
		Card: &model.CardData{
			Number:     "4532015112830366",
			Expiry:     "13/25",
			HolderName: "John Doe",
			CVV:        "123",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Card_MissingData(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCard,
		Name:      "Test",
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Card_EmptyFields(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	tests := []struct {
		name string
		card model.CardData
	}{
		{"empty number", model.CardData{Number: "", Expiry: "12/25", HolderName: "John", CVV: "123"}},
		{"empty expiry", model.CardData{Number: "4532015112830366", Expiry: "", HolderName: "John", CVV: "123"}},
		{"empty holder", model.CardData{Number: "4532015112830366", Expiry: "12/25", HolderName: "", CVV: "123"}},
		{"empty cvv", model.CardData{Number: "4532015112830366", Expiry: "12/25", HolderName: "John", CVV: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &model.Entry{
				UserID:    uuid.New(),
				EntryType: model.EntryTypeCard,
				Name:      "Test",
				Card:      &tt.card,
			}
			err := svc.Create(context.Background(), entry)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestEntryService_Create_Card_EncryptionVerify(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeCard,
		Name:      "Encrypted Card",
		Card: &model.CardData{
			Number:     "4532015112830366",
			Expiry:     "12/25",
			HolderName: "John Doe",
			CVV:        "123",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	stored := repo.entries[entry.ID]
	assert.NotEqual(t, []byte("4532015112830366"), stored.Card.EncryptedNumber)
	assert.NotEqual(t, []byte("12/25"), stored.Card.EncryptedExpiry)
	assert.NotEqual(t, []byte("John Doe"), stored.Card.EncryptedHolderName)
	assert.NotEqual(t, []byte("123"), stored.Card.EncryptedCVV)
	assert.NotEmpty(t, stored.Card.EncryptedNumber)
	assert.NotEmpty(t, stored.Card.EncryptedExpiry)
	assert.NotEmpty(t, stored.Card.EncryptedHolderName)
	assert.NotEmpty(t, stored.Card.EncryptedCVV)
}

func TestEntryService_GetByID_Card_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	entry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeCard,
		Name:      "My Card",
		Card: &model.CardData{
			Number:     "4532015112830366",
			Expiry:     "12/25",
			HolderName: "John Doe",
			CVV:        "123",
		},
	}
	require.NoError(t, svc.Create(context.Background(), entry))

	result, err := svc.GetByID(context.Background(), entry.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, result.ID)
	assert.Equal(t, "4532015112830366", result.Card.Number)
	assert.Equal(t, "12/25", result.Card.Expiry)
	assert.Equal(t, "John Doe", result.Card.HolderName)
	assert.Equal(t, "123", result.Card.CVV)
}

func TestEntryService_GetByID_DecryptionVerify(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

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

func TestEntryService_Create_Binary_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	rawData := []byte("hello binary world")
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeBinary,
		Name:      "My File",
		Binary: &model.BinaryData{
			Data:             b64Data,
			OriginalFilename: "test.bin",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.NotEmpty(t, entry.Binary.EncryptedData)

	stored, ok := repo.entries[entry.ID]
	require.True(t, ok)
	assert.Equal(t, model.EntryTypeBinary, stored.EntryType)
	assert.Equal(t, "My File", stored.Name)
	assert.Equal(t, "test.bin", stored.Binary.OriginalFilename)
}

func TestEntryService_Create_Binary_MissingData(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeBinary,
		Name:      "Test",
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Binary_EmptyData(t *testing.T) {
	svc := newTestEntryService(newMockEntryRepo())

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeBinary,
		Name:      "Test",
		Binary: &model.BinaryData{
			Data: "",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Create_Binary_ExceedsMaxSize(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryServiceWithMaxBinary(repo, 10) // 10 bytes max

	// Create data larger than 10 bytes
	rawData := []byte(strings.Repeat("x", 11))
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeBinary,
		Name:      "Too Big",
		Binary: &model.BinaryData{
			Data:             b64Data,
			OriginalFilename: "big.bin",
		},
	}

	err := svc.Create(context.Background(), entry)
	assert.ErrorIs(t, err, ErrPayloadTooLarge)
}

func TestEntryService_Create_Binary_EncryptionVerify(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	rawData := []byte("secret binary content")
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	entry := &model.Entry{
		UserID:    uuid.New(),
		EntryType: model.EntryTypeBinary,
		Name:      "Encrypted Binary",
		Binary: &model.BinaryData{
			Data:             b64Data,
			OriginalFilename: "secret.bin",
		},
	}

	err := svc.Create(context.Background(), entry)
	require.NoError(t, err)

	stored := repo.entries[entry.ID]
	assert.NotEqual(t, rawData, stored.Binary.EncryptedData)
	assert.NotEmpty(t, stored.Binary.EncryptedData)
}

func TestEntryService_GetByID_Binary_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	rawData := []byte("round trip binary data")
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	entry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeBinary,
		Name:      "My File",
		Binary: &model.BinaryData{
			Data:             b64Data,
			OriginalFilename: "roundtrip.bin",
		},
	}
	require.NoError(t, svc.Create(context.Background(), entry))

	result, err := svc.GetByID(context.Background(), entry.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, result.ID)
	assert.Equal(t, b64Data, result.Binary.Data)
	assert.Equal(t, "roundtrip.bin", result.Binary.OriginalFilename)
}

func TestEntryService_Update_Credential_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	original := createEncryptedEntry(t, repo, svc, userID)

	updated := &model.Entry{
		EntryType: model.EntryTypeCredential,
		Name:      "Updated Website",
		Credential: &model.CredentialData{
			Login:    "newlogin",
			Password: "newpassword",
		},
	}

	err := svc.Update(context.Background(), original.ID, userID, updated)
	require.NoError(t, err)

	assert.Equal(t, original.ID, updated.ID)
	assert.Equal(t, userID, updated.UserID)

	result, err := svc.GetByID(context.Background(), original.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Website", result.Name)
	assert.Equal(t, "newlogin", result.Credential.Login)
	assert.Equal(t, "newpassword", result.Credential.Password)
}

func TestEntryService_Update_Text_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	entry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeText,
		Name:      "Original Note",
		Text:      &model.TextData{Content: "original content"},
	}
	require.NoError(t, svc.Create(context.Background(), entry))

	updated := &model.Entry{
		EntryType: model.EntryTypeText,
		Name:      "Updated Note",
		Text:      &model.TextData{Content: "updated content"},
	}

	err := svc.Update(context.Background(), entry.ID, userID, updated)
	require.NoError(t, err)

	result, err := svc.GetByID(context.Background(), entry.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Note", result.Name)
	assert.Equal(t, "updated content", result.Text.Content)
}

func TestEntryService_Update_NotFound(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	updated := &model.Entry{
		EntryType: model.EntryTypeCredential,
		Name:      "Test",
		Credential: &model.CredentialData{
			Login:    "user",
			Password: "pass",
		},
	}

	err := svc.Update(context.Background(), uuid.New(), uuid.New(), updated)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestEntryService_Update_WrongUser(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	ownerID := uuid.New()
	otherID := uuid.New()

	original := createEncryptedEntry(t, repo, svc, ownerID)

	updated := &model.Entry{
		EntryType: model.EntryTypeCredential,
		Name:      "Hacked",
		Credential: &model.CredentialData{
			Login:    "hacker",
			Password: "hacked",
		},
	}

	err := svc.Update(context.Background(), original.ID, otherID, updated)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestEntryService_Update_TypeMismatch(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	original := createEncryptedEntry(t, repo, svc, userID)

	updated := &model.Entry{
		EntryType: model.EntryTypeText,
		Name:      "Changed Type",
		Text:      &model.TextData{Content: "some text"},
	}

	err := svc.Update(context.Background(), original.ID, userID, updated)
	assert.ErrorIs(t, err, ErrTypeMismatch)
}

func TestEntryService_Update_EmptyName(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	original := createEncryptedEntry(t, repo, svc, userID)

	updated := &model.Entry{
		EntryType: model.EntryTypeCredential,
		Name:      "",
		Credential: &model.CredentialData{
			Login:    "user",
			Password: "pass",
		},
	}

	err := svc.Update(context.Background(), original.ID, userID, updated)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestEntryService_Update_VerifyReEncryption(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	original := createEncryptedEntry(t, repo, svc, userID)
	oldEncLogin := make([]byte, len(original.Credential.EncryptedLogin))
	copy(oldEncLogin, original.Credential.EncryptedLogin)

	updated := &model.Entry{
		EntryType: model.EntryTypeCredential,
		Name:      "Re-encrypted",
		Credential: &model.CredentialData{
			Login:    "newlogin",
			Password: "newpassword",
		},
	}

	err := svc.Update(context.Background(), original.ID, userID, updated)
	require.NoError(t, err)

	stored := repo.entries[original.ID]
	assert.NotEmpty(t, stored.Credential.EncryptedLogin)
	assert.NotEqual(t, oldEncLogin, stored.Credential.EncryptedLogin)

	decLogin, err := crypto.Decrypt([]byte(testEncryptionKey), stored.Credential.EncryptedLogin)
	require.NoError(t, err)
	assert.Equal(t, "newlogin", string(decLogin))
}

func TestEntryService_Delete_Success(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	entry := createEncryptedEntry(t, repo, svc, userID)

	err := svc.Delete(context.Background(), entry.ID, userID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), entry.ID, userID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestEntryService_Delete_NotFound(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestEntryService_Delete_WrongUser(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	ownerID := uuid.New()
	otherID := uuid.New()

	entry := createEncryptedEntry(t, repo, svc, ownerID)

	err := svc.Delete(context.Background(), entry.ID, otherID)
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = svc.GetByID(context.Background(), entry.ID, ownerID)
	assert.NoError(t, err)
}

func TestEntryService_Sync_ReturnsEntriesAfterTimestamp(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	since := time.Now().Add(-time.Hour)

	createEncryptedEntry(t, repo, svc, userID)

	entries, serverTime, err := svc.Sync(context.Background(), userID, since)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.False(t, serverTime.IsZero())
	assert.Equal(t, "mylogin", entries[0].Credential.Login)
	assert.Equal(t, "mypassword", entries[0].Credential.Password)
}

func TestEntryService_Sync_ReturnsEmptyWhenNoChanges(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	createEncryptedEntry(t, repo, svc, userID)

	since := time.Now().Add(time.Hour)
	entries, serverTime, err := svc.Sync(context.Background(), userID, since)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.False(t, serverTime.IsZero())
}

func TestEntryService_Sync_DecryptsAllTypes(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	since := time.Now().Add(-time.Hour)

	createEncryptedEntry(t, repo, svc, userID)

	textEntry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeText,
		Name:      "Note",
		Text:      &model.TextData{Content: "secret note"},
	}
	require.NoError(t, svc.Create(context.Background(), textEntry))

	cardEntry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeCard,
		Name:      "Card",
		Card: &model.CardData{
			Number:     "4532015112830366",
			Expiry:     "12/25",
			HolderName: "John Doe",
			CVV:        "123",
		},
	}
	require.NoError(t, svc.Create(context.Background(), cardEntry))

	rawData := []byte("binary content")
	b64Data := base64.StdEncoding.EncodeToString(rawData)
	binaryEntry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeBinary,
		Name:      "File",
		Binary: &model.BinaryData{
			Data:             b64Data,
			OriginalFilename: "file.bin",
		},
	}
	require.NoError(t, svc.Create(context.Background(), binaryEntry))

	entries, _, err := svc.Sync(context.Background(), userID, since)
	require.NoError(t, err)
	assert.Len(t, entries, 4)

	typeCount := map[model.EntryType]int{}
	for _, e := range entries {
		typeCount[e.EntryType]++
		switch e.EntryType {
		case model.EntryTypeCredential:
			assert.Equal(t, "mylogin", e.Credential.Login)
			assert.Equal(t, "mypassword", e.Credential.Password)
		case model.EntryTypeText:
			assert.Equal(t, "secret note", e.Text.Content)
		case model.EntryTypeCard:
			assert.Equal(t, "4532015112830366", e.Card.Number)
			assert.Equal(t, "12/25", e.Card.Expiry)
			assert.Equal(t, "John Doe", e.Card.HolderName)
			assert.Equal(t, "123", e.Card.CVV)
		case model.EntryTypeBinary:
			assert.Equal(t, b64Data, e.Binary.Data)
			assert.Equal(t, "file.bin", e.Binary.OriginalFilename)
		}
	}
	assert.Equal(t, 1, typeCount[model.EntryTypeCredential])
	assert.Equal(t, 1, typeCount[model.EntryTypeText])
	assert.Equal(t, 1, typeCount[model.EntryTypeCard])
	assert.Equal(t, 1, typeCount[model.EntryTypeBinary])
}

func TestEntryService_Sync_LWW_ReturnsLastUpdate(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	// Create entry
	entry := &model.Entry{
		UserID:    userID,
		EntryType: model.EntryTypeCredential,
		Name:      "Original",
		Credential: &model.CredentialData{
			Login:    "original_login",
			Password: "original_pass",
		},
	}
	require.NoError(t, svc.Create(context.Background(), entry))
	entryID := entry.ID

	// First update
	time.Sleep(time.Millisecond)
	update1 := &model.Entry{
		EntryType: model.EntryTypeCredential,
		Name:      "First Update",
		Credential: &model.CredentialData{
			Login:    "first_login",
			Password: "first_pass",
		},
	}
	require.NoError(t, svc.Update(context.Background(), entryID, userID, update1))

	time.Sleep(time.Millisecond)
	since := time.Now()
	time.Sleep(time.Millisecond)

	update2 := &model.Entry{
		EntryType: model.EntryTypeCredential,
		Name:      "Second Update",
		Credential: &model.CredentialData{
			Login:    "second_login",
			Password: "second_pass",
		},
	}
	require.NoError(t, svc.Update(context.Background(), entryID, userID, update2))

	entries, _, err := svc.Sync(context.Background(), userID, since)
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "Second Update", entries[0].Name)
	assert.Equal(t, "second_login", entries[0].Credential.Login)
	assert.Equal(t, "second_pass", entries[0].Credential.Password)
}

func TestEntryService_Sync_ReturnsServerTime(t *testing.T) {
	repo := newMockEntryRepo()
	svc := newTestEntryService(repo)
	userID := uuid.New()

	before := time.Now()
	_, serverTime, err := svc.Sync(context.Background(), userID, time.Time{})
	require.NoError(t, err)
	after := time.Now()

	assert.True(t, !serverTime.Before(before), "serverTime should be >= before")
	assert.True(t, !serverTime.After(after), "serverTime should be <= after")
}
