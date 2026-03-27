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
	if err := os.MkdirAll(filepath.Join(auditDir, "unknown"), 0o755); err != nil {
		t.Fatalf("mkdir unknown failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(auditDir, "log-only-project"), 0o755); err != nil {
		t.Fatalf("mkdir log-only-project failed: %v", err)
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
	if _, ok := payload.Data["log-only-project"]; !ok {
		t.Fatal("expected log-only project stats in aggregated response")
	}
	if _, ok := payload.Data["unknown"]; ok {
		t.Fatal("did not expect unknown audit bucket in aggregated project response")
	}
}

func TestHandleListProjectsReturnsSortedProjectsWithoutUnknown(t *testing.T) {
	auditDir := t.TempDir()

	for _, projectID := range []string{"project-b", "project-a", "log-only-project", "unknown"} {
		projectDir := filepath.Join(auditDir, projectID)
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir %s failed: %v", projectID, err)
		}
	}

	handler := &AdminHandler{
		cfg: &config.Config{AuditLogDir: auditDir},
		keys: map[string]*KeyInfo{
			"key-b": {ProjectID: "project-b"},
			"key-c": {ProjectID: "project-c"},
		},
	}

	req := httptest.NewRequest("GET", "/v1/admin/projects", nil)
	rec := httptest.NewRecorder()
	handler.handleListProjects(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Code int `json:"code"`
		Data struct {
			TotalProjects int              `json:"total_projects"`
			Projects      []map[string]any `json:"projects"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if payload.Code != 200 {
		t.Fatalf("expected code 200, got %d", payload.Code)
	}
	if payload.Data.TotalProjects != 4 {
		t.Fatalf("expected 4 projects, got %d", payload.Data.TotalProjects)
	}

	gotOrder := make([]string, 0, len(payload.Data.Projects))
	seenLogOnly := false
	for _, project := range payload.Data.Projects {
		projectID, _ := project["project_id"].(string)
		if projectID == "unknown" {
			t.Fatal("did not expect unknown audit bucket in project list response")
		}
		if projectID == "log-only-project" {
			seenLogOnly = true
			continue
		}
		gotOrder = append(gotOrder, projectID)
	}

	if !seenLogOnly {
		t.Fatal("expected log-only-project to be present in project list response")
	}

	// The response now includes projects discovered from the audit log directory.
	// We ignore log-only projects when validating configured project order.
	wantOrder := []string{"project-b", "project-c"}
	configured := make([]string, 0, len(gotOrder))
	for _, projectID := range gotOrder {
		if projectID == "project-a" {
			continue
		}
		configured = append(configured, projectID)
	}

	if len(configured) != len(wantOrder) {
		t.Fatalf("expected %d configured projects in response, got %d", len(wantOrder), len(configured))
	}
	for i := range wantOrder {
		if configured[i] != wantOrder[i] {
			t.Fatalf("expected configured project order %v, got %v", wantOrder, configured)
		}
	}
}
