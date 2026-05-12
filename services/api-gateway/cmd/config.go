package main

import (
	"os"
	"strconv"
	"time"
)

type config struct {
	Listen           string
	JWTSecret        string
	RateLimit        int
	RateLimitTTL     time.Duration
	UserService      string
	OrderService     string
	InventoryService string
	LogisticsService string
	AnalyticsService     string
	NotificationService  string
	SimulatorService     string
	NatsURL              string
}

func loadConfig() config {
	return config{
		Listen:       envOrDefault("LISTEN", ":8080"),
		JWTSecret:    envOrDefault("JWT_SECRET", "dev-secret-change-me"),
		RateLimit:    envInt("RATE_LIMIT", 100),
		RateLimitTTL: envDuration("RATE_LIMIT_TTL", time.Minute),
		UserService:  envOrDefault("USER_SERVICE_URL", "http://localhost:8001"),
		OrderService:     envOrDefault("ORDER_SERVICE_URL", "http://localhost:8002"),
		InventoryService: envOrDefault("INVENTORY_SERVICE_URL", "http://localhost:8003"),
		LogisticsService: envOrDefault("LOGISTICS_SERVICE_URL", "http://localhost:8004"),
		AnalyticsService:     envOrDefault("ANALYTICS_SERVICE_URL", "http://localhost:8005"),
		NotificationService:  envOrDefault("NOTIFICATION_SERVICE_URL", "http://localhost:8006"),
		SimulatorService:     envOrDefault("SIMULATOR_SERVICE_URL", "http://localhost:8007"),
		NatsURL:              envOrDefault("NATS_URL", "nats://localhost:4222"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
