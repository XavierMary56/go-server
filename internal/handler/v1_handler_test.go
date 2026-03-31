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

func TestV1ModerateReturnsLegacyResponseShape(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:   60,
		APITimeout: 1,
		LogDir:     t.TempDir(),
		LogLevel:   "error",
	}
	lg := logger.New(t.TempDir(), "error")
	svc := service.NewModerationService(cfg, lg, nil)
	h := New(svc, lg, cfg, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/moderate", strings.NewReader(`{"content":"ordinary discussion"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Code       int     `json:"code"`
		Verdict    string  `json:"verdict"`
		Category   string  `json:"category"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
		ModelUsed  string  `json:"model_used"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload.Code != 200 {
		t.Fatalf("expected code 200, got %d", payload.Code)
	}
	if payload.Verdict == "" {
		t.Fatal("expected verdict to be present")
	}
	if payload.ModelUsed == "" {
		t.Fatal("expected model_used to be present")
	}
}

func TestV1TaskQueryReturnsLegacyTaskPayload(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:   60,
		APITimeout: 1,
		LogDir:     t.TempDir(),
		LogLevel:   "error",
	}
	lg := logger.New(t.TempDir(), "error")
	svc := service.NewModerationService(cfg, lg, nil)
	h := New(svc, lg, cfg, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	createReq := httptest.NewRequest(http.MethodPost, "/v1/moderate/async", strings.NewReader(`{"content":"ordinary discussion"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", createRec.Code)
	}

	var createPayload struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("unmarshal create response failed: %v", err)
	}
	if createPayload.TaskID == "" {
		t.Fatal("expected task_id in async response")
	}

	queryReq := httptest.NewRequest(http.MethodGet, "/v1/task/"+createPayload.TaskID, nil)
	queryRec := httptest.NewRecorder()
	mux.ServeHTTP(queryRec, queryReq)

	if queryRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", queryRec.Code)
	}

	var queryPayload struct {
		Code int `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(queryRec.Body.Bytes(), &queryPayload); err != nil {
		t.Fatalf("unmarshal query response failed: %v", err)
	}
	if queryPayload.Code != 200 {
		t.Fatalf("expected code 200, got %d", queryPayload.Code)
	}
	status, _ := queryPayload.Data["status"].(string)
	if status == "" {
		t.Fatal("expected task status to be present")
	}
	if status == "done" {
		if gotTaskID, _ := queryPayload.Data["task_id"].(string); gotTaskID != createPayload.TaskID {
			t.Fatalf("expected task_id %q, got %q", createPayload.TaskID, gotTaskID)
		}
	}
}
