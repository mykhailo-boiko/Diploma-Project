package controller

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

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
	log       *zap.Logger
}

func NewService(shipments shipment.Storage, carriers carrier.Storage, nc *natspkg.Client, log *zap.Logger) *Service {
	return &Service{shipments: shipments, carriers: carriers, nc: nc, log: log}
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

	s.publishShipmentStatusChanged(updated, current.Status)

	return updated, nil
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
		return shipment.ReassignResult{}, err
	}
	if from, ferr := s.carriers.GetCarrierByID(ctx, fromCarrierID); ferr == nil {
		result.FromCarrierName = from.Name
	}
	if to, terr := s.carriers.GetCarrierByID(ctx, toCarrierID); terr == nil {
		result.ToCarrierName = to.Name
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
}
