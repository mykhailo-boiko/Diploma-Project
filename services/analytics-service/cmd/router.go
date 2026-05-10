package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/analytics-service/internal/controller"
)

func newRouter(ctrl *controller.AnalyticsController, nc *natspkg.Client) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	metrics := middleware.NewMetrics("analytics_service")
	handler := metrics.Middleware(middleware.UserContext(mux))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /health/nats", nc.HealthHandler())

	mux.HandleFunc("GET /api/v1/analytics/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /api/v1/analytics/sales", ctrl.GetSalesDaily)
	mux.HandleFunc("GET /api/v1/analytics/sales/summary", ctrl.GetSalesSummary)
	mux.HandleFunc("GET /api/v1/analytics/sales/trends", ctrl.GetSalesTrends)
	mux.HandleFunc("GET /api/v1/analytics/inventory", ctrl.GetInventorySnapshots)
	mux.HandleFunc("GET /api/v1/analytics/inventory/summary", ctrl.GetInventorySummary)
	mux.HandleFunc("GET /api/v1/analytics/logistics", ctrl.GetLogisticsDaily)
	mux.HandleFunc("GET /api/v1/analytics/logistics/performance", ctrl.GetLogisticsPerformance)
	mux.HandleFunc("GET /api/v1/analytics/anomalies", ctrl.GetAnomalies)
	mux.HandleFunc("GET /api/v1/analytics/optimization", ctrl.GetOptimizations)
	mux.HandleFunc("GET /api/v1/analytics/quick-cancellations", ctrl.GetQuickCancellations)
	mux.HandleFunc("GET /api/v1/analytics/rebalancing", ctrl.GetRebalancing)
	mux.HandleFunc("GET /api/v1/analytics/carriers-performance", ctrl.GetCarrierPerformance)
	mux.HandleFunc("GET /api/v1/analytics/customers/profile-360", ctrl.GetCustomerProfile360)
	mux.HandleFunc("GET /api/v1/analytics/period-comparison", ctrl.GetPeriodComparison)
	mux.HandleFunc("GET /api/v1/analytics/audit-log", ctrl.QueryAuditLog)
	mux.HandleFunc("GET /api/v1/analytics/forecast", ctrl.GetForecast)
	mux.HandleFunc("POST /api/v1/analytics/report", ctrl.GenerateReport)

	return handler
}
