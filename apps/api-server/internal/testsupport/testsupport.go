// Package testsupport spins up an in-memory SQLite store with all migrations
// applied so individual packages can run table-driven tests without a real DB.
package testsupport

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zboard/api-server/internal/config"
	"github.com/zboard/api-server/internal/db"
	"github.com/zboard/api-server/internal/store"

	_ "modernc.org/sqlite"
)

// NewStore returns a fresh in-memory SQLite store with migrations applied.
// Each call gets a brand new database — tests are fully isolated from each
// other.
func NewStore(t *testing.T) *store.Store {
	t.Helper()
	// `:memory:` plus a unique cache=shared name is unnecessary here because
	// each *sqlx.DB owns one connection pool. We force max-open=1 so multiple
	// statements always hit the same in-memory database.
	//
	// `_time_format=sqlite` tells the modernc driver to parse the TEXT
	// `CURRENT_TIMESTAMP` columns into time.Time on Scan — the production
	// MySQL/MariaDB DSN gets the same effect via `parseTime=true`.
	conn, err := sqlx.Open("sqlite", "file::memory:?_time_format=sqlite")
	if err != nil {
		t.Fatalf("open sqlite memory: %v", err)
	}
	conn.SetMaxOpenConns(1)
	if err := conn.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.Migrate(ctx, conn, config.DialectSQLite); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return store.New(conn, config.DialectSQLite)
}
