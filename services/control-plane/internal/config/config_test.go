package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.CORSOrigin != "http://localhost:3000" {
		t.Fatalf("expected default CORS origin, got %s", cfg.CORSOrigin)
	}
	if cfg.ModuleTimeout != 90*time.Second {
		t.Fatalf("expected default module timeout 90s, got %s", cfg.ModuleTimeout)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Fatalf("expected default read timeout 30s, got %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 180*time.Second {
		t.Fatalf("expected default write timeout 180s, got %s", cfg.WriteTimeout)
	}
	if cfg.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("expected default read header timeout 5s, got %s", cfg.ReadHeaderTimeout)
	}
}

func TestLoadEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("CORS_ORIGIN", "http://example.com")
	t.Setenv("MODULE_TIMEOUT", "2m")
	t.Setenv("HTTP_READ_TIMEOUT", "45s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "5m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.CORSOrigin != "http://example.com" {
		t.Fatalf("expected CORS origin http://example.com, got %s", cfg.CORSOrigin)
	}
	if cfg.ModuleTimeout != 2*time.Minute {
		t.Fatalf("expected module timeout 2m, got %s", cfg.ModuleTimeout)
	}
	if cfg.ReadTimeout != 45*time.Second {
		t.Fatalf("expected read timeout 45s, got %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 5*time.Minute {
		t.Fatalf("expected write timeout 5m, got %s", cfg.WriteTimeout)
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	t.Setenv("MODULE_TIMEOUT", "not-a-duration")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
