package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	api "github.com/XavierMary56/automatic_review/go-server/internal/api"
	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

type syncMap interface {
	Load(key any) (value any, ok bool)
	Store(key, value any)
}

type Handler struct {
	svc   *service.ModerationService
	log   *logger.Logger
	cfg   *config.Config
	db    *storage.DB
	audit *audit.AuditLogger
	tasks syncMap
}

func New(svc *service.ModerationService, log *logger.Logger, cfg *config.Config, db *storage.DB, auditLogger *audit.AuditLogger, tasks syncMap) *Handler {
	return &Handler{svc: svc, log: log, cfg: cfg, db: db, audit: auditLogger, tasks: tasks}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/v1/moderate", middleware(h.handleModerate))
	mux.HandleFunc("/v1/moderate/async", middleware(h.handleModerateAsync))
	mux.HandleFunc("/v1/task/", middleware(h.handleTaskQuery))
	mux.HandleFunc("/v1/models", middleware(h.handleModels))
	mux.HandleFunc("/v1/stats", middleware(h.handleStats))
	mux.HandleFunc("/v1/health", h.handleHealth)
}

func (h *Handler) handleModerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.JSONError(w, http.StatusMethodNotAllowed, "only POST is supported")
		return
	}

	var req service.ModerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		api.JSONError(w, http.StatusBadRequest, "content cannot be empty")
		return
	}

	result := h.svc.Moderate(&req)
	api.JSONOK(w, http.StatusOK, map[string]any{
		"code":       200,
		"verdict":    result.Verdict,
		"category":   result.Category,
		"confidence": result.Confidence,
		"reason":     result.Reason,
		"model_used": result.ModelUsed,
		"latency_ms": result.LatencyMs,
		"from_cache": result.FromCache,
	})
}

func (h *Handler) handleModerateAsync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.JSONError(w, http.StatusMethodNotAllowed, "only POST is supported")
		return
	}

	var req service.ModerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		api.JSONError(w, http.StatusBadRequest, "content cannot be empty")
		return
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	h.tasks.Store(taskID, map[string]any{
		"status":     "pending",
		"created_at": time.Now().Unix(),
	})

	go func() {
		result := h.svc.Moderate(&req)
		taskData := map[string]any{
			"status":     "done",
			"verdict":    result.Verdict,
			"category":   result.Category,
			"confidence": result.Confidence,
			"reason":     result.Reason,
			"model_used": result.ModelUsed,
			"latency_ms": result.LatencyMs,
			"task_id":    taskID,
		}
		h.tasks.Store(taskID, taskData)
		if req.WebhookURL != "" {
			go api.TriggerWebhook(req.WebhookURL, taskData)
		}
	}()

	api.JSONOK(w, http.StatusAccepted, map[string]any{
		"code":    202,
		"task_id": taskID,
		"message": "task accepted",
	})
}

func (h *Handler) handleTaskQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.JSONError(w, http.StatusMethodNotAllowed, "only GET is supported")
		return
	}

	re := regexp.MustCompile(`^/v1/task/(.+)$`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		api.JSONError(w, http.StatusBadRequest, "missing task_id")
		return
	}
	taskID := matches[1]

	val, ok := h.tasks.Load(taskID)
	if !ok {
		api.JSONError(w, http.StatusNotFound, "task not found: "+taskID)
		return
	}
	api.JSONOK(w, http.StatusOK, map[string]any{"code": 200, "data": val})
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
	api.JSONOK(w, http.StatusOK, map[string]any{"code": 200, "models": list})
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	api.JSONOK(w, http.StatusOK, map[string]any{"code": 200, "data": h.svc.GetStats()})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	api.JSONOK(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "2.0.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}
