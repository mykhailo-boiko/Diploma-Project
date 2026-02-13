package warehouse

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

func (s *PostgresStorage) CreateWarehouse(ctx context.Context, w Warehouse) (Warehouse, error) {
	w.ID = uuid.NewString()
	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now
	w.IsActive = true

	query := `
		INSERT INTO inventory.warehouse (id, name, address, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.pool.Exec(ctx, query,
		w.ID, w.Name, w.Address, w.IsActive, w.CreatedAt, w.UpdatedAt,
	)
	if err != nil {
		return Warehouse{}, fmt.Errorf("failed to insert warehouse: %w", err)
	}

	return w, nil
}

func (s *PostgresStorage) GetWarehouseByID(ctx context.Context, id string) (Warehouse, error) {
	query := `
		SELECT id, name, address, is_active, created_at, updated_at
		FROM inventory.warehouse
		WHERE id = $1`

	var w Warehouse
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&w.ID, &w.Name, &w.Address, &w.IsActive, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Warehouse{}, ErrWarehouseNotFound
		}
		return Warehouse{}, fmt.Errorf("failed to get warehouse: %w", err)
	}

	return w, nil
}

func (s *PostgresStorage) ListWarehouses(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Warehouse, int, error) {
	where, args := buildWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM inventory.warehouse" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count warehouses: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, name, address, is_active, created_at, updated_at FROM inventory.warehouse%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list warehouses: %w", err)
	}
	defer rows.Close()

	var warehouses []Warehouse
	for rows.Next() {
		var w Warehouse
		if err := rows.Scan(&w.ID, &w.Name, &w.Address, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan warehouse: %w", err)
		}
		warehouses = append(warehouses, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate warehouses: %w", err)
	}

	return warehouses, total, nil
}

func (s *PostgresStorage) UpdateWarehouse(ctx context.Context, w Warehouse) (Warehouse, error) {
	query := `
		UPDATE inventory.warehouse
		SET name = $1, address = $2, is_active = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, name, address, is_active, created_at, updated_at`

	now := time.Now().UTC()
	err := s.pool.QueryRow(ctx, query,
		w.Name, w.Address, w.IsActive, now, w.ID,
	).Scan(
		&w.ID, &w.Name, &w.Address, &w.IsActive, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Warehouse{}, ErrWarehouseNotFound
		}
		return Warehouse{}, fmt.Errorf("failed to update warehouse: %w", err)
	}

	return w, nil
}

func buildWhereClause(filter Filter) (string, []any) {
	var conditions []string
	var args []any

	if filter.Name != nil {
		args = append(args, "%"+*filter.Name+"%")
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapSortField(field string) string {
	switch field {
	case "name":
		return "name"
	case "created_at":
		return "created_at"
	default:
		return "created_at"
	}
}
