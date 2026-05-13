package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/carrier"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/route"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/shipment"
)

type LogisticsService interface {
	CreateShipment(ctx context.Context, req CreateShipmentRequest) (shipment.Shipment, error)
	GetShipmentByID(ctx context.Context, id string) (shipment.Shipment, error)
	ListShipments(ctx context.Context, filter shipment.Filter, sort pagination.Sort, page pagination.Page) ([]shipment.Shipment, int, error)
	UpdateShipmentStatus(ctx context.Context, id string, newStatus shipment.Status) (shipment.Shipment, error)
	CreateCarrier(ctx context.Context, req CreateCarrierRequest) (carrier.Carrier, error)
	GetCarrierByID(ctx context.Context, id string) (carrier.Carrier, error)
	ListCarriers(ctx context.Context, filter carrier.Filter, sort pagination.Sort, page pagination.Page) ([]carrier.Carrier, int, error)
	UpdateCarrier(ctx context.Context, id string, req UpdateCarrierRequest) (carrier.Carrier, error)
	CalculateRoute(ctx context.Context, req CalculateRouteRequest) (route.Route, error)
	GetPerformance(ctx context.Context) (PerformanceStats, error)
	BulkUpdateStatus(ctx context.Context, updates []BulkStatusItem) []BulkStatusResult
	ReassignCarrierByCity(ctx context.Context, fromCarrierID, toCarrierID, city string, statuses []shipment.Status, dryRun bool) (shipment.ReassignResult, error)

	GetShipmentByTracking(ctx context.Context, trackingNumber string) (shipment.Shipment, error)
	GetTimeline(ctx context.Context, shipmentID string) (shipment.Timeline, error)
	GetTimelineByTracking(ctx context.Context, trackingNumber string) (shipment.Timeline, error)
	UpdateRecipient(ctx context.Context, id string, patch shipment.RecipientPatch) (shipment.Shipment, error)
	UpdateSender(ctx context.Context, id string, patch shipment.RecipientPatch) (shipment.Shipment, error)
	RecordEvent(ctx context.Context, e shipment.ShipmentEvent) (shipment.ShipmentEvent, error)
	Reschedule(ctx context.Context, id string, newETA time.Time, reason string) (shipment.Shipment, error)
	Redirect(ctx context.Context, id string, newAddress shipment.Address, reason string) (shipment.Shipment, error)
	HoldForPickup(ctx context.Context, id, officeAddress, reason string) (shipment.Shipment, error)
	RecordAttempt(ctx context.Context, id, reason, notes string, nextAttemptAt *time.Time) (shipment.DeliveryAttempt, error)
	RecordDelivery(ctx context.Context, id, signature, photoURL string) (shipment.Shipment, error)
	InTransitSummary(ctx context.Context) (shipment.InTransitSummaryResult, error)
}

type LogisticsController struct {
	svc LogisticsService
	log *zap.Logger
}

func NewLogisticsController(svc LogisticsService, log *zap.Logger) *LogisticsController {
	return &LogisticsController{svc: svc, log: log}
}

var shipmentAllowedSortFields = map[string]bool{
	"created_at": true,
	"status":     true,
	"order_id":   true,
}

var carrierAllowedSortFields = map[string]bool{
	"created_at": true,
	"name":       true,
	"type":       true,
	"cost_per_km": true,
}

func (c *LogisticsController) CreateShipment(w http.ResponseWriter, r *http.Request) {
	var req CreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.InvalidBody(w, err.Error())
		return
	}

	if req.OrderID == "" {
		httpresponse.MissingField(w, "order_id",
			"order UUID string",
			"Get an order ID via orders_list. The order should be in 'processing' or 'shipped' status.")
		return
	}
	if req.WarehouseID == "" {
		httpresponse.MissingField(w, "warehouse_id",
			"warehouse UUID string",
			"List available warehouses via warehouses_list and pick one with stock for the order items.")
		return
	}
	if req.CarrierID == "" {
		httpresponse.MissingField(w, "carrier_id",
			"carrier UUID string (must be active)",
			"List active carriers via carriers_list. Inactive carriers cannot be assigned.")
		return
	}
	if req.Address == "" {
		httpresponse.MissingField(w, "address",
			"delivery address as a free-form string",
			"Provide the recipient delivery address in 'Street N, City' format (Latin transliteration).",
			"Khreshchatyk str. 22, Kyiv", "Sahaidachnoho str. 5, Lviv")
		return
	}

	created, err := c.svc.CreateShipment(r.Context(), req)
	if err != nil {
		if errors.Is(err, carrier.ErrCarrierNotFound) {
			httpresponse.NotFoundError(w, httpresponse.LLMError{
				Code:       "carrier_not_found",
				Message:    "carrier with id '" + req.CarrierID + "' was not found or is inactive",
				Field:      "carrier_id",
				Received:   req.CarrierID,
				Suggestion: "Use carriers_list to get a valid active carrier ID.",
			})
			return
		}
		c.log.Error("Failed to create shipment", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *LogisticsController) GetShipmentByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "shipment id is required")
		return
	}

	sh, err := c.svc.GetShipmentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFound(w, "shipment_not_found", "shipment not found")
			return
		}
		c.log.Error("Failed to get shipment", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, sh)
}

func (c *LogisticsController) ListShipments(w http.ResponseWriter, r *http.Request) {
	filter := parseShipmentFilter(r)
	sort := pagination.SortFromRequest(r, shipmentAllowedSortFields, "created_at")
	page := pagination.PageFromRequest(r)

	shipments, total, err := c.svc.ListShipments(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list shipments", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if shipments == nil {
		shipments = []shipment.Shipment{}
	}

	httpresponse.List(w, shipments, total, page.Limit, page.Offset)
}

func (c *LogisticsController) UpdateShipmentStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.MissingField(w, "id (path)",
			"shipment UUID in URL path",
			"Specify shipment ID in path: PUT /api/v1/shipments/{id}/status")
		return
	}

	var req UpdateShipmentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.InvalidBody(w, err.Error())
		return
	}

	if req.Status == "" {
		httpresponse.MissingField(w, "status",
			"one of 15 valid shipment statuses",
			"Use shipments_get to see current status, then choose a valid next status.",
			"label_created", "picked_up", "in_transit", "out_for_delivery", "delivered", "cancelled")
		return
	}

	updated, err := c.svc.UpdateShipmentStatus(r.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFoundError(w, httpresponse.LLMError{
				Code:       "shipment_not_found",
				Message:    "shipment with id '" + id + "' was not found",
				Field:      "id",
				Received:   id,
				Suggestion: "Verify the shipment ID via shipments_list or shipments_tracking (tracking_number).",
			})
			return
		}
		if errors.Is(err, shipment.ErrInvalidTransition) {
			httpresponse.InvalidTransition(w, "shipment", "(see shipment's current status)", string(req.Status),
				[]string{"created→label_created", "label_created→awaiting_pickup→picked_up", "picked_up→in_transit", "in_transit↔at_hub", "in_transit→out_for_delivery", "out_for_delivery→delivered|delivery_attempted|held_at_office", "delivery_attempted×3 → returned_to_sender"})
			return
		}
		c.log.Error("Failed to update shipment status", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *LogisticsController) BulkUpdateShipmentStatus(w http.ResponseWriter, r *http.Request) {
	var req BulkStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if len(req.Updates) == 0 {
		httpresponse.BadRequest(w, "validation_error", "updates list is required and must not be empty")
		return
	}

	for i, item := range req.Updates {
		if item.ShipmentID == "" {
			httpresponse.BadRequest(w, "validation_error", fmt.Sprintf("updates[%d].shipment_id is required", i))
			return
		}
		if item.Status == "" {
			httpresponse.BadRequest(w, "validation_error", fmt.Sprintf("updates[%d].status is required", i))
			return
		}
	}

	results := c.svc.BulkUpdateStatus(r.Context(), req.Updates)

	hasFailures := false
	for _, res := range results {
		if res.Status == "failed" {
			hasFailures = true
			break
		}
	}

	status := http.StatusOK
	if hasFailures {
		status = http.StatusMultiStatus
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": results,
	})
}

func (c *LogisticsController) CreateCarrier(w http.ResponseWriter, r *http.Request) {
	var req CreateCarrierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Name == "" {
		httpresponse.BadRequest(w, "validation_error", "name is required")
		return
	}
	if !carrier.ValidType(req.Type) {
		httpresponse.BadRequest(w, "validation_error", "type must be one of: ground, air, sea")
		return
	}
	if req.CostPerKm <= 0 {
		httpresponse.BadRequest(w, "validation_error", "cost_per_km must be greater than 0")
		return
	}

	created, err := c.svc.CreateCarrier(r.Context(), req)
	if err != nil {
		c.log.Error("Failed to create carrier", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *LogisticsController) GetCarrierByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "carrier id is required")
		return
	}

	found, err := c.svc.GetCarrierByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, carrier.ErrCarrierNotFound) {
			httpresponse.NotFound(w, "carrier_not_found", "carrier not found")
			return
		}
		c.log.Error("Failed to get carrier", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, found)
}

func (c *LogisticsController) ListCarriers(w http.ResponseWriter, r *http.Request) {
	filter := parseCarrierFilter(r)
	sort := pagination.SortFromRequest(r, carrierAllowedSortFields, "created_at")
	page := pagination.PageFromRequest(r)

	carriers, total, err := c.svc.ListCarriers(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list carriers", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if carriers == nil {
		carriers = []carrier.Carrier{}
	}

	httpresponse.List(w, carriers, total, page.Limit, page.Offset)
}

func (c *LogisticsController) UpdateCarrier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "carrier id is required")
		return
	}

	var req UpdateCarrierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Name == "" {
		httpresponse.BadRequest(w, "validation_error", "name is required")
		return
	}
	if !carrier.ValidType(req.Type) {
		httpresponse.BadRequest(w, "validation_error", "type must be one of: ground, air, sea")
		return
	}
	if req.CostPerKm <= 0 {
		httpresponse.BadRequest(w, "validation_error", "cost_per_km must be greater than 0")
		return
	}

	updated, err := c.svc.UpdateCarrier(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, carrier.ErrCarrierNotFound) {
			httpresponse.NotFound(w, "carrier_not_found", "carrier not found")
			return
		}
		c.log.Error("Failed to update carrier", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *LogisticsController) ReassignCarrier(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromCarrierID string   `json:"from_carrier_id"`
		ToCarrierID   string   `json:"to_carrier_id"`
		City          string   `json:"city"`
		Statuses      []string `json:"statuses"`
		DryRun        bool     `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}
	if req.FromCarrierID == "" || req.ToCarrierID == "" {
		httpresponse.BadRequest(w, "validation_error", "from_carrier_id and to_carrier_id are required")
		return
	}
	if req.FromCarrierID == req.ToCarrierID {
		httpresponse.BadRequest(w, "validation_error", "from_carrier_id and to_carrier_id must differ")
		return
	}

	statuses := make([]shipment.Status, 0, len(req.Statuses))
	for _, s := range req.Statuses {
		statuses = append(statuses, shipment.Status(s))
	}

	result, err := c.svc.ReassignCarrierByCity(r.Context(), req.FromCarrierID, req.ToCarrierID, req.City, statuses, req.DryRun)
	if err != nil {
		c.log.Error("Failed to reassign carrier", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func (c *LogisticsController) CalculateRoute(w http.ResponseWriter, r *http.Request) {
	var req CalculateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Origin == "" {
		httpresponse.BadRequest(w, "validation_error", "origin is required")
		return
	}
	if req.Destination == "" {
		httpresponse.BadRequest(w, "validation_error", "destination is required")
		return
	}
	if req.CarrierID == "" {
		httpresponse.BadRequest(w, "validation_error", "carrier_id is required")
		return
	}

	result, err := c.svc.CalculateRoute(r.Context(), req)
	if err != nil {
		if errors.Is(err, carrier.ErrCarrierNotFound) {
			httpresponse.NotFound(w, "carrier_not_found", "carrier not found")
			return
		}
		c.log.Error("Failed to calculate route", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func (c *LogisticsController) GetPerformance(w http.ResponseWriter, r *http.Request) {
	stats, err := c.svc.GetPerformance(r.Context())
	if err != nil {
		c.log.Error("Failed to get performance stats", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, stats)
}

func (c *LogisticsController) TrackingByNumber(w http.ResponseWriter, r *http.Request) {
	trackingNumber := r.PathValue("tracking_number")
	if trackingNumber == "" {
		httpresponse.BadRequest(w, "validation_error", "tracking_number is required")
		return
	}
	tl, err := c.svc.GetTimelineByTracking(r.Context(), trackingNumber)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFound(w, "shipment_not_found", "shipment not found")
			return
		}
		c.log.Error("Failed to fetch tracking", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}
	httpresponse.OK(w, tl)
}

func (c *LogisticsController) PublicTracking(w http.ResponseWriter, r *http.Request) {
	trackingNumber := r.PathValue("tracking_number")
	if trackingNumber == "" {
		httpresponse.BadRequest(w, "validation_error", "tracking_number is required")
		return
	}
	last4 := r.URL.Query().Get("last4")
	if last4 == "" || len(last4) != 4 {
		httpresponse.BadRequest(w, "validation_error", "?last4=<recipient phone last 4 digits> is required")
		return
	}
	tl, err := c.svc.GetTimelineByTracking(r.Context(), trackingNumber)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFound(w, "shipment_not_found", "shipment not found")
			return
		}
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}
	phone := tl.Shipment.Recipient.Phone
	if len(phone) < 4 || phone[len(phone)-4:] != last4 {
		httpresponse.Forbidden(w, "verification_failed", "phone digits do not match")
		return
	}

	maskedName := tl.Shipment.Recipient.FullName
	if len(maskedName) > 2 {
		maskedName = maskedName[:1] + strings.Repeat("*", len(maskedName)-2) + maskedName[len(maskedName)-1:]
	}
	publicTl := struct {
		TrackingNumber      string                    `json:"tracking_number"`
		Status              shipment.Status           `json:"status"`
		CurrentLocationCity string                    `json:"current_location_city,omitempty"`
		CurrentLocationHub  string                    `json:"current_location_hub,omitempty"`
		EstimatedDeliveryAt *time.Time                `json:"estimated_delivery_at,omitempty"`
		DeliveredAt         *time.Time                `json:"delivered_at,omitempty"`
		DeliveryAttempts    int                       `json:"delivery_attempts"`
		RecipientName       string                    `json:"recipient_name"`
		RecipientCity       string                    `json:"recipient_city,omitempty"`
		Events              []shipment.ShipmentEvent  `json:"events"`
	}{
		TrackingNumber:      tl.Shipment.TrackingNumber,
		Status:              tl.Shipment.Status,
		CurrentLocationCity: tl.Shipment.CurrentLocationCity,
		CurrentLocationHub:  tl.Shipment.CurrentLocationHub,
		EstimatedDeliveryAt: tl.Shipment.EstimatedDeliveryAt,
		DeliveredAt:         tl.Shipment.DeliveredAt,
		DeliveryAttempts:    tl.Shipment.DeliveryAttempts,
		RecipientName:       maskedName,
		RecipientCity:       tl.Shipment.Recipient.City,
		Events:              tl.Events,
	}
	httpresponse.OK(w, publicTl)
}

func (c *LogisticsController) GetTimeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "shipment id is required")
		return
	}
	tl, err := c.svc.GetTimeline(r.Context(), id)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFound(w, "shipment_not_found", "shipment not found")
			return
		}
		c.log.Error("Failed to fetch timeline", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}
	httpresponse.OK(w, tl)
}

func (c *LogisticsController) UpdateRecipient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var patch shipment.RecipientPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	updated, err := c.svc.UpdateRecipient(r.Context(), id, patch)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFound(w, "shipment_not_found", "shipment not found")
			return
		}
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, updated)
}

func (c *LogisticsController) UpdateSender(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var patch shipment.RecipientPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	updated, err := c.svc.UpdateSender(r.Context(), id, patch)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFound(w, "shipment_not_found", "shipment not found")
			return
		}
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, updated)
}

func (c *LogisticsController) AddEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Type         string `json:"type"`
		LocationCity string `json:"location_city"`
		LocationHub  string `json:"location_hub"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	if req.Type == "" {
		httpresponse.BadRequest(w, "validation_error", "type is required")
		return
	}
	ev, err := c.svc.RecordEvent(r.Context(), shipment.ShipmentEvent{
		ShipmentID:   id,
		Type:         req.Type,
		LocationCity: req.LocationCity,
		LocationHub:  req.LocationHub,
		Notes:        req.Notes,
	})
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, ev)
}

func (c *LogisticsController) Reschedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		NewETA string `json:"new_eta"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	t, err := time.Parse(time.RFC3339, req.NewETA)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", "new_eta must be RFC3339")
		return
	}
	if t.Before(time.Now().UTC()) {
		httpresponse.InvalidField(w, "new_eta",
			"future RFC3339 datetime", req.NewETA,
			"new_eta must be in the future. Use a date after current UTC time.",
			"2026-06-01T10:00:00Z")
		return
	}
	updated, err := c.svc.Reschedule(r.Context(), id, t, req.Reason)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, updated)
}

func (c *LogisticsController) Redirect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		NewAddress shipment.Address `json:"new_address"`
		Reason     string           `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	updated, err := c.svc.Redirect(r.Context(), id, req.NewAddress, req.Reason)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, updated)
}

func (c *LogisticsController) HoldForPickup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		OfficeAddress string `json:"office_address"`
		Reason        string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	updated, err := c.svc.HoldForPickup(r.Context(), id, req.OfficeAddress, req.Reason)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, updated)
}

func (c *LogisticsController) RecordAttempt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Reason        string  `json:"reason"`
		Notes         string  `json:"notes"`
		NextAttemptAt *string `json:"next_attempt_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	if req.Reason == "" {
		httpresponse.BadRequest(w, "validation_error", "reason is required")
		return
	}
	var nextAt *time.Time
	if req.NextAttemptAt != nil && *req.NextAttemptAt != "" {
		t, err := time.Parse(time.RFC3339, *req.NextAttemptAt)
		if err != nil {
			httpresponse.BadRequest(w, "validation_error", "next_attempt_at must be RFC3339")
			return
		}
		nextAt = &t
	}
	attempt, err := c.svc.RecordAttempt(r.Context(), id, req.Reason, req.Notes, nextAt)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentTerminalState) {
			httpresponse.Err(w, http.StatusConflict, "terminal_state",
				err.Error())
			return
		}
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, attempt)
}

func (c *LogisticsController) RecordDelivery(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Signature     string `json:"signature"`
		SignatureName string `json:"signature_name"`
		PhotoURL      string `json:"photo_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	signature := req.Signature
	if signature == "" {
		signature = req.SignatureName
	}
	if signature == "" && req.PhotoURL == "" {
		httpresponse.MissingField(w, "signature_or_photo_url",
			"either 'signature' (recipient name) or 'photo_url' is required",
			"Provide at least one delivery proof: a printed signature name or a photo URL.",
			"John Doe", "https://cdn.example.com/proof.jpg")
		return
	}
	updated, err := c.svc.RecordDelivery(r.Context(), id, signature, req.PhotoURL)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentAlreadyDelivered) {
			httpresponse.Err(w, http.StatusConflict, "already_delivered", err.Error())
			return
		}
		if errors.Is(err, shipment.ErrShipmentTerminalState) {
			httpresponse.Err(w, http.StatusConflict, "terminal_state", err.Error())
			return
		}
		httpresponse.BadRequest(w, "validation_error", err.Error())
		return
	}
	httpresponse.OK(w, updated)
}

func (c *LogisticsController) InTransitSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := c.svc.InTransitSummary(r.Context())
	if err != nil {
		c.log.Error("Failed to get in-transit summary", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}
	httpresponse.OK(w, summary)
}

func parseShipmentFilter(r *http.Request) shipment.Filter {
	var filter shipment.Filter

	if s := r.URL.Query().Get("status"); s != "" {
		status := shipment.Status(s)
		filter.Status = &status
	}
	if s := r.URL.Query().Get("carrier_id"); s != "" {
		filter.CarrierID = &s
	}
	if s := r.URL.Query().Get("order_id"); s != "" {
		filter.OrderID = &s
	}
	if s := r.URL.Query().Get("warehouse_id"); s != "" {
		filter.WarehouseID = &s
	}
	if s := r.URL.Query().Get("date_from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.DateFrom = &t
		}
	}
	if s := r.URL.Query().Get("date_to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.DateTo = &t
		}
	}

	return filter
}

func parseCarrierFilter(r *http.Request) carrier.Filter {
	var filter carrier.Filter

	if s := r.URL.Query().Get("type"); s != "" {
		t := carrier.Type(s)
		filter.Type = &t
	}
	if s := r.URL.Query().Get("is_active"); s != "" {
		active := s == "true"
		filter.IsActive = &active
	}
	if s := r.URL.Query().Get("name"); s != "" {
		filter.Name = &s
	}

	return filter
}
