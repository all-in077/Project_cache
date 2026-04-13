package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	CacheCapacity int
	CacheTTL      time.Duration
	HTTPPort      string
	GRPCPort      string
}

func Load() *Config {
	ttl, err := time.ParseDuration(getEnv("CACHE_TTL", "5m"))
	if err != nil {
		ttl = 5 * time.Minute
	}

	capacity, err := strconv.Atoi(getEnv("CACHE_CAPACITY", "100"))
	if err != nil {
		capacity = 100
	}

	return &Config{
		CacheCapacity: capacity,
		CacheTTL:      ttl,
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
