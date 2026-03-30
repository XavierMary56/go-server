package apiv2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

type Handler struct {
	svc   *service.ModerationService
	log   *logger.Logger
	cfg   *config.Config
	db    *storage.DB
	audit *audit.AuditLogger
	tasks *syncTaskStore
}

type syncTaskStore struct {
	data syncMap
}

type syncMap interface {
	Load(key any) (value any, ok bool)
	Store(key, value any)
}

type moderationResult struct {
	Verdict    string  `json:"verdict"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	ModelUsed  string  `json:"model_used"`
	LatencyMs  int64   `json:"latency_ms"`
	FromCache  bool    `json:"from_cache"`
}

func New(svc *service.ModerationService, log *logger.Logger, cfg *config.Config, db *storage.DB, auditLogger *audit.AuditLogger, tasks syncMap) *Handler {
	return &Handler{
		svc:   svc,
		log:   log,
		cfg:   cfg,
		db:    db,
		audit: auditLogger,
		tasks: &syncTaskStore{data: tasks},
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/v2/moderations", middleware(h.handleModeration))
	mux.HandleFunc("/v2/moderations/async", middleware(h.handleModerationAsync))
	mux.HandleFunc("/v2/tasks/", middleware(h.handleTaskQuery))
	mux.HandleFunc("/v2/models", middleware(h.handleModels))
	mux.HandleFunc("/v2/health", h.handleHealth)
}

func (h *Handler) handleModeration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, http.StatusMethodNotAllowed, "only POST is supported")
		return
	}

	var req service.ModerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		h.jsonError(w, http.StatusBadRequest, "content cannot be empty")
		return
	}

	result := h.svc.Moderate(&req)
	responseID := fmt.Sprintf("mod_%d", time.Now().UnixNano())
	h.jsonOK(w, http.StatusOK, map[string]any{
		"code":    200,
		"message": "ok",
		"data": map[string]any{
			"id":     responseID,
			"status": "completed",
			"result": moderationResult{
				Verdict:    result.Verdict,
				Category:   result.Category,
				Confidence: result.Confidence,
				Reason:     result.Reason,
				ModelUsed:  result.ModelUsed,
				LatencyMs:  result.LatencyMs,
				FromCache:  result.FromCache,
			},
		},
	})
}

func (h *Handler) handleModerationAsync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, http.StatusMethodNotAllowed, "only POST is supported")
		return
	}

	var req service.ModerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		h.jsonError(w, http.StatusBadRequest, "content cannot be empty")
		return
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	h.tasks.data.Store(taskID, map[string]any{
		"task_id":    taskID,
		"status":     "pending",
		"created_at": time.Now().Unix(),
	})

	go func() {
		result := h.svc.Moderate(&req)
		taskData := map[string]any{
			"task_id": taskID,
			"status":  "done",
			"result": moderationResult{
				Verdict:    result.Verdict,
				Category:   result.Category,
				Confidence: result.Confidence,
				Reason:     result.Reason,
				ModelUsed:  result.ModelUsed,
				LatencyMs:  result.LatencyMs,
				FromCache:  result.FromCache,
			},
		}
		h.tasks.data.Store(taskID, taskData)
		if req.WebhookURL != "" {
			go triggerWebhook(req.WebhookURL, taskData)
		}
	}()

	h.jsonOK(w, http.StatusAccepted, map[string]any{
		"code":    202,
		"message": "accepted",
		"data": map[string]any{
			"task_id": taskID,
			"status":  "pending",
		},
	})
}

func (h *Handler) handleTaskQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, http.StatusMethodNotAllowed, "only GET is supported")
		return
	}

	prefix := "/v2/tasks/"
	if !strings.HasPrefix(r.URL.Path, prefix) || len(r.URL.Path) <= len(prefix) {
		h.jsonError(w, http.StatusBadRequest, "missing task_id")
		return
	}
	taskID := strings.TrimPrefix(r.URL.Path, prefix)

	val, ok := h.tasks.data.Load(taskID)
	if !ok {
		h.jsonError(w, http.StatusNotFound, "task not found: "+taskID)
		return
	}

	h.jsonOK(w, http.StatusOK, map[string]any{
		"code":    200,
		"message": "ok",
		"data":    val,
	})
}

func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	models := h.svc.GetModels()
	list := make([]map[string]any, 0, len(models))
	for _, m := range models {
		list = append(list, map[string]any{
			"id":       m.ID,
			"name":     m.Name,
			"weight":   m.Weight,
			"priority": m.Priority,
			"status":   "active",
		})
	}
	h.jsonOK(w, http.StatusOK, map[string]any{
		"code":    200,
		"message": "ok",
		"data": map[string]any{
			"models": list,
		},
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	h.jsonOK(w, http.StatusOK, map[string]any{
		"code":    200,
		"message": "ok",
		"data": map[string]any{
			"status":  "ok",
			"version": "2.0.0",
			"time":    time.Now().Format(time.RFC3339),
		},
	})
}

func (h *Handler) jsonOK(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) jsonError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    status,
		"message": msg,
		"error":   msg,
	})
}

func triggerWebhook(webhookURL string, data map[string]any) {
	body, _ := json.Marshal(data)
	_, _ = http.Post(webhookURL, "application/json", strings.NewReader(string(body)))
}
