package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/admin"
	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/handler"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	lg := logger.New(cfg.LogDir, cfg.LogLevel)
	lg.Info("system starting", map[string]interface{}{
		"version": "2.0.0",
		"port":    cfg.Port,
		"models":  len(cfg.Models),
	})

	svc := service.NewModerationService(cfg, lg)

	var db *storage.DB
	if cfg.EnableAuth || cfg.EnableAdminAPI {
		db, err = storage.New("/data")
		if err != nil {
			lg.Error("database init failed: " + err.Error())
			db = nil
		} else {
			// When DB is available, project key auth is sourced from SQLite at runtime.
			cfg.AllowedKeys = nil
			defer db.Close()
		}
	}

	mux := http.NewServeMux()
	h := handler.New(svc, lg, cfg, db)
	h.RegisterRoutes(mux)

	if cfg.EnableAdminAPI {
		auditLogger := audit.New(cfg.AuditLogDir, cfg.EnableAudit)
		adminHandler := admin.New(cfg, lg, auditLogger, db)
		adminHandler.RegisterRoutes(mux)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		lg.Info(fmt.Sprintf("server listening on %d", cfg.Port), nil)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lg.Error("server start failed: " + err.Error())
			os.Exit(1)
		}
	}()

	<-quit
	lg.Info("shutting down server", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		lg.Error("server forced shutdown: " + err.Error())
	}

	lg.Info("server exited", nil)
}
