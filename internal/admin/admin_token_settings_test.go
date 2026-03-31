package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

func TestIsValidAdminTokenPrefersDatabaseSetting(t *testing.T) {
	db := storage.NewForTest(t)
	defer db.Close()

	if err := db.SetAdminSetting(adminTokenSettingKey, hashAdminToken("db-token-123456")); err != nil {
		t.Fatalf("set admin token failed: %v", err)
	}

	handler := &AdminHandler{
		cfg: &config.Config{AdminToken: "env-token-default"},
		db:  db,
	}

	if !handler.isValidAdminToken("db-token-123456") {
		t.Fatal("expected database token to authenticate")
	}
	if handler.isValidAdminToken("env-token-default") {
		t.Fatal("did not expect env token to authenticate when database token exists")
	}
}

func TestHandleAdminTokenSettingsUpdateAndReadback(t *testing.T) {
	db := storage.NewForTest(t)
	defer db.Close()

	handler := &AdminHandler{
		cfg:         &config.Config{AdminToken: "env-token-default", AuditLogDir: t.TempDir()},
		db:          db,
		auditLogger: audit.New(t.TempDir(), false),
	}

	reqBody, _ := json.Marshal(map[string]string{
		"new_token":     "new-admin-token-123",
		"confirm_token": "new-admin-token-123",
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/settings/admin-token", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	handler.handleAdminTokenSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !handler.isValidAdminToken("new-admin-token-123") {
		t.Fatal("expected updated admin token to authenticate")
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/admin/settings/admin-token", nil)
	rec = httptest.NewRecorder()
	handler.handleAdminTokenSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Data struct {
			Source     string `json:"source"`
			Configured bool   `json:"configured"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload.Data.Source != "database" {
		t.Fatalf("expected source database, got %s", payload.Data.Source)
	}
	if !payload.Data.Configured {
		t.Fatal("expected configured=true after update")
	}
}

func TestHandleAdminTokenSettingsRejectsInvalidToken(t *testing.T) {
	db := storage.NewForTest(t)
	defer db.Close()

	handler := &AdminHandler{
		cfg:         &config.Config{AdminToken: "env-token-default", AuditLogDir: t.TempDir()},
		db:          db,
		auditLogger: audit.New(t.TempDir(), false),
	}

	reqBody, _ := json.Marshal(map[string]string{
		"new_token":     "short",
		"confirm_token": "short",
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/settings/admin-token", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	handler.handleAdminTokenSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}
