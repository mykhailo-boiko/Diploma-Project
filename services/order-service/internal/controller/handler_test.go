package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/services/order-service/internal/order"
)

func setupOrderController() (*OrderController, *Service) {
	svc, _ := newTestService()
	log := zap.NewNop()
	ctrl := NewOrderController(svc, log)
	return ctrl, svc
}

func TestOrderController_Create_Success(t *testing.T) {
	ctrl, _ := setupOrderController()

	body, _ := json.Marshal(CreateOrderRequest{
		CustomerName: "John Doe",
		Items: []CreateItemInput{
			{ProductID: "prod-1", Name: "Widget", Quantity: 2, UnitPrice: 10.50},
			{ProductID: "prod-2", Name: "Gadget", Quantity: 1, UnitPrice: 25.00},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

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

func TestOrderController_Create_MissingCustomerName(t *testing.T) {
	ctrl, _ := setupOrderController()

	body, _ := json.Marshal(CreateOrderRequest{
		Items: []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 5.00}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestOrderController_Create_NoItems(t *testing.T) {
	ctrl, _ := setupOrderController()

	body, _ := json.Marshal(CreateOrderRequest{CustomerName: "John"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestOrderController_Create_InvalidItem(t *testing.T) {
	ctrl, _ := setupOrderController()

	body, _ := json.Marshal(CreateOrderRequest{
		CustomerName: "John",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "", Quantity: 0, UnitPrice: 0}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestOrderController_GetByID_NotFound(t *testing.T) {
	ctrl, _ := setupOrderController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestOrderController_GetByID_Success(t *testing.T) {
	ctrl, svc := setupOrderController()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test User",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_List_Success(t *testing.T) {
	ctrl, svc := setupOrderController()

	_, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 5.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?limit=20&offset=0", nil)
	rec := httptest.NewRecorder()

	ctrl.List(rec, req)

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

func TestOrderController_UpdateStatus_Success(t *testing.T) {
	ctrl, svc := setupOrderController()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	body, _ := json.Marshal(UpdateStatusRequest{Status: order.StatusConfirmed})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+created.ID+"/status", bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_UpdateStatus_InvalidTransition(t *testing.T) {
	ctrl, svc := setupOrderController()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	body, _ := json.Marshal(UpdateStatusRequest{Status: order.StatusDelivered})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+created.ID+"/status", bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateStatus(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 (invalid transition), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_UpdateStatus_NotFound(t *testing.T) {
	ctrl, _ := setupOrderController()

	body, _ := json.Marshal(UpdateStatusRequest{Status: order.StatusConfirmed})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/nonexistent/status", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.UpdateStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_List_WithStatusFilter(t *testing.T) {
	ctrl, _ := setupOrderController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?status=pending", nil)
	rec := httptest.NewRecorder()

	ctrl.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestOrderController_Cancel_Success(t *testing.T) {
	ctrl, svc := setupOrderController()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test Cancel",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	body, _ := json.Marshal(CancelOrderRequest{Reason: "customer changed mind"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+created.ID+"/cancel", bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.Cancel(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_Cancel_MissingReason(t *testing.T) {
	ctrl, svc := setupOrderController()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	body, _ := json.Marshal(CancelOrderRequest{Reason: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+created.ID+"/cancel", bytes.NewReader(body))
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()

	ctrl.Cancel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_Cancel_NotFound(t *testing.T) {
	ctrl, _ := setupOrderController()

	body, _ := json.Marshal(CancelOrderRequest{Reason: "reason"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/nonexistent/cancel", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	ctrl.Cancel(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_Search_Success(t *testing.T) {
	ctrl, svc := setupOrderController()

	_, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "John Smith",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/search?q=John&limit=20", nil)
	rec := httptest.NewRecorder()

	ctrl.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_Search_TooShort(t *testing.T) {
	ctrl, _ := setupOrderController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/search?q=J", nil)
	rec := httptest.NewRecorder()

	ctrl.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderController_Stats_Success(t *testing.T) {
	ctrl, svc := setupOrderController()

	_, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/stats", nil)
	rec := httptest.NewRecorder()

	ctrl.Stats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
}
