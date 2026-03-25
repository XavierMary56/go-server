package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetProjectStatsReturnsZeroSummaryForMissingProjectLogDir(t *testing.T) {
	baseDir := t.TempDir()

	stats, err := GetProjectStats(baseDir, "project-a")
	if err != nil {
		t.Fatalf("expected no error for missing log dir, got %v", err)
	}

	if got := stats["api_calls"]; got != 0 {
		t.Fatalf("expected zero api_calls, got %v", got)
	}
	if got := stats["auth_attempts"]; got != 0 {
		t.Fatalf("expected zero auth_attempts, got %v", got)
	}
	if got := stats["rate_limited"]; got != 0 {
		t.Fatalf("expected zero rate_limited, got %v", got)
	}
	if got := stats["errors"]; got != 0 {
		t.Fatalf("expected zero errors, got %v", got)
	}
}

func TestGetProjectStatsBuildsFrontendFriendlyCounters(t *testing.T) {
	baseDir := t.TempDir()
	projectID := "project-a"
	projectDir := filepath.Join(baseDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	events := []AuditEvent{
		{Timestamp: time.Now(), EventType: "auth_attempt", ProjectID: projectID, StatusCode: 200},
		{Timestamp: time.Now(), EventType: "api_call", ProjectID: projectID, StatusCode: 200},
		{Timestamp: time.Now(), EventType: "api_call", ProjectID: projectID, StatusCode: 500, ErrorMsg: "boom"},
		{Timestamp: time.Now(), EventType: "rate_limit_exceeded", ProjectID: projectID, StatusCode: 429},
	}

	filename := filepath.Join(projectDir, "audit_"+time.Now().Format("2006-01-02")+".log")
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	defer f.Close()

	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	stats, err := GetProjectStats(baseDir, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := stats["api_calls"]; got != 2 {
		t.Fatalf("expected 2 api_calls, got %v", got)
	}
	if got := stats["auth_attempts"]; got != 1 {
		t.Fatalf("expected 1 auth_attempt, got %v", got)
	}
	if got := stats["rate_limited"]; got != 1 {
		t.Fatalf("expected 1 rate_limited, got %v", got)
	}
	if got := stats["errors"]; got != 2 {
		t.Fatalf("expected 2 errors (500 api call + 429 limit), got %v", got)
	}
}
