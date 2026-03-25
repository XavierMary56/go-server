package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

func TestAuthenticatedRequestWritesProjectAuditStats(t *testing.T) {
	dataDir := t.TempDir()
	auditDir := t.TempDir()
	logDir := t.TempDir()

	db, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	if _, err := db.AddProjectKey("project-a", "key-a", 60); err != nil {
		t.Fatalf("add project key failed: %v", err)
	}

	cfg := &config.Config{
		EnableAuth:  true,
		EnableAudit: true,
		CacheTTL:    60,
		APITimeout:  1,
		AuditLogDir: auditDir,
		LogDir:      logDir,
		LogLevel:    "error",
	}
	lg := logger.New(logDir, "error")
	svc := service.NewModerationService(cfg, lg, db)
	auditLogger := audit.New(auditDir, true)
	defer auditLogger.Close()

	h := New(svc, lg, cfg, db, auditLogger)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/moderate", strings.NewReader(`{"content":"ordinary discussion"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Project-Key", "key-a")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	time.Sleep(150 * time.Millisecond)

	stats, err := audit.GetProjectStats(auditDir, "project-a")
	if err != nil {
		t.Fatalf("get project stats failed: %v", err)
	}
	if got := stats["auth_attempts"]; got != 1 {
		t.Fatalf("expected 1 auth_attempt, got %v", got)
	}
	if got := stats["api_calls"]; got != 1 {
		t.Fatalf("expected 1 api_call, got %v", got)
	}
}
