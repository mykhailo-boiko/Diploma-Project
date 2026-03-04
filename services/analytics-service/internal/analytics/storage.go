package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ Storage = (*PostgresStorage)(nil)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

func (s *PostgresStorage) UpsertSalesDaily(ctx context.Context, record SalesDaily) (SalesDaily, error) {
	now := time.Now().UTC()

	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	record.UpdatedAt = now

	query := `
		INSERT INTO analytics.sales_daily (id, date, total_orders, total_revenue, avg_order_size, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (date) DO UPDATE SET
			total_orders = EXCLUDED.total_orders,
			total_revenue = EXCLUDED.total_revenue,
			avg_order_size = EXCLUDED.avg_order_size,
			updated_at = EXCLUDED.updated_at
		RETURNING id, date, total_orders, total_revenue, avg_order_size, created_at, updated_at
	`

	var result SalesDaily
	err := s.pool.QueryRow(ctx, query,
		record.ID, record.Date, record.TotalOrders, record.TotalRevenue,
		record.AvgOrderSize, now, now,
	).Scan(
		&result.ID, &result.Date, &result.TotalOrders, &result.TotalRevenue,
		&result.AvgOrderSize, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return SalesDaily{}, fmt.Errorf("failed to upsert sales_daily: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) UpsertInventorySnapshot(ctx context.Context, record InventorySnapshot) (InventorySnapshot, error) {
	now := time.Now().UTC()

	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	record.UpdatedAt = now

	query := `
		INSERT INTO analytics.inventory_snapshot (id, date, total_products, total_quantity, total_reserved, total_available, low_stock_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (date) DO UPDATE SET
			total_products = EXCLUDED.total_products,
			total_quantity = EXCLUDED.total_quantity,
			total_reserved = EXCLUDED.total_reserved,
			total_available = EXCLUDED.total_available,
			low_stock_count = EXCLUDED.low_stock_count,
			updated_at = EXCLUDED.updated_at
		RETURNING id, date, total_products, total_quantity, total_reserved, total_available, low_stock_count, created_at, updated_at
	`

	var result InventorySnapshot
	err := s.pool.QueryRow(ctx, query,
		record.ID, record.Date, record.TotalProducts, record.TotalQuantity,
		record.TotalReserved, record.TotalAvailable, record.LowStockCount, now, now,
	).Scan(
		&result.ID, &result.Date, &result.TotalProducts, &result.TotalQuantity,
		&result.TotalReserved, &result.TotalAvailable, &result.LowStockCount,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("failed to upsert inventory_snapshot: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) UpsertLogisticsDaily(ctx context.Context, record LogisticsDaily) (LogisticsDaily, error) {
	now := time.Now().UTC()

	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	record.UpdatedAt = now

	query := `
		INSERT INTO analytics.logistics_daily (id, date, total_shipments, delivered_count, failed_count, avg_delivery_hours, on_time_rate, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (date) DO UPDATE SET
			total_shipments = EXCLUDED.total_shipments,
			delivered_count = EXCLUDED.delivered_count,
			failed_count = EXCLUDED.failed_count,
			avg_delivery_hours = EXCLUDED.avg_delivery_hours,
			on_time_rate = EXCLUDED.on_time_rate,
			updated_at = EXCLUDED.updated_at
		RETURNING id, date, total_shipments, delivered_count, failed_count, avg_delivery_hours, on_time_rate, created_at, updated_at
	`

	var result LogisticsDaily
	err := s.pool.QueryRow(ctx, query,
		record.ID, record.Date, record.TotalShipments, record.DeliveredCount,
		record.FailedCount, record.AvgDeliveryH, record.OnTimeRate, now, now,
	).Scan(
		&result.ID, &result.Date, &result.TotalShipments, &result.DeliveredCount,
		&result.FailedCount, &result.AvgDeliveryH, &result.OnTimeRate,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return LogisticsDaily{}, fmt.Errorf("failed to upsert logistics_daily: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) GetSalesDaily(ctx context.Context, from, to time.Time) ([]SalesDaily, error) {
	query := `
		SELECT id, date, total_orders, total_revenue, avg_order_size, created_at, updated_at
		FROM analytics.sales_daily
		WHERE date >= $1 AND date <= $2
		ORDER BY date ASC
	`

	rows, err := s.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query sales_daily: %w", err)
	}
	defer rows.Close()

	var results []SalesDaily
	for rows.Next() {
		var r SalesDaily
		if err := rows.Scan(&r.ID, &r.Date, &r.TotalOrders, &r.TotalRevenue, &r.AvgOrderSize, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sales_daily: %w", err)
		}
		results = append(results, r)
	}

	return results, nil
}

func (s *PostgresStorage) GetInventorySnapshots(ctx context.Context, from, to time.Time) ([]InventorySnapshot, error) {
	query := `
		SELECT id, date, total_products, total_quantity, total_reserved, total_available, low_stock_count, created_at, updated_at
		FROM analytics.inventory_snapshot
		WHERE date >= $1 AND date <= $2
		ORDER BY date ASC
	`

	rows, err := s.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory_snapshot: %w", err)
	}
	defer rows.Close()

	var results []InventorySnapshot
	for rows.Next() {
		var r InventorySnapshot
		if err := rows.Scan(
			&r.ID, &r.Date, &r.TotalProducts, &r.TotalQuantity,
			&r.TotalReserved, &r.TotalAvailable, &r.LowStockCount,
			&r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan inventory_snapshot: %w", err)
		}
		results = append(results, r)
	}

	return results, nil
}

func (s *PostgresStorage) GetLogisticsDaily(ctx context.Context, from, to time.Time) ([]LogisticsDaily, error) {
	query := `
		SELECT id, date, total_shipments, delivered_count, failed_count, avg_delivery_hours, on_time_rate, created_at, updated_at
		FROM analytics.logistics_daily
		WHERE date >= $1 AND date <= $2
		ORDER BY date ASC
	`

	rows, err := s.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query logistics_daily: %w", err)
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (LogisticsDaily, error) {
		var r LogisticsDaily
		err := row.Scan(
			&r.ID, &r.Date, &r.TotalShipments, &r.DeliveredCount,
			&r.FailedCount, &r.AvgDeliveryH, &r.OnTimeRate,
			&r.CreatedAt, &r.UpdatedAt,
		)
		return r, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to collect logistics_daily: %w", err)
	}

	return results, nil
}
