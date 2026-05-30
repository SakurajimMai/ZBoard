package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zboard/api-server/internal/subtoken"
)

type Dialect string

const (
	DialectMySQL    Dialect = "mysql"
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

type Config struct {
	Host            string
	Port            int
	DBDialect       Dialect
	DBDSN           string
	AdminSetupToken string
	AdminEmail      string
	AdminPassword   string
	TokenSecret     string
	CORSOrigins     []string
	// TrustedProxies lists CIDR ranges (or bare IPs) of reverse proxies allowed
	// to set X-Forwarded-For. Empty means trust no proxy: ClientIP falls back to
	// the direct TCP peer, so a spoofed X-Forwarded-For can't shift the per-IP
	// rate-limit bucket. Set this only to the CIDRs of your own proxy/CDN tier
	// so legitimate clients are counted by their real IP instead of all sharing
	// the proxy's address.
	TrustedProxies []string

	// SMTP for transactional emails. When SMTPHost/From are empty, the mailer
	// is disabled and code sends become no-ops (logged only).
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

func Load() (*Config, error) {
	port, err := strconv.Atoi(getenv("ZBOARD_PORT", "3000"))
	if err != nil {
		return nil, fmt.Errorf("ZBOARD_PORT: %w", err)
	}
	dialect := Dialect(strings.ToLower(getenv("ZBOARD_DB_DIALECT", "sqlite")))
	switch dialect {
	case DialectMySQL, DialectPostgres, DialectSQLite:
	default:
		return nil, fmt.Errorf("unsupported ZBOARD_DB_DIALECT: %s", dialect)
	}
	dsn := os.Getenv("ZBOARD_DB_DSN")
	if dsn == "" {
		if dialect == DialectSQLite {
			dsn = getenv("ZBOARD_DB_PATH", "./data/zboard.sqlite")
		} else {
			return nil, fmt.Errorf("ZBOARD_DB_DSN is required for dialect %s", dialect)
		}
	}
	tokenSecret := strings.TrimSpace(os.Getenv("ZBOARD_TOKEN_SECRET"))
	if err := subtoken.ValidateSigningSecret(tokenSecret); err != nil {
		return nil, fmt.Errorf("ZBOARD_TOKEN_SECRET: %w", err)
	}
	return &Config{
		Host:            getenv("ZBOARD_HOST", "127.0.0.1"),
		Port:            port,
		DBDialect:       dialect,
		DBDSN:           dsn,
		AdminSetupToken: os.Getenv("ZBOARD_ADMIN_SETUP_TOKEN"),
		AdminEmail:      os.Getenv("ZBOARD_ADMIN_EMAIL"),
		AdminPassword:   os.Getenv("ZBOARD_ADMIN_PASSWORD"),
		TokenSecret:     tokenSecret,
		CORSOrigins:     parseCORSOrigins(os.Getenv("ZBOARD_CORS_ORIGINS")),
		TrustedProxies:  parseTrustedProxies(os.Getenv("ZBOARD_TRUSTED_PROXIES")),

		SMTPHost: os.Getenv("ZBOARD_SMTP_HOST"),
		SMTPPort: atoiOr(os.Getenv("ZBOARD_SMTP_PORT"), 587),
		SMTPUser: os.Getenv("ZBOARD_SMTP_USER"),
		SMTPPass: os.Getenv("ZBOARD_SMTP_PASS"),
		SMTPFrom: os.Getenv("ZBOARD_SMTP_FROM"),
	}, nil
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// parseCORSOrigins splits a comma-separated list of origins.
// Example: "http://localhost:3001,https://panel.example.com"
// A single "*" means allow all origins.
func parseCORSOrigins(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// parseTrustedProxies splits a comma-separated list of proxy CIDRs or IPs.
// Example: "127.0.0.1/32,10.0.0.0/8,172.18.0.0/16"
// Empty input returns nil, which means "trust no proxy".
func parseTrustedProxies(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
