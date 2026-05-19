package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/config"
	"github.com/zboard/api-server/internal/db"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/server"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 60*time.Second)
	if err := db.Migrate(migrateCtx, database, cfg.DBDialect); err != nil {
		cancelMigrate()
		log.Fatalf("migrate: %v", err)
	}
	cancelMigrate()

	st := store.New(database, cfg.DBDialect)
	auth := authsvc.New(st, cfg.AdminSetupToken)
	biz := bizsvc.New(st)
	nodes := nodesvc.New(st)
	wk := worker.New(st)

	r := server.New(server.Deps{DB: database, Store: st, Auth: auth, Biz: biz, Nodes: nodes, Worker: wk})
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Zboard API listening on http://%s:%d (dialect=%s)", cfg.Host, cfg.Port, cfg.DBDialect)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
