package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EntryType string

const (
	EntryTypeCredential EntryType = "credential"
	EntryTypeText       EntryType = "text"
	EntryTypeBinary     EntryType = "binary"
	EntryTypeCard       EntryType = "card"
)

func (t EntryType) Valid() bool {
	switch t {
	case EntryTypeCredential, EntryTypeText, EntryTypeBinary, EntryTypeCard:
		return true
	}
	return false
}

type Entry struct {
	ID        uuid.UUID        `json:"id"`
	UserID    uuid.UUID        `json:"user_id"`
	EntryType EntryType        `json:"entry_type"`
	Name      string           `json:"name"`
	Metadata  *json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`

	Credential *CredentialData `json:"credential,omitempty"`
	Text       *TextData       `json:"text,omitempty"`
	Binary     *BinaryData     `json:"binary,omitempty"`
	Card       *CardData       `json:"card,omitempty"`
}

type CredentialData struct {
	EntryID           uuid.UUID `json:"-"`
	EncryptedLogin    []byte    `json:"-"`
	EncryptedPassword []byte    `json:"-"`
	Login             string    `json:"login,omitempty"`
	Password          string    `json:"password,omitempty"`
}

type TextData struct {
	EntryID          uuid.UUID `json:"-"`
	EncryptedContent []byte    `json:"-"`
	Content          string    `json:"content,omitempty"`
}

type BinaryData struct {
	EntryID          uuid.UUID `json:"-"`
	EncryptedData    []byte    `json:"-"`
	OriginalFilename string    `json:"original_filename,omitempty"`
	Data             string    `json:"data,omitempty"`
}

type CardData struct {
	EntryID             uuid.UUID `json:"-"`
	EncryptedNumber     []byte    `json:"-"`
	EncryptedExpiry     []byte    `json:"-"`
	EncryptedHolderName []byte    `json:"-"`
	EncryptedCVV        []byte    `json:"-"`
	Number              string    `json:"number,omitempty"`
	Expiry              string    `json:"expiry,omitempty"`
	HolderName          string    `json:"holder_name,omitempty"`
	CVV                 string    `json:"cvv,omitempty"`
}

type Metadata map[string]string
