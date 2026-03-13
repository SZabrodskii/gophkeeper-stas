package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
)

// Module предоставляет *sql.DB в DI-контейнер.
var Module = fx.Module("db",
	fx.Provide(NewDB),
)

func NewDB(lc fx.Lifecycle, cfg config.DBConfig, logger *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("ping db: %w", err)
			}
			logger.Info("database connected")

			if err := runMigrations(db, logger); err != nil {
				return fmt.Errorf("run migrations: %w", err)
			}
			return nil
		},
		OnStop: func(_ context.Context) error {
			logger.Info("closing database connection")
			return db.Close()
		},
	})

	return db, nil
}

func runMigrations(db *sql.DB, logger *zap.Logger) error {
	entries, err := os.ReadDir("migrations")
	if err != nil || len(entries) == 0 {
		logger.Info("no migration files found, skipping")
		return nil
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}

	logger.Info("migrations applied")
	return nil
}
