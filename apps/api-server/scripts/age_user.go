// Test helper to mutate the DB directly (age expiry, etc.)
//go:build ignore

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/zboard/api-server/internal/config"
	"github.com/zboard/api-server/internal/db"
)

func main() {
	dialect := flag.String("dialect", "mysql", "dialect")
	dsn := flag.String("dsn", "", "DSN")
	userID := flag.Int64("user", 0, "user id to age")
	flag.Parse()

	cfg := &config.Config{DBDialect: config.Dialect(*dialect), DBDSN: *dsn}
	conn, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer conn.Close()

	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	res, err := conn.ExecContext(context.Background(),
		conn.Rebind(`UPDATE users SET expired_at = ? WHERE id = ?`),
		past, *userID)
	if err != nil {
		log.Fatalf("update: %v", err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("aged user %d: %d row(s)\n", *userID, n)
}
