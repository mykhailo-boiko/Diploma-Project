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
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	cfg := loadConfig()

	router, err := newRouter(cfg, log)
	if err != nil {
		log.Fatal("Failed to create router", zap.Error(err))
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
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

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	log.Info("API Gateway stopped")
}
