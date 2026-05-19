package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/zboard/api-server/internal/config"
)

//go:embed all:migrations
var migrationsFS embed.FS

// Migrate applies pending migrations for the active dialect.
func Migrate(ctx context.Context, db *sqlx.DB, dialect config.Dialect) error {
	if _, err := db.ExecContext(ctx, ensureMigrationsTable(dialect)); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := loadMigrations(string(dialect))
	if err != nil {
		return err
	}

	applied, err := loadApplied(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range files {
		if applied[m.version] {
			continue
		}
		if err := applyMigration(ctx, db, m, dialect); err != nil {
			return fmt.Errorf("apply %s: %w", m.version, err)
		}
	}
	return nil
}

type migration struct {
	version string
	sql     string
}

func loadMigrations(dialect string) ([]migration, error) {
	dir := "migrations/" + dialect
	entries, err := fs.ReadDir(migrationsFS, dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	var out []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		body, err := fs.ReadFile(migrationsFS, dir+"/"+e.Name())
		if err != nil {
			return nil, err
		}
		version := strings.TrimSuffix(e.Name(), ".sql")
		out = append(out, migration{version: version, sql: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func loadApplied(ctx context.Context, db *sqlx.DB) (map[string]bool, error) {
	rows, err := db.QueryxContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func applyMigration(ctx context.Context, db *sqlx.DB, m migration, dialect config.Dialect) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitSQL(m.sql) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("statement: %s\nerror: %w", firstLine(stmt), err)
		}
	}
	insert := "INSERT INTO schema_migrations(version) VALUES (?)"
	if dialect == config.DialectPostgres {
		insert = "INSERT INTO schema_migrations(version) VALUES ($1)"
	}
	if _, err := tx.ExecContext(ctx, insert, m.version); err != nil {
		return fmt.Errorf("record version: %w", err)
	}
	return tx.Commit()
}

// splitSQL splits on `;` boundaries. Sufficient for the plain DDL used here.
func splitSQL(s string) []string {
	parts := strings.Split(s, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func ensureMigrationsTable(dialect config.Dialect) string {
	switch dialect {
	case config.DialectMySQL:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(64) PRIMARY KEY,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case config.DialectPostgres:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`
	default:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
)`
	}
}
