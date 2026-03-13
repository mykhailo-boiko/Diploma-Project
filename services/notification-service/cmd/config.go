package main

import (
	"os"

	"github.com/haradrim/chainorchestra/internal/pkg/postgres"
)

type config struct {
	Listen   string
	Postgres postgres.Config
	NatsURL  string
}

func loadConfig() config {
	return config{
		Listen:  envOrDefault("LISTEN", ":8006"),
		NatsURL: envOrDefault("NATS_URL", "nats://localhost:4222"),
		Postgres: postgres.Config{
			Host:     envOrDefault("POSTGRES_HOST", "localhost"),
			Port:     envInt("POSTGRES_PORT", 5432),
			User:     envOrDefault("POSTGRES_USER", "notification_service"),
			Password: envOrDefault("POSTGRES_PASSWORD", "notification_service_pass"),
			Database: envOrDefault("POSTGRES_DB", "chainorchestra"),
			Schema:   envOrDefault("POSTGRES_SCHEMA", "notifications"),
			SSLMode:  envOrDefault("POSTGRES_SSLMODE", "disable"),
		},
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
