package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
	"github.com/SZabrodskii/gophkeeper-stas/internal/crypto"
	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
)

var EntryModule = fx.Module("service.entry",
	fx.Provide(NewEntryService),
)

type entryServiceParams struct {
	fx.In

	EntryRepo  repository.EntryRepository
	AuthConfig config.AuthConfig
}

type EntryService struct {
	entryRepo     repository.EntryRepository
	encryptionKey []byte
}

func NewEntryService(params entryServiceParams) *EntryService {
	return &EntryService{
		entryRepo:     params.EntryRepo,
		encryptionKey: []byte(params.AuthConfig.EncryptionKey),
	}
}

func NewEntryServiceFromRaw(entryRepo repository.EntryRepository, encryptionKey string) *EntryService {
	return &EntryService{
		entryRepo:     entryRepo,
		encryptionKey: []byte(encryptionKey),
	}
}

func (s *EntryService) Create(ctx context.Context, entry *model.Entry) error {
	if entry.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if !entry.EntryType.Valid() {
		return fmt.Errorf("%w: invalid entry type", ErrValidation)
	}

	entry.ID = uuid.New()

	switch entry.EntryType {
	case model.EntryTypeCredential:
		if err := s.encryptCredential(entry); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unsupported entry type: %s", ErrValidation, entry.EntryType)
	}

	if err := s.entryRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("create entry: %w", err)
	}

	return nil
}

func (s *EntryService) encryptCredential(entry *model.Entry) error {
	if entry.Credential == nil {
		return fmt.Errorf("%w: credential data is required", ErrValidation)
	}
	if entry.Credential.Login == "" {
		return fmt.Errorf("%w: login is required for credential entry", ErrValidation)
	}
	if entry.Credential.Password == "" {
		return fmt.Errorf("%w: password is required for credential entry", ErrValidation)
	}

	encLogin, err := crypto.Encrypt(s.encryptionKey, []byte(entry.Credential.Login))
	if err != nil {
		return fmt.Errorf("encrypt login: %w", err)
	}
	encPassword, err := crypto.Encrypt(s.encryptionKey, []byte(entry.Credential.Password))
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}

	entry.Credential.EncryptedLogin = encLogin
	entry.Credential.EncryptedPassword = encPassword
	entry.Credential.EntryID = entry.ID

	return nil
}
