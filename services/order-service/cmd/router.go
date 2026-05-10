package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/order-service/internal/controller"
)

func newRouter(orderCtrl *controller.OrderController, nc *natspkg.Client) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	metrics := middleware.NewMetrics("order_service")
	handler := metrics.Middleware(middleware.UserContext(mux))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /health/nats", nc.HealthHandler())

	mux.HandleFunc("POST /api/v1/orders", orderCtrl.Create)
	mux.HandleFunc("GET /api/v1/orders", orderCtrl.List)
	mux.HandleFunc("GET /api/v1/orders/search", orderCtrl.Search)
	mux.HandleFunc("GET /api/v1/orders/stats", orderCtrl.Stats)
	mux.HandleFunc("GET /api/v1/orders/sales-by-product", orderCtrl.SalesByProduct)
	mux.HandleFunc("GET /api/v1/orders/customers", orderCtrl.CustomerSummary)
	mux.HandleFunc("GET /api/v1/orders/{id}", orderCtrl.GetByID)
	mux.HandleFunc("PUT /api/v1/orders/{id}/status", orderCtrl.UpdateStatus)
	mux.HandleFunc("POST /api/v1/orders/bulk-status", orderCtrl.BulkUpdateStatus)
	mux.HandleFunc("POST /api/v1/orders/{id}/cancel", orderCtrl.Cancel)

	return handler
}
