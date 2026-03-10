package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/services/analytics-service/internal/analytics"
)

func setupAnalyticsController() (*AnalyticsController, *Service) {
	svc, _ := newTestService()
	log := zap.NewNop()
	ctrl := NewAnalyticsController(svc, log)
	return ctrl, svc
}

func TestAnalyticsController_GetSalesDaily_Success(t *testing.T) {
	ctrl, svc := setupAnalyticsController()

	_ = svc
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales?date_from=2026-04-01&date_to=2026-04-30", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesDaily(rec, req)

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

func TestAnalyticsController_GetSalesDaily_MissingDates(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesDaily(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GetSalesDaily_InvalidDateFormat(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales?date_from=not-a-date&date_to=2026-04-30", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesDaily(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GetInventorySnapshots_Success(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/inventory?date_from=2026-04-01&date_to=2026-04-30", nil)
	rec := httptest.NewRecorder()

	ctrl.GetInventorySnapshots(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAnalyticsController_GetInventorySnapshots_MissingDates(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/inventory?date_from=2026-04-01", nil)
	rec := httptest.NewRecorder()

	ctrl.GetInventorySnapshots(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GetLogisticsDaily_Success(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/logistics?date_from=2026-04-01&date_to=2026-04-30", nil)
	rec := httptest.NewRecorder()

	ctrl.GetLogisticsDaily(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAnalyticsController_GetLogisticsDaily_MissingDates(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/logistics", nil)
	rec := httptest.NewRecorder()

	ctrl.GetLogisticsDaily(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GetSalesDaily_EmptyResponse(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales?date_from=2026-04-01&date_to=2026-04-30", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesDaily(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(data) != 0 {
		t.Errorf("expected empty slice, got %d items", len(data))
	}
}

func TestAnalyticsController_ReturnsEmptyArrayForAllEndpoints(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	endpoints := []struct {
		name   string
		path   string
		method func(http.ResponseWriter, *http.Request)
	}{
		{"sales", "/api/v1/analytics/sales?date_from=2026-01-01&date_to=2026-01-31", ctrl.GetSalesDaily},
		{"inventory", "/api/v1/analytics/inventory?date_from=2026-01-01&date_to=2026-01-31", ctrl.GetInventorySnapshots},
		{"logistics", "/api/v1/analytics/logistics?date_from=2026-01-01&date_to=2026-01-31", ctrl.GetLogisticsDaily},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, ep.path, nil)
			rec := httptest.NewRecorder()
			ep.method(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", rec.Code)
			}

			var resp httpresponse.Response
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			data, ok := resp.Data.([]any)
			if !ok {
				t.Fatalf("expected data to be a slice, got %T", resp.Data)
			}
			if len(data) != 0 {
				t.Errorf("expected empty array, got %d items", len(data))
			}
		})
	}
}

func TestAnalyticsController_GetSalesDaily_WithData(t *testing.T) {
	svc, storage := newTestService()
	ctrl := NewAnalyticsController(svc, zap.NewNop())

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: parseDate("2026-04-10"), TotalOrders: 10, TotalRevenue: 500.00, AvgOrderSize: 50.00},
		{ID: "s2", Date: parseDate("2026-04-11"), TotalOrders: 15, TotalRevenue: 750.00, AvgOrderSize: 50.00},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales?date_from=2026-04-10&date_to=2026-04-11", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesDaily(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 items, got %d", len(data))
	}
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestAnalyticsController_GetSalesSummary_Success(t *testing.T) {
	svc, storage := newTestService()
	ctrl := NewAnalyticsController(svc, zap.NewNop())

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: parseDate("2026-04-10"), TotalOrders: 10, TotalRevenue: 500.00, AvgOrderSize: 50.00},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales/summary?date_from=2026-04-10&date_to=2026-04-10", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesSummary(rec, req)

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

func TestAnalyticsController_GetSalesSummary_MissingDates(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales/summary", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesSummary(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GetSalesTrends_Daily(t *testing.T) {
	svc, storage := newTestService()
	ctrl := NewAnalyticsController(svc, zap.NewNop())

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: parseDate("2026-04-10"), TotalOrders: 10, TotalRevenue: 500.00, AvgOrderSize: 50.00},
		{ID: "s2", Date: parseDate("2026-04-11"), TotalOrders: 15, TotalRevenue: 750.00, AvgOrderSize: 50.00},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales/trends?date_from=2026-04-10&date_to=2026-04-11&granularity=day", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesTrends(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 items, got %d", len(data))
	}
}

func TestAnalyticsController_GetSalesTrends_InvalidGranularity(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/sales/trends?date_from=2026-04-10&date_to=2026-04-11&granularity=month", nil)
	rec := httptest.NewRecorder()

	ctrl.GetSalesTrends(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GetInventorySummary_Success(t *testing.T) {
	svc, storage := newTestService()
	ctrl := NewAnalyticsController(svc, zap.NewNop())

	storage.inventorySnapshots = []analytics.InventorySnapshot{
		{ID: "inv1", Date: parseDate("2026-04-10"), TotalProducts: 100, TotalQuantity: 5000, TotalReserved: 200, TotalAvailable: 4800, LowStockCount: 5},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/inventory/summary?date_from=2026-04-10&date_to=2026-04-10", nil)
	rec := httptest.NewRecorder()

	ctrl.GetInventorySummary(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAnalyticsController_GetLogisticsPerformance_Success(t *testing.T) {
	svc, storage := newTestService()
	ctrl := NewAnalyticsController(svc, zap.NewNop())

	storage.logisticsDaily = []analytics.LogisticsDaily{
		{ID: "l1", Date: parseDate("2026-04-10"), TotalShipments: 20, DeliveredCount: 18, FailedCount: 2, AvgDeliveryH: 24.0, OnTimeRate: 90.0},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/logistics/performance?date_from=2026-04-10&date_to=2026-04-10", nil)
	rec := httptest.NewRecorder()

	ctrl.GetLogisticsPerformance(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAnalyticsController_GetAnomalies_Success(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/anomalies?date_from=2026-04-01&date_to=2026-04-30", nil)
	rec := httptest.NewRecorder()

	ctrl.GetAnomalies(rec, req)

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

func TestAnalyticsController_GetAnomalies_MissingDates(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/anomalies", nil)
	rec := httptest.NewRecorder()

	ctrl.GetAnomalies(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GetOptimizations_Success(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/optimization?date_from=2026-04-01&date_to=2026-04-30", nil)
	rec := httptest.NewRecorder()

	ctrl.GetOptimizations(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAnalyticsController_GenerateReport_Success(t *testing.T) {
	svc, _ := newTestService()
	ctrl := NewAnalyticsController(svc, zap.NewNop())

	body := `{"report_type":"sales","date_from":"2026-04-01","date_to":"2026-04-30"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctrl.GenerateReport(rec, req)

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

func TestAnalyticsController_GenerateReport_MissingType(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	body := `{"date_from":"2026-04-01","date_to":"2026-04-30"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctrl.GenerateReport(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GenerateReport_InvalidBody(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/report", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctrl.GenerateReport(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticsController_GenerateReport_UnsupportedType(t *testing.T) {
	ctrl, _ := setupAnalyticsController()

	body := `{"report_type":"invalid","date_from":"2026-04-01","date_to":"2026-04-30"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctrl.GenerateReport(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
