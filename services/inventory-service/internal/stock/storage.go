package stock

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

func (s *PostgresStorage) ListStock(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Stock, int, error) {
	where, args := buildWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM inventory.stock" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count stock: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at FROM inventory.stock%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list stock: %w", err)
	}
	defer rows.Close()

	var stocks []Stock
	for rows.Next() {
		var st Stock
		if err := rows.Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan stock: %w", err)
		}
		stocks = append(stocks, st)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate stock: %w", err)
	}

	return stocks, total, nil
}

func (s *PostgresStorage) GetStockByProductAndWarehouse(ctx context.Context, productID, warehouseID string) (Stock, error) {
	var st Stock
	err := s.pool.QueryRow(ctx,
		"SELECT id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at FROM inventory.stock WHERE product_id = $1 AND warehouse_id = $2",
		productID, warehouseID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, ErrStockNotFound
		}
		return Stock{}, fmt.Errorf("failed to get stock: %w", err)
	}

	return st, nil
}

func (s *PostgresStorage) GetOrCreateStock(ctx context.Context, productID, warehouseID string) (Stock, error) {
	st, err := s.GetStockByProductAndWarehouse(ctx, productID, warehouseID)
	if err == nil {
		return st, nil
	}
	if !errors.Is(err, ErrStockNotFound) {
		return Stock{}, err
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	_, err = s.pool.Exec(ctx,
		"INSERT INTO inventory.stock (id, product_id, warehouse_id, quantity, reserved, min_threshold, updated_at) VALUES ($1, $2, $3, 0, 0, 0, $4)",
		id, productID, warehouseID, now,
	)
	if err != nil {
		return Stock{}, fmt.Errorf("failed to create stock record: %w", err)
	}

	return Stock{
		ID:          id,
		ProductID:   productID,
		WarehouseID: warehouseID,
		Quantity:    0,
		Reserved:    0,
		Available:   0,
		UpdatedAt:   now,
	}, nil
}

func (s *PostgresStorage) ReserveStock(ctx context.Context, productID, warehouseID string, quantity int, reference string) (Stock, error) {
	if quantity <= 0 {
		return Stock{}, ErrInvalidQuantity
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Stock{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var st Stock
	err = tx.QueryRow(ctx,
		"SELECT id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at FROM inventory.stock WHERE product_id = $1 AND warehouse_id = $2 FOR UPDATE",
		productID, warehouseID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, ErrStockNotFound
		}
		return Stock{}, fmt.Errorf("failed to lock stock: %w", err)
	}

	if st.Available < quantity {
		return Stock{}, ErrInsufficientStock
	}

	now := time.Now().UTC()
	err = tx.QueryRow(ctx,
		"UPDATE inventory.stock SET reserved = reserved + $1, updated_at = $2 WHERE id = $3 RETURNING id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at",
		quantity, now, st.ID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		return Stock{}, fmt.Errorf("failed to update stock: %w", err)
	}

	if err := insertMovement(ctx, tx, st.ID, st.ProductID, st.WarehouseID, MovementTypeReserve, quantity, reference); err != nil {
		return Stock{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Stock{}, fmt.Errorf("failed to commit reserve: %w", err)
	}

	return st, nil
}

func (s *PostgresStorage) ReleaseStock(ctx context.Context, productID, warehouseID string, quantity int, reference string) (Stock, error) {
	if quantity <= 0 {
		return Stock{}, ErrInvalidQuantity
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Stock{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var st Stock
	err = tx.QueryRow(ctx,
		"SELECT id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at FROM inventory.stock WHERE product_id = $1 AND warehouse_id = $2 FOR UPDATE",
		productID, warehouseID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, ErrStockNotFound
		}
		return Stock{}, fmt.Errorf("failed to lock stock: %w", err)
	}

	if st.Reserved == 0 {
		return Stock{}, ErrNothingToRelease
	}
	if quantity > st.Reserved {
		return Stock{}, ErrReleaseExceedsReserved
	}

	now := time.Now().UTC()
	err = tx.QueryRow(ctx,
		"UPDATE inventory.stock SET reserved = reserved - $1, updated_at = $2 WHERE id = $3 RETURNING id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at",
		quantity, now, st.ID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		return Stock{}, fmt.Errorf("failed to update stock: %w", err)
	}

	if err := insertMovement(ctx, tx, st.ID, st.ProductID, st.WarehouseID, MovementTypeRelease, quantity, reference); err != nil {
		return Stock{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Stock{}, fmt.Errorf("failed to commit release: %w", err)
	}

	return st, nil
}

func (s *PostgresStorage) AdjustStock(ctx context.Context, productID, warehouseID string, quantity int, movementType, reference string) (Stock, error) {
	if quantity <= 0 {
		return Stock{}, ErrInvalidQuantity
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Stock{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var st Stock
	err = tx.QueryRow(ctx,
		"SELECT id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at FROM inventory.stock WHERE product_id = $1 AND warehouse_id = $2 FOR UPDATE",
		productID, warehouseID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, ErrStockNotFound
		}
		return Stock{}, fmt.Errorf("failed to lock stock: %w", err)
	}

	var delta int
	switch movementType {
	case MovementTypeOutbound:
		if st.Available < quantity {
			return Stock{}, ErrInsufficientStock
		}
		delta = -quantity
	case MovementTypeInbound, MovementTypeAdjustment:
		delta = quantity
	default:
		return Stock{}, fmt.Errorf("invalid movement type: %s", movementType)
	}

	now := time.Now().UTC()
	err = tx.QueryRow(ctx,
		"UPDATE inventory.stock SET quantity = quantity + $1, updated_at = $2 WHERE id = $3 RETURNING id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at",
		delta, now, st.ID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		return Stock{}, fmt.Errorf("failed to update stock: %w", err)
	}

	if err := insertMovement(ctx, tx, st.ID, st.ProductID, st.WarehouseID, movementType, quantity, reference); err != nil {
		return Stock{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Stock{}, fmt.Errorf("failed to commit adjust: %w", err)
	}

	return st, nil
}

func (s *PostgresStorage) ListMovements(ctx context.Context, filter MovementFilter, sort pagination.Sort, page pagination.Page) ([]Movement, int, error) {
	where, args := buildMovementWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM inventory.stock_movement" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count movements: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapMovementSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, stock_id, product_id, warehouse_id, type, quantity, reference, created_at FROM inventory.stock_movement%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list movements: %w", err)
	}
	defer rows.Close()

	var movements []Movement
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.StockID, &m.ProductID, &m.WarehouseID, &m.Type, &m.Quantity, &m.Reference, &m.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan movement: %w", err)
		}
		movements = append(movements, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate movements: %w", err)
	}

	return movements, total, nil
}

func (s *PostgresStorage) ListLowStock(ctx context.Context, page pagination.Page) ([]LowStockItem, int, error) {
	countQuery := `
		SELECT COUNT(*) FROM inventory.stock s
		JOIN inventory.product p ON p.id = s.product_id AND p.deleted_at IS NULL
		WHERE s.min_threshold > 0 AND (s.quantity - s.reserved) < s.min_threshold`

	var total int
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count low stock: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	query := `
		SELECT s.id, s.product_id, s.warehouse_id, s.quantity, s.reserved, (s.quantity - s.reserved) AS available, s.min_threshold, s.updated_at, p.name, p.sku, w.name
		FROM inventory.stock s
		JOIN inventory.product p ON p.id = s.product_id AND p.deleted_at IS NULL
		JOIN inventory.warehouse w ON w.id = s.warehouse_id
		WHERE s.min_threshold > 0 AND (s.quantity - s.reserved) < s.min_threshold
		ORDER BY (s.quantity - s.reserved) ASC
		LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list low stock: %w", err)
	}
	defer rows.Close()

	var items []LowStockItem
	for rows.Next() {
		var item LowStockItem
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.WarehouseID, &item.Quantity, &item.Reserved,
			&item.Available, &item.MinThreshold, &item.UpdatedAt, &item.ProductName, &item.ProductSKU, &item.WarehouseName,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan low stock item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate low stock: %w", err)
	}

	return items, total, nil
}

func (s *PostgresStorage) GetInventoryReport(ctx context.Context) (InventoryReport, error) {
	var report InventoryReport

	err := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT COUNT(*) FROM inventory.product WHERE deleted_at IS NULL), 0),
			COALESCE((SELECT COUNT(*) FROM inventory.warehouse), 0),
			COALESCE(SUM(quantity), 0),
			COALESCE(SUM(reserved), 0),
			COALESCE(SUM(quantity - reserved), 0),
			COALESCE((SELECT COUNT(*) FROM inventory.stock WHERE min_threshold > 0 AND (quantity - reserved) < min_threshold), 0)
		FROM inventory.stock
	`).Scan(&report.TotalProducts, &report.TotalWarehouses, &report.TotalQuantity, &report.TotalReserved, &report.TotalAvailable, &report.LowStockCount)
	if err != nil {
		return InventoryReport{}, fmt.Errorf("failed to get report totals: %w", err)
	}

	whRows, err := s.pool.Query(ctx, `
		SELECT s.warehouse_id, w.name, COALESCE(SUM(s.quantity), 0), COALESCE(SUM(s.reserved), 0), COALESCE(SUM(s.quantity - s.reserved), 0), COUNT(DISTINCT s.product_id)
		FROM inventory.stock s
		JOIN inventory.warehouse w ON w.id = s.warehouse_id
		GROUP BY s.warehouse_id, w.name
		ORDER BY w.name
	`)
	if err != nil {
		return InventoryReport{}, fmt.Errorf("failed to query warehouse summary: %w", err)
	}
	defer whRows.Close()

	for whRows.Next() {
		var ws WarehouseSummary
		if err := whRows.Scan(&ws.WarehouseID, &ws.WarehouseName, &ws.TotalQuantity, &ws.TotalReserved, &ws.TotalAvailable, &ws.ProductCount); err != nil {
			return InventoryReport{}, fmt.Errorf("failed to scan warehouse summary: %w", err)
		}
		report.ByWarehouse = append(report.ByWarehouse, ws)
	}
	if err := whRows.Err(); err != nil {
		return InventoryReport{}, fmt.Errorf("failed to iterate warehouse summary: %w", err)
	}

	catRows, err := s.pool.Query(ctx, `
		SELECT COALESCE(p.category, ''), COALESCE(SUM(s.quantity), 0), COALESCE(SUM(s.reserved), 0), COALESCE(SUM(s.quantity - s.reserved), 0), COUNT(DISTINCT s.product_id)
		FROM inventory.stock s
		JOIN inventory.product p ON p.id = s.product_id AND p.deleted_at IS NULL
		GROUP BY p.category
		ORDER BY p.category
	`)
	if err != nil {
		return InventoryReport{}, fmt.Errorf("failed to query category summary: %w", err)
	}
	defer catRows.Close()

	for catRows.Next() {
		var cs CategorySummary
		if err := catRows.Scan(&cs.Category, &cs.TotalQuantity, &cs.TotalReserved, &cs.TotalAvailable, &cs.ProductCount); err != nil {
			return InventoryReport{}, fmt.Errorf("failed to scan category summary: %w", err)
		}
		report.ByCategory = append(report.ByCategory, cs)
	}
	if err := catRows.Err(); err != nil {
		return InventoryReport{}, fmt.Errorf("failed to iterate category summary: %w", err)
	}

	if report.ByWarehouse == nil {
		report.ByWarehouse = []WarehouseSummary{}
	}
	if report.ByCategory == nil {
		report.ByCategory = []CategorySummary{}
	}

	return report, nil
}

func (s *PostgresStorage) UpdateMinThreshold(ctx context.Context, productID, warehouseID string, threshold int) (Stock, error) {
	var st Stock
	err := s.pool.QueryRow(ctx,
		"UPDATE inventory.stock SET min_threshold = $1, updated_at = NOW() WHERE product_id = $2 AND warehouse_id = $3 RETURNING id, product_id, warehouse_id, quantity, reserved, (quantity - reserved) AS available, min_threshold, updated_at",
		threshold, productID, warehouseID,
	).Scan(&st.ID, &st.ProductID, &st.WarehouseID, &st.Quantity, &st.Reserved, &st.Available, &st.MinThreshold, &st.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, ErrStockNotFound
		}
		return Stock{}, fmt.Errorf("failed to update min_threshold: %w", err)
	}

	return st, nil
}

func insertMovement(ctx context.Context, tx pgx.Tx, stockID, productID, warehouseID, movementType string, quantity int, reference string) error {
	id := uuid.New().String()
	_, err := tx.Exec(ctx,
		"INSERT INTO inventory.stock_movement (id, stock_id, product_id, warehouse_id, type, quantity, reference, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())",
		id, stockID, productID, warehouseID, movementType, quantity, reference,
	)
	if err != nil {
		return fmt.Errorf("failed to insert movement: %w", err)
	}

	return nil
}

func buildWhereClause(filter Filter) (string, []any) {
	var conditions []string
	var args []any

	if filter.ProductID != nil {
		args = append(args, *filter.ProductID)
		conditions = append(conditions, fmt.Sprintf("product_id = $%d", len(args)))
	}
	if filter.WarehouseID != nil {
		args = append(args, *filter.WarehouseID)
		conditions = append(conditions, fmt.Sprintf("warehouse_id = $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func buildMovementWhereClause(filter MovementFilter) (string, []any) {
	var conditions []string
	var args []any

	if filter.StockID != nil {
		args = append(args, *filter.StockID)
		conditions = append(conditions, fmt.Sprintf("stock_id = $%d", len(args)))
	}
	if filter.ProductID != nil {
		args = append(args, *filter.ProductID)
		conditions = append(conditions, fmt.Sprintf("product_id = $%d", len(args)))
	}
	if filter.WarehouseID != nil {
		args = append(args, *filter.WarehouseID)
		conditions = append(conditions, fmt.Sprintf("warehouse_id = $%d", len(args)))
	}
	if filter.Type != nil {
		args = append(args, *filter.Type)
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}
	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapSortField(field string) string {
	switch field {
	case "product_id":
		return "product_id"
	case "warehouse_id":
		return "warehouse_id"
	case "quantity":
		return "quantity"
	case "available":
		return "(quantity - reserved)"
	case "updated_at":
		return "updated_at"
	default:
		return "updated_at"
	}
}

func mapMovementSortField(field string) string {
	switch field {
	case "type":
		return "type"
	case "quantity":
		return "quantity"
	case "created_at":
		return "created_at"
	default:
		return "created_at"
	}
}
