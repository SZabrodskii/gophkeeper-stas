package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
)

var EntryModule = fx.Module("repository.entry",
	fx.Provide(NewPostgresEntryRepository),
)

type EntryRepositoryOut struct {
	fx.Out

	Repo EntryRepository
}

type PostgresEntryRepository struct {
	db *sql.DB
}

func NewPostgresEntryRepository(db *sql.DB) EntryRepositoryOut {
	return EntryRepositoryOut{
		Repo: newPostgresEntryRepository(db),
	}
}

func newPostgresEntryRepository(db *sql.DB) *PostgresEntryRepository {
	return &PostgresEntryRepository{db: db}
}

func (r *PostgresEntryRepository) Create(ctx context.Context, entry *model.Entry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var metadataBytes []byte
	if entry.Metadata != nil {
		metadataBytes = []byte(*entry.Metadata)
	}

	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	query := `INSERT INTO entries (id, user_id, entry_type, name, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.ExecContext(ctx, query,
		entry.ID, entry.UserID, entry.EntryType, entry.Name, metadataBytes, entry.CreatedAt, entry.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert entry: %w", err)
	}

	switch entry.EntryType {
	case model.EntryTypeCredential:
		if entry.Credential == nil {
			return fmt.Errorf("credential data is required for credential entry type")
		}
		credQuery := `INSERT INTO credential_data (entry_id, encrypted_login, encrypted_password)
			VALUES ($1, $2, $3)`
		_, err = tx.ExecContext(ctx, credQuery,
			entry.ID, entry.Credential.EncryptedLogin, entry.Credential.EncryptedPassword)
		if err != nil {
			return fmt.Errorf("insert credential data: %w", err)
		}
	case model.EntryTypeText:
		if entry.Text == nil {
			return fmt.Errorf("text data is required for text entry type")
		}
		textQuery := `INSERT INTO text_data (entry_id, encrypted_content)
			VALUES ($1, $2)`
		_, err = tx.ExecContext(ctx, textQuery,
			entry.ID, entry.Text.EncryptedContent)
		if err != nil {
			return fmt.Errorf("insert text data: %w", err)
		}
	case model.EntryTypeCard:
		if entry.Card == nil {
			return fmt.Errorf("card data is required for card entry type")
		}
		cardQuery := `INSERT INTO card_data (entry_id, encrypted_number, encrypted_expiry, encrypted_holder_name, encrypted_cvv)
          VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.ExecContext(ctx, cardQuery,
			entry.ID, entry.Card.EncryptedNumber, entry.Card.EncryptedExpiry,
			entry.Card.EncryptedHolderName, entry.Card.EncryptedCVV)
		if err != nil {
			return fmt.Errorf("insert card data: %w", err)
		}
	case model.EntryTypeBinary:
		if entry.Binary == nil {
			return fmt.Errorf("binary data is required for binary entry type")
		}
		binaryQuery := `INSERT INTO binary_data (entry_id, encrypted_data, original_filename)  VALUES ($1, $2, $3)`
		_, err = tx.ExecContext(ctx, binaryQuery, entry.ID, entry.Binary.EncryptedData, entry.Binary.OriginalFilename)
		if err != nil {
			return fmt.Errorf("insert binary data: %w", err)
		}
	default:
		return fmt.Errorf("unsupported entry type: %s", entry.EntryType)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *PostgresEntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error) {
	query := `SELECT id, user_id, entry_type, name, metadata, created_at, updated_at
		FROM entries WHERE id = $1`

	var entry model.Entry
	var metadataBytes []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID, &entry.UserID, &entry.EntryType, &entry.Name,
		&metadataBytes, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get entry: %w", err)
	}

	if metadataBytes != nil {
		raw := json.RawMessage(metadataBytes)
		entry.Metadata = &raw
	}

	switch entry.EntryType {
	case model.EntryTypeCredential:
		cred, err := r.getCredentialData(ctx, entry.ID)
		if err != nil {
			return nil, fmt.Errorf("get credential data: %w", err)
		}
		entry.Credential = cred
	case model.EntryTypeText:
		text, err := r.getTextData(ctx, entry.ID)
		if err != nil {
			return nil, fmt.Errorf("get text data: %w", err)
		}
		entry.Text = text
	case model.EntryTypeCard:
		card, err := r.getCardData(ctx, entry.ID)
		if err != nil {
			return nil, fmt.Errorf("get card data: %w", err)
		}
		entry.Card = card
	case model.EntryTypeBinary:
		binary, err := r.getBinaryData(ctx, entry.ID)
		if err != nil {
			return nil, fmt.Errorf("get binary data: %w", err)
		}
		entry.Binary = binary
	}

	return &entry, nil
}

func (r *PostgresEntryRepository) getCredentialData(ctx context.Context, entryID uuid.UUID) (*model.CredentialData, error) {
	query := `SELECT entry_id, encrypted_login, encrypted_password FROM credential_data WHERE entry_id = $1`
	var cred model.CredentialData
	err := r.db.QueryRowContext(ctx, query, entryID).Scan(
		&cred.EntryID, &cred.EncryptedLogin, &cred.EncryptedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get credential data: %w", err)
	}
	return &cred, nil
}

func (r *PostgresEntryRepository) getTextData(ctx context.Context, entryID uuid.UUID) (*model.TextData, error) {
	query := `SELECT entry_id, encrypted_content FROM text_data WHERE entry_id = $1`
	var text model.TextData
	err := r.db.QueryRowContext(ctx, query, entryID).Scan(
		&text.EntryID, &text.EncryptedContent)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get text data: %w", err)
	}
	return &text, nil
}

func (r *PostgresEntryRepository) getCardData(ctx context.Context, entryID uuid.UUID) (*model.CardData, error) {
	query := `SELECT entry_id, encrypted_number, encrypted_expiry, encrypted_holder_name, encrypted_cvv FROM card_data WHERE entry_id = $1`
	var card model.CardData
	err := r.db.QueryRowContext(ctx, query, entryID).Scan(
		&card.EntryID, &card.EncryptedNumber, &card.EncryptedExpiry,
		&card.EncryptedHolderName, &card.EncryptedCVV)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get card data: %w", err)
	}
	return &card, nil
}

func (r *PostgresEntryRepository) getBinaryData(ctx context.Context, entryID uuid.UUID) (*model.BinaryData, error) {
	query := `SELECT entry_id, encrypted_data, original_filename FROM binary_data WHERE entry_id = $1`
	var binary model.BinaryData
	err := r.db.QueryRowContext(ctx, query, entryID).Scan(
		&binary.EntryID, &binary.EncryptedData, &binary.OriginalFilename)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get binary data: %w", err)
	}
	return &binary, nil
}

func (r *PostgresEntryRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Entry, error) {
	query := `SELECT id, user_id, entry_type, name, metadata, created_at, updated_at
		FROM entries WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	var entries []model.Entry
	for rows.Next() {
		var entry model.Entry
		var metadataBytes []byte
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.EntryType, &entry.Name,
			&metadataBytes, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		if metadataBytes != nil {
			raw := json.RawMessage(metadataBytes)
			entry.Metadata = &raw
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *PostgresEntryRepository) Update(ctx context.Context, entry *model.Entry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var metadataByte []byte
	if entry.Metadata != nil {
		metadataByte = []byte(*entry.Metadata)
	}
	entry.UpdatedAt = time.Now()

	query := `UPDATE entries SET name = $1, metadata = $2, updated_at = $3 WHERE id = $4`
	res, err := tx.ExecContext(ctx, query, entry.Name, metadataByte, entry.UpdatedAt, entry.ID)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	switch entry.EntryType {
	case model.EntryTypeCredential:
		if entry.Credential == nil {
			return fmt.Errorf("credential data is required for credential entry type")
		}
		q := `UPDATE credential_data SET encrypted_login = $1, encrypted_password = $2 WHERE entry_id = $3`
		_, err = tx.ExecContext(ctx, q, entry.Credential.EncryptedLogin, entry.Credential.EncryptedPassword, entry.ID)
	case model.EntryTypeText:
		if entry.Text == nil {
			return fmt.Errorf("text data is required for text entry type")
		}
		q := `UPDATE text_data SET encrypted_content = $1 WHERE entry_id = $2`
		_, err = tx.ExecContext(ctx, q, entry.Text.EncryptedContent, entry.ID)
	case model.EntryTypeCard:
		if entry.Card == nil {
			return fmt.Errorf("card data is required for card entry type")
		}
		q := `UPDATE card_data SET encrypted_number = $1, encrypted_expiry = $2, encrypted_holder_name = $3, encrypted_cvv = $4 WHERE entry_id = $5`
		_, err = tx.ExecContext(ctx, q, entry.Card.EncryptedNumber, entry.Card.EncryptedExpiry, entry.Card.EncryptedHolderName, entry.Card.EncryptedCVV, entry.ID)
	case model.EntryTypeBinary:
		if entry.Binary == nil {
			return fmt.Errorf("binary data is required for binary entry type")
		}
		q := `UPDATE binary_data SET encrypted_data = $1, original_filename = $2 WHERE entry_id = $3`
		_, err = tx.ExecContext(ctx, q, entry.Binary.EncryptedData, entry.Binary.OriginalFilename, entry.ID)
	default:
		return fmt.Errorf("unsupported entry type: %s", entry.EntryType)
	}

	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil

}

func (r *PostgresEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM entries WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresEntryRepository) ListUpdatedAfter(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	return nil, fmt.Errorf("todo")
}
