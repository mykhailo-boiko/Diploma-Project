package controller

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"strings"

	"github.com/haradrim/chainorchestra/internal/pkg/audit"
	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/carrier"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/route"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/shipment"
)

type Service struct {
	shipments shipment.Storage
	carriers  carrier.Storage
	nc        *natspkg.Client
	audit     *audit.Logger
	log       *zap.Logger
}

func NewService(shipments shipment.Storage, carriers carrier.Storage, nc *natspkg.Client, auditLog *audit.Logger, log *zap.Logger) *Service {
	return &Service{shipments: shipments, carriers: carriers, nc: nc, audit: auditLog, log: log}
}

type CreateShipmentRequest struct {
	OrderID     string `json:"order_id"`
	WarehouseID string `json:"warehouse_id"`
	CarrierID   string `json:"carrier_id"`
	Address     string `json:"address"`
}

type UpdateShipmentStatusRequest struct {
	Status shipment.Status `json:"status"`
}

type CreateCarrierRequest struct {
	Name      string       `json:"name"`
	Type      carrier.Type `json:"type"`
	CostPerKm float64      `json:"cost_per_km"`
}

type UpdateCarrierRequest struct {
	Name      string       `json:"name"`
	Type      carrier.Type `json:"type"`
	CostPerKm float64      `json:"cost_per_km"`
	IsActive  bool         `json:"is_active"`
}

type CalculateRouteRequest struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	CarrierID   string `json:"carrier_id"`
}

type BulkStatusRequest struct {
	Updates []BulkStatusItem `json:"updates"`
}

type BulkStatusItem struct {
	ShipmentID string          `json:"shipment_id"`
	Status     shipment.Status `json:"status"`
}

type BulkStatusResult struct {
	ShipmentID string           `json:"shipment_id"`
	Status     string           `json:"status"`
	Shipment   *shipment.Shipment `json:"shipment,omitempty"`
	Error      string           `json:"error,omitempty"`
}

type PerformanceStats struct {
	TotalDelivered int     `json:"total_delivered"`
	OnTime         int     `json:"on_time"`
	Late           int     `json:"late"`
	OnTimeRate     float64 `json:"on_time_rate"`
}

func (s *Service) CreateShipment(ctx context.Context, req CreateShipmentRequest) (shipment.Shipment, error) {
	if _, err := s.carriers.GetCarrierByID(ctx, req.CarrierID); err != nil {
		return shipment.Shipment{}, fmt.Errorf("failed to verify carrier: %w", err)
	}

	sh := shipment.Shipment{
		OrderID:     req.OrderID,
		WarehouseID: req.WarehouseID,
		CarrierID:   req.CarrierID,
		Address:     req.Address,
	}

	created, err := s.shipments.CreateShipment(ctx, sh)
	if err != nil {
		return shipment.Shipment{}, fmt.Errorf("failed to create shipment: %w", err)
	}

	s.publishShipmentCreated(created)

	return created, nil
}

func (s *Service) GetShipmentByID(ctx context.Context, id string) (shipment.Shipment, error) {
	return s.shipments.GetShipmentByID(ctx, id)
}

func (s *Service) ListShipments(ctx context.Context, filter shipment.Filter, sort pagination.Sort, page pagination.Page) ([]shipment.Shipment, int, error) {
	return s.shipments.ListShipments(ctx, filter, sort, page)
}

func (s *Service) UpdateShipmentStatus(ctx context.Context, id string, newStatus shipment.Status) (shipment.Shipment, error) {
	current, err := s.shipments.GetShipmentByID(ctx, id)
	if err != nil {
		return shipment.Shipment{}, err
	}

	if !shipment.CanTransition(current.Status, newStatus) {
		return shipment.Shipment{}, shipment.ErrInvalidTransition
	}

	updated, err := s.shipments.UpdateShipmentStatus(ctx, id, newStatus)
	if err != nil {
		return shipment.Shipment{}, fmt.Errorf("failed to update shipment status: %w", err)
	}

	_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
		ShipmentID: id,
		Type:       string(newStatus),
		Notes:      fmt.Sprintf("status changed from %s to %s", current.Status, newStatus),
		RecordedBy: actorEmail(ctx),
	})

	s.publishShipmentStatusChanged(updated, current.Status)
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.update_status",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       map[string]any{"old_status": current.Status, "new_status": newStatus},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})

	return updated, nil
}

func (s *Service) GetShipmentByTracking(ctx context.Context, trackingNumber string) (shipment.Shipment, error) {
	return s.shipments.GetShipmentByTracking(ctx, trackingNumber)
}

func (s *Service) GetTimeline(ctx context.Context, shipmentID string) (shipment.Timeline, error) {
	sh, err := s.shipments.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return shipment.Timeline{}, err
	}
	events, err := s.shipments.GetTimeline(ctx, shipmentID)
	if err != nil {
		return shipment.Timeline{}, err
	}
	attempts, err := s.shipments.GetDeliveryAttempts(ctx, shipmentID)
	if err != nil {
		return shipment.Timeline{}, err
	}
	if events == nil {
		events = []shipment.ShipmentEvent{}
	}
	if attempts == nil {
		attempts = []shipment.DeliveryAttempt{}
	}
	return shipment.Timeline{Shipment: sh, Events: events, DeliveryAttempts: attempts}, nil
}

func (s *Service) GetTimelineByTracking(ctx context.Context, trackingNumber string) (shipment.Timeline, error) {
	sh, err := s.shipments.GetShipmentByTracking(ctx, trackingNumber)
	if err != nil {
		return shipment.Timeline{}, err
	}
	return s.GetTimeline(ctx, sh.ID)
}

func (s *Service) UpdateRecipient(ctx context.Context, id string, patch shipment.RecipientPatch) (shipment.Shipment, error) {
	if err := validatePatch(patch); err != nil {
		return shipment.Shipment{}, err
	}
	updated, err := s.shipments.UpdateRecipient(ctx, id, patch)
	if err != nil {
		return shipment.Shipment{}, err
	}
	_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
		ShipmentID: id,
		Type:       "recipient_updated",
		Notes:      "recipient details modified",
		RecordedBy: actorEmail(ctx),
		Payload:    patchToMap(patch),
	})
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.update_recipient",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       patchToMap(patch),
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return updated, nil
}

func (s *Service) UpdateSender(ctx context.Context, id string, patch shipment.RecipientPatch) (shipment.Shipment, error) {
	if err := validatePatch(patch); err != nil {
		return shipment.Shipment{}, err
	}
	updated, err := s.shipments.UpdateSender(ctx, id, patch)
	if err != nil {
		return shipment.Shipment{}, err
	}
	_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
		ShipmentID: id,
		Type:       "sender_updated",
		Notes:      "sender details modified",
		RecordedBy: actorEmail(ctx),
		Payload:    patchToMap(patch),
	})
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.update_sender",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       patchToMap(patch),
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return updated, nil
}

func (s *Service) RecordEvent(ctx context.Context, e shipment.ShipmentEvent) (shipment.ShipmentEvent, error) {
	if e.Type == "" {
		return shipment.ShipmentEvent{}, fmt.Errorf("event_type is required")
	}
	if e.RecordedBy == "" {
		e.RecordedBy = actorEmail(ctx)
	}
	rec, err := s.shipments.RecordEvent(ctx, e)
	if err != nil {
		return shipment.ShipmentEvent{}, err
	}
	s.audit.Log(ctx, audit.Entry{
		Action:     "shipments.add_event",
		EntityType: "shipment",
		EntityIDs:  []string{e.ShipmentID},
		Params: map[string]any{
			"event_type":    e.Type,
			"location_city": e.LocationCity,
			"location_hub":  e.LocationHub,
		},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return rec, nil
}

func (s *Service) Reschedule(ctx context.Context, id string, newETA time.Time, reason string) (shipment.Shipment, error) {
	updated, err := s.shipments.UpdateEstimatedDelivery(ctx, id, newETA)
	if err != nil {
		return shipment.Shipment{}, err
	}
	_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
		ShipmentID: id,
		Type:       "rescheduled",
		Notes:      reason,
		RecordedBy: actorEmail(ctx),
		Payload:    map[string]any{"new_eta": newETA.Format(time.RFC3339), "reason": reason},
	})
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.reschedule",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       map[string]any{"new_eta": newETA, "reason": reason},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return updated, nil
}

func (s *Service) recentlyUpdated(id string) shipment.Shipment {
	sh, err := s.shipments.GetShipmentByID(context.Background(), id)
	if err != nil {
		return shipment.Shipment{}
	}
	return sh
}

func (s *Service) Redirect(ctx context.Context, id string, newAddress shipment.Address, reason string) (shipment.Shipment, error) {
	if newAddress.City == "" || newAddress.Street == "" {
		return shipment.Shipment{}, fmt.Errorf("new address must include at least street and city")
	}
	current, err := s.shipments.GetShipmentByID(ctx, id)
	if err != nil {
		return shipment.Shipment{}, err
	}
	switch current.Status {
	case shipment.StatusDelivered, shipment.StatusReturnedToSender, shipment.StatusCancelled, shipment.StatusReturned:
		return shipment.Shipment{}, fmt.Errorf("cannot redirect a shipment in status %s", current.Status)
	}
	updated, err := s.shipments.RedirectAddress(ctx, id, newAddress, reason)
	if err != nil {
		return shipment.Shipment{}, err
	}
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.redirect",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       map[string]any{"new_city": newAddress.City, "new_street": newAddress.Street, "reason": reason},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return updated, nil
}

func (s *Service) HoldForPickup(ctx context.Context, id, officeAddress, reason string) (shipment.Shipment, error) {
	updated, err := s.shipments.UpdateShipmentStatus(ctx, id, shipment.StatusHeldAtOffice)
	if err != nil {
		return shipment.Shipment{}, err
	}
	_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
		ShipmentID: id,
		Type:       "held_at_office",
		LocationHub: officeAddress,
		Notes:      reason,
		RecordedBy: actorEmail(ctx),
	})
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.hold_for_pickup",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       map[string]any{"office": officeAddress, "reason": reason},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return updated, nil
}

func (s *Service) RecordAttempt(ctx context.Context, id, reason, notes string, nextAttemptAt *time.Time) (shipment.DeliveryAttempt, error) {
	current, err := s.shipments.GetShipmentByID(ctx, id)
	if err != nil {
		return shipment.DeliveryAttempt{}, err
	}
	attempt, err := s.shipments.RecordDeliveryAttempt(ctx, shipment.DeliveryAttempt{
		ShipmentID:    id,
		Reason:        reason,
		Notes:         notes,
		NextAttemptAt: nextAttemptAt,
	})
	if err != nil {
		return shipment.DeliveryAttempt{}, err
	}
	_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
		ShipmentID: id,
		Type:       "delivery_attempted",
		Notes:      reason,
		RecordedBy: actorEmail(ctx),
		Payload: map[string]any{
			"attempt_number":  attempt.AttemptNumber,
			"reason":          reason,
			"next_attempt_at": formatTimePtr(nextAttemptAt),
		},
	})
	if attempt.AttemptNumber >= 3 {
		_, _ = s.shipments.UpdateShipmentStatus(ctx, id, shipment.StatusReturnedToSender)
		_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
			ShipmentID: id,
			Type:       "returned_to_sender",
			Notes:      "automatically returned after 3 failed delivery attempts",
			RecordedBy: "system",
		})
	} else if current.Status != shipment.StatusDeliveryAttempted {
		_, _ = s.shipments.UpdateShipmentStatus(ctx, id, shipment.StatusDeliveryAttempted)
	}
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.record_attempt",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       map[string]any{"reason": reason, "attempt_number": attempt.AttemptNumber},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return attempt, nil
}

func (s *Service) RecordDelivery(ctx context.Context, id, signature, photoURL string) (shipment.Shipment, error) {
	updated, err := s.shipments.RecordDelivery(ctx, id, signature, photoURL)
	if err != nil {
		return shipment.Shipment{}, err
	}
	_, _ = s.shipments.RecordEvent(ctx, shipment.ShipmentEvent{
		ShipmentID: id,
		Type:       "delivered",
		Notes:      "delivery confirmed",
		RecordedBy: actorEmail(ctx),
		Payload:    map[string]any{"signature": signature, "photo_url": photoURL},
	})
	s.publishShipmentMilestone(updated, "delivered")
	s.audit.Log(ctx, audit.Entry{
		Action:       "shipments.record_delivery",
		EntityType:   "shipment",
		EntityIDs:    []string{id},
		Params:       map[string]any{"signature": signature, "has_photo": photoURL != ""},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})
	return updated, nil
}

func actorEmail(ctx context.Context) string {
	if e := middleware.GetUserEmail(ctx); e != "" {
		return e
	}
	return "system"
}

func validatePatch(p shipment.RecipientPatch) error {
	if p.Phone != nil && *p.Phone != "" {
		v := strings.TrimSpace(*p.Phone)
		if !strings.HasPrefix(v, "+") || len(v) < 6 {
			return fmt.Errorf("invalid phone format: expected E.164 (e.g. +380501112233), received %q", *p.Phone)
		}
	}
	if p.Email != nil && *p.Email != "" {
		v := strings.TrimSpace(*p.Email)
		if !strings.Contains(v, "@") || !strings.Contains(v, ".") {
			return fmt.Errorf("invalid email format: received %q", *p.Email)
		}
	}
	return nil
}

func patchToMap(p shipment.RecipientPatch) map[string]any {
	m := map[string]any{}
	if p.FullName != nil {
		m["full_name"] = *p.FullName
	}
	if p.Phone != nil {
		m["phone"] = *p.Phone
	}
	if p.Email != nil {
		m["email"] = *p.Email
	}
	if p.Company != nil {
		m["company"] = *p.Company
	}
	if p.Street != nil {
		m["street"] = *p.Street
	}
	if p.City != nil {
		m["city"] = *p.City
	}
	if p.Region != nil {
		m["region"] = *p.Region
	}
	if p.Postcode != nil {
		m["postcode"] = *p.Postcode
	}
	if p.Country != nil {
		m["country"] = *p.Country
	}
	if p.DeliveryNotes != nil {
		m["delivery_notes"] = *p.DeliveryNotes
	}
	return m
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func (s *Service) CreateCarrier(ctx context.Context, req CreateCarrierRequest) (carrier.Carrier, error) {
	c := carrier.Carrier{
		Name:      req.Name,
		Type:      req.Type,
		CostPerKm: req.CostPerKm,
	}

	created, err := s.carriers.CreateCarrier(ctx, c)
	if err != nil {
		return carrier.Carrier{}, fmt.Errorf("failed to create carrier: %w", err)
	}

	return created, nil
}

func (s *Service) GetCarrierByID(ctx context.Context, id string) (carrier.Carrier, error) {
	return s.carriers.GetCarrierByID(ctx, id)
}

func (s *Service) ListCarriers(ctx context.Context, filter carrier.Filter, sort pagination.Sort, page pagination.Page) ([]carrier.Carrier, int, error) {
	return s.carriers.ListCarriers(ctx, filter, sort, page)
}

func (s *Service) ReassignCarrierByCity(ctx context.Context, fromCarrierID, toCarrierID, city string, statuses []shipment.Status, dryRun bool) (shipment.ReassignResult, error) {
	result, err := s.shipments.ReassignCarrierByCity(ctx, fromCarrierID, toCarrierID, city, statuses, dryRun)
	if err != nil {
		s.audit.Log(ctx, audit.Entry{
			Action:       "shipments.reassign_carrier",
			EntityType:   "shipment",
			Params:       map[string]any{"from": fromCarrierID, "to": toCarrierID, "city": city, "dry_run": dryRun},
			ResultStatus: audit.StatusFailed,
			ErrorMessage: err.Error(),
		})
		return shipment.ReassignResult{}, err
	}
	if from, ferr := s.carriers.GetCarrierByID(ctx, fromCarrierID); ferr == nil {
		result.FromCarrierName = from.Name
	}
	if to, terr := s.carriers.GetCarrierByID(ctx, toCarrierID); terr == nil {
		result.ToCarrierName = to.Name
	}
	if !dryRun {
		s.audit.Log(ctx, audit.Entry{
			Action:     "shipments.reassign_carrier",
			EntityType: "shipment",
			EntityIDs:  result.ReassignedIDs,
			Params: map[string]any{
				"from_carrier_id":   fromCarrierID,
				"to_carrier_id":     toCarrierID,
				"from_carrier_name": result.FromCarrierName,
				"to_carrier_name":   result.ToCarrierName,
				"city":              city,
				"total":             result.Total,
			},
			ResultStatus: audit.StatusSuccess,
			SuccessCount: result.Total,
		})
	}
	return result, nil
}

func (s *Service) UpdateCarrier(ctx context.Context, id string, req UpdateCarrierRequest) (carrier.Carrier, error) {
	if _, err := s.carriers.GetCarrierByID(ctx, id); err != nil {
		return carrier.Carrier{}, err
	}

	c := carrier.Carrier{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		CostPerKm: req.CostPerKm,
		IsActive:  req.IsActive,
	}

	updated, err := s.carriers.UpdateCarrier(ctx, c)
	if err != nil {
		return carrier.Carrier{}, fmt.Errorf("failed to update carrier: %w", err)
	}

	s.audit.Log(ctx, audit.Entry{
		Action:       "carriers.update",
		EntityType:   "carrier",
		EntityIDs:    []string{id},
		Params:       map[string]any{"name": req.Name, "is_active": req.IsActive, "cost_per_km": req.CostPerKm},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})

	return updated, nil
}

func (s *Service) CalculateRoute(ctx context.Context, req CalculateRouteRequest) (route.Route, error) {
	c, err := s.carriers.GetCarrierByID(ctx, req.CarrierID)
	if err != nil {
		return route.Route{}, fmt.Errorf("failed to get carrier: %w", err)
	}

	distanceKm := mockDistance(req.Origin, req.Destination)
	durationH := mockDuration(distanceKm, c.Type)
	cost := distanceKm * c.CostPerKm

	return route.Route{
		ID:          uuid.NewString(),
		Origin:      req.Origin,
		Destination: req.Destination,
		DistanceKm:  math.Round(distanceKm*100) / 100,
		DurationH:   math.Round(durationH*100) / 100,
		Cost:        math.Round(cost*100) / 100,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

func (s *Service) GetPerformance(ctx context.Context) (PerformanceStats, error) {
	deliveredStatus := shipment.StatusDelivered
	filter := shipment.Filter{Status: &deliveredStatus}

	delivered, _, err := s.shipments.ListShipments(ctx, filter, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 10000, Offset: 0})
	if err != nil {
		return PerformanceStats{}, fmt.Errorf("failed to list delivered shipments: %w", err)
	}

	const onTimeThreshold = 72 * time.Hour

	var stats PerformanceStats
	stats.TotalDelivered = len(delivered)

	for _, sh := range delivered {
		if sh.UpdatedAt.Sub(sh.CreatedAt) <= onTimeThreshold {
			stats.OnTime++
		} else {
			stats.Late++
		}
	}

	if stats.TotalDelivered > 0 {
		stats.OnTimeRate = math.Round(float64(stats.OnTime)/float64(stats.TotalDelivered)*10000) / 100
	}

	return stats, nil
}

func (s *Service) BulkUpdateStatus(ctx context.Context, updates []BulkStatusItem) []BulkStatusResult {
	results := make([]BulkStatusResult, 0, len(updates))

	for _, item := range updates {
		updated, err := s.UpdateShipmentStatus(ctx, item.ShipmentID, item.Status)
		if err != nil {
			results = append(results, BulkStatusResult{
				ShipmentID: item.ShipmentID,
				Status:     "failed",
				Error:      err.Error(),
			})
		} else {
			results = append(results, BulkStatusResult{
				ShipmentID: item.ShipmentID,
				Status:     "success",
				Shipment:   &updated,
			})
		}
	}

	return results
}

func mockDistance(origin, destination string) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(origin + ":" + destination))
	return 50.0 + float64(h.Sum64()%1950)
}

func mockDuration(distanceKm float64, carrierType carrier.Type) float64 {
	var speedKmH float64
	switch carrierType {
	case carrier.TypeAir:
		speedKmH = 800
	case carrier.TypeSea:
		speedKmH = 30
	default:
		speedKmH = 80
	}
	return distanceKm / speedKmH
}

func (s *Service) publishShipmentCreated(sh shipment.Shipment) {
	if s.nc == nil {
		return
	}

	data := map[string]any{
		"shipment_id":  sh.ID,
		"order_id":     sh.OrderID,
		"warehouse_id": sh.WarehouseID,
		"carrier_id":   sh.CarrierID,
		"status":       string(sh.Status),
	}

	if err := s.nc.Publish("logistics.shipment_created", "logistics.shipment_created", data); err != nil {
		s.log.Error("Failed to publish logistics.shipment_created", zap.String("shipment_id", sh.ID), zap.Error(err))
	}
}

func (s *Service) publishShipmentStatusChanged(sh shipment.Shipment, previousStatus shipment.Status) {
	if s.nc == nil {
		return
	}

	data := map[string]any{
		"shipment_id":     sh.ID,
		"order_id":        sh.OrderID,
		"previous_status": string(previousStatus),
		"new_status":      string(sh.Status),
	}

	if err := s.nc.Publish("logistics.shipment_status_changed", "logistics.shipment_status_changed", data); err != nil {
		s.log.Error("Failed to publish logistics.shipment_status_changed", zap.String("shipment_id", sh.ID), zap.Error(err))
	}

	switch sh.Status {
	case shipment.StatusOutForDelivery:
		s.publishShipmentMilestone(sh, "out_for_delivery")
	case shipment.StatusDelivered:
		s.publishShipmentMilestone(sh, "delivered")
	case shipment.StatusDeliveryAttempted:
		s.publishShipmentMilestone(sh, "attempted")
	case shipment.StatusReturnedToSender, shipment.StatusReturned:
		s.publishShipmentMilestone(sh, "returned")
	case shipment.StatusRedirected:
		s.publishShipmentMilestone(sh, "redirected")
	}
}

func (s *Service) publishShipmentMilestone(sh shipment.Shipment, milestone string) {
	if s.nc == nil {
		return
	}
	data := map[string]any{
		"shipment_id":     sh.ID,
		"order_id":        sh.OrderID,
		"tracking_number": sh.TrackingNumber,
		"milestone":       milestone,
		"status":          string(sh.Status),
		"recipient_email": sh.Recipient.Email,
		"recipient_phone": sh.Recipient.Phone,
		"recipient_city":  sh.Recipient.City,
		"current_city":    sh.CurrentLocationCity,
		"current_hub":     sh.CurrentLocationHub,
	}
	subject := "logistics.shipment_" + milestone
	if err := s.nc.Publish(subject, subject, data); err != nil {
		s.log.Error("Failed to publish milestone",
			zap.String("subject", subject),
			zap.String("shipment_id", sh.ID),
			zap.Error(err))
	}
}
