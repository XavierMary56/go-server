package service

import (
	"testing"

	"github.com/XavierMary56/automatic_review/go-server/internal/config"
)

func TestGetActiveModelsFallsBackToConfigWhenDBUnavailable(t *testing.T) {
	svc := NewModerationService(&config.Config{
		CacheTTL: 60,
		Models: []config.ModelConfig{
			{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Weight: 100, Priority: 1, Provider: "anthropic"},
		},
	}, nil, nil)

	models := svc.getActiveModels()
	if len(models) != 1 {
		t.Fatalf("expected 1 fallback model, got %d", len(models))
	}
	if models[0].ID != "claude-sonnet-4-20250514" {
		t.Fatalf("expected fallback model claude-sonnet-4-20250514, got %q", models[0].ID)
	}
}

func TestApplyRequestDefaultsFillsMissingFields(t *testing.T) {
	svc := NewModerationService(&config.Config{CacheTTL: 60}, nil, nil)
	req := &ModerateRequest{
		Content: "normal discussion only",
	}

	svc.applyRequestDefaults(req)

	if req.Type != "post" {
		t.Fatalf("expected default type post, got %q", req.Type)
	}
	if req.Strictness != "standard" {
		t.Fatalf("expected default strictness standard, got %q", req.Strictness)
	}
	if req.Model != "auto" {
		t.Fatalf("expected default model auto, got %q", req.Model)
	}
}

func TestBuildAuditContentIncludesStructuredContext(t *testing.T) {
	svc := NewModerationService(&config.Config{CacheTTL: 60}, nil, nil)
	req := &ModerateRequest{
		Content: "main body",
		Context: map[string]interface{}{
			"scene": "comment_review",
			"payload": map[string]interface{}{
				"title":   "hello title",
				"content": "hello detail",
			},
		},
	}

	got := svc.buildAuditContent(req)
	want := "review body:\nmain body\n\nscene: comment_review\n\ntitle:\nhello title\n\nbody:\nhello detail"
	if got != want {
		t.Fatalf("unexpected audit content:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestModerateReturnsCloneForCachedResult(t *testing.T) {
	svc := NewModerationService(&config.Config{CacheTTL: 60}, nil, nil)
	req := &ModerateRequest{
		Content:    "ordinary review text",
		Type:       "comment",
		Strictness: "standard",
		Model:      "auto",
	}

	auditContent := svc.buildAuditContent(req)
	cacheKey := svc.cacheKey(auditContent, req.Type, req.Strictness)
	svc.cache.Set(cacheKey, &ModerateResult{
		Verdict:    "approved",
		Category:   "none",
		Confidence: 0.91,
		Reason:     "cached decision",
		ModelUsed:  "cache-seed",
	})

	first := svc.Moderate(req)
	if !first.FromCache {
		t.Fatal("expected first result to come from cache")
	}

	first.Reason = "mutated outside"
	first.Category = "spam"

	second := svc.Moderate(req)
	if !second.FromCache {
		t.Fatal("expected second result to come from cache")
	}
	if second.Reason != "cached decision" {
		t.Fatalf("expected cached clone to stay unchanged, got %q", second.Reason)
	}
	if second.Category != "none" {
		t.Fatalf("expected cached category none, got %q", second.Category)
	}
}

func TestBeginInflightSharesCallAndFinishCleansUp(t *testing.T) {
	svc := NewModerationService(&config.Config{CacheTTL: 60}, nil, nil)
	cacheKey := "same-request"

	first, leader := svc.beginInflight(cacheKey)
	if !leader {
		t.Fatal("expected first caller to be leader")
	}
	if first == nil || first.done == nil {
		t.Fatal("expected inflight call to be initialized")
	}

	second, leader := svc.beginInflight(cacheKey)
	if leader {
		t.Fatal("expected second caller to join existing inflight call")
	}
	if second != first {
		t.Fatal("expected callers with same cache key to share inflight call")
	}

	svc.finishInflight(cacheKey, first)

	third, leader := svc.beginInflight(cacheKey)
	if !leader {
		t.Fatal("expected new caller to become leader after inflight cleanup")
	}
	if third == first {
		t.Fatal("expected cleaned-up inflight call to be replaced with a new one")
	}
}
