package model

import (
	"encoding/json"
	"strconv"
	"strings"
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

func ValidateLuhn(number string) bool {
	cleaned := strings.ReplaceAll(number, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	if len(cleaned) < 2 {
		return false
	}

	for _, r := range cleaned {
		if r < '0' || r > '9' {
			return false
		}
	}

	var sum int
	var alternate bool
	for i := len(cleaned) - 1; i >= 0; i-- {
		digit := int(cleaned[i] - '0')
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alternate = !alternate

	}
	return sum%10 == 0
}

func ValidateExpiry(expiry string) bool {
	parts := strings.Split(expiry, "/")
	if len(parts) != 2 {
		return false
	}

	if len(parts[0]) != 2 {
		return false
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return false
	}
	if len(parts[1]) != 2 {
		return false
	}

	_, err = strconv.Atoi(parts[1])

	return err == nil
}

type Metadata map[string]string
