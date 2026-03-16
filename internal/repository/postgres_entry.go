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
	return fmt.Errorf("todo")
}

func (r *PostgresEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("todo")
}

func (r *PostgresEntryRepository) ListUpdatedAfter(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	return nil, fmt.Errorf("todo")
}
