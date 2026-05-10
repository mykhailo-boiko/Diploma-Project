package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/analytics-service/internal/analytics"
)

type mockStorage struct {
	mu                 sync.Mutex
	salesDaily         []analytics.SalesDaily
	inventorySnapshots []analytics.InventorySnapshot
	logisticsDaily     []analytics.LogisticsDaily
}

func newMockStorage() *mockStorage {
	return &mockStorage{}
}

func (m *mockStorage) UpsertSalesDaily(_ context.Context, record analytics.SalesDaily) (analytics.SalesDaily, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, r := range m.salesDaily {
		if r.Date.Equal(record.Date) {
			m.salesDaily[i] = record
			return record, nil
		}
	}
	m.salesDaily = append(m.salesDaily, record)
	return record, nil
}

func (m *mockStorage) UpsertInventorySnapshot(_ context.Context, record analytics.InventorySnapshot) (analytics.InventorySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, r := range m.inventorySnapshots {
		if r.Date.Equal(record.Date) {
			m.inventorySnapshots[i] = record
			return record, nil
		}
	}
	m.inventorySnapshots = append(m.inventorySnapshots, record)
	return record, nil
}

func (m *mockStorage) UpsertLogisticsDaily(_ context.Context, record analytics.LogisticsDaily) (analytics.LogisticsDaily, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, r := range m.logisticsDaily {
		if r.Date.Equal(record.Date) {
			m.logisticsDaily[i] = record
			return record, nil
		}
	}
	m.logisticsDaily = append(m.logisticsDaily, record)
	return record, nil
}

func (m *mockStorage) GetSalesDaily(_ context.Context, from, to time.Time) ([]analytics.SalesDaily, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var results []analytics.SalesDaily
	for _, r := range m.salesDaily {
		if !r.Date.Before(from) && !r.Date.After(to) {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *mockStorage) GetInventorySnapshots(_ context.Context, from, to time.Time) ([]analytics.InventorySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var results []analytics.InventorySnapshot
	for _, r := range m.inventorySnapshots {
		if !r.Date.Before(from) && !r.Date.After(to) {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *mockStorage) GetLogisticsDaily(_ context.Context, from, to time.Time) ([]analytics.LogisticsDaily, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var results []analytics.LogisticsDaily
	for _, r := range m.logisticsDaily {
		if !r.Date.Before(from) && !r.Date.After(to) {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *mockStorage) GetQuickCancellations(_ context.Context, _, _ time.Time, _ int) ([]analytics.QuickCancellation, error) {
	return []analytics.QuickCancellation{}, nil
}

func (m *mockStorage) GetRebalancingRecommendations(_ context.Context, _ analytics.RebalancingParams) ([]analytics.RebalancingRecommendation, error) {
	return []analytics.RebalancingRecommendation{}, nil
}

func (m *mockStorage) GetCustomerProfile360(_ context.Context, _ string, _ int, _ int) (analytics.CustomerProfile360, error) {
	return analytics.CustomerProfile360{}, nil
}

func (m *mockStorage) GetMetricValue(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}

func (m *mockStorage) GetCarrierPerformance(_ context.Context, _, _ time.Time, _ int, _ int) ([]analytics.CarrierPerformance, error) {
	return []analytics.CarrierPerformance{}, nil
}

func newTestService() (*Service, *mockStorage) {
	storage := newMockStorage()
	return NewService(storage, zap.NewNop()), storage
}

func TestGetSalesDaily(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 10, TotalRevenue: 500.00, AvgOrderSize: 50.00},
		{ID: "s2", Date: date2, TotalOrders: 15, TotalRevenue: 750.00, AvgOrderSize: 50.00},
		{ID: "s3", Date: date3, TotalOrders: 8, TotalRevenue: 320.00, AvgOrderSize: 40.00},
	}

	results, err := svc.GetSalesDaily(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("GetSalesDaily failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestGetSalesDaily_Empty(t *testing.T) {
	svc, _ := newTestService()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	results, err := svc.GetSalesDaily(t.Context(), from, to)
	if err != nil {
		t.Fatalf("GetSalesDaily failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGetInventorySnapshots(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	storage.inventorySnapshots = []analytics.InventorySnapshot{
		{ID: "inv1", Date: date1, TotalProducts: 100, TotalQuantity: 5000, TotalReserved: 200, TotalAvailable: 4800, LowStockCount: 5},
	}

	results, err := svc.GetInventorySnapshots(t.Context(), date1, date1)
	if err != nil {
		t.Fatalf("GetInventorySnapshots failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].TotalProducts != 100 {
		t.Errorf("expected 100 total_products, got %d", results[0].TotalProducts)
	}
}

func TestGetLogisticsDaily(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.logisticsDaily = []analytics.LogisticsDaily{
		{ID: "l1", Date: date1, TotalShipments: 20, DeliveredCount: 18, FailedCount: 2, AvgDeliveryH: 24.5, OnTimeRate: 90.0},
		{ID: "l2", Date: date2, TotalShipments: 25, DeliveredCount: 24, FailedCount: 1, AvgDeliveryH: 20.0, OnTimeRate: 96.0},
	}

	results, err := svc.GetLogisticsDaily(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("GetLogisticsDaily failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestGetLogisticsDaily_Empty(t *testing.T) {
	svc, _ := newTestService()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	results, err := svc.GetLogisticsDaily(t.Context(), from, to)
	if err != nil {
		t.Fatalf("GetLogisticsDaily failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGetSalesSummary(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 10, TotalRevenue: 500.00, AvgOrderSize: 50.00},
		{ID: "s2", Date: date2, TotalOrders: 15, TotalRevenue: 750.00, AvgOrderSize: 50.00},
	}

	summary, err := svc.GetSalesSummary(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("GetSalesSummary failed: %v", err)
	}

	if summary.OrderCount != 25 {
		t.Errorf("expected 25 orders, got %d", summary.OrderCount)
	}
	if summary.TotalRevenue != 1250.00 {
		t.Errorf("expected 1250.00 revenue, got %.2f", summary.TotalRevenue)
	}
	if summary.AvgOrderValue != 50.00 {
		t.Errorf("expected 50.00 avg, got %.2f", summary.AvgOrderValue)
	}
}

func TestGetSalesSummary_Empty(t *testing.T) {
	svc, _ := newTestService()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	summary, err := svc.GetSalesSummary(t.Context(), from, to)
	if err != nil {
		t.Fatalf("GetSalesSummary failed: %v", err)
	}

	if summary.OrderCount != 0 {
		t.Errorf("expected 0 orders, got %d", summary.OrderCount)
	}
	if summary.AvgOrderValue != 0 {
		t.Errorf("expected 0 avg, got %.2f", summary.AvgOrderValue)
	}
}

func TestGetSalesTrends_Daily(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 10, TotalRevenue: 500.00, AvgOrderSize: 50.00},
		{ID: "s2", Date: date2, TotalOrders: 15, TotalRevenue: 750.00, AvgOrderSize: 50.00},
	}

	trends, err := svc.GetSalesTrends(t.Context(), date1, date2, "day")
	if err != nil {
		t.Fatalf("GetSalesTrends failed: %v", err)
	}

	if len(trends) != 2 {
		t.Fatalf("expected 2 trends, got %d", len(trends))
	}
	if trends[0].Period != "2026-04-10" {
		t.Errorf("expected period 2026-04-10, got %s", trends[0].Period)
	}
}

func TestGetSalesTrends_Weekly(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 10, TotalRevenue: 500.00, AvgOrderSize: 50.00},
		{ID: "s2", Date: date2, TotalOrders: 15, TotalRevenue: 750.00, AvgOrderSize: 50.00},
		{ID: "s3", Date: date3, TotalOrders: 8, TotalRevenue: 320.00, AvgOrderSize: 40.00},
	}

	trends, err := svc.GetSalesTrends(t.Context(), date1, date3, "week")
	if err != nil {
		t.Fatalf("GetSalesTrends failed: %v", err)
	}

	if len(trends) != 2 {
		t.Fatalf("expected 2 weekly trends, got %d", len(trends))
	}
	if trends[0].TotalOrders != 25 {
		t.Errorf("expected 25 orders in first week, got %d", trends[0].TotalOrders)
	}
}

func TestGetInventorySummary(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.inventorySnapshots = []analytics.InventorySnapshot{
		{ID: "inv1", Date: date1, TotalProducts: 100, TotalQuantity: 5000, TotalReserved: 200, TotalAvailable: 4800, LowStockCount: 3},
		{ID: "inv2", Date: date2, TotalProducts: 110, TotalQuantity: 5500, TotalReserved: 300, TotalAvailable: 5200, LowStockCount: 5},
	}

	summary, err := svc.GetInventorySummary(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("GetInventorySummary failed: %v", err)
	}

	if summary.TotalProducts != 110 {
		t.Errorf("expected 110 products, got %d", summary.TotalProducts)
	}
	if summary.LowStockCount != 5 {
		t.Errorf("expected 5 low stock, got %d", summary.LowStockCount)
	}
	if summary.TurnoverRate == 0 {
		t.Error("expected non-zero turnover rate")
	}
}

func TestGetLogisticsPerformance(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.logisticsDaily = []analytics.LogisticsDaily{
		{ID: "l1", Date: date1, TotalShipments: 20, DeliveredCount: 18, FailedCount: 2, AvgDeliveryH: 24.0, OnTimeRate: 90.0},
		{ID: "l2", Date: date2, TotalShipments: 30, DeliveredCount: 28, FailedCount: 2, AvgDeliveryH: 20.0, OnTimeRate: 93.0},
	}

	perf, err := svc.GetLogisticsPerformance(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("GetLogisticsPerformance failed: %v", err)
	}

	if perf.TotalShipments != 50 {
		t.Errorf("expected 50 shipments, got %d", perf.TotalShipments)
	}
	if perf.DeliveredCount != 46 {
		t.Errorf("expected 46 delivered, got %d", perf.DeliveredCount)
	}
	if perf.FailedCount != 4 {
		t.Errorf("expected 4 failed, got %d", perf.FailedCount)
	}
}

func TestDetectAnomalies_NoAnomalies(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 10, TotalRevenue: 500.00},
		{ID: "s2", Date: date2, TotalOrders: 11, TotalRevenue: 520.00},
	}
	storage.logisticsDaily = []analytics.LogisticsDaily{
		{ID: "l1", Date: date1, TotalShipments: 20, DeliveredCount: 18, FailedCount: 1, OnTimeRate: 90.0},
	}
	storage.inventorySnapshots = []analytics.InventorySnapshot{
		{ID: "inv1", Date: date1, TotalProducts: 100, LowStockCount: 5},
	}

	anomalies, err := svc.DetectAnomalies(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	if anomalies == nil {
		t.Error("expected non-nil anomalies slice")
	}
}

func TestDetectAnomalies_ZeroOrders(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 10, TotalRevenue: 500.00},
		{ID: "s2", Date: date2, TotalOrders: 0, TotalRevenue: 0},
	}

	anomalies, err := svc.DetectAnomalies(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	found := false
	for _, a := range anomalies {
		if a.Type == "sales" && a.Metric == "total_orders" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected zero-order anomaly")
	}
}

func TestDetectAnomalies_HighFailureRate(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	storage.logisticsDaily = []analytics.LogisticsDaily{
		{ID: "l1", Date: date1, TotalShipments: 10, DeliveredCount: 5, FailedCount: 5, OnTimeRate: 50.0},
	}

	anomalies, err := svc.DetectAnomalies(t.Context(), date1, date1)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	found := false
	for _, a := range anomalies {
		if a.Type == "logistics" && a.Metric == "failure_rate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected high failure rate anomaly")
	}
}

func TestGetOptimizations_BelowReorderPoint(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 100, TotalRevenue: 5000.00},
		{ID: "s2", Date: date2, TotalOrders: 100, TotalRevenue: 5000.00},
	}
	storage.inventorySnapshots = []analytics.InventorySnapshot{
		{ID: "inv1", Date: date2, TotalProducts: 50, TotalQuantity: 10, TotalReserved: 5, TotalAvailable: 5, LowStockCount: 3},
	}

	opts, err := svc.GetOptimizations(t.Context(), date1, date2)
	if err != nil {
		t.Fatalf("GetOptimizations failed: %v", err)
	}

	if len(opts) == 0 {
		t.Fatal("expected at least one optimization")
	}

	foundReorder := false
	foundLow := false
	for _, o := range opts {
		if o.Type == "reorder" {
			foundReorder = true
		}
		if o.Type == "low_stock_alert" {
			foundLow = true
		}
	}
	if !foundReorder {
		t.Error("expected reorder optimization")
	}
	if !foundLow {
		t.Error("expected low_stock_alert optimization")
	}
}

func TestGetOptimizations_Empty(t *testing.T) {
	svc, _ := newTestService()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	opts, err := svc.GetOptimizations(t.Context(), from, to)
	if err != nil {
		t.Fatalf("GetOptimizations failed: %v", err)
	}

	if len(opts) != 0 {
		t.Errorf("expected empty optimizations, got %d", len(opts))
	}
}

func TestGenerateReport_Sales(t *testing.T) {
	svc, storage := newTestService()

	date1 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	storage.salesDaily = []analytics.SalesDaily{
		{ID: "s1", Date: date1, TotalOrders: 10, TotalRevenue: 500.00},
	}

	report, err := svc.GenerateReport(t.Context(), analytics.ReportRequest{
		ReportType: "sales",
		DateFrom:   "2026-04-10",
		DateTo:     "2026-04-10",
	})
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if report.ReportType != "sales" {
		t.Errorf("expected sales report type, got %s", report.ReportType)
	}
	if report.Data == nil {
		t.Error("expected non-nil data")
	}
}

func TestGenerateReport_Full(t *testing.T) {
	svc, _ := newTestService()

	report, err := svc.GenerateReport(t.Context(), analytics.ReportRequest{
		ReportType: "full",
		DateFrom:   "2026-04-01",
		DateTo:     "2026-04-30",
	})
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if report.ReportType != "full" {
		t.Errorf("expected full report type, got %s", report.ReportType)
	}

	data, ok := report.Data.(map[string]any)
	if !ok {
		t.Fatal("expected map data for full report")
	}
	if _, ok := data["sales"]; !ok {
		t.Error("expected sales key in full report")
	}
}

func TestGenerateReport_UnsupportedType(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.GenerateReport(t.Context(), analytics.ReportRequest{
		ReportType: "invalid",
		DateFrom:   "2026-04-01",
		DateTo:     "2026-04-30",
	})
	if err == nil {
		t.Error("expected error for unsupported report type")
	}
}

func TestGenerateReport_InvalidDate(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.GenerateReport(t.Context(), analytics.ReportRequest{
		ReportType: "sales",
		DateFrom:   "not-a-date",
		DateTo:     "2026-04-30",
	})
	if err == nil {
		t.Error("expected error for invalid date")
	}
}
