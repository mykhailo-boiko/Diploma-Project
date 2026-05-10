package controller

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/carrier"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/shipment"
)

type mockShipmentStorage struct {
	mu        sync.Mutex
	shipments map[string]shipment.Shipment
	nextID    int
}

func newMockShipmentStorage() *mockShipmentStorage {
	return &mockShipmentStorage{shipments: make(map[string]shipment.Shipment)}
}

func (m *mockShipmentStorage) CreateShipment(_ context.Context, s shipment.Shipment) (shipment.Shipment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	s.ID = fmt.Sprintf("shipment-%d", m.nextID)
	s.Status = shipment.StatusCreated
	s.CreatedAt = time.Now().UTC()
	s.UpdatedAt = s.CreatedAt
	m.shipments[s.ID] = s
	return s, nil
}

func (m *mockShipmentStorage) GetShipmentByID(_ context.Context, id string) (shipment.Shipment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.shipments[id]
	if !ok {
		return shipment.Shipment{}, shipment.ErrShipmentNotFound
	}
	return s, nil
}

func (m *mockShipmentStorage) ListShipments(_ context.Context, filter shipment.Filter, _ pagination.Sort, _ pagination.Page) ([]shipment.Shipment, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []shipment.Shipment
	for _, s := range m.shipments {
		if filter.Status != nil && s.Status != *filter.Status {
			continue
		}
		result = append(result, s)
	}
	return result, len(result), nil
}

func (m *mockShipmentStorage) UpdateShipmentStatus(_ context.Context, id string, status shipment.Status) (shipment.Shipment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.shipments[id]
	if !ok {
		return shipment.Shipment{}, shipment.ErrShipmentNotFound
	}
	s.Status = status
	s.UpdatedAt = time.Now().UTC()
	m.shipments[id] = s
	return s, nil
}

func (m *mockShipmentStorage) ReassignCarrierByCity(_ context.Context, _, _, _ string, _ []shipment.Status) (shipment.ReassignResult, error) {
	return shipment.ReassignResult{}, nil
}

type mockCarrierStorage struct {
	mu       sync.Mutex
	carriers map[string]carrier.Carrier
	nextID   int
}

func newMockCarrierStorage() *mockCarrierStorage {
	return &mockCarrierStorage{carriers: make(map[string]carrier.Carrier)}
}

func (m *mockCarrierStorage) CreateCarrier(_ context.Context, c carrier.Carrier) (carrier.Carrier, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	c.ID = fmt.Sprintf("carrier-%d", m.nextID)
	c.IsActive = true
	m.carriers[c.ID] = c
	return c, nil
}

func (m *mockCarrierStorage) GetCarrierByID(_ context.Context, id string) (carrier.Carrier, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.carriers[id]
	if !ok {
		return carrier.Carrier{}, carrier.ErrCarrierNotFound
	}
	return c, nil
}

func (m *mockCarrierStorage) ListCarriers(_ context.Context, _ carrier.Filter, _ pagination.Sort, _ pagination.Page) ([]carrier.Carrier, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []carrier.Carrier
	for _, c := range m.carriers {
		result = append(result, c)
	}
	return result, len(result), nil
}

func (m *mockCarrierStorage) UpdateCarrier(_ context.Context, c carrier.Carrier) (carrier.Carrier, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.carriers[c.ID]; !ok {
		return carrier.Carrier{}, carrier.ErrCarrierNotFound
	}
	m.carriers[c.ID] = c
	return c, nil
}

func newTestService() (*Service, *mockShipmentStorage, *mockCarrierStorage) {
	ss := newMockShipmentStorage()
	cs := newMockCarrierStorage()
	return NewService(ss, cs, nil, zap.NewNop()), ss, cs
}

func createTestCarrier(t *testing.T, svc *Service) carrier.Carrier {
	t.Helper()
	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{
		Name:      "Test Carrier",
		Type:      carrier.TypeGround,
		CostPerKm: 2.50,
	})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}
	return c
}

func createTestShipment(t *testing.T, svc *Service, carrierID string) shipment.Shipment {
	t.Helper()
	sh, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   carrierID,
		Address:     "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}
	return sh
}

func TestCreateShipment(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	created, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   c.ID,
		Address:     "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	if created.Status != shipment.StatusCreated {
		t.Errorf("expected status created, got %s", created.Status)
	}
	if created.OrderID != "order-1" {
		t.Errorf("expected order_id 'order-1', got %q", created.OrderID)
	}
}

func TestCreateShipment_CarrierNotFound(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   "nonexistent",
		Address:     "123 Main St",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent carrier")
	}
	if !errors.Is(err, carrier.ErrCarrierNotFound) {
		t.Errorf("expected ErrCarrierNotFound, got %v", err)
	}
}

func TestGetShipmentByID(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	created, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   c.ID,
		Address:     "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	fetched, err := svc.GetShipmentByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetShipmentByID failed: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, fetched.ID)
	}
}

func TestGetShipmentByID_NotFound(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.GetShipmentByID(t.Context(), "nonexistent")
	if !errors.Is(err, shipment.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got %v", err)
	}
}

func TestUpdateShipmentStatus_ValidTransition(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	created, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   c.ID,
		Address:     "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	updated, err := svc.UpdateShipmentStatus(t.Context(), created.ID, shipment.StatusPickedUp)
	if err != nil {
		t.Fatalf("UpdateShipmentStatus failed: %v", err)
	}
	if updated.Status != shipment.StatusPickedUp {
		t.Errorf("expected status picked_up, got %s", updated.Status)
	}
}

func TestUpdateShipmentStatus_InvalidTransition(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	created, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   c.ID,
		Address:     "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	_, err = svc.UpdateShipmentStatus(t.Context(), created.ID, shipment.StatusDelivered)
	if !errors.Is(err, shipment.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestListShipments(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	for i := 0; i < 3; i++ {
		_, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
			OrderID:     fmt.Sprintf("order-%d", i),
			WarehouseID: "warehouse-1",
			CarrierID:   c.ID,
			Address:     fmt.Sprintf("Address %d", i),
		})
		if err != nil {
			t.Fatalf("CreateShipment failed: %v", err)
		}
	}

	shipments, total, err := svc.ListShipments(t.Context(), shipment.Filter{}, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListShipments failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(shipments) != 3 {
		t.Errorf("expected 3 shipments, got %d", len(shipments))
	}
}

func TestCreateCarrier(t *testing.T) {
	svc, _, _ := newTestService()

	created, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{
		Name:      "Express Delivery",
		Type:      carrier.TypeAir,
		CostPerKm: 5.00,
	})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}
	if created.Name != "Express Delivery" {
		t.Errorf("expected name 'Express Delivery', got %q", created.Name)
	}
	if created.Type != carrier.TypeAir {
		t.Errorf("expected type air, got %s", created.Type)
	}
	if !created.IsActive {
		t.Error("expected carrier to be active")
	}
}

func TestGetCarrierByID_NotFound(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.GetCarrierByID(t.Context(), "nonexistent")
	if !errors.Is(err, carrier.ErrCarrierNotFound) {
		t.Errorf("expected ErrCarrierNotFound, got %v", err)
	}
}

func TestUpdateCarrier(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	updated, err := svc.UpdateCarrier(t.Context(), c.ID, UpdateCarrierRequest{
		Name:      "Updated Carrier",
		Type:      carrier.TypeSea,
		CostPerKm: 1.00,
		IsActive:  false,
	})
	if err != nil {
		t.Fatalf("UpdateCarrier failed: %v", err)
	}
	if updated.Name != "Updated Carrier" {
		t.Errorf("expected name 'Updated Carrier', got %q", updated.Name)
	}
	if updated.Type != carrier.TypeSea {
		t.Errorf("expected type sea, got %s", updated.Type)
	}
}

func TestUpdateCarrier_NotFound(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.UpdateCarrier(t.Context(), "nonexistent", UpdateCarrierRequest{
		Name:      "Test",
		Type:      carrier.TypeGround,
		CostPerKm: 1.00,
		IsActive:  true,
	})
	if !errors.Is(err, carrier.ErrCarrierNotFound) {
		t.Errorf("expected ErrCarrierNotFound, got %v", err)
	}
}

func TestListCarriers(t *testing.T) {
	svc, _, _ := newTestService()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{
			Name:      fmt.Sprintf("Carrier %d", i),
			Type:      carrier.TypeGround,
			CostPerKm: float64(i+1) * 1.50,
		})
		if err != nil {
			t.Fatalf("CreateCarrier failed: %v", err)
		}
	}

	carriers, total, err := svc.ListCarriers(t.Context(), carrier.Filter{}, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListCarriers failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(carriers) != 3 {
		t.Errorf("expected 3 carriers, got %d", len(carriers))
	}
}

func TestCalculateRoute(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	result, err := svc.CalculateRoute(t.Context(), CalculateRouteRequest{
		Origin:      "Kyiv",
		Destination: "Lviv",
		CarrierID:   c.ID,
	})
	if err != nil {
		t.Fatalf("CalculateRoute failed: %v", err)
	}

	if result.Origin != "Kyiv" {
		t.Errorf("expected origin Kyiv, got %q", result.Origin)
	}
	if result.Destination != "Lviv" {
		t.Errorf("expected destination Lviv, got %q", result.Destination)
	}
	if result.DistanceKm <= 0 {
		t.Error("expected positive distance")
	}
	if result.DurationH <= 0 {
		t.Error("expected positive duration")
	}
	if result.Cost <= 0 {
		t.Error("expected positive cost")
	}

	result2, err := svc.CalculateRoute(t.Context(), CalculateRouteRequest{
		Origin:      "Kyiv",
		Destination: "Lviv",
		CarrierID:   c.ID,
	})
	if err != nil {
		t.Fatalf("CalculateRoute (2nd call) failed: %v", err)
	}
	if result.DistanceKm != result2.DistanceKm {
		t.Errorf("expected deterministic distance, got %f and %f", result.DistanceKm, result2.DistanceKm)
	}
	if result.Cost != result2.Cost {
		t.Errorf("expected deterministic cost, got %f and %f", result.Cost, result2.Cost)
	}
}

func TestCalculateRoute_CarrierNotFound(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.CalculateRoute(t.Context(), CalculateRouteRequest{
		Origin:      "Kyiv",
		Destination: "Lviv",
		CarrierID:   "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent carrier")
	}
	if !errors.Is(err, carrier.ErrCarrierNotFound) {
		t.Errorf("expected ErrCarrierNotFound, got %v", err)
	}
}

func TestCalculateRoute_CostFormula(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	result, err := svc.CalculateRoute(t.Context(), CalculateRouteRequest{
		Origin:      "A",
		Destination: "B",
		CarrierID:   c.ID,
	})
	if err != nil {
		t.Fatalf("CalculateRoute failed: %v", err)
	}

	expectedCost := math.Round(result.DistanceKm*c.CostPerKm*100) / 100
	if result.Cost != expectedCost {
		t.Errorf("expected cost %f (distance %f * cost_per_km %f), got %f", expectedCost, result.DistanceKm, c.CostPerKm, result.Cost)
	}
}

func TestGetPerformance_Empty(t *testing.T) {
	svc, _, _ := newTestService()

	stats, err := svc.GetPerformance(t.Context())
	if err != nil {
		t.Fatalf("GetPerformance failed: %v", err)
	}
	if stats.TotalDelivered != 0 {
		t.Errorf("expected 0 delivered, got %d", stats.TotalDelivered)
	}
	if stats.OnTime != 0 {
		t.Errorf("expected 0 on_time, got %d", stats.OnTime)
	}
	if stats.Late != 0 {
		t.Errorf("expected 0 late, got %d", stats.Late)
	}
}

func TestGetPerformance_WithDeliveries(t *testing.T) {
	svc, ss, _ := newTestService()
	c := createTestCarrier(t, svc)

	for i := 0; i < 3; i++ {
		sh := createTestShipment(t, svc, c.ID)
		now := time.Now().UTC()
		ss.mu.Lock()
		s := ss.shipments[sh.ID]
		s.Status = shipment.StatusDelivered
		s.CreatedAt = now.Add(-24 * time.Hour)
		s.UpdatedAt = now
		ss.shipments[sh.ID] = s
		ss.mu.Unlock()
	}

	for i := 0; i < 2; i++ {
		sh := createTestShipment(t, svc, c.ID)
		now := time.Now().UTC()
		ss.mu.Lock()
		s := ss.shipments[sh.ID]
		s.Status = shipment.StatusDelivered
		s.CreatedAt = now.Add(-200 * time.Hour)
		s.UpdatedAt = now
		ss.shipments[sh.ID] = s
		ss.mu.Unlock()
	}

	stats, err := svc.GetPerformance(t.Context())
	if err != nil {
		t.Fatalf("GetPerformance failed: %v", err)
	}
	if stats.TotalDelivered != 5 {
		t.Errorf("expected 5 delivered, got %d", stats.TotalDelivered)
	}
	if stats.OnTime != 3 {
		t.Errorf("expected 3 on_time, got %d", stats.OnTime)
	}
	if stats.Late != 2 {
		t.Errorf("expected 2 late, got %d", stats.Late)
	}
	if stats.OnTimeRate != 60.0 {
		t.Errorf("expected on_time_rate 60.0, got %f", stats.OnTimeRate)
	}
}

func TestBulkUpdateStatus_AllSuccess(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	sh1 := createTestShipment(t, svc, c.ID)
	sh2 := createTestShipment(t, svc, c.ID)

	results := svc.BulkUpdateStatus(t.Context(), []BulkStatusItem{
		{ShipmentID: sh1.ID, Status: shipment.StatusPickedUp},
		{ShipmentID: sh2.ID, Status: shipment.StatusPickedUp},
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != "success" {
			t.Errorf("expected success for %s, got %s: %s", r.ShipmentID, r.Status, r.Error)
		}
		if r.Shipment == nil {
			t.Errorf("expected shipment in result for %s", r.ShipmentID)
		}
	}
}

func TestBulkUpdateStatus_PartialFailure(t *testing.T) {
	svc, _, _ := newTestService()
	c := createTestCarrier(t, svc)

	sh1 := createTestShipment(t, svc, c.ID)

	results := svc.BulkUpdateStatus(t.Context(), []BulkStatusItem{
		{ShipmentID: sh1.ID, Status: shipment.StatusPickedUp},
		{ShipmentID: "nonexistent", Status: shipment.StatusPickedUp},
		{ShipmentID: sh1.ID, Status: shipment.StatusDelivered},
	})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Status != "success" {
		t.Errorf("expected success for first update, got %s: %s", results[0].Status, results[0].Error)
	}
	if results[1].Status != "failed" {
		t.Errorf("expected failed for nonexistent, got %s", results[1].Status)
	}
	if results[2].Status != "failed" {
		t.Errorf("expected failed for invalid transition, got %s", results[2].Status)
	}
}
