package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/actors"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/httpclient"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	cfg := loadConfig()
	log = log.With(zap.String("service", "simulator-service"))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	st := state.New(state.ParseScenario(cfg.DefaultScenario), cfg.DefaultSpeed)
	if cfg.AutoStart {
		st.Start(state.ParseScenario(cfg.DefaultScenario), cfg.DefaultSpeed)
		log.Info("Auto-started simulator", zap.String("scenario", cfg.DefaultScenario), zap.Float64("speed", cfg.DefaultSpeed))
	}

	hc := httpclient.New(cfg.GatewayURL, cfg.ServiceEmail, cfg.ServicePassword, log.Named("httpclient"))
	catalog := actors.NewCatalog(hc)

	ctrl := controller.New(st, log.Named("controller"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /api/v1/simulator/status", ctrl.Status)
	mux.HandleFunc("POST /api/v1/simulator/start", ctrl.Start)
	mux.HandleFunc("POST /api/v1/simulator/stop", ctrl.Stop)
	mux.HandleFunc("POST /api/v1/simulator/speed", ctrl.SetSpeed)
	mux.HandleFunc("POST /api/v1/simulator/scenario", ctrl.SetScenario)

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("Simulator service starting", zap.String("addr", cfg.Listen))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	var wg sync.WaitGroup
	launch := func(a actors.Actor, intervalFn func(actors.ScenarioTuning) float64) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			actors.RunActor(ctx, a, st, intervalFn, log.Named("actor"))
		}()
	}

	launch(actors.NewOrderSpawner(hc, catalog, st, log.Named("order_spawner")),
		func(t actors.ScenarioTuning) float64 { return t.OrderSpawnIntervalSec })
	launch(actors.NewOrderProgressor(hc, catalog, st, log.Named("order_progressor")),
		func(t actors.ScenarioTuning) float64 { return t.OrderProgressIntervalSec })
	launch(actors.NewShipmentProgressor(hc, st, log.Named("shipment_progressor")),
		func(t actors.ScenarioTuning) float64 { return t.ShipmentTickIntervalSec })
	launch(actors.NewInventoryFluctuator(hc, catalog, st, log.Named("inventory_fluctuator")),
		func(t actors.ScenarioTuning) float64 { return t.InventoryTickIntervalSec })
	launch(actors.NewNotificationActor(hc, st, log.Named("notification_actor")),
		func(t actors.ScenarioTuning) float64 { return t.NotificationIntervalSec })

	wg.Add(1)
	go func() {
		defer wg.Done()
		actors.RunSupervisor(ctx, st, log)
	}()

	<-ctx.Done()
	log.Info("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	wg.Wait()
	log.Info("Simulator service stopped")
}
