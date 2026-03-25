package admin

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
)

func TestHandleProjectKeyDetailAllowsZeroRateLimit(t *testing.T) {
	handler := &AdminHandler{
		cfg:         &config.Config{AuditLogDir: t.TempDir()},
		auditLogger: audit.New(t.TempDir(), false),
		keys: map[string]*KeyInfo{
			"key-a": {
				ProjectID: "project-a",
				Key:       "key-a",
				RateLimit: 10,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Enabled:   true,
			},
		},
	}

	reqBody, err := json.Marshal(map[string]any{
		"project_id": "project-a",
		"key":        "key-a",
		"rate_limit": 0,
		"enabled":    true,
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	req := httptest.NewRequest("PUT", "/v1/admin/keys/key-a", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	handler.handleProjectKeyDetail(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	updated := handler.keys["key-a"]
	if updated == nil {
		t.Fatal("expected updated key to remain in handler map")
	}
	if updated.RateLimit != 0 {
		t.Fatalf("expected rate limit 0, got %d", updated.RateLimit)
	}

	var payload struct {
		Code int      `json:"code"`
		Data *KeyInfo `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload.Code != 200 {
		t.Fatalf("expected code 200, got %d", payload.Code)
	}
	if payload.Data == nil || payload.Data.RateLimit != 0 {
		t.Fatalf("expected response rate limit 0, got %+v", payload.Data)
	}
}
