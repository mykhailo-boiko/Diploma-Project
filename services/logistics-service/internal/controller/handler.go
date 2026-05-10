package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	ReassignCarrierByCity(ctx context.Context, fromCarrierID, toCarrierID, city string, statuses []shipment.Status) (shipment.ReassignResult, error)
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
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.OrderID == "" {
		httpresponse.BadRequest(w, "validation_error", "order_id is required")
		return
	}
	if req.WarehouseID == "" {
		httpresponse.BadRequest(w, "validation_error", "warehouse_id is required")
		return
	}
	if req.CarrierID == "" {
		httpresponse.BadRequest(w, "validation_error", "carrier_id is required")
		return
	}
	if req.Address == "" {
		httpresponse.BadRequest(w, "validation_error", "address is required")
		return
	}

	created, err := c.svc.CreateShipment(r.Context(), req)
	if err != nil {
		if errors.Is(err, carrier.ErrCarrierNotFound) {
			httpresponse.NotFound(w, "carrier_not_found", "carrier not found")
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
		httpresponse.BadRequest(w, "validation_error", "shipment id is required")
		return
	}

	var req UpdateShipmentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Status == "" {
		httpresponse.BadRequest(w, "validation_error", "status is required")
		return
	}

	updated, err := c.svc.UpdateShipmentStatus(r.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, shipment.ErrShipmentNotFound) {
			httpresponse.NotFound(w, "shipment_not_found", "shipment not found")
			return
		}
		if errors.Is(err, shipment.ErrInvalidTransition) {
			httpresponse.BadRequest(w, "invalid_transition", "invalid status transition")
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

	result, err := c.svc.ReassignCarrierByCity(r.Context(), req.FromCarrierID, req.ToCarrierID, req.City, statuses)
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
