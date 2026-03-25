package admin

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/config"
)

func TestHandleProjectStatsReturnsAllProjectsWhenQueryIsMissing(t *testing.T) {
	auditDir := t.TempDir()
	projectID := "project-a"
	projectDir := filepath.Join(auditDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	filename := filepath.Join(projectDir, "audit_"+time.Now().Format("2006-01-02")+".log")
	if err := os.WriteFile(filename, []byte(`{"timestamp":"2026-03-25T12:00:00Z","event_type":"api_call","project_id":"project-a","status_code":200}`+"\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	handler := &AdminHandler{
		cfg: &config.Config{AuditLogDir: auditDir},
		keys: map[string]*KeyInfo{
			"key-a": {ProjectID: projectID},
			"key-b": {ProjectID: "project-b"},
		},
	}

	req := httptest.NewRequest("GET", "/v1/admin/projects/stats", nil)
	rec := httptest.NewRecorder()
	handler.handleProjectStats(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Code int                       `json:"code"`
		Data map[string]map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if payload.Code != 200 {
		t.Fatalf("expected code 200, got %d", payload.Code)
	}
	if _, ok := payload.Data["project-a"]; !ok {
		t.Fatal("expected project-a stats in aggregated response")
	}
	if _, ok := payload.Data["project-b"]; !ok {
		t.Fatal("expected configured project-b zero stats in aggregated response")
	}
}
