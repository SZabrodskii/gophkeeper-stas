package model

import (
	"encoding/json"
	"iter"
	"slices"
	"strconv"
	"strings"
	"time"
	"unique"

	"github.com/google/uuid"
)

// EntryType represents the kind of secret stored in an entry.
type EntryType string

// Supported entry types.
const (
	EntryTypeCredential EntryType = "credential"
	EntryTypeText       EntryType = "text"
	EntryTypeBinary     EntryType = "binary"
	EntryTypeCard       EntryType = "card"
)

// EntryTypes returns an iterator over all supported entry types.
func EntryTypes() iter.Seq[EntryType] {
	return slices.Values([]EntryType{
		EntryTypeCredential, EntryTypeText, EntryTypeBinary, EntryTypeCard,
	})
}

// validTypeHandles is a set of interned entry type handles for O(1) lookup.
var validTypeHandles = func() map[unique.Handle[string]]struct{} {
	m := make(map[unique.Handle[string]]struct{}, 4)
	for et := range EntryTypes() {
		m[unique.Make(string(et))] = struct{}{}
	}
	return m
}()

// Valid reports whether t is one of the known entry types.
func (t EntryType) Valid() bool {
	_, ok := validTypeHandles[unique.Make(string(t))]
	return ok
}

// Entry is the core domain object representing a user's secret.
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

// CredentialData holds login/password pair for a credential entry.
type CredentialData struct {
	EntryID           uuid.UUID `json:"-"`
	EncryptedLogin    []byte    `json:"-"`
	EncryptedPassword []byte    `json:"-"`
	Login             string    `json:"login,omitempty"`
	Password          string    `json:"password,omitempty"`
}

// TextData holds free-form text content for a text entry.
type TextData struct {
	EntryID          uuid.UUID `json:"-"`
	EncryptedContent []byte    `json:"-"`
	Content          string    `json:"content,omitempty"`
}

// BinaryData holds an arbitrary file for a binary entry.
type BinaryData struct {
	EntryID          uuid.UUID `json:"-"`
	EncryptedData    []byte    `json:"-"`
	OriginalFilename string    `json:"original_filename,omitempty"`
	Data             string    `json:"data,omitempty"`
}

// CardData holds payment card details for a card entry.
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

// ValidateLuhn checks a card number using the Luhn algorithm.
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

// ValidateExpiry validates card expiry in MM/YY format.
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

// Metadata is a set of arbitrary key-value pairs attached to an entry.
type Metadata map[string]string
