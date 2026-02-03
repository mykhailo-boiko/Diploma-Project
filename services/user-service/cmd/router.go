package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/user-service/internal/controller"
)

func newRouter(authCtrl *controller.AuthController, userCtrl *controller.UserController, nc *natspkg.Client) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	metrics := middleware.NewMetrics("user_service")
	handler := metrics.Middleware(middleware.UserContext(mux))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /health/nats", nc.HealthHandler())

	mux.HandleFunc("POST /api/v1/auth/login", authCtrl.Login)
	mux.HandleFunc("POST /api/v1/auth/register", authCtrl.Register)
	mux.HandleFunc("POST /api/v1/auth/refresh", authCtrl.Refresh)
	mux.HandleFunc("POST /api/v1/auth/password-reset", userCtrl.RequestPasswordReset)
	mux.HandleFunc("POST /api/v1/auth/password-reset/confirm", userCtrl.ConfirmPasswordReset)

	mux.HandleFunc("GET /api/v1/users/me", userCtrl.GetProfile)
	mux.HandleFunc("PUT /api/v1/users/me", userCtrl.UpdateProfile)

	mux.HandleFunc("GET /api/v1/users", userCtrl.ListUsers)
	mux.HandleFunc("POST /api/v1/users", userCtrl.CreateUser)
	mux.HandleFunc("PUT /api/v1/users/{id}", userCtrl.UpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/{id}", userCtrl.DeleteUser)

	return handler
}
