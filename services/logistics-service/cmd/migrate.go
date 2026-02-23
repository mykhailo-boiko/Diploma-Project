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
		version: "000001_create_logistics_tables",
		sql: `
			CREATE TABLE IF NOT EXISTS logistics.carrier (
				id         UUID PRIMARY KEY,
				name       VARCHAR(255) NOT NULL,
				type       VARCHAR(50)  NOT NULL CHECK (type IN ('ground', 'air', 'sea')),
				cost_per_km NUMERIC(10,2) NOT NULL CHECK (cost_per_km > 0),
				is_active  BOOLEAN NOT NULL DEFAULT true,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_carrier_type ON logistics.carrier (type);
			CREATE INDEX IF NOT EXISTS idx_carrier_is_active ON logistics.carrier (is_active);

			CREATE TABLE IF NOT EXISTS logistics.shipment (
				id           UUID PRIMARY KEY,
				order_id     VARCHAR(255) NOT NULL,
				warehouse_id VARCHAR(255) NOT NULL,
				carrier_id   UUID NOT NULL REFERENCES logistics.carrier(id),
				status       VARCHAR(50)  NOT NULL DEFAULT 'created',
				address      TEXT NOT NULL,
				created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
				deleted_at   TIMESTAMPTZ
			);

			CREATE INDEX IF NOT EXISTS idx_shipment_status ON logistics.shipment (status) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_shipment_order_id ON logistics.shipment (order_id) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_shipment_carrier_id ON logistics.shipment (carrier_id) WHERE deleted_at IS NULL;
			CREATE INDEX IF NOT EXISTS idx_shipment_created_at ON logistics.shipment (created_at) WHERE deleted_at IS NULL;

			CREATE TABLE IF NOT EXISTS logistics.route (
				id          UUID PRIMARY KEY,
				shipment_id UUID NOT NULL REFERENCES logistics.shipment(id) ON DELETE CASCADE,
				origin      VARCHAR(255) NOT NULL,
				destination VARCHAR(255) NOT NULL,
				distance_km NUMERIC(10,2),
				duration_h  NUMERIC(10,2),
				cost        NUMERIC(12,2),
				created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_route_shipment_id ON logistics.route (shipment_id);
		`,
	},
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS logistics.schema_migrations (
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
			"SELECT EXISTS(SELECT 1 FROM logistics.schema_migrations WHERE version = $1)",
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

		if _, err := tx.Exec(ctx, "INSERT INTO logistics.schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.version, err)
		}
	}

	return nil
}
