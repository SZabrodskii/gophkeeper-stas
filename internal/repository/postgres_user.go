package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
)

// UserModule provides the UserRepository via fx DI.
var UserModule = fx.Module("repository.user",
	fx.Provide(NewPostgresUserRepository),
)

const pgUniqueViolation = "23505"

// UserRepositoryOut wraps UserRepository for fx dependency injection.
type UserRepositoryOut struct {
	fx.Out

	Repo UserRepository
}

// PostgresUserRepository implements UserRepository backed by PostgreSQL.
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a PostgreSQL-backed UserRepository.
func NewPostgresUserRepository(db *sql.DB) UserRepositoryOut {
	return UserRepositoryOut{
		Repo: newPostgresUserRepository(db),
	}
}

func newPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (id, login, password_hash, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Login, user.PasswordHash, user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	query := `SELECT id, login, password_hash, created_at FROM users WHERE login = $1`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, login).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by login: %w", err)
	}
	return &user, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `SELECT id, login, password_hash, created_at FROM users WHERE id = $1`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}
