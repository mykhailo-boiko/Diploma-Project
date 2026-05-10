package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
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

func (s *PostgresStorage) GetDailyMetricSeries(ctx context.Context, metric string, from, to time.Time) ([]ForecastPoint, error) {
	var query string
	switch metric {
	case "revenue":
		query = `
			SELECT date_trunc('day', created_at)::date AS d,
			       COALESCE(SUM(total_amount), 0)::float8
			FROM orders.orders
			WHERE deleted_at IS NULL
				AND status NOT IN ('cancelled','returned')
				AND created_at >= $1 AND created_at <= $2
			GROUP BY d ORDER BY d`
	case "order_count":
		query = `
			SELECT date_trunc('day', created_at)::date AS d,
			       COUNT(*)::float8
			FROM orders.orders
			WHERE deleted_at IS NULL
				AND status NOT IN ('cancelled','returned')
				AND created_at >= $1 AND created_at <= $2
			GROUP BY d ORDER BY d`
	case "shipment_count":
		query = `
			SELECT date_trunc('day', created_at)::date AS d,
			       COUNT(*)::float8
			FROM logistics.shipment
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at <= $2
			GROUP BY d ORDER BY d`
	default:
		return nil, fmt.Errorf("unsupported metric for series: %s (supported: revenue, order_count, shipment_count)", metric)
	}

	rows, err := s.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily series for %s: %w", metric, err)
	}
	defer rows.Close()

	var points []ForecastPoint
	for rows.Next() {
		var d time.Time
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, fmt.Errorf("failed to scan series point: %w", err)
		}
		points = append(points, ForecastPoint{Date: d, Value: v})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate series: %w", err)
	}
	return points, nil
}

func (s *PostgresStorage) QueryAuditLog(ctx context.Context, f AuditFilter) ([]AuditEntry, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	conds := []string{"1=1"}
	args := []any{}
	if f.ActorEmail != "" {
		args = append(args, f.ActorEmail)
		conds = append(conds, fmt.Sprintf("actor_email = $%d", len(args)))
	}
	if f.Action != "" {
		args = append(args, f.Action)
		conds = append(conds, fmt.Sprintf("action = $%d", len(args)))
	}
	if f.EntityID != "" {
		args = append(args, f.EntityID)
		conds = append(conds, fmt.Sprintf("$%d = ANY(entity_ids)", len(args)))
	}
	if f.From != nil {
		args = append(args, *f.From)
		conds = append(conds, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		conds = append(conds, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT id::text, actor_user_id, actor_email, actor_role, service_name, action,
			COALESCE(entity_type,''), COALESCE(entity_ids,'{}'),
			COALESCE(params_snip,''), result_status,
			success_count, failure_count, COALESCE(error_message,''), created_at
		FROM audit.action_log
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d`,
		strings.Join(conds, " AND "), len(args),
	)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var results []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.ActorEmail, &e.ActorRole,
			&e.ServiceName, &e.Action, &e.EntityType, &e.EntityIDs,
			&e.ParamsSnip, &e.ResultStatus,
			&e.SuccessCount, &e.FailureCount, &e.ErrorMessage, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate audit log: %w", err)
	}
	return results, nil
}

func (s *PostgresStorage) GetMetricValue(ctx context.Context, metric string, from, to time.Time) (float64, error) {
	var query string
	switch metric {
	case "revenue":
		query = `
			SELECT COALESCE(SUM(total_amount), 0)::float8
			FROM orders.orders
			WHERE deleted_at IS NULL
				AND status NOT IN ('cancelled','returned')
				AND created_at >= $1 AND created_at <= $2`
	case "order_count":
		query = `
			SELECT COUNT(*)::float8
			FROM orders.orders
			WHERE deleted_at IS NULL
				AND status NOT IN ('cancelled','returned')
				AND created_at >= $1 AND created_at <= $2`
	case "aov":
		query = `
			SELECT COALESCE(AVG(total_amount), 0)::float8
			FROM orders.orders
			WHERE deleted_at IS NULL
				AND status NOT IN ('cancelled','returned')
				AND created_at >= $1 AND created_at <= $2`
	case "cancellation_rate":
		query = `
			SELECT COALESCE(
				100.0 * COUNT(*) FILTER (WHERE status = 'cancelled')::float8
				/ NULLIF(COUNT(*), 0)::float8,
			0)::float8
			FROM orders.orders
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at <= $2`
	case "on_time_rate":
		query = `
			SELECT COALESCE(
				100.0 * COUNT(*) FILTER (
					WHERE status = 'delivered'
					AND EXTRACT(EPOCH FROM (updated_at - created_at))/3600 <= 168
				)::float8
				/ NULLIF(COUNT(*) FILTER (WHERE status = 'delivered'), 0)::float8,
			0)::float8
			FROM logistics.shipment
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at <= $2`
	case "shipment_count":
		query = `
			SELECT COUNT(*)::float8
			FROM logistics.shipment
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at <= $2`
	case "low_stock_count":
		query = `
			SELECT COUNT(*)::float8
			FROM inventory.stock
			WHERE quantity < min_threshold AND min_threshold > 0`
		var v float64
		if err := s.pool.QueryRow(ctx, query).Scan(&v); err != nil {
			return 0, fmt.Errorf("failed to query metric %s: %w", metric, err)
		}
		return v, nil
	default:
		return 0, fmt.Errorf("unsupported metric: %s (supported: revenue, order_count, aov, cancellation_rate, on_time_rate, shipment_count, low_stock_count)", metric)
	}

	var v float64
	if err := s.pool.QueryRow(ctx, query, from, to).Scan(&v); err != nil {
		return 0, fmt.Errorf("failed to query metric %s: %w", metric, err)
	}
	return v, nil
}

func (s *PostgresStorage) GetCustomerProfile360(ctx context.Context, customerName string, recentN int, topCategoriesN int) (CustomerProfile360, error) {
	if recentN <= 0 {
		recentN = 5
	}
	if topCategoriesN <= 0 {
		topCategoriesN = 5
	}

	var profile CustomerProfile360
	profile.CustomerName = customerName
	profile.StatusBreakdown = map[string]int{}
	profile.TopCategories = []CategorySpend{}
	profile.RecentOrders = []OrderHeader{}

	lifetimeQuery := `
		SELECT
			MIN(created_at) AS first_order,
			MAX(created_at) AS last_order,
			COUNT(*)::int AS order_count,
			COALESCE(SUM(total_amount), 0)::float8 AS lifetime_value,
			COALESCE(AVG(total_amount), 0)::float8 AS aov,
			COALESCE((
				SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM gap)) / 86400.0
				FROM (
					SELECT created_at - LAG(created_at) OVER (ORDER BY created_at) AS gap
					FROM orders.orders o2
					WHERE o2.customer_name = $1 AND o2.deleted_at IS NULL
					  AND o2.status NOT IN ('cancelled','returned')
				) g
				WHERE gap IS NOT NULL
			), 0)::float8 AS median_inter_order_days
		FROM orders.orders
		WHERE customer_name = $1
			AND deleted_at IS NULL
			AND status NOT IN ('cancelled','returned')`

	var firstOrder, lastOrder *time.Time
	if err := s.pool.QueryRow(ctx, lifetimeQuery, customerName).Scan(
		&firstOrder, &lastOrder,
		&profile.OrderCount, &profile.LifetimeValue, &profile.AvgOrderValue,
		&profile.MedianInterOrderD,
	); err != nil {
		return CustomerProfile360{}, fmt.Errorf("failed to load lifetime aggregates: %w", err)
	}
	if profile.OrderCount == 0 || firstOrder == nil {
		return CustomerProfile360{}, ErrCustomerNotFound
	}
	profile.FirstOrderDate = *firstOrder
	profile.LastOrderDate = *lastOrder
	profile.DaysSinceLastOrder = int(time.Since(*lastOrder).Hours() / 24)
	profile.IsNewCustomer90Days = time.Since(*firstOrder) < 90*24*time.Hour

	if profile.MedianInterOrderD > 0 {
		x := float64(profile.DaysSinceLastOrder) / profile.MedianInterOrderD
		profile.ChurnRiskScore = 1 - math.Exp(-x/2.0)
	} else {
		profile.ChurnRiskScore = 0
	}
	if profile.ChurnRiskScore < 0 {
		profile.ChurnRiskScore = 0
	}
	if profile.ChurnRiskScore > 1 {
		profile.ChurnRiskScore = 1
	}

	statusQuery := `
		SELECT status, COUNT(*)::int
		FROM orders.orders
		WHERE customer_name = $1 AND deleted_at IS NULL
		GROUP BY status`
	rows, err := s.pool.Query(ctx, statusQuery, customerName)
	if err != nil {
		return CustomerProfile360{}, fmt.Errorf("failed to load status breakdown: %w", err)
	}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			rows.Close()
			return CustomerProfile360{}, fmt.Errorf("failed to scan status: %w", err)
		}
		profile.StatusBreakdown[st] = n
	}
	rows.Close()

	catQuery := `
		SELECT
			COALESCE(NULLIF(p.category, ''), 'Uncategorized') AS category,
			COALESCE(SUM(oi.subtotal), 0)::float8 AS revenue,
			COALESCE(SUM(oi.quantity), 0)::int AS units
		FROM orders.order_items oi
		JOIN orders.orders o ON o.id = oi.order_id
		LEFT JOIN inventory.product p ON p.id::text = oi.product_id
		WHERE o.customer_name = $1 AND o.deleted_at IS NULL
			AND o.status NOT IN ('cancelled','returned')
		GROUP BY COALESCE(NULLIF(p.category, ''), 'Uncategorized')
		ORDER BY revenue DESC
		LIMIT $2`
	catRows, err := s.pool.Query(ctx, catQuery, customerName, topCategoriesN)
	if err != nil {
		return CustomerProfile360{}, fmt.Errorf("failed to load top categories: %w", err)
	}
	for catRows.Next() {
		var c CategorySpend
		if err := catRows.Scan(&c.Category, &c.Revenue, &c.UnitsSold); err != nil {
			catRows.Close()
			return CustomerProfile360{}, fmt.Errorf("failed to scan category: %w", err)
		}
		profile.TopCategories = append(profile.TopCategories, c)
	}
	catRows.Close()

	recentQuery := `
		SELECT id::text, status, total_amount::float8, created_at
		FROM orders.orders
		WHERE customer_name = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2`
	recentRows, err := s.pool.Query(ctx, recentQuery, customerName, recentN)
	if err != nil {
		return CustomerProfile360{}, fmt.Errorf("failed to load recent orders: %w", err)
	}
	for recentRows.Next() {
		var o OrderHeader
		if err := recentRows.Scan(&o.ID, &o.Status, &o.TotalAmount, &o.CreatedAt); err != nil {
			recentRows.Close()
			return CustomerProfile360{}, fmt.Errorf("failed to scan recent order: %w", err)
		}
		profile.RecentOrders = append(profile.RecentOrders, o)
	}
	recentRows.Close()

	return profile, nil
}

func (s *PostgresStorage) GetCarrierPerformance(ctx context.Context, from, to time.Time, slaHours int, worstCitiesPerCarrier int) ([]CarrierPerformance, error) {
	if slaHours <= 0 {
		slaHours = 168
	}
	if worstCitiesPerCarrier <= 0 {
		worstCitiesPerCarrier = 5
	}

	carrierQuery := `
		SELECT
			c.id::text AS carrier_id,
			c.name AS carrier_name,
			c.is_active,
			COUNT(s.id)::int AS total,
			COUNT(*) FILTER (WHERE s.status = 'delivered')::int AS delivered,
			COUNT(*) FILTER (WHERE s.status = 'delivered'
				AND EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600 <= $3)::int AS on_time,
			COUNT(*) FILTER (WHERE s.status = 'delivered'
				AND EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600 > $3)::int AS late,
			COUNT(*) FILTER (WHERE s.status = 'cancelled')::int AS cancelled,
			COUNT(*) FILTER (WHERE s.status = 'returned')::int AS returned,
			COALESCE(AVG(EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600)
				FILTER (WHERE s.status = 'delivered'), 0)::float8 AS avg_h
		FROM logistics.carrier c
		LEFT JOIN logistics.shipment s
			ON s.carrier_id = c.id
			AND s.deleted_at IS NULL
			AND s.created_at >= $1 AND s.created_at <= $2
		GROUP BY c.id, c.name, c.is_active`

	rows, err := s.pool.Query(ctx, carrierQuery, from, to, slaHours)
	if err != nil {
		return nil, fmt.Errorf("failed to query carrier performance: %w", err)
	}
	defer rows.Close()

	var carriers []CarrierPerformance
	carrierByID := map[string]*CarrierPerformance{}
	for rows.Next() {
		var c CarrierPerformance
		if err := rows.Scan(
			&c.CarrierID, &c.CarrierName, &c.IsActive,
			&c.TotalShipments, &c.Delivered, &c.OnTime, &c.Late, &c.Cancelled, &c.Returned,
			&c.AvgDeliveryHours,
		); err != nil {
			return nil, fmt.Errorf("failed to scan carrier performance: %w", err)
		}
		if c.Delivered > 0 {
			c.OnTimeRate = float64(c.OnTime) / float64(c.Delivered)
		}
		c.WorstCities = []CarrierCityStat{}
		carriers = append(carriers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate carrier performance: %w", err)
	}
	for i := range carriers {
		carrierByID[carriers[i].CarrierID] = &carriers[i]
	}

	cityQuery := `
		SELECT
			c.id::text AS carrier_id,
			COALESCE(NULLIF(trim(split_part(s.address, ',', 3)), ''), 'Unknown') AS city,
			COUNT(*) FILTER (WHERE s.status = 'delivered')::int AS delivered,
			COUNT(*) FILTER (WHERE s.status = 'delivered'
				AND EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600 <= $3)::int AS on_time,
			COUNT(*) FILTER (WHERE s.status = 'delivered'
				AND EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600 > $3)::int AS late,
			COALESCE(AVG(EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600)
				FILTER (WHERE s.status = 'delivered'), 0)::float8 AS avg_h
		FROM logistics.shipment s
		JOIN logistics.carrier c ON c.id = s.carrier_id
		WHERE s.deleted_at IS NULL
			AND s.created_at >= $1 AND s.created_at <= $2
			AND s.status = 'delivered'
		GROUP BY c.id, COALESCE(NULLIF(trim(split_part(s.address, ',', 3)), ''), 'Unknown')
		HAVING COUNT(*) FILTER (WHERE s.status = 'delivered'
			AND EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600 > $3) > 0
		ORDER BY c.id,
			(COUNT(*) FILTER (WHERE s.status = 'delivered'
				AND EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600 > $3)::float8
				/ NULLIF(COUNT(*) FILTER (WHERE s.status = 'delivered'), 0)) DESC,
			COUNT(*) FILTER (WHERE s.status = 'delivered'
				AND EXTRACT(EPOCH FROM (s.updated_at - s.created_at))/3600 > $3) DESC`

	cityRows, err := s.pool.Query(ctx, cityQuery, from, to, slaHours)
	if err != nil {
		return nil, fmt.Errorf("failed to query carrier city perf: %w", err)
	}
	defer cityRows.Close()

	for cityRows.Next() {
		var carrierID string
		var stat CarrierCityStat
		if err := cityRows.Scan(&carrierID, &stat.City, &stat.Delivered, &stat.OnTime, &stat.Late, &stat.AvgHours); err != nil {
			return nil, fmt.Errorf("failed to scan carrier city perf: %w", err)
		}
		if stat.Delivered > 0 {
			stat.OnTimeRate = float64(stat.OnTime) / float64(stat.Delivered)
		}
		if c, ok := carrierByID[carrierID]; ok {
			if len(c.WorstCities) < worstCitiesPerCarrier {
				c.WorstCities = append(c.WorstCities, stat)
			}
		}
	}
	if err := cityRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate carrier city perf: %w", err)
	}

	sort.SliceStable(carriers, func(i, j int) bool {
		return carriers[i].OnTimeRate < carriers[j].OnTimeRate
	})

	return carriers, nil
}

func (s *PostgresStorage) GetRebalancingRecommendations(ctx context.Context, p RebalancingParams) ([]RebalancingRecommendation, error) {
	if p.OverstockMultiplier <= 0 {
		p.OverstockMultiplier = 3.0
	}
	if p.HoldingDailyRate <= 0 {
		p.HoldingDailyRate = 0.0005
	}
	if p.HoldingHorizonDays <= 0 {
		p.HoldingHorizonDays = 30
	}
	if p.TransferBaseFee < 0 {
		p.TransferBaseFee = 50.0
	}
	if p.TransferPerUnit < 0 {
		p.TransferPerUnit = 1.5
	}
	if p.Limit <= 0 {
		p.Limit = 50
	}

	query := `
WITH
donors AS (
  SELECT s.product_id, s.warehouse_id AS donor_wh_id, w.name AS donor_wh_name,
         s.quantity AS donor_qty, s.min_threshold AS donor_thr
  FROM inventory.stock s
  JOIN inventory.warehouse w ON w.id = s.warehouse_id
  WHERE s.min_threshold > 0
    AND s.quantity > s.min_threshold * $1
),
acceptors AS (
  SELECT s.product_id, s.warehouse_id AS acceptor_wh_id, w.name AS acceptor_wh_name,
         s.quantity AS acceptor_qty, s.min_threshold AS acceptor_thr
  FROM inventory.stock s
  JOIN inventory.warehouse w ON w.id = s.warehouse_id
  WHERE s.min_threshold > 0
    AND s.quantity < s.min_threshold
),
pairs AS (
  SELECT
    d.product_id, p.sku, p.name AS product_name, p.category, p.unit_price::float8 AS unit_price,
    d.donor_wh_id, d.donor_wh_name, d.donor_qty, d.donor_thr,
    a.acceptor_wh_id, a.acceptor_wh_name, a.acceptor_qty, a.acceptor_thr,
    LEAST(
      GREATEST(d.donor_qty - d.donor_thr * 2, 0),
      GREATEST(a.acceptor_thr * 2 - a.acceptor_qty, 0)
    )::int AS transfer_qty
  FROM donors d
  JOIN acceptors a
    ON a.product_id = d.product_id
   AND a.acceptor_wh_id <> d.donor_wh_id
  JOIN inventory.product p ON p.id = d.product_id
  WHERE p.deleted_at IS NULL
),
costed AS (
  SELECT
    *,
    (transfer_qty * unit_price * $2 * $3)::float8 AS holding_savings,
    ($4 + transfer_qty * $5)::float8 AS transfer_cost
  FROM pairs
  WHERE transfer_qty > 0
),
final AS (
  SELECT
    *,
    (holding_savings - transfer_cost)::float8 AS net_benefit,
    CASE WHEN transfer_cost > 0
         THEN ((holding_savings - transfer_cost) / transfer_cost * 100.0)::float8
         ELSE 0::float8
    END AS roi_pct
  FROM costed
),
ranked AS (
  SELECT *,
    ROW_NUMBER() OVER (PARTITION BY product_id, acceptor_wh_id ORDER BY net_benefit DESC, donor_qty DESC) AS rn
  FROM final
)
SELECT product_id::text, sku, product_name, category, unit_price,
       donor_wh_name, donor_qty, donor_thr,
       acceptor_wh_name, acceptor_qty, acceptor_thr,
       transfer_qty, holding_savings, transfer_cost, net_benefit, roi_pct
FROM ranked
WHERE rn = 1`

	if p.OnlyPositiveROI {
		query += " AND net_benefit > 0"
	}

	query += " ORDER BY net_benefit DESC LIMIT $6"

	rows, err := s.pool.Query(ctx, query,
		p.OverstockMultiplier, p.HoldingDailyRate, p.HoldingHorizonDays,
		p.TransferBaseFee, p.TransferPerUnit, p.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query rebalancing recommendations: %w", err)
	}
	defer rows.Close()

	var results []RebalancingRecommendation
	for rows.Next() {
		var r RebalancingRecommendation
		if err := rows.Scan(
			&r.ProductID, &r.SKU, &r.ProductName, &r.Category, &r.UnitPrice,
			&r.DonorWarehouse, &r.DonorQuantity, &r.DonorThreshold,
			&r.AcceptorWarehouse, &r.AcceptorQuantity, &r.AcceptorThreshold,
			&r.TransferQty, &r.HoldingSavings, &r.TransferCost, &r.NetBenefit, &r.ROIPct,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rebalancing recommendation: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rebalancing recommendations: %w", err)
	}

	return results, nil
}

func (s *PostgresStorage) GetQuickCancellations(ctx context.Context, from, to time.Time, maxMinutes int) ([]QuickCancellation, error) {
	if maxMinutes <= 0 {
		maxMinutes = 60
	}

	query := `
		SELECT
			c.name AS carrier_name,
			COALESCE(NULLIF(trim(split_part(s.address, ',', 3)), ''), 'Unknown') AS city,
			COUNT(*)::int AS cnt,
			AVG(EXTRACT(EPOCH FROM (o.updated_at - s.created_at))/60.0)::float8 AS avg_min,
			MIN(EXTRACT(EPOCH FROM (o.updated_at - s.created_at))/60.0)::float8 AS min_min,
			MAX(EXTRACT(EPOCH FROM (o.updated_at - s.created_at))/60.0)::float8 AS max_min,
			COALESCE(SUM(o.total_amount), 0)::float8 AS lost_revenue,
			(array_agg(o.id::text ORDER BY o.updated_at DESC))[1:5] AS sample_order_ids,
			(array_agg(DISTINCT o.cancel_reason) FILTER (WHERE o.cancel_reason IS NOT NULL))[1:3] AS sample_reasons
		FROM orders.orders o
		JOIN logistics.shipment s ON s.order_id = o.id::text
		JOIN logistics.carrier c ON c.id = s.carrier_id
		WHERE o.deleted_at IS NULL
			AND s.deleted_at IS NULL
			AND o.status = 'cancelled'
			AND s.created_at >= $1
			AND s.created_at <= $2
			AND o.updated_at > s.created_at
			AND (o.updated_at - s.created_at) <= make_interval(mins => $3)
		GROUP BY c.name, COALESCE(NULLIF(trim(split_part(s.address, ',', 3)), ''), 'Unknown')
		ORDER BY cnt DESC, lost_revenue DESC`

	rows, err := s.pool.Query(ctx, query, from, to, maxMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to query quick cancellations: %w", err)
	}
	defer rows.Close()

	var results []QuickCancellation
	for rows.Next() {
		var qc QuickCancellation
		if err := rows.Scan(
			&qc.CarrierName, &qc.City, &qc.Count,
			&qc.AvgMinutes, &qc.MinMinutes, &qc.MaxMinutes,
			&qc.LostRevenue, &qc.SampleOrderIDs, &qc.SampleCancelReasons,
		); err != nil {
			return nil, fmt.Errorf("failed to scan quick cancellation: %w", err)
		}
		results = append(results, qc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate quick cancellations: %w", err)
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
