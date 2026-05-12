package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/stock"
)

func setupController() (*InventoryController, *Service, *mockStockStorage) {
	svc, _, _, ss := newTestService()
	log := zap.NewNop()
	ctrl := NewInventoryController(svc, log)
	return ctrl, svc, ss
}

func TestProductController_Create_Success(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(CreateProductRequest{
		SKU:       "TEST-001",
		Name:      "Test Product",
		Category:  "Test",
		UnitPrice: 19.99,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateProduct(rec, req)

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

func TestProductController_Create_MissingSKU(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(CreateProductRequest{Name: "Test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestProductController_Create_MissingName(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(CreateProductRequest{SKU: "SKU-001"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestProductController_Create_DuplicateSKU(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(CreateProductRequest{SKU: "DUP-001", Name: "First"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctrl.CreateProduct(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first create, got %d", rec.Code)
	}

	body, _ = json.Marshal(CreateProductRequest{SKU: "DUP-001", Name: "Second"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctrl.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate SKU, got %d", rec.Code)
	}
}

func TestProductController_GetByID_NotFound(t *testing.T) {
	ctrl, _, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.GetProduct(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProductController_GetByID_Success(t *testing.T) {
	ctrl, svc, _ := setupController()

	created, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "GET-001",
		Name: "Get Test",
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.GetProduct(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProductController_List_Success(t *testing.T) {
	ctrl, svc, _ := setupController()

	_, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "LIST-001",
		Name: "List Test",
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?limit=20&offset=0", nil)
	rec := httptest.NewRecorder()

	ctrl.ListProducts(rec, req)

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

func TestProductController_Update_Success(t *testing.T) {
	ctrl, svc, _ := setupController()

	created, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "UPD-001",
		Name: "Original",
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	body, _ := json.Marshal(UpdateProductRequest{Name: "Updated", UnitPrice: 99.99})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+created.ID, bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateProduct(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProductController_Update_NotFound(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(UpdateProductRequest{Name: "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/products/nonexistent", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.UpdateProduct(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProductController_Delete_Success(t *testing.T) {
	ctrl, svc, _ := setupController()

	created, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "DEL-001",
		Name: "To Delete",
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.DeleteProduct(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProductController_Delete_NotFound(t *testing.T) {
	ctrl, _, _ := setupController()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.DeleteProduct(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestWarehouseController_Create_Success(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(CreateWarehouseRequest{
		Name:    "Main Warehouse",
		Address: "123 Main St",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warehouses", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateWarehouse(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWarehouseController_Create_MissingName(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(CreateWarehouseRequest{Address: "123 Main St"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warehouses", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.CreateWarehouse(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWarehouseController_GetByID_NotFound(t *testing.T) {
	ctrl, _, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/warehouses/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.GetWarehouse(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestWarehouseController_List_Success(t *testing.T) {
	ctrl, svc, _ := setupController()

	_, err := svc.CreateWarehouse(t.Context(), CreateWarehouseRequest{Name: "Test"})
	if err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/warehouses?limit=20&offset=0", nil)
	rec := httptest.NewRecorder()

	ctrl.ListWarehouses(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWarehouseController_Update_Success(t *testing.T) {
	ctrl, svc, _ := setupController()

	created, err := svc.CreateWarehouse(t.Context(), CreateWarehouseRequest{Name: "Original"})
	if err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}

	body, _ := json.Marshal(UpdateWarehouseRequest{Name: "Updated", IsActive: true})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/warehouses/"+created.ID, bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateWarehouse(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWarehouseController_Update_NotFound(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(UpdateWarehouseRequest{Name: "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/warehouses/nonexistent", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.UpdateWarehouse(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestStockController_List_Empty(t *testing.T) {
	ctrl, _, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stock", nil)
	rec := httptest.NewRecorder()

	ctrl.ListStock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_List_WithFilters(t *testing.T) {
	ctrl, _, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stock?product_id=abc&warehouse_id=def", nil)
	rec := httptest.NewRecorder()

	ctrl.ListStock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestStockController_Reserve_Success(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	body, _ := json.Marshal(ReserveStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    30,
		Reference:   "order-123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/reserve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ReserveStock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_Reserve_MissingFields(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(ReserveStockRequest{Quantity: 10})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/reserve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ReserveStock(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStockController_Reserve_ZeroQuantity(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(ReserveStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    0,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/reserve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ReserveStock(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStockController_Reserve_InsufficientStock(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 10, 0, 0)

	body, _ := json.Marshal(ReserveStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    50,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/reserve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ReserveStock(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_Reserve_NotFound(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(ReserveStockRequest{
		ProductID:   "nonexistent",
		WarehouseID: "wh-1",
		Quantity:    10,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/reserve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ReserveStock(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestStockController_Release_Success(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 100, 30, 0)

	body, _ := json.Marshal(ReleaseStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    30,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/release", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ReleaseStock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_Release_NothingToRelease(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	body, _ := json.Marshal(ReleaseStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    10,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/release", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ReleaseStock(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_Adjust_Inbound(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	body, _ := json.Marshal(AdjustStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    50,
		Type:        stock.MovementTypeInbound,
		Reference:   "shipment-456",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/adjust", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.AdjustStock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_Adjust_InvalidType(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(AdjustStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    50,
		Type:        "invalid",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock/adjust", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.AdjustStock(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStockController_ListMovements_Empty(t *testing.T) {
	ctrl, _, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stock/movements", nil)
	rec := httptest.NewRecorder()

	ctrl.ListMovements(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_ListLowStock_Empty(t *testing.T) {
	ctrl, _, _ := setupController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stock/low", nil)
	rec := httptest.NewRecorder()

	ctrl.ListLowStock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_ListLowStock_WithData(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 10, 0, 200)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stock/low", nil)
	rec := httptest.NewRecorder()

	ctrl.ListLowStock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_GetInventoryReport(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 100, 20, 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/report", nil)
	rec := httptest.NewRecorder()

	ctrl.GetInventoryReport(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_UpdateMinThreshold_Success(t *testing.T) {
	ctrl, _, ss := setupController()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	body, _ := json.Marshal(UpdateMinThresholdRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Threshold:   50,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/stock/threshold", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.UpdateMinThreshold(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStockController_UpdateMinThreshold_MissingFields(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(UpdateMinThresholdRequest{Threshold: 50})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/stock/threshold", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.UpdateMinThreshold(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStockController_UpdateMinThreshold_Negative(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(UpdateMinThresholdRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Threshold:   -1,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/stock/threshold", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.UpdateMinThreshold(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStockController_UpdateMinThreshold_NotFound(t *testing.T) {
	ctrl, _, _ := setupController()

	body, _ := json.Marshal(UpdateMinThresholdRequest{
		ProductID:   "nonexistent",
		WarehouseID: "wh-1",
		Threshold:   50,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/stock/threshold", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.UpdateMinThreshold(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
