package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
)

func TestV2ModerationsReturnsStructuredResponse(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:  60,
		APITimeout: 1,
		LogDir:    t.TempDir(),
		LogLevel:  "error",
	}
	lg := logger.New(t.TempDir(), "error")
	svc := service.NewModerationService(cfg, lg, nil)
	h := New(svc, lg, cfg, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/v2/moderations", strings.NewReader(`{"content":"ordinary discussion"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Result struct {
				Verdict   string `json:"verdict"`
				Category  string `json:"category"`
				ModelUsed string `json:"model_used"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload.Code != 200 {
		t.Fatalf("expected code 200, got %d", payload.Code)
	}
	if payload.Message != "ok" {
		t.Fatalf("expected message ok, got %q", payload.Message)
	}
	if payload.Data.ID == "" {
		t.Fatal("expected moderation id to be set")
	}
	if payload.Data.Status != "completed" {
		t.Fatalf("expected status completed, got %q", payload.Data.Status)
	}
	if payload.Data.Result.Verdict == "" {
		t.Fatal("expected result verdict to be present")
	}
}

func TestV2TaskQueryWrapsAsyncResultInData(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:  60,
		APITimeout: 1,
		LogDir:    t.TempDir(),
		LogLevel:  "error",
	}
	lg := logger.New(t.TempDir(), "error")
	svc := service.NewModerationService(cfg, lg, nil)
	h := New(svc, lg, cfg, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	createReq := httptest.NewRequest(http.MethodPost, "/v2/moderations/async", strings.NewReader(`{"content":"ordinary discussion"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", createRec.Code)
	}

	var createPayload struct {
		Data struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("unmarshal create response failed: %v", err)
	}
	if createPayload.Data.TaskID == "" {
		t.Fatal("expected task_id in async response")
	}

	queryReq := httptest.NewRequest(http.MethodGet, "/v2/tasks/"+createPayload.Data.TaskID, nil)
	queryRec := httptest.NewRecorder()
	mux.ServeHTTP(queryRec, queryReq)

	if queryRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", queryRec.Code)
	}

	var queryPayload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(queryRec.Body.Bytes(), &queryPayload); err != nil {
		t.Fatalf("unmarshal query response failed: %v", err)
	}
	if queryPayload.Code != 200 {
		t.Fatalf("expected code 200, got %d", queryPayload.Code)
	}
	if queryPayload.Message != "ok" {
		t.Fatalf("expected message ok, got %q", queryPayload.Message)
	}
	if queryPayload.Data.TaskID != createPayload.Data.TaskID {
		t.Fatalf("expected task_id %q, got %q", createPayload.Data.TaskID, queryPayload.Data.TaskID)
	}
	if queryPayload.Data.Status == "" {
		t.Fatal("expected task status to be present")
	}
}
