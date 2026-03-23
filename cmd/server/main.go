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
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	lg := logger.New(cfg.LogDir, cfg.LogLevel)
	lg.Info("系统启动", map[string]interface{}{
		"version": "2.0.0",
		"port":    cfg.Port,
		"models":  len(cfg.Models),
	})

	// 初始化审核服务
	svc := service.NewModerationService(cfg, lg)

	// 注册路由
	mux := http.NewServeMux()
	h := handler.New(svc, lg, cfg)
	h.RegisterRoutes(mux)

	// 注册 Admin 路由
	if cfg.EnableAdminAPI {
		auditLogger := audit.New(cfg.AuditLogDir, cfg.EnableAudit)
		adminHandler := admin.New(cfg, lg, auditLogger)
		adminHandler.RegisterRoutes(mux)
	}

	// 启动 HTTP 服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		lg.Info(fmt.Sprintf("服务器启动，监听端口 %d", cfg.Port), nil)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lg.Error("服务器启动失败: " + err.Error())
			os.Exit(1)
		}
	}()

	<-quit
	lg.Info("正在关闭服务器...", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		lg.Error("服务器强制关闭: " + err.Error())
	}
	lg.Info("服务器已退出", nil)
}
