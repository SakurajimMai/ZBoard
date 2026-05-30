package config

import (
	"strings"
	"testing"
)

func TestLoadRequiresExplicitTokenSecret(t *testing.T) {
	t.Setenv("ZBOARD_DB_DIALECT", "sqlite")
	t.Setenv("ZBOARD_DB_DSN", "")
	t.Setenv("ZBOARD_DB_PATH", ":memory:")
	t.Setenv("ZBOARD_TOKEN_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected ZBOARD_TOKEN_SECRET to be required")
	}
	if !strings.Contains(err.Error(), "ZBOARD_TOKEN_SECRET") {
		t.Fatalf("expected token secret error, got %v", err)
	}
}

func TestLoadRejectsDefaultTokenSecret(t *testing.T) {
	t.Setenv("ZBOARD_DB_DIALECT", "sqlite")
	t.Setenv("ZBOARD_DB_DSN", "")
	t.Setenv("ZBOARD_DB_PATH", ":memory:")
	t.Setenv("ZBOARD_TOKEN_SECRET", "dev-token-secret")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected default ZBOARD_TOKEN_SECRET to be rejected")
	}
	if !strings.Contains(err.Error(), "ZBOARD_TOKEN_SECRET") {
		t.Fatalf("expected token secret error, got %v", err)
	}
}

func TestLoadAcceptsExplicitTokenSecret(t *testing.T) {
	t.Setenv("ZBOARD_DB_DIALECT", "sqlite")
	t.Setenv("ZBOARD_DB_DSN", "")
	t.Setenv("ZBOARD_DB_PATH", ":memory:")
	t.Setenv("ZBOARD_TOKEN_SECRET", "test-token-secret-not-default")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.TokenSecret != "test-token-secret-not-default" {
		t.Fatalf("unexpected token secret %q", cfg.TokenSecret)
	}
}
