// Package config reads runtime settings from environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime settings.
type Config struct {
	Port            string
	CacheTTL        time.Duration
	AllowedOrigin   string
	ITunesBaseURL   string
	UpstreamTimeout time.Duration
}

// Load reads configuration from the environment, applying defaults.
func Load() (Config, error) {
	cfg := Config{
		Port:            getenv("PORT", "8080"),
		AllowedOrigin:   getenv("ALLOWED_ORIGIN", "http://localhost:5173"),
		ITunesBaseURL:   getenv("ITUNES_BASE_URL", "https://itunes.apple.com"),
		UpstreamTimeout: 5 * time.Second,
	}

	ttl, err := time.ParseDuration(getenv("CACHE_TTL", "1h"))
	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf("invalid CACHE_TTL %q: want a positive Go duration like 1h", os.Getenv("CACHE_TTL"))
	}
	cfg.CacheTTL = ttl
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
