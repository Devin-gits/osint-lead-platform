// Package config loads control-plane settings from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the runtime configuration.
type Config struct {
	DatabaseURL       string
	Port              string
	ListenHost        string
	CORSOrigin        string
	ModuleTimeout     time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
}

// Load reads configuration from the environment and returns sane defaults.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Port:              os.Getenv("PORT"),
		ListenHost:        os.Getenv("LISTEN_HOST"),
		CORSOrigin:        os.Getenv("CORS_ORIGIN"),
		ModuleTimeout:     90 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      180 * time.Second,
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.ListenHost == "" {
		cfg.ListenHost = "127.0.0.1"
	}
	if cfg.CORSOrigin == "" {
		cfg.CORSOrigin = "http://localhost:3000"
	}

	timeouts := []struct {
		env string
		d   *time.Duration
	}{
		{"MODULE_TIMEOUT", &cfg.ModuleTimeout},
		{"HTTP_READ_TIMEOUT", &cfg.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", &cfg.WriteTimeout},
	}
	for _, t := range timeouts {
		if v := os.Getenv(t.env); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", t.env, err)
			}
			*t.d = d
		}
	}

	return cfg, nil
}

// ParseBool is a small helper for optional boolean env vars.
func ParseBool(key string, def bool) bool {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return b
}
