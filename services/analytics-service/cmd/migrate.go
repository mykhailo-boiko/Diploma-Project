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
		version: "000001_create_analytics_tables",
		sql: `
			CREATE TABLE IF NOT EXISTS analytics.sales_daily (
				id             UUID PRIMARY KEY,
				date           DATE NOT NULL UNIQUE,
				total_orders   INT NOT NULL DEFAULT 0,
				total_revenue  NUMERIC(14,2) NOT NULL DEFAULT 0,
				avg_order_size NUMERIC(14,2) NOT NULL DEFAULT 0,
				created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_sales_daily_date ON analytics.sales_daily (date);

			CREATE TABLE IF NOT EXISTS analytics.inventory_snapshot (
				id              UUID PRIMARY KEY,
				date            DATE NOT NULL UNIQUE,
				total_products  INT NOT NULL DEFAULT 0,
				total_quantity  INT NOT NULL DEFAULT 0,
				total_reserved  INT NOT NULL DEFAULT 0,
				total_available INT NOT NULL DEFAULT 0,
				low_stock_count INT NOT NULL DEFAULT 0,
				created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_inventory_snapshot_date ON analytics.inventory_snapshot (date);

			CREATE TABLE IF NOT EXISTS analytics.logistics_daily (
				id                 UUID PRIMARY KEY,
				date               DATE NOT NULL UNIQUE,
				total_shipments    INT NOT NULL DEFAULT 0,
				delivered_count    INT NOT NULL DEFAULT 0,
				failed_count       INT NOT NULL DEFAULT 0,
				avg_delivery_hours NUMERIC(10,2) NOT NULL DEFAULT 0,
				on_time_rate       NUMERIC(5,2) NOT NULL DEFAULT 0,
				created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_logistics_daily_date ON analytics.logistics_daily (date);
		`,
	},
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS analytics.schema_migrations (
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
			"SELECT EXISTS(SELECT 1 FROM analytics.schema_migrations WHERE version = $1)",
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

		if _, err := tx.Exec(ctx, "INSERT INTO analytics.schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.version, err)
		}
	}

	return nil
}
