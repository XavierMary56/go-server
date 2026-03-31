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
	})

	// 保存原始的 AllowedKeys，因为可能需要导入到数据库
	originalAllowedKeys := cfg.AllowedKeys

	var db *storage.DB
	if cfg.EnableAuth || cfg.EnableAdminAPI {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=Local",
			cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
		db, err = storage.New(dsn)
		if err != nil {
			if cfg.DBRequired {
				log.Fatalf("database init failed (DB_REQUIRED=true): %v", err)
			}
			lg.Error("database init failed, continuing without DB: " + err.Error())
			db = nil
		} else {
			// When DB is available, project key auth is sourced from SQLite at runtime.
			cfg.AllowedKeys = nil
			defer db.Close()
		}
	}

	svc := service.NewModerationService(cfg, lg, db)
	auditLogger := audit.New(cfg.AuditLogDir, cfg.EnableAudit)
	defer auditLogger.Close()

	mux := http.NewServeMux()
	h := handler.New(svc, lg, cfg, db, auditLogger)
	h.RegisterRoutes(mux)

	// 启动后台 Key 健康检测（每 5 分钟一次）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartHealthChecker(ctx, 5*time.Minute)

	if cfg.EnableAdminAPI {
		adminHandler := admin.New(cfg, lg, auditLogger, db, svc)
		// 确保 AdminHandler 能够访问原始的 AllowedKeys 用于数据库初始化
		adminHandler.SetOriginalAllowedKeys(originalAllowedKeys)
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

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		lg.Error("server forced shutdown: " + err.Error())
	}

	lg.Info("server exited", nil)
}
