package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zboard/api-server/internal/config"
)

// Store wraps a sqlx.DB plus the active dialect for write helpers that need
// dialect-specific SQL (e.g. INSERT … RETURNING vs LAST_INSERT_ID).
type Store struct {
	DB      *sqlx.DB
	Dialect config.Dialect
}

func New(db *sqlx.DB, dialect config.Dialect) *Store { return &Store{DB: db, Dialect: dialect} }

// Rebind converts ?-style placeholders into the dialect-native ones via sqlx.
func (s *Store) Rebind(q string) string { return s.DB.Rebind(q) }

// InsertReturningID runs an insert and returns the new auto-id. It uses
// RETURNING for postgres and LastInsertId for mysql/sqlite.
func (s *Store) InsertReturningID(ctx context.Context, query string, args ...any) (int64, error) {
	switch s.Dialect {
	case config.DialectPostgres:
		q := s.Rebind(query + " RETURNING id")
		var id int64
		if err := s.DB.QueryRowxContext(ctx, q, args...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	default:
		res, err := s.DB.ExecContext(ctx, s.Rebind(query), args...)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
}

// IsUniqueViolation tries to detect dialect-specific unique-key collisions.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "1062") || // MySQL
		strings.Contains(msg, "23505") // Postgres
}

// IsNoRows returns true when err is sql.ErrNoRows.
func IsNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// Now returns a UTC time formatted for cross-dialect storage.
func Now() time.Time { return time.Now().UTC() }

// AssertSingle returns ErrNotFound (caller wraps to httpx) when no rows.
func AssertSingle(err error, what string) error {
	if IsNoRows(err) {
		return fmt.Errorf("%s not found", what)
	}
	return err
}
