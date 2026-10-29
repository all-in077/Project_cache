package config

import (
	"fmt"
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

func Load() (*Config, error) {
	ttl, err := time.ParseDuration(getEnv("CACHE_TTL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid CACHE_TTL: %w", err)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("CACHE_TTL must be positive, got %s", ttl)
	}

	capacity, err := strconv.Atoi(getEnv("CACHE_CAPACITY", "100"))
	if err != nil {
		return nil, fmt.Errorf("invalid CACHE_CAPACITY: %w", err)
	}
	if capacity <= 0 {
		return nil, fmt.Errorf("CACHE_CAPACITY must be positive, got %d", capacity)
	}

	return &Config{
		CacheCapacity: capacity,
		CacheTTL:      ttl,
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
