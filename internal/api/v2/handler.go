package apiv2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	api "github.com/XavierMary56/automatic_review/go-server/internal/api"
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
	responseID := fmt.Sprintf("mod_%d", time.Now().UnixNano())

	h.logModerationEvent(r, &req, result, responseID)

	api.JSONOK(w, http.StatusOK, map[string]any{
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
	h.tasks.data.Store(taskID, map[string]any{
		"task_id":    taskID,
		"status":     "pending",
		"created_at": time.Now().Unix(),
	})

	projectName := r.Header.Get("X-Project-Name")
	clientIP := getClientIP(r)

	go func() {
		result := h.svc.Moderate(&req)
		h.logModerationEventDirect(projectName, r.Method, r.URL.Path, clientIP, &req, result, taskID)
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
			go api.TriggerWebhook(req.WebhookURL, taskData)
		}
	}()

	api.JSONOK(w, http.StatusAccepted, map[string]any{
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
		api.JSONError(w, http.StatusMethodNotAllowed, "only GET is supported")
		return
	}

	prefix := "/v2/tasks/"
	if !strings.HasPrefix(r.URL.Path, prefix) || len(r.URL.Path) <= len(prefix) {
		api.JSONError(w, http.StatusBadRequest, "missing task_id")
		return
	}
	taskID := strings.TrimPrefix(r.URL.Path, prefix)

	val, ok := h.tasks.data.Load(taskID)
	if !ok {
		api.JSONError(w, http.StatusNotFound, "task not found: "+taskID)
		return
	}

	api.JSONOK(w, http.StatusOK, map[string]any{
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
	api.JSONOK(w, http.StatusOK, map[string]any{
		"code":    200,
		"message": "ok",
		"data": map[string]any{
			"models": list,
		},
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	api.JSONOK(w, http.StatusOK, map[string]any{
		"code":    200,
		"message": "ok",
		"data": map[string]any{
			"status":  "ok",
			"version": "2.0.0",
			"time":    time.Now().Format(time.RFC3339),
		},
	})
}

// logModerationEvent 记录同步审核请求的完整日志
func (h *Handler) logModerationEvent(r *http.Request, req *service.ModerateRequest, result *service.ModerateResult, responseID string) {
	if h.audit == nil {
		return
	}
	h.audit.LogEvent(&audit.AuditEvent{
		Timestamp:   time.Now(),
		EventType:   "moderation_request",
		ProjectName: r.Header.Get("X-Project-Name"),
		Method:      r.Method,
		Path:        r.URL.Path,
		IPAddress:   getClientIP(r),
		RequestBody: map[string]interface{}{
			"content":    req.Content,
			"type":       req.Type,
			"model":      req.Model,
			"strictness": req.Strictness,
		},
		Metadata: map[string]interface{}{
			"response_id": responseID,
			"verdict":     result.Verdict,
			"category":    result.Category,
			"confidence":  result.Confidence,
			"reason":      result.Reason,
			"model_used":  result.ModelUsed,
			"latency_ms":  result.LatencyMs,
			"from_cache":  result.FromCache,
		},
	})
}

// logModerationEventDirect 记录异步审核请求的日志（goroutine 内调用，不能依赖 http.Request）
func (h *Handler) logModerationEventDirect(projectName, method, path, clientIP string, req *service.ModerateRequest, result *service.ModerateResult, taskID string) {
	if h.audit == nil {
		return
	}
	h.audit.LogEvent(&audit.AuditEvent{
		Timestamp:   time.Now(),
		EventType:   "moderation_request",
		ProjectName: projectName,
		Method:      method,
		Path:        path,
		IPAddress:   clientIP,
		RequestBody: map[string]interface{}{
			"content":    req.Content,
			"type":       req.Type,
			"model":      req.Model,
			"strictness": req.Strictness,
		},
		Metadata: map[string]interface{}{
			"task_id":    taskID,
			"verdict":    result.Verdict,
			"category":   result.Category,
			"confidence": result.Confidence,
			"reason":     result.Reason,
			"model_used": result.ModelUsed,
			"latency_ms": result.LatencyMs,
			"from_cache": result.FromCache,
		},
	})
}

// getClientIP 从请求中提取客户端 IP
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}
