package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/zboard/api-server/internal/config"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func Open(cfg *config.Config) (*sqlx.DB, error) {
	driver, dsn := driverDSN(cfg)
	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", driver, err)
	}
	return db, nil
}

func driverDSN(cfg *config.Config) (string, string) {
	switch cfg.DBDialect {
	case config.DialectMySQL:
		return "mysql", cfg.DBDSN
	case config.DialectPostgres:
		return "pgx", cfg.DBDSN
	default:
		return "sqlite", cfg.DBDSN
	}
}
