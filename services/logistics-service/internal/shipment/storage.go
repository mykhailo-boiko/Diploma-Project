package shipment

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

func (s *PostgresStorage) CreateShipment(ctx context.Context, sh Shipment) (Shipment, error) {
	sh.ID = uuid.NewString()
	now := time.Now().UTC()
	sh.CreatedAt = now
	sh.UpdatedAt = now
	sh.Status = StatusCreated

	query := `
		INSERT INTO logistics.shipment (id, order_id, warehouse_id, carrier_id, status, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if _, err := s.pool.Exec(ctx, query,
		sh.ID, sh.OrderID, sh.WarehouseID, sh.CarrierID, string(sh.Status), sh.Address, sh.CreatedAt, sh.UpdatedAt,
	); err != nil {
		return Shipment{}, fmt.Errorf("failed to insert shipment: %w", err)
	}

	return sh, nil
}

func (s *PostgresStorage) GetShipmentByID(ctx context.Context, id string) (Shipment, error) {
	query := `
		SELECT id, order_id, warehouse_id, carrier_id, status, address, created_at, updated_at
		FROM logistics.shipment
		WHERE id = $1 AND deleted_at IS NULL`

	var sh Shipment
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&sh.ID, &sh.OrderID, &sh.WarehouseID, &sh.CarrierID, &sh.Status, &sh.Address, &sh.CreatedAt, &sh.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Shipment{}, ErrShipmentNotFound
		}
		return Shipment{}, fmt.Errorf("failed to get shipment: %w", err)
	}

	return sh, nil
}

func (s *PostgresStorage) ListShipments(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Shipment, int, error) {
	where, args := buildShipmentWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM logistics.shipment" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count shipments: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapShipmentSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, order_id, warehouse_id, carrier_id, status, address, created_at, updated_at FROM logistics.shipment%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shipments: %w", err)
	}
	defer rows.Close()

	var shipments []Shipment
	for rows.Next() {
		var sh Shipment
		if err := rows.Scan(&sh.ID, &sh.OrderID, &sh.WarehouseID, &sh.CarrierID, &sh.Status, &sh.Address, &sh.CreatedAt, &sh.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan shipment: %w", err)
		}
		shipments = append(shipments, sh)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate shipments: %w", err)
	}

	return shipments, total, nil
}

func (s *PostgresStorage) UpdateShipmentStatus(ctx context.Context, id string, status Status) (Shipment, error) {
	query := `
		UPDATE logistics.shipment
		SET status = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, order_id, warehouse_id, carrier_id, status, address, created_at, updated_at`

	now := time.Now().UTC()
	var sh Shipment
	err := s.pool.QueryRow(ctx, query, string(status), now, id).Scan(
		&sh.ID, &sh.OrderID, &sh.WarehouseID, &sh.CarrierID, &sh.Status, &sh.Address, &sh.CreatedAt, &sh.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Shipment{}, ErrShipmentNotFound
		}
		return Shipment{}, fmt.Errorf("failed to update shipment status: %w", err)
	}

	return sh, nil
}

func (s *PostgresStorage) ReassignCarrierByCity(ctx context.Context, fromCarrierID, toCarrierID, city string, statuses []Status, dryRun bool) (ReassignResult, error) {
	res := ReassignResult{
		FromCarrierID: fromCarrierID,
		ToCarrierID:   toCarrierID,
		City:          city,
		DryRun:        dryRun,
		ReassignedIDs: []string{},
	}

	if len(statuses) == 0 {
		statuses = []Status{StatusCreated, StatusPickedUp, StatusInTransit}
	}
	statusStrings := make([]string, 0, len(statuses))
	for _, st := range statuses {
		statusStrings = append(statusStrings, string(st))
	}

	if dryRun {
		query := `
			SELECT id::text
			FROM logistics.shipment
			WHERE deleted_at IS NULL
				AND carrier_id = $1
				AND status = ANY($2)`
		args := []any{fromCarrierID, statusStrings}
		if city != "" {
			query += " AND trim(split_part(address, ',', 3)) ILIKE $3"
			args = append(args, city)
		}
		rows, err := s.pool.Query(ctx, query, args...)
		if err != nil {
			return res, fmt.Errorf("failed to scan candidates for dry-run: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return res, fmt.Errorf("failed to scan candidate id: %w", err)
			}
			res.ReassignedIDs = append(res.ReassignedIDs, id)
		}
		if err := rows.Err(); err != nil {
			return res, fmt.Errorf("failed to iterate candidate ids: %w", err)
		}
		res.Total = len(res.ReassignedIDs)
		return res, nil
	}

	query := `
		UPDATE logistics.shipment
		SET carrier_id = $2, updated_at = $3
		WHERE deleted_at IS NULL
			AND carrier_id = $1
			AND status = ANY($4)`
	args := []any{fromCarrierID, toCarrierID, time.Now().UTC(), statusStrings}

	if city != "" {
		query += " AND trim(split_part(address, ',', 3)) ILIKE $5"
		args = append(args, city)
	}

	query += " RETURNING id::text"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return res, fmt.Errorf("failed to reassign carrier: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return res, fmt.Errorf("failed to scan reassigned id: %w", err)
		}
		res.ReassignedIDs = append(res.ReassignedIDs, id)
	}
	if err := rows.Err(); err != nil {
		return res, fmt.Errorf("failed to iterate reassigned ids: %w", err)
	}
	res.Total = len(res.ReassignedIDs)
	return res, nil
}

func buildShipmentWhereClause(filter Filter) (string, []any) {
	var conditions []string
	var args []any

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.CarrierID != nil {
		args = append(args, *filter.CarrierID)
		conditions = append(conditions, fmt.Sprintf("carrier_id = $%d", len(args)))
	}
	if filter.OrderID != nil {
		args = append(args, *filter.OrderID)
		conditions = append(conditions, fmt.Sprintf("order_id = $%d", len(args)))
	}
	if filter.WarehouseID != nil {
		args = append(args, *filter.WarehouseID)
		conditions = append(conditions, fmt.Sprintf("warehouse_id = $%d", len(args)))
	}
	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", len(args)))
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapShipmentSortField(field string) string {
	switch field {
	case "created_at":
		return "created_at"
	case "status":
		return "status"
	case "order_id":
		return "order_id"
	default:
		return "created_at"
	}
}
