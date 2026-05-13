package shipment

import (
	"context"
	"encoding/json"
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

const shipmentColumns = `
	id, order_id, warehouse_id, carrier_id, status, tracking_number, address,
	recipient, sender,
	estimated_delivery_at, delivered_at, delivery_attempts,
	COALESCE(delivery_signature,''), COALESCE(delivery_photo_url,''),
	COALESCE(current_location_city,''), COALESCE(current_location_hub,''),
	created_at, updated_at`

func scanShipment(row pgx.Row, sh *Shipment) error {
	var recipientBytes, senderBytes []byte
	err := row.Scan(
		&sh.ID, &sh.OrderID, &sh.WarehouseID, &sh.CarrierID, &sh.Status,
		&sh.TrackingNumber, &sh.Address,
		&recipientBytes, &senderBytes,
		&sh.EstimatedDeliveryAt, &sh.DeliveredAt, &sh.DeliveryAttempts,
		&sh.DeliverySignature, &sh.DeliveryPhotoURL,
		&sh.CurrentLocationCity, &sh.CurrentLocationHub,
		&sh.CreatedAt, &sh.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if len(recipientBytes) > 0 {
		_ = json.Unmarshal(recipientBytes, &sh.Recipient)
	}
	if len(senderBytes) > 0 {
		_ = json.Unmarshal(senderBytes, &sh.Sender)
	}
	return nil
}

func generateTrackingNumber() string {
	id := uuid.NewString()
	suffix := strings.ToUpper(strings.ReplaceAll(id, "-", ""))[:6]
	return fmt.Sprintf("CO-%d-%s", time.Now().UTC().Year(), suffix)
}

func (s *PostgresStorage) CreateShipment(ctx context.Context, sh Shipment) (Shipment, error) {
	sh.ID = uuid.NewString()
	now := time.Now().UTC()
	sh.CreatedAt = now
	sh.UpdatedAt = now
	if sh.Status == "" {
		sh.Status = StatusLabelCreated
	}
	if sh.TrackingNumber == "" {
		sh.TrackingNumber = generateTrackingNumber()
	}

	recipientJSON, _ := json.Marshal(sh.Recipient)
	senderJSON, _ := json.Marshal(sh.Sender)

	query := `
		INSERT INTO logistics.shipment
			(id, order_id, warehouse_id, carrier_id, status, tracking_number, address,
			 recipient, sender, estimated_delivery_at, delivery_attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, $11, $12)`

	if _, err := s.pool.Exec(ctx, query,
		sh.ID, sh.OrderID, sh.WarehouseID, sh.CarrierID, string(sh.Status),
		sh.TrackingNumber, sh.Address,
		recipientJSON, senderJSON,
		sh.EstimatedDeliveryAt,
		sh.CreatedAt, sh.UpdatedAt,
	); err != nil {
		return Shipment{}, fmt.Errorf("failed to insert shipment: %w", err)
	}

	return sh, nil
}

func (s *PostgresStorage) FindByOrderID(ctx context.Context, orderID string) ([]Shipment, error) {
	query := `SELECT ` + shipmentColumns + ` FROM logistics.shipment WHERE order_id = $1 AND deleted_at IS NULL LIMIT 16`
	rows, err := s.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shipments by order: %w", err)
	}
	defer rows.Close()
	var out []Shipment
	for rows.Next() {
		var sh Shipment
		if err := scanShipment(rows, &sh); err != nil {
			return nil, fmt.Errorf("failed to scan shipment: %w", err)
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) GetShipmentByID(ctx context.Context, id string) (Shipment, error) {
	query := `SELECT ` + shipmentColumns + ` FROM logistics.shipment WHERE id = $1 AND deleted_at IS NULL`
	var sh Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, id), &sh); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Shipment{}, ErrShipmentNotFound
		}
		return Shipment{}, fmt.Errorf("failed to get shipment: %w", err)
	}
	return sh, nil
}

func (s *PostgresStorage) GetShipmentByTracking(ctx context.Context, trackingNumber string) (Shipment, error) {
	query := `SELECT ` + shipmentColumns + ` FROM logistics.shipment WHERE tracking_number = $1 AND deleted_at IS NULL`
	var sh Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, trackingNumber), &sh); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Shipment{}, ErrShipmentNotFound
		}
		return Shipment{}, fmt.Errorf("failed to get shipment by tracking: %w", err)
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
		"SELECT %s FROM logistics.shipment%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		shipmentColumns, where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
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
		if err := scanShipment(rows, &sh); err != nil {
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
		SET status = $1::text, updated_at = $2,
		    delivered_at = CASE WHEN $1::text = 'delivered' THEN $2 ELSE delivered_at END
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING ` + shipmentColumns
	now := time.Now().UTC()
	var sh Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, string(status), now, id), &sh); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Shipment{}, ErrShipmentNotFound
		}
		return Shipment{}, fmt.Errorf("failed to update shipment status: %w", err)
	}
	return sh, nil
}

func (s *PostgresStorage) RecordEvent(ctx context.Context, e ShipmentEvent) (ShipmentEvent, error) {
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if e.RecordedBy == "" {
		e.RecordedBy = "system"
	}
	payloadJSON, _ := json.Marshal(e.Payload)
	if len(payloadJSON) == 0 || string(payloadJSON) == "null" {
		payloadJSON = []byte("{}")
	}

	query := `
		INSERT INTO logistics.shipment_event
			(shipment_id, event_type, location_city, location_hub, notes, occurred_at, recorded_by, payload)
		VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), NULLIF($5,''), $6, $7, $8)
		RETURNING id`
	if err := s.pool.QueryRow(ctx, query,
		e.ShipmentID, e.Type, e.LocationCity, e.LocationHub, e.Notes, e.OccurredAt, e.RecordedBy, payloadJSON,
	).Scan(&e.ID); err != nil {
		return ShipmentEvent{}, fmt.Errorf("failed to insert shipment event: %w", err)
	}

	if e.LocationCity != "" || e.LocationHub != "" {
		_, _ = s.pool.Exec(ctx, `
			UPDATE logistics.shipment
			SET current_location_city = NULLIF($2,''),
			    current_location_hub = NULLIF($3,''),
			    updated_at = $4
			WHERE id = $1 AND deleted_at IS NULL`,
			e.ShipmentID, e.LocationCity, e.LocationHub, time.Now().UTC(),
		)
	}

	return e, nil
}

func (s *PostgresStorage) GetTimeline(ctx context.Context, shipmentID string) ([]ShipmentEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, shipment_id::text, event_type,
		       COALESCE(location_city,''), COALESCE(location_hub,''),
		       COALESCE(notes,''), occurred_at, recorded_by, payload
		FROM logistics.shipment_event
		WHERE shipment_id = $1
		ORDER BY occurred_at ASC`, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline: %w", err)
	}
	defer rows.Close()

	var events []ShipmentEvent
	for rows.Next() {
		var e ShipmentEvent
		var payloadBytes []byte
		if err := rows.Scan(&e.ID, &e.ShipmentID, &e.Type,
			&e.LocationCity, &e.LocationHub, &e.Notes, &e.OccurredAt, &e.RecordedBy, &payloadBytes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		if len(payloadBytes) > 0 {
			_ = json.Unmarshal(payloadBytes, &e.Payload)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func mergeAddress(existing Address, patch RecipientPatch) Address {
	if patch.FullName != nil {
		existing.FullName = *patch.FullName
	}
	if patch.Phone != nil {
		existing.Phone = *patch.Phone
	}
	if patch.Email != nil {
		existing.Email = *patch.Email
	}
	if patch.Company != nil {
		existing.Company = *patch.Company
	}
	if patch.Street != nil {
		existing.Street = *patch.Street
	}
	if patch.City != nil {
		existing.City = *patch.City
	}
	if patch.Region != nil {
		existing.Region = *patch.Region
	}
	if patch.Postcode != nil {
		existing.Postcode = *patch.Postcode
	}
	if patch.Country != nil {
		existing.Country = *patch.Country
	}
	if patch.DeliveryNotes != nil {
		existing.DeliveryNotes = *patch.DeliveryNotes
	}
	return existing
}

func (s *PostgresStorage) UpdateRecipient(ctx context.Context, id string, patch RecipientPatch) (Shipment, error) {
	sh, err := s.GetShipmentByID(ctx, id)
	if err != nil {
		return Shipment{}, err
	}
	merged := mergeAddress(sh.Recipient, patch)
	mergedJSON, _ := json.Marshal(merged)

	query := `UPDATE logistics.shipment SET recipient = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL RETURNING ` + shipmentColumns
	var out Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, mergedJSON, time.Now().UTC(), id), &out); err != nil {
		return Shipment{}, fmt.Errorf("failed to update recipient: %w", err)
	}
	return out, nil
}

func (s *PostgresStorage) UpdateSender(ctx context.Context, id string, patch RecipientPatch) (Shipment, error) {
	sh, err := s.GetShipmentByID(ctx, id)
	if err != nil {
		return Shipment{}, err
	}
	merged := mergeAddress(sh.Sender, patch)
	mergedJSON, _ := json.Marshal(merged)

	query := `UPDATE logistics.shipment SET sender = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL RETURNING ` + shipmentColumns
	var out Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, mergedJSON, time.Now().UTC(), id), &out); err != nil {
		return Shipment{}, fmt.Errorf("failed to update sender: %w", err)
	}
	return out, nil
}

func (s *PostgresStorage) UpdateEstimatedDelivery(ctx context.Context, id string, eta time.Time) (Shipment, error) {
	query := `UPDATE logistics.shipment SET estimated_delivery_at = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL RETURNING ` + shipmentColumns
	var out Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, eta, time.Now().UTC(), id), &out); err != nil {
		return Shipment{}, fmt.Errorf("failed to update ETA: %w", err)
	}
	return out, nil
}

func (s *PostgresStorage) UpdateCurrentLocation(ctx context.Context, id, city, hub string) (Shipment, error) {
	query := `
		UPDATE logistics.shipment
		SET current_location_city = NULLIF($1,''),
		    current_location_hub = NULLIF($2,''),
		    updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING ` + shipmentColumns
	var out Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, city, hub, time.Now().UTC(), id), &out); err != nil {
		return Shipment{}, fmt.Errorf("failed to update location: %w", err)
	}
	return out, nil
}

func (s *PostgresStorage) RecordDelivery(ctx context.Context, id, signature, photoURL string) (Shipment, error) {
	existing, err := s.GetShipmentByID(ctx, id)
	if err != nil {
		return Shipment{}, err
	}
	if existing.Status == "delivered" {
		return Shipment{}, ErrShipmentAlreadyDelivered
	}
	if isTerminalStatus(existing.Status) {
		return Shipment{}, fmt.Errorf("cannot record delivery for shipment in terminal status %q: %w",
			existing.Status, ErrShipmentTerminalState)
	}
	now := time.Now().UTC()
	query := `
		UPDATE logistics.shipment
		SET status = 'delivered',
		    delivered_at = $1,
		    delivery_signature = NULLIF($2,''),
		    delivery_photo_url = NULLIF($3,''),
		    updated_at = $1
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING ` + shipmentColumns
	var out Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, now, signature, photoURL, id), &out); err != nil {
		return Shipment{}, fmt.Errorf("failed to record delivery: %w", err)
	}
	return out, nil
}

func isTerminalStatus(status Status) bool {
	switch string(status) {
	case "delivered", "returned_to_sender", "returned", "cancelled":
		return true
	}
	return false
}

func (s *PostgresStorage) RecordDeliveryAttempt(ctx context.Context, a DeliveryAttempt) (DeliveryAttempt, error) {
	existing, err := s.GetShipmentByID(ctx, a.ShipmentID)
	if err != nil {
		return DeliveryAttempt{}, err
	}
	if isTerminalStatus(existing.Status) {
		return DeliveryAttempt{}, fmt.Errorf("cannot record attempt for shipment in terminal status %q: %w",
			existing.Status, ErrShipmentTerminalState)
	}
	if a.OccurredAt.IsZero() {
		a.OccurredAt = time.Now().UTC()
	}
	if a.AttemptNumber == 0 {
		if err := s.pool.QueryRow(ctx, `
			SELECT COALESCE(MAX(attempt_number),0) + 1
			FROM logistics.delivery_attempt
			WHERE shipment_id = $1`, a.ShipmentID).Scan(&a.AttemptNumber); err != nil {
			return DeliveryAttempt{}, fmt.Errorf("failed to compute attempt number: %w", err)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DeliveryAttempt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.QueryRow(ctx, `
		INSERT INTO logistics.delivery_attempt
			(shipment_id, attempt_number, reason, notes, next_attempt_at, occurred_at)
		VALUES ($1, $2, $3, NULLIF($4,''), $5, $6)
		RETURNING id::text`,
		a.ShipmentID, a.AttemptNumber, a.Reason, a.Notes, a.NextAttemptAt, a.OccurredAt,
	).Scan(&a.ID); err != nil {
		return DeliveryAttempt{}, fmt.Errorf("failed to insert delivery attempt: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE logistics.shipment
		SET delivery_attempts = delivery_attempts + 1, updated_at = $1
		WHERE id = $2`, time.Now().UTC(), a.ShipmentID); err != nil {
		return DeliveryAttempt{}, fmt.Errorf("failed to bump attempts counter: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return DeliveryAttempt{}, err
	}
	return a, nil
}

func (s *PostgresStorage) GetDeliveryAttempts(ctx context.Context, shipmentID string) ([]DeliveryAttempt, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, shipment_id::text, attempt_number, reason, COALESCE(notes,''),
		       next_attempt_at, occurred_at
		FROM logistics.delivery_attempt
		WHERE shipment_id = $1
		ORDER BY attempt_number ASC`, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attempts: %w", err)
	}
	defer rows.Close()

	var attempts []DeliveryAttempt
	for rows.Next() {
		var a DeliveryAttempt
		if err := rows.Scan(&a.ID, &a.ShipmentID, &a.AttemptNumber, &a.Reason, &a.Notes,
			&a.NextAttemptAt, &a.OccurredAt,
		); err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, rows.Err()
}

func (s *PostgresStorage) RedirectAddress(ctx context.Context, id string, newAddress Address, reason string) (Shipment, error) {
	existing, err := s.GetShipmentByID(ctx, id)
	if err != nil {
		return Shipment{}, err
	}
	merged := mergeAddressFromStruct(existing.Recipient, newAddress)
	addrJSON, _ := json.Marshal(merged)
	now := time.Now().UTC()
	flatAddress := strings.Join([]string{
		merged.Street, merged.City, merged.Country, merged.Postcode,
	}, ", ")
	query := `
		UPDATE logistics.shipment
		SET recipient = $1, address = $2, status = 'redirected', updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING ` + shipmentColumns
	var out Shipment
	if err := scanShipment(s.pool.QueryRow(ctx, query, addrJSON, flatAddress, now, id), &out); err != nil {
		return Shipment{}, fmt.Errorf("failed to redirect: %w", err)
	}
	_, _ = s.RecordEvent(ctx, ShipmentEvent{
		ShipmentID:   id,
		Type:         "redirected",
		LocationCity: newAddress.City,
		Notes:        reason,
		RecordedBy:   "system",
		Payload: map[string]any{
			"new_city":    newAddress.City,
			"new_street":  newAddress.Street,
			"reason":      reason,
		},
	})
	return out, nil
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
		statuses = []Status{StatusCreated, StatusLabelCreated, StatusAwaitingPickup, StatusPickedUp, StatusInTransit, StatusAtHub}
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
			query += " AND (recipient->>'city' ILIKE $3 OR trim(split_part(address, ',', 3)) ILIKE $3)"
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
		query += " AND (recipient->>'city' ILIKE $5 OR trim(split_part(address, ',', 3)) ILIKE $5)"
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
	case "estimated_delivery_at":
		return "estimated_delivery_at"
	default:
		return "created_at"
	}
}

func mergeAddressFromStruct(existing Address, patch Address) Address {
	if patch.FullName != "" {
		existing.FullName = patch.FullName
	}
	if patch.Phone != "" {
		existing.Phone = patch.Phone
	}
	if patch.Email != "" {
		existing.Email = patch.Email
	}
	if patch.Company != "" {
		existing.Company = patch.Company
	}
	if patch.Street != "" {
		existing.Street = patch.Street
	}
	if patch.City != "" {
		existing.City = patch.City
	}
	if patch.Region != "" {
		existing.Region = patch.Region
	}
	if patch.Postcode != "" {
		existing.Postcode = patch.Postcode
	}
	if patch.Country != "" {
		existing.Country = patch.Country
	}
	if patch.DeliveryNotes != "" {
		existing.DeliveryNotes = patch.DeliveryNotes
	}
	return existing
}

func (s *PostgresStorage) InTransitSummary(ctx context.Context) (InTransitSummaryResult, error) {
	const activeStatusesSQL = `('in_transit','at_hub','out_for_delivery','picked_up','delivery_attempted','held_at_office','awaiting_pickup','label_created','created')`

	res := InTransitSummaryResult{}

	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM logistics.shipment
		WHERE deleted_at IS NULL AND status IN `+activeStatusesSQL).Scan(&res.Total); err != nil {
		return res, fmt.Errorf("failed to count in-transit: %w", err)
	}

	rowsByStatus, err := s.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM logistics.shipment
		WHERE deleted_at IS NULL AND status IN `+activeStatusesSQL+`
		GROUP BY status ORDER BY status`)
	if err != nil {
		return res, fmt.Errorf("failed to group by status: %w", err)
	}
	defer rowsByStatus.Close()
	for rowsByStatus.Next() {
		var b StatusBucket
		if err := rowsByStatus.Scan(&b.Status, &b.Count); err != nil {
			return res, err
		}
		res.ByStatus = append(res.ByStatus, b)
	}

	rowsByCarrier, err := s.pool.Query(ctx, `
		SELECT s.carrier_id::text, COALESCE(c.name, ''), COUNT(*)
		FROM logistics.shipment s
		LEFT JOIN logistics.carrier c ON c.id = s.carrier_id
		WHERE s.deleted_at IS NULL AND s.status IN `+activeStatusesSQL+`
		GROUP BY s.carrier_id, c.name
		ORDER BY COUNT(*) DESC
		LIMIT 20`)
	if err != nil {
		return res, fmt.Errorf("failed to group by carrier: %w", err)
	}
	defer rowsByCarrier.Close()
	for rowsByCarrier.Next() {
		var b CarrierBucket
		if err := rowsByCarrier.Scan(&b.CarrierID, &b.CarrierName, &b.Count); err != nil {
			return res, err
		}
		res.ByCarrier = append(res.ByCarrier, b)
	}

	rowsByHub, err := s.pool.Query(ctx, `
		SELECT COALESCE(current_location_hub, ''), COUNT(*)
		FROM logistics.shipment
		WHERE deleted_at IS NULL AND status IN `+activeStatusesSQL+`
		GROUP BY current_location_hub
		ORDER BY COUNT(*) DESC
		LIMIT 20`)
	if err != nil {
		return res, fmt.Errorf("failed to group by hub: %w", err)
	}
	defer rowsByHub.Close()
	for rowsByHub.Next() {
		var b HubBucket
		if err := rowsByHub.Scan(&b.Hub, &b.Count); err != nil {
			return res, err
		}
		res.ByHub = append(res.ByHub, b)
	}

	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM logistics.shipment
		WHERE deleted_at IS NULL AND status IN `+activeStatusesSQL+`
		  AND estimated_delivery_at::date = CURRENT_DATE`).Scan(&res.EstimatedToDeliverToday); err != nil {
		return res, fmt.Errorf("failed to count today: %w", err)
	}

	return res, nil
}
