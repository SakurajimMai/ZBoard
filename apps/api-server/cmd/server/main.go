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
	"github.com/zboard/api-server/internal/captchasvc"
	"github.com/zboard/api-server/internal/config"
	"github.com/zboard/api-server/internal/db"
	"github.com/zboard/api-server/internal/mailer"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/payment/registry"
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
	mail := mailer.New(mailer.Config{
		Host: cfg.SMTPHost,
		Port: cfg.SMTPPort,
		User: cfg.SMTPUser,
		Pass: cfg.SMTPPass,
		From: cfg.SMTPFrom,
	})
	if mail.Enabled() {
		log.Printf("SMTP enabled: %s:%d (from=%s)", cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)
	} else {
		log.Printf("SMTP disabled — verification codes will be logged only")
	}
	auth := authsvc.New(st, cfg.AdminSetupToken, mail)
	biz := bizsvc.New(st)
	nodes := nodesvc.New(st)
	wk := worker.New(st)

	// Auto-bootstrap: create initial admin if ZBOARD_ADMIN_EMAIL is set and
	// no admin exists yet. This replaces the manual POST /auth/bootstrap step.
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if _, err := auth.BootstrapAdmin(context.Background(), cfg.AdminSetupToken, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Printf("auto-bootstrap: %v (already initialized or token mismatch)", err)
		} else {
			log.Printf("auto-bootstrap: admin %s created", cfg.AdminEmail)
		}
	}

	// Payment providers — loaded from DB (payment_providers table).
	// Admin manages them via POST/PUT/DELETE /api/admin/v1/payment-providers.
	payments := registry.New(st)
	if err := payments.Reload(context.Background()); err != nil {
		log.Printf("payment providers: %v (will retry on first request)", err)
	}

	r := server.New(server.Deps{DB: database, Store: st, Auth: auth, Biz: biz, Nodes: nodes, Worker: wk, Payments: payments, Captcha: captchasvc.New(st), CORSOrigins: cfg.CORSOrigins})
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
