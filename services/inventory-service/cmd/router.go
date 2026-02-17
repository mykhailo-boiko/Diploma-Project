package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/controller"
)

func newRouter(ctrl *controller.InventoryController, nc *natspkg.Client) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	metrics := middleware.NewMetrics("inventory_service")
	handler := metrics.Middleware(middleware.UserContext(mux))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /health/nats", nc.HealthHandler())

	mux.HandleFunc("POST /api/v1/products", ctrl.CreateProduct)
	mux.HandleFunc("GET /api/v1/products", ctrl.ListProducts)
	mux.HandleFunc("GET /api/v1/products/{id}", ctrl.GetProduct)
	mux.HandleFunc("PUT /api/v1/products/{id}", ctrl.UpdateProduct)
	mux.HandleFunc("DELETE /api/v1/products/{id}", ctrl.DeleteProduct)

	mux.HandleFunc("POST /api/v1/warehouses", ctrl.CreateWarehouse)
	mux.HandleFunc("GET /api/v1/warehouses", ctrl.ListWarehouses)
	mux.HandleFunc("GET /api/v1/warehouses/{id}", ctrl.GetWarehouse)
	mux.HandleFunc("PUT /api/v1/warehouses/{id}", ctrl.UpdateWarehouse)

	mux.HandleFunc("GET /api/v1/stock", ctrl.ListStock)
	mux.HandleFunc("POST /api/v1/stock/reserve", ctrl.ReserveStock)
	mux.HandleFunc("POST /api/v1/stock/release", ctrl.ReleaseStock)
	mux.HandleFunc("POST /api/v1/stock/adjust", ctrl.AdjustStock)
	mux.HandleFunc("GET /api/v1/stock/movements", ctrl.ListMovements)
	mux.HandleFunc("GET /api/v1/stock/low", ctrl.ListLowStock)
	mux.HandleFunc("PUT /api/v1/stock/threshold", ctrl.UpdateMinThreshold)

	mux.HandleFunc("GET /api/v1/inventory/report", ctrl.GetInventoryReport)

	return handler
}
