package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/ws"
)

func newRouter(notifCtrl *controller.NotificationController, nc *natspkg.Client, wsHub *ws.Hub) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	metrics := middleware.NewMetrics("notification_service")
	handler := metrics.Middleware(middleware.UserContext(mux))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /health/nats", nc.HealthHandler())

	mux.HandleFunc("POST /api/v1/notifications", notifCtrl.Create)
	mux.HandleFunc("GET /api/v1/notifications", notifCtrl.List)
	mux.HandleFunc("GET /api/v1/notifications/unread-count", notifCtrl.UnreadCount)
	mux.HandleFunc("GET /api/v1/notifications/preferences", notifCtrl.GetPreferences)
	mux.HandleFunc("PUT /api/v1/notifications/preferences", notifCtrl.UpdatePreference)
	mux.HandleFunc("POST /api/v1/notifications/bulk", notifCtrl.Bulk)
	mux.HandleFunc("PUT /api/v1/notifications/{id}/read", notifCtrl.MarkAsRead)

	mux.HandleFunc("/ws/notifications", wsHub.HandleWebSocket)

	return handler
}
