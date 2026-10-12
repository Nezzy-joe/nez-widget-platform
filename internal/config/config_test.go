package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv(
		"DATABASE_URL",
		"postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable",
	)
	t.Setenv(
		"JWT_SECRET",
		"test-secret-that-is-long-enough-for-tests",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("expected port 8080, got %q", cfg.Port)
	}

	if cfg.DatabaseURL == "" {
		t.Fatal("expected database URL to be configured")
	}

	if cfg.JWTSecret == "" {
		t.Fatal("expected JWT secret to be configured")
	}
}
