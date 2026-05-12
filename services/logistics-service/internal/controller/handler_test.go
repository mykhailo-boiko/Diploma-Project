package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/carrier"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/shipment"
)

func setupController() (*LogisticsController, *Service) {
	svc, _, _ := newTestService()
	log := zap.NewNop()
	ctrl := NewLogisticsController(svc, log)
	return ctrl, svc
}

func TestController_CreateCarrier_Success(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(CreateCarrierRequest{
		Name:      "Fast Logistics",
		Type:      carrier.TypeGround,
		CostPerKm: 2.50,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/carriers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateCarrier(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
}

func TestController_CreateCarrier_MissingName(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(CreateCarrierRequest{Type: carrier.TypeGround, CostPerKm: 2.50})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/carriers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateCarrier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestController_CreateCarrier_InvalidType(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(map[string]any{"name": "Test", "type": "bicycle", "cost_per_km": 1.0})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/carriers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateCarrier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestController_CreateCarrier_InvalidCost(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(CreateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 0})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/carriers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateCarrier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestController_GetCarrierByID_NotFound(t *testing.T) {
	ctrl, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/carriers/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.GetCarrierByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestController_ListCarriers_Success(t *testing.T) {
	ctrl, svc := setupController()

	_, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 1.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/carriers?limit=20&offset=0", nil)
	rec := httptest.NewRecorder()

	ctrl.ListCarriers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta == nil {
		t.Error("expected meta in response")
	}
}

func TestController_CreateShipment_Success(t *testing.T) {
	ctrl, svc := setupController()

	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Test Carrier", Type: carrier.TypeGround, CostPerKm: 2.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	body, _ := json.Marshal(CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   c.ID,
		Address:     "123 Main St",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipments", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateShipment(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_CreateShipment_MissingFields(t *testing.T) {
	ctrl, _ := setupController()

	tests := []struct {
		name string
		req  CreateShipmentRequest
	}{
		{name: "missing order_id", req: CreateShipmentRequest{WarehouseID: "w1", CarrierID: "c1", Address: "addr"}},
		{name: "missing warehouse_id", req: CreateShipmentRequest{OrderID: "o1", CarrierID: "c1", Address: "addr"}},
		{name: "missing carrier_id", req: CreateShipmentRequest{OrderID: "o1", WarehouseID: "w1", Address: "addr"}},
		{name: "missing address", req: CreateShipmentRequest{OrderID: "o1", WarehouseID: "w1", CarrierID: "c1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/shipments", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			ctrl.CreateShipment(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestController_CreateShipment_CarrierNotFound(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   "nonexistent",
		Address:     "123 Main St",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipments", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateShipment(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_GetShipmentByID_NotFound(t *testing.T) {
	ctrl, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipments/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.GetShipmentByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestController_ListShipments_Success(t *testing.T) {
	ctrl, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipments?limit=20&offset=0", nil)
	rec := httptest.NewRecorder()

	ctrl.ListShipments(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_UpdateShipmentStatus_Success(t *testing.T) {
	ctrl, svc := setupController()

	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 2.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	created, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   c.ID,
		Address:     "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	body, _ := json.Marshal(UpdateShipmentStatusRequest{Status: shipment.StatusPickedUp})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipments/"+created.ID+"/status", bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateShipmentStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_UpdateShipmentStatus_InvalidTransition(t *testing.T) {
	ctrl, svc := setupController()

	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 2.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	created, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID:     "order-1",
		WarehouseID: "warehouse-1",
		CarrierID:   c.ID,
		Address:     "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	body, _ := json.Marshal(UpdateShipmentStatusRequest{Status: shipment.StatusDelivered})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipments/"+created.ID+"/status", bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateShipmentStatus(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 (invalid transition), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_UpdateShipmentStatus_NotFound(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(UpdateShipmentStatusRequest{Status: shipment.StatusPickedUp})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipments/nonexistent/status", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.UpdateShipmentStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_UpdateCarrier_Success(t *testing.T) {
	ctrl, svc := setupController()

	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Old Name", Type: carrier.TypeGround, CostPerKm: 1.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	body, _ := json.Marshal(UpdateCarrierRequest{Name: "New Name", Type: carrier.TypeAir, CostPerKm: 5.0, IsActive: true})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/carriers/"+c.ID, bytes.NewReader(body))
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateCarrier(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_UpdateCarrier_NotFound(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(UpdateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 1.0, IsActive: true})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/carriers/nonexistent", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.UpdateCarrier(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_ListShipments_WithStatusFilter(t *testing.T) {
	ctrl, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipments?status=created", nil)
	rec := httptest.NewRecorder()

	ctrl.ListShipments(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestController_CalculateRoute_Success(t *testing.T) {
	ctrl, svc := setupController()

	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 2.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	body, _ := json.Marshal(CalculateRouteRequest{
		Origin:      "Kyiv",
		Destination: "Lviv",
		CarrierID:   c.ID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routes/calculate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CalculateRoute(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_CalculateRoute_MissingFields(t *testing.T) {
	ctrl, _ := setupController()

	tests := []struct {
		name string
		req  CalculateRouteRequest
	}{
		{name: "missing origin", req: CalculateRouteRequest{Destination: "Lviv", CarrierID: "c1"}},
		{name: "missing destination", req: CalculateRouteRequest{Origin: "Kyiv", CarrierID: "c1"}},
		{name: "missing carrier_id", req: CalculateRouteRequest{Origin: "Kyiv", Destination: "Lviv"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/routes/calculate", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			ctrl.CalculateRoute(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestController_CalculateRoute_CarrierNotFound(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(CalculateRouteRequest{
		Origin:      "Kyiv",
		Destination: "Lviv",
		CarrierID:   "nonexistent",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routes/calculate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CalculateRoute(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_GetPerformance_Success(t *testing.T) {
	ctrl, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logistics/performance", nil)
	rec := httptest.NewRecorder()

	ctrl.GetPerformance(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_BulkUpdateShipmentStatus_Success(t *testing.T) {
	ctrl, svc := setupController()

	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 2.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	sh1, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID: "o1", WarehouseID: "w1", CarrierID: c.ID, Address: "addr1",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}
	sh2, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID: "o2", WarehouseID: "w1", CarrierID: c.ID, Address: "addr2",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	body, _ := json.Marshal(BulkStatusRequest{
		Updates: []BulkStatusItem{
			{ShipmentID: sh1.ID, Status: shipment.StatusPickedUp},
			{ShipmentID: sh2.ID, Status: shipment.StatusPickedUp},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipments/bulk-status", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.BulkUpdateShipmentStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_BulkUpdateShipmentStatus_PartialFailure(t *testing.T) {
	ctrl, svc := setupController()

	c, err := svc.CreateCarrier(t.Context(), CreateCarrierRequest{Name: "Test", Type: carrier.TypeGround, CostPerKm: 2.0})
	if err != nil {
		t.Fatalf("CreateCarrier failed: %v", err)
	}

	sh1, err := svc.CreateShipment(t.Context(), CreateShipmentRequest{
		OrderID: "o1", WarehouseID: "w1", CarrierID: c.ID, Address: "addr1",
	})
	if err != nil {
		t.Fatalf("CreateShipment failed: %v", err)
	}

	body, _ := json.Marshal(BulkStatusRequest{
		Updates: []BulkStatusItem{
			{ShipmentID: sh1.ID, Status: shipment.StatusPickedUp},
			{ShipmentID: "nonexistent", Status: shipment.StatusPickedUp},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipments/bulk-status", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.BulkUpdateShipmentStatus(rec, req)

	if rec.Code != http.StatusMultiStatus {
		t.Errorf("expected 207, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestController_BulkUpdateShipmentStatus_EmptyUpdates(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(BulkStatusRequest{Updates: []BulkStatusItem{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipments/bulk-status", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.BulkUpdateShipmentStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestController_BulkUpdateShipmentStatus_MissingFields(t *testing.T) {
	ctrl, _ := setupController()

	body, _ := json.Marshal(BulkStatusRequest{
		Updates: []BulkStatusItem{
			{ShipmentID: "", Status: shipment.StatusPickedUp},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipments/bulk-status", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.BulkUpdateShipmentStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
