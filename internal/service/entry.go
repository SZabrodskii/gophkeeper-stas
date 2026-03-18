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
	case model.EntryTypeCard:
		if err := s.encryptCard(entry); err != nil {
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
	case model.EntryTypeCard:
		if err := s.decryptCard(entry); err != nil {
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

func (s *EntryService) encryptCard(entry *model.Entry) error {
	if entry.Card == nil {
		return fmt.Errorf("%w: card data is required", ErrValidation)
	}
	if entry.Card.Number == "" {
		return fmt.Errorf("%w: number is required for card entry", ErrValidation)
	}
	if !model.ValidateLuhn(entry.Card.Number) {
		return fmt.Errorf("%w: invalid card number", ErrValidation)
	}
	if entry.Card.Expiry == "" {
		return fmt.Errorf("%w: expiry is required for card entry", ErrValidation)
	}
	if !model.ValidateExpiry(entry.Card.Expiry) {
		return fmt.Errorf("%w: invalid card expiry", ErrValidation)
	}
	if entry.Card.HolderName == "" {
		return fmt.Errorf("%w: holder name is required for card entry", ErrValidation)
	}
	if entry.Card.CVV == "" {
		return fmt.Errorf("%w: CVV is required for card entry", ErrValidation)
	}

	encNumber, err := crypto.Encrypt(s.encryptionKey, []byte(entry.Card.Number))
	if err != nil {
		return fmt.Errorf("encrypt number: %w", err)
	}
	encExpiry, err := crypto.Encrypt(s.encryptionKey, []byte(entry.Card.Expiry))
	if err != nil {
		return fmt.Errorf("encrypt expiry: %w", err)
	}
	encHolderName, err := crypto.Encrypt(s.encryptionKey, []byte(entry.Card.HolderName))
	if err != nil {
		return fmt.Errorf("encrypt holder name: %w", err)
	}
	encCVV, err := crypto.Encrypt(s.encryptionKey, []byte(entry.Card.CVV))
	if err != nil {
		return fmt.Errorf("encrypt CVV: %w", err)
	}

	entry.Card.EncryptedNumber = encNumber
	entry.Card.EncryptedExpiry = encExpiry
	entry.Card.EncryptedHolderName = encHolderName
	entry.Card.EncryptedCVV = encCVV
	entry.Card.EntryID = entry.ID

	return nil
}

func (s *EntryService) decryptCard(entry *model.Entry) error {
	if entry.Card == nil {
		return nil
	}
	if len(entry.Card.EncryptedNumber) > 0 {
		number, err := crypto.Decrypt(s.encryptionKey, entry.Card.EncryptedNumber)
		if err != nil {
			return fmt.Errorf("decrypt number: %w", err)
		}
		entry.Card.Number = string(number)

		expiry, err := crypto.Decrypt(s.encryptionKey, entry.Card.EncryptedExpiry)
		if err != nil {
			return fmt.Errorf("decrypt expiry: %w", err)
		}
		entry.Card.Expiry = string(expiry)

		holderName, err := crypto.Decrypt(s.encryptionKey, entry.Card.EncryptedHolderName)
		if err != nil {
			return fmt.Errorf("decrypt holder name: %w", err)
		}
		entry.Card.HolderName = string(holderName)

		cvv, err := crypto.Decrypt(s.encryptionKey, entry.Card.EncryptedCVV)
		if err != nil {
			return fmt.Errorf("decrypt CVV: %w", err)
		}
		entry.Card.CVV = string(cvv)

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
