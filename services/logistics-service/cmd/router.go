package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/controller"
)

func newRouter(ctrl *controller.LogisticsController, nc *natspkg.Client) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	metrics := middleware.NewMetrics("logistics_service")
	handler := metrics.Middleware(middleware.UserContext(mux))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /health/nats", nc.HealthHandler())

	mux.HandleFunc("POST /api/v1/shipments/bulk-status", ctrl.BulkUpdateShipmentStatus)
	mux.HandleFunc("POST /api/v1/shipments", ctrl.CreateShipment)
	mux.HandleFunc("GET /api/v1/shipments", ctrl.ListShipments)
	mux.HandleFunc("GET /api/v1/shipments/{id}", ctrl.GetShipmentByID)
	mux.HandleFunc("PUT /api/v1/shipments/{id}/status", ctrl.UpdateShipmentStatus)

	mux.HandleFunc("POST /api/v1/carriers", ctrl.CreateCarrier)
	mux.HandleFunc("GET /api/v1/carriers", ctrl.ListCarriers)
	mux.HandleFunc("GET /api/v1/carriers/{id}", ctrl.GetCarrierByID)
	mux.HandleFunc("PUT /api/v1/carriers/{id}", ctrl.UpdateCarrier)

	mux.HandleFunc("POST /api/v1/routes/calculate", ctrl.CalculateRoute)

	mux.HandleFunc("GET /api/v1/logistics/performance", ctrl.GetPerformance)

	return handler
}
