package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (s *PostgresStorage) CreateProduct(ctx context.Context, p Product) (Product, error) {
	p.ID = uuid.NewString()
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	query := `
		INSERT INTO inventory.product (id, sku, name, description, category, unit_price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := s.pool.Exec(ctx, query,
		p.ID, p.SKU, p.Name, p.Description, p.Category, p.UnitPrice, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Product{}, ErrSKUExists
		}
		return Product{}, fmt.Errorf("failed to insert product: %w", err)
	}

	return p, nil
}

func (s *PostgresStorage) GetProductByID(ctx context.Context, id string) (Product, error) {
	query := `
		SELECT id, sku, name, description, category, unit_price, created_at, updated_at
		FROM inventory.product
		WHERE id = $1 AND deleted_at IS NULL`

	var p Product
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Category, &p.UnitPrice, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, ErrProductNotFound
		}
		return Product{}, fmt.Errorf("failed to get product: %w", err)
	}

	return p, nil
}

func (s *PostgresStorage) ListProducts(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Product, int, error) {
	where, args := buildWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM inventory.product" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, sku, name, description, category, unit_price, created_at, updated_at FROM inventory.product%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.Category, &p.UnitPrice, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate products: %w", err)
	}

	return products, total, nil
}

func (s *PostgresStorage) UpdateProduct(ctx context.Context, p Product) (Product, error) {
	query := `
		UPDATE inventory.product
		SET name = $1, description = $2, category = $3, unit_price = $4, updated_at = $5
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING id, sku, name, description, category, unit_price, created_at, updated_at`

	now := time.Now().UTC()
	err := s.pool.QueryRow(ctx, query,
		p.Name, p.Description, p.Category, p.UnitPrice, now, p.ID,
	).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Category, &p.UnitPrice, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, ErrProductNotFound
		}
		return Product{}, fmt.Errorf("failed to update product: %w", err)
	}

	return p, nil
}

func (s *PostgresStorage) DeleteProduct(ctx context.Context, id string) error {
	query := `
		UPDATE inventory.product
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL`

	now := time.Now().UTC()
	result, err := s.pool.Exec(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrProductNotFound
	}

	return nil
}

func buildWhereClause(filter Filter) (string, []any) {
	var conditions []string
	var args []any

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.SKU != nil {
		args = append(args, *filter.SKU)
		conditions = append(conditions, fmt.Sprintf("sku = $%d", len(args)))
	}
	if filter.Name != nil {
		args = append(args, "%"+*filter.Name+"%")
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	if filter.Category != nil {
		args = append(args, *filter.Category)
		conditions = append(conditions, fmt.Sprintf("category = $%d", len(args)))
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapSortField(field string) string {
	switch field {
	case "name":
		return "name"
	case "sku":
		return "sku"
	case "category":
		return "category"
	case "unit_price":
		return "unit_price"
	case "created_at":
		return "created_at"
	default:
		return "created_at"
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
