package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	CORSOrigins     []string // allowed origins, e.g. ["http://localhost:3001","https://panel.example.com"]
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
	return &Config{
		Host:            getenv("ZBOARD_HOST", "127.0.0.1"),
		Port:            port,
		DBDialect:       dialect,
		DBDSN:           dsn,
		AdminSetupToken: os.Getenv("ZBOARD_ADMIN_SETUP_TOKEN"),
		AdminEmail:      os.Getenv("ZBOARD_ADMIN_EMAIL"),
		AdminPassword:   os.Getenv("ZBOARD_ADMIN_PASSWORD"),
		TokenSecret:     getenv("ZBOARD_TOKEN_SECRET", "dev-token-secret"),
		CORSOrigins:     parseCORSOrigins(os.Getenv("ZBOARD_CORS_ORIGINS")),
	}, nil
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
