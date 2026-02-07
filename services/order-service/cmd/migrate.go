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
		version: "000001_create_orders_tables",
		sql: `
			CREATE TABLE IF NOT EXISTS orders.orders (
				id            UUID PRIMARY KEY,
				customer_name VARCHAR(255) NOT NULL,
				status        VARCHAR(50)  NOT NULL DEFAULT 'pending',
				total_amount  NUMERIC(12,2) NOT NULL DEFAULT 0,
				created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				deleted_at    TIMESTAMPTZ
			);

			CREATE INDEX IF NOT EXISTS idx_orders_status ON orders.orders (status) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_orders_customer_name ON orders.orders (customer_name) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders.orders (created_at) WHERE deleted_at IS NULL;

			CREATE TABLE IF NOT EXISTS orders.order_items (
				id         UUID PRIMARY KEY,
				order_id   UUID NOT NULL REFERENCES orders.orders(id) ON DELETE CASCADE,
				product_id VARCHAR(255) NOT NULL,
				name       VARCHAR(255) NOT NULL,
				quantity   INT NOT NULL CHECK (quantity > 0),
				unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price > 0),
				subtotal   NUMERIC(12,2) NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON orders.order_items (order_id);
		`,
	},
	{
		version: "000002_add_cancel_reason_and_search",
		sql: `
			ALTER TABLE orders.orders ADD COLUMN IF NOT EXISTS cancel_reason TEXT;

			CREATE EXTENSION IF NOT EXISTS pg_trgm;

			CREATE INDEX IF NOT EXISTS idx_orders_customer_name_trgm
				ON orders.orders USING gin (customer_name gin_trgm_ops)
				WHERE deleted_at IS NULL;

			CREATE INDEX IF NOT EXISTS idx_orders_id_text
				ON orders.orders (CAST(id AS TEXT))
				WHERE deleted_at IS NULL;
		`,
	},
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS orders.schema_migrations (
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
			"SELECT EXISTS(SELECT 1 FROM orders.schema_migrations WHERE version = $1)",
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

		if _, err := tx.Exec(ctx, "INSERT INTO orders.schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.version, err)
		}
	}

	return nil
}
