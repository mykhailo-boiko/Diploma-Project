package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/internal/pkg/postgres"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/consumer"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/delivery"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/ws"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	cfg := loadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		log.Fatal("Failed to connect to postgres", zap.Error(err))
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		log.Fatal("Failed to run migrations", zap.Error(err))
	}
	log.Info("Migrations applied successfully")

	natsCfg := natspkg.DefaultConfig(cfg.NatsURL, "notification-service")
	nc, err := natspkg.NewClient(natsCfg, log.Named("nats"))
	if err != nil {
		log.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()

	storage := notification.NewPostgresStorage(pool)
	emailAdapter := delivery.NewEmailAdapter(log.Named("email"))
	smsAdapter := delivery.NewSMSAdapter(log.Named("sms"))
	wsHub := ws.NewHub(log.Named("websocket"))

	svc := controller.NewService(storage, emailAdapter, smsAdapter, wsHub, log.Named("service"))
	notifCtrl := controller.NewNotificationController(svc, log.Named("controller"))

	cons := consumer.NewConsumer(svc, nc, log.Named("consumer"))
	if err := cons.Start(); err != nil {
		log.Fatal("Failed to start NATS consumer", zap.Error(err))
	}

	router := newRouter(notifCtrl, nc, wsHub)

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("Notification service starting", zap.String("addr", cfg.Listen))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	log.Info("Notification service stopped")
}
