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
	{
		version: "000002_postal_tracking_fields",
		sql: `
			ALTER TABLE logistics.shipment
				ADD COLUMN IF NOT EXISTS tracking_number       VARCHAR(32),
				ADD COLUMN IF NOT EXISTS recipient              JSONB NOT NULL DEFAULT '{}'::jsonb,
				ADD COLUMN IF NOT EXISTS sender                 JSONB NOT NULL DEFAULT '{}'::jsonb,
				ADD COLUMN IF NOT EXISTS estimated_delivery_at  TIMESTAMPTZ,
				ADD COLUMN IF NOT EXISTS delivered_at           TIMESTAMPTZ,
				ADD COLUMN IF NOT EXISTS delivery_attempts      INT NOT NULL DEFAULT 0,
				ADD COLUMN IF NOT EXISTS delivery_signature     VARCHAR(255),
				ADD COLUMN IF NOT EXISTS delivery_photo_url     TEXT,
				ADD COLUMN IF NOT EXISTS current_location_city  VARCHAR(100),
				ADD COLUMN IF NOT EXISTS current_location_hub   VARCHAR(100);

			UPDATE logistics.shipment
			SET tracking_number = 'CO-' ||
			    EXTRACT(YEAR FROM created_at)::text || '-' ||
			    UPPER(SUBSTR(MD5(id::text), 1, 6))
			WHERE tracking_number IS NULL;

			ALTER TABLE logistics.shipment
				ALTER COLUMN tracking_number SET NOT NULL;

			CREATE UNIQUE INDEX IF NOT EXISTS uniq_shipment_tracking_number
				ON logistics.shipment (tracking_number);

			CREATE INDEX IF NOT EXISTS idx_shipment_estimated_delivery
				ON logistics.shipment (estimated_delivery_at) WHERE deleted_at IS NULL;

			CREATE INDEX IF NOT EXISTS idx_shipment_recipient_city
				ON logistics.shipment ((recipient->>'city')) WHERE deleted_at IS NULL;

			CREATE TABLE IF NOT EXISTS logistics.shipment_event (
				id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				shipment_id   UUID NOT NULL REFERENCES logistics.shipment(id) ON DELETE CASCADE,
				event_type    VARCHAR(50) NOT NULL,
				location_city VARCHAR(100),
				location_hub  VARCHAR(100),
				notes         TEXT,
				occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				recorded_by   VARCHAR(100) NOT NULL DEFAULT 'system',
				payload       JSONB NOT NULL DEFAULT '{}'::jsonb,
				created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_shipment_event_shipment_time
				ON logistics.shipment_event (shipment_id, occurred_at DESC);
			CREATE INDEX IF NOT EXISTS idx_shipment_event_type
				ON logistics.shipment_event (event_type, occurred_at DESC);

			CREATE TABLE IF NOT EXISTS logistics.delivery_attempt (
				id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				shipment_id     UUID NOT NULL REFERENCES logistics.shipment(id) ON DELETE CASCADE,
				attempt_number  INT NOT NULL,
				reason          VARCHAR(50) NOT NULL,
				notes           TEXT,
				next_attempt_at TIMESTAMPTZ,
				occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_delivery_attempt_shipment
				ON logistics.delivery_attempt (shipment_id, attempt_number);
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
