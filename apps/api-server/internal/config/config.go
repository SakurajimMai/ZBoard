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

	// TrustedPlatform names a request header whose value gin trusts as the real
	// client IP, read directly without consulting TrustedProxies. Set this when
	// your origin sits behind a CDN that injects a real-client-IP header (e.g.
	// Cloudflare's CF-Connecting-IP) AND your origin only accepts traffic from
	// that CDN — otherwise a direct client could spoof the header to dodge rate
	// limits. Empty disables it. Mutually preferable to TrustedProxies for CDNs
	// whose egress IP ranges change frequently.
	TrustedPlatform string

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
		TrustedPlatform: parseTrustedPlatform(os.Getenv("ZBOARD_TRUSTED_PLATFORM")),

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

// parseTrustedPlatform maps the ZBOARD_TRUSTED_PLATFORM env value to the request
// header gin should trust as the real client IP. Friendly aliases are accepted
// for the common CDNs; any other non-empty value is treated as a literal header
// name so future platforms work without a code change. The returned strings
// match gin's Platform* constants but are kept literal here to avoid coupling
// the config package to the web framework.
//
//	cloudflare / cf   -> CF-Connecting-IP
//	google / gae      -> X-Appengine-Remote-Addr
//	fly / flyio       -> Fly-Client-IP
//	<anything else>   -> used verbatim as the header name
//	"" (empty)        -> disabled
func parseTrustedPlatform(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	switch strings.ToLower(s) {
	case "cloudflare", "cf":
		return "CF-Connecting-IP"
	case "google", "gae", "appengine":
		return "X-Appengine-Remote-Addr"
	case "fly", "flyio", "fly.io":
		return "Fly-Client-IP"
	default:
		return s
	}
}
