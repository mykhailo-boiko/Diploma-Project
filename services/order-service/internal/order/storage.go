package order

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

var _ Storage = (*PostgresStorage)(nil)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

func (s *PostgresStorage) CreateOrder(ctx context.Context, o Order) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	o.ID = uuid.NewString()
	now := time.Now().UTC()
	o.CreatedAt = now
	o.UpdatedAt = now
	o.Status = StatusPending

	var total float64
	for i := range o.Items {
		o.Items[i].ID = uuid.NewString()
		o.Items[i].OrderID = o.ID
		o.Items[i].Subtotal = float64(o.Items[i].Quantity) * o.Items[i].UnitPrice
		total += o.Items[i].Subtotal
	}
	o.TotalAmount = total

	orderQuery := `
		INSERT INTO orders.orders (id, customer_name, status, total_amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	if _, err := tx.Exec(ctx, orderQuery,
		o.ID, o.CustomerName, string(o.Status), o.TotalAmount, o.CreatedAt, o.UpdatedAt,
	); err != nil {
		return Order{}, fmt.Errorf("failed to insert order: %w", err)
	}

	itemQuery := `
		INSERT INTO orders.order_items (id, order_id, product_id, name, quantity, unit_price, subtotal)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	for _, item := range o.Items {
		if _, err := tx.Exec(ctx, itemQuery,
			item.ID, item.OrderID, item.ProductID, item.Name, item.Quantity, item.UnitPrice, item.Subtotal,
		); err != nil {
			return Order{}, fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return o, nil
}

func (s *PostgresStorage) GetOrderByID(ctx context.Context, id string) (Order, error) {
	orderQuery := `
		SELECT id, customer_name, status, total_amount, cancel_reason, created_at, updated_at
		FROM orders.orders
		WHERE id = $1 AND deleted_at IS NULL`

	var o Order
	err := s.pool.QueryRow(ctx, orderQuery, id).Scan(
		&o.ID, &o.CustomerName, &o.Status, &o.TotalAmount, &o.CancelReason, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, fmt.Errorf("failed to get order: %w", err)
	}

	items, err := s.getOrderItems(ctx, id)
	if err != nil {
		return Order{}, err
	}
	o.Items = items

	return o, nil
}

func (s *PostgresStorage) ListOrders(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Order, int, error) {
	where, args := s.buildWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM orders.orders" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, customer_name, status, total_amount, cancel_reason, created_at, updated_at FROM orders.orders%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerName, &o.Status, &o.TotalAmount, &o.CancelReason, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate orders: %w", err)
	}

	return orders, total, nil
}

func (s *PostgresStorage) UpdateOrderStatus(ctx context.Context, id string, status Status) (Order, error) {
	query := `
		UPDATE orders.orders
		SET status = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, customer_name, status, total_amount, cancel_reason, created_at, updated_at`

	now := time.Now().UTC()
	var o Order
	err := s.pool.QueryRow(ctx, query, string(status), now, id).Scan(
		&o.ID, &o.CustomerName, &o.Status, &o.TotalAmount, &o.CancelReason, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, fmt.Errorf("failed to update order status: %w", err)
	}

	items, err := s.getOrderItems(ctx, id)
	if err != nil {
		return Order{}, err
	}
	o.Items = items

	return o, nil
}

func (s *PostgresStorage) getOrderItems(ctx context.Context, orderID string) ([]Item, error) {
	query := `
		SELECT id, order_id, product_id, name, quantity, unit_price, subtotal
		FROM orders.order_items
		WHERE order_id = $1`

	rows, err := s.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate order items: %w", err)
	}

	return items, nil
}

func (s *PostgresStorage) CancelOrder(ctx context.Context, id string, reason string) (Order, error) {
	query := `
		UPDATE orders.orders
		SET status = $1, cancel_reason = $2, updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING id, customer_name, status, total_amount, cancel_reason, created_at, updated_at`

	now := time.Now().UTC()
	var o Order
	err := s.pool.QueryRow(ctx, query, string(StatusCancelled), reason, now, id).Scan(
		&o.ID, &o.CustomerName, &o.Status, &o.TotalAmount, &o.CancelReason, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, fmt.Errorf("failed to cancel order: %w", err)
	}

	items, err := s.getOrderItems(ctx, id)
	if err != nil {
		return Order{}, err
	}
	o.Items = items

	return o, nil
}

func (s *PostgresStorage) SearchOrders(ctx context.Context, query string, page pagination.Page) ([]Order, int, error) {
	searchPattern := "%" + query + "%"

	countQuery := `
		SELECT COUNT(*) FROM orders.orders
		WHERE deleted_at IS NULL
			AND (customer_name ILIKE $1 OR CAST(id AS TEXT) ILIKE $1)`

	var total int
	if err := s.pool.QueryRow(ctx, countQuery, searchPattern).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	dataQuery := `
		SELECT id, customer_name, status, total_amount, cancel_reason, created_at, updated_at
		FROM orders.orders
		WHERE deleted_at IS NULL
			AND (customer_name ILIKE $1 OR CAST(id AS TEXT) ILIKE $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, dataQuery, searchPattern, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerName, &o.Status, &o.TotalAmount, &o.CancelReason, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan search result: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate search results: %w", err)
	}

	return orders, total, nil
}

func (s *PostgresStorage) GetOrderStats(ctx context.Context) (OrderStats, error) {
	query := `
		SELECT status, COUNT(*) as count, COALESCE(SUM(total_amount), 0) as revenue
		FROM orders.orders
		WHERE deleted_at IS NULL
		GROUP BY status
		ORDER BY status`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return OrderStats{}, fmt.Errorf("failed to get order stats: %w", err)
	}
	defer rows.Close()

	var stats OrderStats
	for rows.Next() {
		var ss StatusStats
		if err := rows.Scan(&ss.Status, &ss.Count, &ss.Revenue); err != nil {
			return OrderStats{}, fmt.Errorf("failed to scan order stats: %w", err)
		}
		stats.TotalOrders += ss.Count
		stats.TotalRevenue += ss.Revenue
		stats.ByStatus = append(stats.ByStatus, ss)
	}
	if err := rows.Err(); err != nil {
		return OrderStats{}, fmt.Errorf("failed to iterate order stats: %w", err)
	}

	if stats.ByStatus == nil {
		stats.ByStatus = []StatusStats{}
	}

	return stats, nil
}

func (s *PostgresStorage) GetSalesByProduct(ctx context.Context, from, to time.Time, includeStatuses []Status) ([]ProductSales, error) {
	if len(includeStatuses) == 0 {
		includeStatuses = []Status{
			StatusConfirmed, StatusProcessing, StatusShipped,
			StatusDelivered, StatusCompleted,
		}
	}

	statusStrings := make([]string, 0, len(includeStatuses))
	for _, s := range includeStatuses {
		statusStrings = append(statusStrings, string(s))
	}

	query := `
		SELECT
			oi.product_id,
			MAX(oi.name) AS name,
			SUM(oi.quantity)::int AS units_sold,
			SUM(oi.subtotal)::float8 AS revenue,
			COUNT(DISTINCT oi.order_id)::int AS order_count
		FROM orders.order_items oi
		JOIN orders.orders o ON o.id = oi.order_id
		WHERE o.deleted_at IS NULL
			AND o.created_at >= $1
			AND o.created_at <= $2
			AND o.status = ANY($3)
		GROUP BY oi.product_id
		ORDER BY revenue DESC`

	rows, err := s.pool.Query(ctx, query, from, to, statusStrings)
	if err != nil {
		return nil, fmt.Errorf("failed to query sales by product: %w", err)
	}
	defer rows.Close()

	days := to.Sub(from).Hours() / 24
	if days < 1 {
		days = 1
	}

	var results []ProductSales
	for rows.Next() {
		var ps ProductSales
		if err := rows.Scan(&ps.ProductID, &ps.Name, &ps.UnitsSold, &ps.Revenue, &ps.OrderCount); err != nil {
			return nil, fmt.Errorf("failed to scan sales by product: %w", err)
		}
		ps.DailyDemand = float64(ps.UnitsSold) / days
		results = append(results, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate sales by product: %w", err)
	}

	return results, nil
}

func (s *PostgresStorage) BulkUpdateStatus(ctx context.Context, orderIDs []string, newStatus Status, note string, dryRun bool) (BulkStatusResult, error) {
	result := BulkStatusResult{
		Total:      len(orderIDs),
		DryRun:     dryRun,
		UpdatedIDs: make([]string, 0, len(orderIDs)),
		Successes:  make([]BulkStatusItem, 0, len(orderIDs)),
		Failures:   make([]BulkStatusItem, 0),
	}
	if len(orderIDs) == 0 {
		return result, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id::text, status FROM orders.orders WHERE id::text = ANY($1) AND deleted_at IS NULL`,
		orderIDs,
	)
	if err != nil {
		return result, fmt.Errorf("failed to load orders for bulk update: %w", err)
	}

	currentStatus := make(map[string]Status, len(orderIDs))
	for rows.Next() {
		var id string
		var st Status
		if err := rows.Scan(&id, &st); err != nil {
			rows.Close()
			return result, fmt.Errorf("failed to scan order: %w", err)
		}
		currentStatus[id] = st
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("failed to iterate orders: %w", err)
	}

	now := time.Now().UTC()
	noteValue := strings.TrimSpace(note)
	for _, id := range orderIDs {
		from, found := currentStatus[id]
		if !found {
			result.Failures = append(result.Failures, BulkStatusItem{
				OrderID: id, Error: ErrOrderNotFound.Error(),
			})
			continue
		}
		if !CanTransition(from, newStatus) {
			result.Failures = append(result.Failures, BulkStatusItem{
				OrderID: id, OldStatus: from, NewStatus: newStatus,
				Error: fmt.Sprintf("invalid transition: %s -> %s", from, newStatus),
			})
			continue
		}

		if dryRun {
			result.Successes = append(result.Successes, BulkStatusItem{
				OrderID: id, OldStatus: from, NewStatus: newStatus,
			})
			result.UpdatedIDs = append(result.UpdatedIDs, id)
			continue
		}

		var execErr error
		if noteValue != "" {
			_, execErr = s.pool.Exec(ctx,
				`UPDATE orders.orders SET status = $1, cancel_reason = $2, updated_at = $3
				 WHERE id::text = $4 AND deleted_at IS NULL`,
				string(newStatus), noteValue, now, id,
			)
		} else {
			_, execErr = s.pool.Exec(ctx,
				`UPDATE orders.orders SET status = $1, updated_at = $2
				 WHERE id::text = $3 AND deleted_at IS NULL`,
				string(newStatus), now, id,
			)
		}
		if execErr != nil {
			result.Failures = append(result.Failures, BulkStatusItem{
				OrderID: id, OldStatus: from, NewStatus: newStatus,
				Error: execErr.Error(),
			})
			continue
		}

		result.Successes = append(result.Successes, BulkStatusItem{
			OrderID: id, OldStatus: from, NewStatus: newStatus,
		})
		result.UpdatedIDs = append(result.UpdatedIDs, id)
	}

	return result, nil
}

func (s *PostgresStorage) GetCustomerSummary(ctx context.Context, filter CustomerFilter) ([]CustomerSummary, error) {
	hasWindow := filter.WindowFrom != nil && filter.WindowTo != nil

	var args []any
	var windowFromExpr, windowToExpr string
	if hasWindow {
		args = append(args, *filter.WindowFrom, *filter.WindowTo)
		windowFromExpr = "$1"
		windowToExpr = "$2"
	}

	query := fmt.Sprintf(`
		SELECT
			customer_name,
			MIN(created_at) AS first_order_date,
			MAX(created_at) AS last_order_date,
			COUNT(*)::int AS total_orders,
			COALESCE(SUM(total_amount), 0)::float8 AS total_revenue,
			%s AS orders_in_window,
			%s AS revenue_in_window
		FROM orders.orders
		WHERE deleted_at IS NULL
			AND status NOT IN ('cancelled','returned')
		GROUP BY customer_name`,
		ifElse(hasWindow,
			fmt.Sprintf("COUNT(*) FILTER (WHERE created_at >= %s AND created_at <= %s)::int", windowFromExpr, windowToExpr),
			"0::int",
		),
		ifElse(hasWindow,
			fmt.Sprintf("COALESCE(SUM(total_amount) FILTER (WHERE created_at >= %s AND created_at <= %s), 0)::float8", windowFromExpr, windowToExpr),
			"0::float8",
		),
	)

	if hasWindow {
		query += " HAVING COUNT(*) FILTER (WHERE created_at >= " + windowFromExpr + " AND created_at <= " + windowToExpr + ") > 0"
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query customer summary: %w", err)
	}
	defer rows.Close()

	var results []CustomerSummary
	for rows.Next() {
		var c CustomerSummary
		if err := rows.Scan(
			&c.CustomerName, &c.FirstOrderDate, &c.LastOrderDate,
			&c.TotalOrders, &c.TotalRevenue,
			&c.OrdersInWindow, &c.RevenueInWindow,
		); err != nil {
			return nil, fmt.Errorf("failed to scan customer summary: %w", err)
		}
		if c.TotalOrders > 0 {
			c.AvgOrderValue = c.TotalRevenue / float64(c.TotalOrders)
		}
		if hasWindow {
			c.NewInWindow = !c.FirstOrderDate.Before(*filter.WindowFrom)
		}
		results = append(results, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate customer summary: %w", err)
	}

	if filter.OnlyNew && hasWindow {
		filtered := results[:0]
		for _, c := range results {
			if c.NewInWindow {
				filtered = append(filtered, c)
			}
		}
		results = filtered
	}

	sortKey := func(c CustomerSummary) float64 {
		switch filter.SortBy {
		case CustomerSortOrders:
			return float64(c.TotalOrders)
		case CustomerSortRevenueWindow:
			return c.RevenueInWindow
		case CustomerSortLastOrder:
			return float64(c.LastOrderDate.UnixNano())
		case CustomerSortFirstOrder:
			return float64(c.FirstOrderDate.UnixNano())
		default:
			return c.TotalRevenue
		}
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := sortKey(results[i]), sortKey(results[j])
		if filter.SortDesc {
			return a > b
		}
		return a < b
	})

	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}

func ifElse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func (s *PostgresStorage) buildWhereClause(filter Filter) (string, []any) {
	var conditions []string
	var args []any

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	if filter.CustomerName != nil {
		args = append(args, "%"+*filter.CustomerName+"%")
		conditions = append(conditions, fmt.Sprintf("customer_name ILIKE $%d", len(args)))
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapSortField(field string) string {
	switch field {
	case "created_at":
		return "created_at"
	case "total_amount":
		return "total_amount"
	case "status":
		return "status"
	case "customer_name":
		return "customer_name"
	default:
		return "created_at"
	}
}
