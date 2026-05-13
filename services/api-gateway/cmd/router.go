package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/auth"
	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/services/api-gateway/internal/gateway"
	"github.com/haradrim/chainorchestra/services/api-gateway/internal/proxy"
	"github.com/haradrim/chainorchestra/services/api-gateway/internal/ratelimit"
	"github.com/haradrim/chainorchestra/services/api-gateway/internal/realtime"
)

func newRouter(cfg config, hub *realtime.Hub, log *zap.Logger) (http.Handler, error) {
	userProxy, err := proxy.New(cfg.UserService, log.Named("proxy.user"))
	if err != nil {
		return nil, err
	}

	orderProxy, err := proxy.New(cfg.OrderService, log.Named("proxy.order"))
	if err != nil {
		return nil, err
	}

	inventoryProxy, err := proxy.New(cfg.InventoryService, log.Named("proxy.inventory"))
	if err != nil {
		return nil, err
	}

	logisticsProxy, err := proxy.New(cfg.LogisticsService, log.Named("proxy.logistics"))
	if err != nil {
		return nil, err
	}

	analyticsProxy, err := proxy.New(cfg.AnalyticsService, log.Named("proxy.analytics"))
	if err != nil {
		return nil, err
	}

	notificationProxy, err := proxy.New(cfg.NotificationService, log.Named("proxy.notification"))
	if err != nil {
		return nil, err
	}

	simulatorProxy, err := proxy.New(cfg.SimulatorService, log.Named("proxy.simulator"))
	if err != nil {
		return nil, err
	}

	validator := auth.NewValidator(cfg.JWTSecret)
	jwtMW := gateway.NewJWTMiddleware(validator)

	limiter := ratelimit.NewLimiter(cfg.RateLimit, cfg.RateLimitTTL)

	corsConfig := middleware.DefaultCORSConfig()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Handle("GET /metrics", promhttp.Handler())

	mux.Handle("/api/v1/auth/", userProxy)
	mux.Handle("/api/v1/users/", userProxy)
	mux.Handle("/api/v1/users", userProxy)

	mux.Handle("/api/v1/orders/", orderProxy)
	mux.Handle("/api/v1/orders", orderProxy)

	mux.Handle("/api/v1/products/", inventoryProxy)
	mux.Handle("/api/v1/products", inventoryProxy)
	mux.Handle("/api/v1/warehouses/", inventoryProxy)
	mux.Handle("/api/v1/warehouses", inventoryProxy)
	mux.Handle("/api/v1/stock/", inventoryProxy)
	mux.Handle("/api/v1/stock", inventoryProxy)
	mux.Handle("/api/v1/inventory/", inventoryProxy)

	mux.Handle("/api/v1/shipments/", logisticsProxy)
	mux.Handle("/api/v1/shipments", logisticsProxy)
	mux.Handle("/api/v1/carriers/", logisticsProxy)
	mux.Handle("/api/v1/carriers", logisticsProxy)
	mux.Handle("/api/v1/routes/", logisticsProxy)
	mux.Handle("/api/v1/logistics/", logisticsProxy)
	mux.Handle("/api/v1/tracking/", logisticsProxy)
	mux.Handle("/api/v1/public/tracking/", logisticsProxy)

	mux.Handle("GET /api/v1/analytics/audit-log", adminOnly(analyticsProxy))
	mux.Handle("GET /api/v1/analytics/trace/by-entity", adminOnly(analyticsProxy))
	mux.Handle("/api/v1/analytics/", analyticsProxy)

	mux.Handle("/api/v1/notifications/", notificationProxy)
	mux.Handle("/api/v1/notifications", notificationProxy)

	mux.Handle("/api/v1/simulator/", adminOnly(simulatorProxy))

	if hub != nil {
		mux.HandleFunc("GET /api/v1/events/stream", hub.Handle)
	} else {
		mux.HandleFunc("GET /api/v1/events/stream", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"realtime disabled (no NATS URL configured)"}`))
		})
	}

	skipPrefixes := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/password-reset",
		"/api/v1/public/",
		"/metrics",
		"/health",
	}

	metrics := middleware.NewMetrics("api_gateway")

	var handler http.Handler = mux
	handler = jwtMW.Middleware(skipPrefixes)(handler)
	handler = limiter.Middleware(handler)
	handler = metrics.Middleware(handler)
	handler = middleware.BodySize(middleware.DefaultMaxBodyBytes)(handler)
	handler = middleware.Logging(log.Named("http"))(handler)
	handler = middleware.Recovery(log.Named("recovery"))(handler)
	handler = middleware.CORS(corsConfig)(handler)
	handler = middleware.RequestID(handler)

	return handler, nil
}

func adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-Role") != "admin" {
			http.Error(w, `{"error":"admin role required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
