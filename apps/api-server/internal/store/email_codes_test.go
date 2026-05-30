package store

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zboard/api-server/internal/config"
	"github.com/zboard/api-server/internal/db"

	_ "modernc.org/sqlite"
)

func newConcurrentSQLiteStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "email-codes.sqlite")
	conn, err := sqlx.Open("sqlite", fmt.Sprintf("file:%s?_time_format=sqlite&_pragma=busy_timeout(5000)", dbPath))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	conn.SetMaxOpenConns(8)
	if err := conn.Ping(); err != nil {
		t.Fatalf("ping sqlite: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.Migrate(ctx, conn, config.DialectSQLite); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return New(conn, config.DialectSQLite)
}

func TestVerifyEmailCodeCountsConcurrentWrongAttemptsAtomically(t *testing.T) {
	st := newConcurrentSQLiteStore(t)
	ctx := context.Background()
	email := "concurrent-code@example.com"
	if err := st.CreateEmailCode(ctx, email, "123456", "reset_password", time.Hour); err != nil {
		t.Fatalf("create email code: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, EmailCodeMaxAttempts*3)
	var wg sync.WaitGroup
	for i := 0; i < EmailCodeMaxAttempts*3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := st.VerifyEmailCode(context.Background(), email, "000000", "reset_password")
			if err != nil {
				errs <- err
				return
			}
			if ok {
				errs <- fmt.Errorf("wrong code unexpectedly succeeded")
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("verify wrong code concurrently: %v", err)
	}

	row, err := st.FindLatestEmailCode(ctx, email, "reset_password")
	if err != nil {
		t.Fatalf("find latest code: %v", err)
	}
	if row.FailedAttempts != EmailCodeMaxAttempts {
		t.Fatalf("failed attempts = %d, want %d", row.FailedAttempts, EmailCodeMaxAttempts)
	}
	if row.LockedAt == nil {
		t.Fatalf("code should be locked after concurrent wrong attempts")
	}
}
