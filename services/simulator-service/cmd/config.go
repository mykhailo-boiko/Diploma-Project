package main

import (
	"os"
	"strconv"
)

type config struct {
	Listen           string
	GatewayURL       string
	ServiceEmail     string
	ServicePassword  string
	NatsURL          string
	AutoStart        bool
	DefaultScenario  string
	DefaultSpeed     float64
	MaxOrdersPerMin  int
	MaxEventsPerMin  int
}

func loadConfig() config {
	return config{
		Listen:          envOrDefault("LISTEN", ":8007"),
		GatewayURL:      envOrDefault("GATEWAY_URL", "http://api-gateway:8080"),
		ServiceEmail:    envOrDefault("SERVICE_USER_EMAIL", "admin@chainorchestra.local"),
		ServicePassword: envOrDefault("SERVICE_USER_PASSWORD", ""),
		NatsURL:         envOrDefault("NATS_URL", "nats://nats:4222"),
		AutoStart:       envBool("SIMULATOR_AUTOSTART", false),
		DefaultScenario: envOrDefault("SIMULATOR_SCENARIO", "steady"),
		DefaultSpeed:    envFloat("SIMULATOR_SPEED", 1.0),
		MaxOrdersPerMin: envInt("SIMULATOR_MAX_ORDERS_PER_MIN", 60),
		MaxEventsPerMin: envInt("SIMULATOR_MAX_EVENTS_PER_MIN", 300),
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

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
