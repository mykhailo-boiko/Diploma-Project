package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/api-gateway/internal/realtime"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	cfg := loadConfig()

	bootCtx, bootCancel := context.WithCancel(context.Background())
	defer bootCancel()

	var ncRef *nats.Conn
	hub := realtime.NewLazyHub(log.Named("realtime"))
	if cfg.NatsURL != "" {
		go func() {
			attempt := 0
			for {
				attempt++
				nc, err := nats.Connect(cfg.NatsURL,
					nats.Name("api-gateway"),
					nats.MaxReconnects(-1),
					nats.ReconnectWait(2*time.Second),
				)
				if err == nil {
					ncRef = nc
					if attachErr := hub.Attach(nc); attachErr != nil {
						log.Error("Hub attach failed", zap.Error(attachErr))
						nc.Close()
					} else {
						log.Info("Realtime hub online", zap.Int("attempts", attempt))
						return
					}
				} else {
					log.Warn("NATS connect retry", zap.Int("attempt", attempt), zap.Error(err))
				}
				select {
				case <-bootCtx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
		}()
	} else {
		log.Warn("NATS URL not configured; realtime stream permanently disabled")
	}

	router, err := newRouter(cfg, hub, log)
	if err != nil {
		log.Fatal("Failed to create router", zap.Error(err))
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		log.Info("API Gateway starting", zap.String("addr", cfg.Listen))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	bootCancel()
	if hub != nil {
		hub.Stop()
	}
	if ncRef != nil {
		ncRef.Close()
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	log.Info("API Gateway stopped")
}
