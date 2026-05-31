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

func TestParseTrustedPlatform(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"cloudflare", "CF-Connecting-IP"},
		{"CF", "CF-Connecting-IP"},
		{"  Cloudflare  ", "CF-Connecting-IP"},
		{"google", "X-Appengine-Remote-Addr"},
		{"gae", "X-Appengine-Remote-Addr"},
		{"fly", "Fly-Client-IP"},
		{"flyio", "Fly-Client-IP"},
		{"X-Custom-Real-IP", "X-Custom-Real-IP"},
	}
	for _, tc := range cases {
		if got := parseTrustedPlatform(tc.in); got != tc.want {
			t.Errorf("parseTrustedPlatform(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLoadReadsTrustedPlatform(t *testing.T) {
	t.Setenv("ZBOARD_DB_DIALECT", "sqlite")
	t.Setenv("ZBOARD_DB_DSN", "")
	t.Setenv("ZBOARD_DB_PATH", ":memory:")
	t.Setenv("ZBOARD_TOKEN_SECRET", "test-token-secret-not-default")
	t.Setenv("ZBOARD_TRUSTED_PLATFORM", "cloudflare")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.TrustedPlatform != "CF-Connecting-IP" {
		t.Fatalf("TrustedPlatform = %q, want CF-Connecting-IP", cfg.TrustedPlatform)
	}
}
