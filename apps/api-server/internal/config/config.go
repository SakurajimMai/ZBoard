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
	Host             string
	Port             int
	DBDialect        Dialect
	DBDSN            string
	AdminSetupToken  string
	TokenSecret      string

	// EasyPay (易支付)
	EpayAPIURL string
	EpayPID    string
	EpayKey    string

	// Creem
	CreemAPIKey        string
	CreemWebhookSecret string
	CreemAPIURL        string

	// NOWPayments
	NowPayAPIKey    string
	NowPayIPNSecret string
	NowPayAPIURL    string
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
		AdminSetupToken: getenv("ZBOARD_ADMIN_SETUP_TOKEN", "dev-admin-token"),
		TokenSecret:     getenv("ZBOARD_TOKEN_SECRET", "dev-token-secret"),

		EpayAPIURL: os.Getenv("ZBOARD_EPAY_API_URL"),
		EpayPID:    os.Getenv("ZBOARD_EPAY_PID"),
		EpayKey:    os.Getenv("ZBOARD_EPAY_KEY"),

		CreemAPIKey:        os.Getenv("ZBOARD_CREEM_API_KEY"),
		CreemWebhookSecret: os.Getenv("ZBOARD_CREEM_WEBHOOK_SECRET"),
		CreemAPIURL:        os.Getenv("ZBOARD_CREEM_API_URL"),

		NowPayAPIKey:    os.Getenv("ZBOARD_NOWPAY_API_KEY"),
		NowPayIPNSecret: os.Getenv("ZBOARD_NOWPAY_IPN_SECRET"),
		NowPayAPIURL:    os.Getenv("ZBOARD_NOWPAY_API_URL"),
	}, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
