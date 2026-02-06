package order

import (
	"context"
	"errors"
	"fmt"
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
