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
		version: "000001_create_products_table",
		sql: `
			CREATE TABLE IF NOT EXISTS inventory.product (
				id          UUID PRIMARY KEY,
				sku         VARCHAR(100) NOT NULL,
				name        VARCHAR(255) NOT NULL,
				description TEXT NOT NULL DEFAULT '',
				category    VARCHAR(100) NOT NULL DEFAULT '',
				unit_price  NUMERIC(12,2) NOT NULL DEFAULT 0,
				created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				deleted_at  TIMESTAMPTZ
			);

			CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sku ON inventory.product (sku) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_product_category ON inventory.product (category) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_product_name ON inventory.product (name) WHERE deleted_at IS NULL;
		`,
	},
	{
		version: "000002_create_warehouse_table",
		sql: `
			CREATE TABLE IF NOT EXISTS inventory.warehouse (
				id         UUID PRIMARY KEY,
				name       VARCHAR(255) NOT NULL,
				address    TEXT NOT NULL DEFAULT '',
				is_active  BOOLEAN NOT NULL DEFAULT TRUE,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_warehouse_name ON inventory.warehouse (name);
		`,
	},
	{
		version: "000003_create_stock_table",
		sql: `
			CREATE TABLE IF NOT EXISTS inventory.stock (
				id           UUID PRIMARY KEY,
				product_id   UUID NOT NULL REFERENCES inventory.product(id),
				warehouse_id UUID NOT NULL REFERENCES inventory.warehouse(id),
				quantity     INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
				reserved     INT NOT NULL DEFAULT 0 CHECK (reserved >= 0),
				updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_product_warehouse
				ON inventory.stock (product_id, warehouse_id);
			CREATE INDEX IF NOT EXISTS idx_stock_product_id ON inventory.stock (product_id);
			CREATE INDEX IF NOT EXISTS idx_stock_warehouse_id ON inventory.stock (warehouse_id);
		`,
	},
	{
		version: "000004_create_stock_movement_table",
		sql: `
			CREATE TABLE IF NOT EXISTS inventory.stock_movement (
				id           UUID PRIMARY KEY,
				stock_id     UUID NOT NULL REFERENCES inventory.stock(id),
				product_id   UUID NOT NULL,
				warehouse_id UUID NOT NULL,
				type         VARCHAR(50) NOT NULL,
				quantity     INT NOT NULL,
				reference    VARCHAR(255) NOT NULL DEFAULT '',
				created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_stock_movement_stock_id ON inventory.stock_movement (stock_id);
			CREATE INDEX IF NOT EXISTS idx_stock_movement_created_at ON inventory.stock_movement (created_at);
		`,
	},
	{
		version: "000005_add_stock_min_threshold",
		sql: `
			ALTER TABLE inventory.stock ADD COLUMN IF NOT EXISTS min_threshold INT NOT NULL DEFAULT 0;
			CREATE INDEX IF NOT EXISTS idx_stock_movement_type ON inventory.stock_movement (type);
		`,
	},
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory.schema_migrations (
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
			"SELECT EXISTS(SELECT 1 FROM inventory.schema_migrations WHERE version = $1)",
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

		if _, err := tx.Exec(ctx, "INSERT INTO inventory.schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.version, err)
		}
	}

	return nil
}
