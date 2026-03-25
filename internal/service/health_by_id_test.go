package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

func TestCheckAnthropicKeyByIDReturnsMatchedKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	key, err := db.AddAnthropicKey("anthropic-check", "sk-ant-test")
	if err != nil {
		t.Fatalf("add anthropic key failed: %v", err)
	}

	svc := NewModerationService(&config.Config{
		AnthropicAPIURL: server.URL + "/v1/messages",
		AnthropicVer:    "2023-06-01",
	}, nil, db)

	result := svc.CheckAnthropicKeyByID(key.ID)
	if result.Provider != "anthropic" {
		t.Fatalf("expected provider anthropic, got %q", result.Provider)
	}
	if result.Name != key.Name {
		t.Fatalf("expected name %q, got %q", key.Name, result.Name)
	}
	if result.Status != "healthy" {
		t.Fatalf("expected healthy status, got %q (%s)", result.Status, result.Error)
	}
}

func TestCheckProviderKeyByIDReturnsMatchedKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	key, err := db.AddProviderKey("openai", "openai-check", "sk-openai-test")
	if err != nil {
		t.Fatalf("add provider key failed: %v", err)
	}

	svc := NewModerationService(&config.Config{
		OpenAIAPIURL: server.URL + "/v1/chat/completions",
		GrokAPIURL:   server.URL + "/v1/chat/completions",
	}, nil, db)

	result := svc.CheckProviderKeyByID(key.ID)
	if result.Provider != "openai" {
		t.Fatalf("expected provider openai, got %q", result.Provider)
	}
	if result.Name != key.Name {
		t.Fatalf("expected name %q, got %q", key.Name, result.Name)
	}
	if result.Status != "healthy" {
		t.Fatalf("expected healthy status, got %q (%s)", result.Status, result.Error)
	}
}
