package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheCapacity != 100 || cfg.CacheTTL != 5*time.Minute ||
		cfg.HTTPPort != "8080" || cfg.GRPCPort != "50051" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("CACHE_CAPACITY", "5")
	t.Setenv("CACHE_TTL", "30s")
	t.Setenv("HTTP_PORT", "9000")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheCapacity != 5 || cfg.CacheTTL != 30*time.Second || cfg.HTTPPort != "9000" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadInvalid(t *testing.T) {
	cases := map[string][2]string{
		"bad ttl":       {"CACHE_TTL", "abc"},
		"zero ttl":      {"CACHE_TTL", "0s"},
		"negative ttl":  {"CACHE_TTL", "-1m"},
		"bad capacity":  {"CACHE_CAPACITY", "abc"},
		"zero capacity": {"CACHE_CAPACITY", "0"},
	}
	for name, kv := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(kv[0], kv[1])
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
