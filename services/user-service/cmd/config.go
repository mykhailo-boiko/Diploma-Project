package main

import (
	"os"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/postgres"
	"github.com/haradrim/chainorchestra/services/user-service/internal/auth"
)

type config struct {
	Listen        string
	Postgres      postgres.Config
	Auth          auth.Config
	NatsURL       string
	AdminEmail    string
	AdminPassword string
}

func loadConfig() config {
	return config{
		Listen:  envOrDefault("LISTEN", ":8001"),
		NatsURL: envOrDefault("NATS_URL", "nats://localhost:4222"),
		Postgres: postgres.Config{
			Host:     envOrDefault("POSTGRES_HOST", "localhost"),
			Port:     envInt("POSTGRES_PORT", 5432),
			User:     envOrDefault("POSTGRES_USER", "user_service"),
			Password: envOrDefault("POSTGRES_PASSWORD", "user_service_pass"),
			Database: envOrDefault("POSTGRES_DB", "chainorchestra"),
			Schema:   envOrDefault("POSTGRES_SCHEMA", "users"),
			SSLMode:  envOrDefault("POSTGRES_SSLMODE", "disable"),
		},
		Auth: auth.Config{
			Secret:     envOrDefault("JWT_SECRET", "dev-secret-change-me"),
			AccessTTL:  envDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: envDuration("JWT_REFRESH_TTL", 168*time.Hour),
		},
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
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
	var n int
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
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
