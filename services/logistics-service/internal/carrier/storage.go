package carrier

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

func (s *PostgresStorage) CreateCarrier(ctx context.Context, c Carrier) (Carrier, error) {
	c.ID = uuid.NewString()
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	c.IsActive = true

	query := `
		INSERT INTO logistics.carrier (id, name, type, cost_per_km, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err := s.pool.Exec(ctx, query,
		c.ID, c.Name, string(c.Type), c.CostPerKm, c.IsActive, c.CreatedAt, c.UpdatedAt,
	); err != nil {
		return Carrier{}, fmt.Errorf("failed to insert carrier: %w", err)
	}

	return c, nil
}

func (s *PostgresStorage) GetCarrierByID(ctx context.Context, id string) (Carrier, error) {
	query := `
		SELECT id, name, type, cost_per_km, is_active, created_at, updated_at
		FROM logistics.carrier
		WHERE id = $1`

	var c Carrier
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Type, &c.CostPerKm, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Carrier{}, ErrCarrierNotFound
		}
		return Carrier{}, fmt.Errorf("failed to get carrier: %w", err)
	}

	return c, nil
}

func (s *PostgresStorage) ListCarriers(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Carrier, int, error) {
	where, args := buildCarrierWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM logistics.carrier" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count carriers: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapCarrierSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, name, type, cost_per_km, is_active, created_at, updated_at FROM logistics.carrier%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list carriers: %w", err)
	}
	defer rows.Close()

	var carriers []Carrier
	for rows.Next() {
		var c Carrier
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.CostPerKm, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan carrier: %w", err)
		}
		carriers = append(carriers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate carriers: %w", err)
	}

	return carriers, total, nil
}

func (s *PostgresStorage) PickDefaultActive(ctx context.Context) (Carrier, error) {
	query := `
		SELECT id, name, type, cost_per_km, is_active, created_at, updated_at
		FROM logistics.carrier
		WHERE is_active = true
		ORDER BY random()
		LIMIT 1`

	var c Carrier
	err := s.pool.QueryRow(ctx, query).Scan(
		&c.ID, &c.Name, &c.Type, &c.CostPerKm, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Carrier{}, ErrNoActiveCarrierFound
		}
		return Carrier{}, fmt.Errorf("failed to pick default carrier: %w", err)
	}
	return c, nil
}

func (s *PostgresStorage) UpdateCarrier(ctx context.Context, c Carrier) (Carrier, error) {
	query := `
		UPDATE logistics.carrier
		SET name = $1, type = $2, cost_per_km = $3, is_active = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, name, type, cost_per_km, is_active, created_at, updated_at`

	now := time.Now().UTC()
	var updated Carrier
	err := s.pool.QueryRow(ctx, query, c.Name, string(c.Type), c.CostPerKm, c.IsActive, now, c.ID).Scan(
		&updated.ID, &updated.Name, &updated.Type, &updated.CostPerKm, &updated.IsActive, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Carrier{}, ErrCarrierNotFound
		}
		return Carrier{}, fmt.Errorf("failed to update carrier: %w", err)
	}

	return updated, nil
}

func buildCarrierWhereClause(filter Filter) (string, []any) {
	var conditions []string
	var args []any

	if filter.Type != nil {
		args = append(args, string(*filter.Type))
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", len(args)))
	}
	if filter.Name != nil {
		args = append(args, "%"+*filter.Name+"%")
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapCarrierSortField(field string) string {
	switch field {
	case "created_at":
		return "created_at"
	case "name":
		return "name"
	case "type":
		return "type"
	case "cost_per_km":
		return "cost_per_km"
	default:
		return "created_at"
	}
}
