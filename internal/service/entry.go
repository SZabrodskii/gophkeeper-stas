package service

import (
	"context"
	"errors"
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
	case model.EntryTypeText:
		if err := s.encryptText(entry); err != nil {
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

func (s *EntryService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Entry, error) {
	entry, err := s.entryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get entry: %w", err)
	}

	if entry.UserID != userID {
		return nil, ErrNotFound
	}

	switch entry.EntryType {
	case model.EntryTypeCredential:
		if err := s.decryptCredential(entry); err != nil {
			return nil, err
		}
	case model.EntryTypeText:
		if err := s.decryptText(entry); err != nil {
			return nil, err
		}
	}

	return entry, nil
}

func (s *EntryService) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Entry, error) {
	result, err := s.entryRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}

	return result, nil
}

func (s *EntryService) encryptText(entry *model.Entry) error {
	if entry.Text == nil {
		return fmt.Errorf("%w: text data is required", ErrValidation)
	}
	if entry.Text.Content == "" {
		return fmt.Errorf("%w: content is required for text entry", ErrValidation)
	}

	encContent, err := crypto.Encrypt(s.encryptionKey, []byte(entry.Text.Content))
	if err != nil {
		return fmt.Errorf("encrypt content: %w", err)
	}

	entry.Text.EncryptedContent = encContent
	entry.Text.EntryID = entry.ID

	return nil
}

func (s *EntryService) decryptText(entry *model.Entry) error {
	if entry.Text == nil {
		return nil
	}

	if len(entry.Text.EncryptedContent) > 0 {
		content, err := crypto.Decrypt(s.encryptionKey, entry.Text.EncryptedContent)
		if err != nil {
			return fmt.Errorf("decrypt content: %w", err)
		}
		entry.Text.Content = string(content)
	}

	return nil
}

func (s *EntryService) decryptCredential(entry *model.Entry) error {
	if entry.Credential == nil {
		return nil
	}

	if len(entry.Credential.EncryptedLogin) > 0 {
		login, err := crypto.Decrypt(s.encryptionKey, entry.Credential.EncryptedLogin)
		if err != nil {
			return fmt.Errorf("decrypt login: %w", err)
		}
		entry.Credential.Login = string(login)
	}

	if len(entry.Credential.EncryptedPassword) > 0 {
		pass, err := crypto.Decrypt(s.encryptionKey, entry.Credential.EncryptedPassword)
		if err != nil {
			return fmt.Errorf("decrypt password: %w", err)
		}
		entry.Credential.Password = string(pass)
	}

	return nil

}
