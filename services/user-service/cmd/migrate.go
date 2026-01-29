package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var migrations = []struct {
	version string
	sql     string
}{
	{
		version: "000001_create_users_table",
		sql: `
			CREATE TABLE IF NOT EXISTS users.users (
				id          UUID PRIMARY KEY,
				email       VARCHAR(255) NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				first_name  VARCHAR(100) NOT NULL,
				last_name   VARCHAR(100) NOT NULL,
				role        VARCHAR(50)  NOT NULL DEFAULT 'operator',
				created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				deleted_at  TIMESTAMPTZ
			);

			CREATE INDEX IF NOT EXISTS idx_users_email ON users.users (email) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_users_role ON users.users (role) WHERE deleted_at IS NULL;
		`,
	},
	{
		version: "000002_add_password_reset_tokens",
		sql: `
			CREATE TABLE IF NOT EXISTS users.password_reset_tokens (
				id         UUID PRIMARY KEY,
				user_id    UUID NOT NULL REFERENCES users.users(id),
				token      VARCHAR(255) NOT NULL UNIQUE,
				expires_at TIMESTAMPTZ NOT NULL,
				used_at    TIMESTAMPTZ,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token
				ON users.password_reset_tokens (token) WHERE used_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id
				ON users.password_reset_tokens (user_id);
		`,
	},
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users.schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM users.schema_migrations WHERE version = $1)",
			m.version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", m.version, err)
		}
		if exists {
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", m.version, err)
		}

		if _, err := tx.Exec(ctx, m.sql); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", m.version, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO users.schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.version, err)
		}
	}

	return nil
}
