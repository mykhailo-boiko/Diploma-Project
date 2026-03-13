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
		version: "000001_create_notification_table",
		sql: `
			CREATE TABLE IF NOT EXISTS notifications.notification (
				id         UUID PRIMARY KEY,
				user_id    VARCHAR(255) NOT NULL,
				type       VARCHAR(50)  NOT NULL,
				title      VARCHAR(500) NOT NULL,
				message    TEXT         NOT NULL,
				status     VARCHAR(20)  NOT NULL DEFAULT 'pending',
				read_at    TIMESTAMPTZ,
				created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_notification_user_id ON notifications.notification (user_id);
			CREATE INDEX IF NOT EXISTS idx_notification_user_status ON notifications.notification (user_id, status);
			CREATE INDEX IF NOT EXISTS idx_notification_type ON notifications.notification (type);
			CREATE INDEX IF NOT EXISTS idx_notification_created_at ON notifications.notification (created_at);
		`,
	},
	{
		version: "000002_create_preference_table",
		sql: `
			CREATE TABLE IF NOT EXISTS notifications.notification_preference (
				id         UUID PRIMARY KEY,
				user_id    VARCHAR(255) NOT NULL,
				type       VARCHAR(50)  NOT NULL,
				in_app     BOOLEAN NOT NULL DEFAULT TRUE,
				email      BOOLEAN NOT NULL DEFAULT TRUE,
				sms        BOOLEAN NOT NULL DEFAULT FALSE,
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE(user_id, type)
			);

			CREATE INDEX IF NOT EXISTS idx_notification_preference_user_id ON notifications.notification_preference (user_id);
		`,
	},
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notifications.schema_migrations (
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
			"SELECT EXISTS(SELECT 1 FROM notifications.schema_migrations WHERE version = $1)",
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

		if _, err := tx.Exec(ctx, "INSERT INTO notifications.schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.version, err)
		}
	}

	return nil
}
