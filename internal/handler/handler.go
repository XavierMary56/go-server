package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

// Handler handles all HTTP requests.
type Handler struct {
	svc   *service.ModerationService
	log   *logger.Logger
	cfg   *config.Config
	db    *storage.DB
	tasks sync.Map
	usage sync.Map // key -> *rateCounter
}

type rateCounter struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
}

// New creates a new handler instance.
func New(svc *service.ModerationService, log *logger.Logger, cfg *config.Config, db *storage.DB) *Handler {
	return &Handler{
		svc: svc,
		log: log,
		cfg: cfg,
		db:  db,
	}
}

// RegisterRoutes registers all public API routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/moderate", h.withMiddleware(h.handleModerate))
	mux.HandleFunc("/v1/moderate/async", h.withMiddleware(h.handleModerateAsync))
	mux.HandleFunc("/v1/task/", h.withMiddleware(h.handleTaskQuery))
	mux.HandleFunc("/v1/models", h.withMiddleware(h.handleModels))
	mux.HandleFunc("/v1/stats", h.withMiddleware(h.handleStats))
	mux.HandleFunc("/v1/health", h.handleHealth)
}

func (h *Handler) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Project-Key")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if h.cfg.EnableAuth {
			projectKey, err := h.validateProjectKey(r.Header.Get("X-Project-Key"))
			if err != nil {
				h.jsonError(w, http.StatusUnauthorized, "invalid project key")
				return
			}
			if err := h.checkRateLimit(projectKey); err != nil {
				h.jsonError(w, http.StatusTooManyRequests, err.Error())
				return
			}
		}

		next(w, r)
	}
}

func (h *Handler) validateProjectKey(key string) (*storage.ProjectKey, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("empty project key")
	}

	if h.db != nil {
		projectKey, err := h.db.GetEnabledProjectKey(key)
		if err != nil {
			h.log.Error("database project key lookup failed: " + err.Error())
			return nil, err
		}
		if projectKey != nil {
			return projectKey, nil
		}
	}

	for _, allowed := range h.cfg.AllowedKeys {
		if strings.TrimSpace(allowed) == key {
			return &storage.ProjectKey{
				ProjectID: key,
				Key:       key,
				Enabled:   true,
			}, nil
		}
	}

	return nil, fmt.Errorf("project key not found")
}

func (h *Handler) checkRateLimit(projectKey *storage.ProjectKey) error {
	if projectKey == nil || projectKey.RateLimit <= 0 {
		return nil
	}

	value, _ := h.usage.LoadOrStore(projectKey.Key, &rateCounter{
		resetAt: time.Now().Add(time.Minute),
	})
	counter := value.(*rateCounter)

	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	if now.After(counter.resetAt) {
		counter.count = 0
		counter.resetAt = now.Add(time.Minute)
	}

	if counter.count >= projectKey.RateLimit {
		return fmt.Errorf("rate limit exceeded for project %s", projectKey.ProjectID)
	}

	counter.count++
	return nil
}

// POST /v1/moderate
func (h *Handler) handleModerate(w http.ResponseWriter, r *http.Request) {
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
	h.jsonOK(w, http.StatusOK, map[string]interface{}{
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

// POST /v1/moderate/async
func (h *Handler) handleModerateAsync(w http.ResponseWriter, r *http.Request) {
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
	h.tasks.Store(taskID, map[string]interface{}{
		"status":     "pending",
		"created_at": time.Now().Unix(),
	})

	go func() {
		result := h.svc.Moderate(&req)
		taskData := map[string]interface{}{
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
			go triggerWebhook(req.WebhookURL, taskData)
		}
	}()

	h.jsonOK(w, http.StatusAccepted, map[string]interface{}{
		"code":    202,
		"task_id": taskID,
		"message": "task accepted",
	})
}

// GET /v1/task/{id}
func (h *Handler) handleTaskQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, http.StatusMethodNotAllowed, "only GET is supported")
		return
	}

	re := regexp.MustCompile(`^/v1/task/(.+)$`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		h.jsonError(w, http.StatusBadRequest, "missing task_id")
		return
	}
	taskID := matches[1]

	val, ok := h.tasks.Load(taskID)
	if !ok {
		h.jsonError(w, http.StatusNotFound, "task not found: "+taskID)
		return
	}
	h.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": val})
}

// GET /v1/models
func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	models := h.svc.GetModels()
	list := make([]map[string]interface{}, 0, len(models))
	for _, m := range models {
		list = append(list, map[string]interface{}{
			"id":       m.ID,
			"name":     m.Name,
			"weight":   m.Weight,
			"priority": m.Priority,
			"status":   "active",
		})
	}
	h.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "models": list})
}

// GET /v1/stats
func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	h.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": h.svc.GetStats()})
}

// GET /v1/health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	h.jsonOK(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": "2.0.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) jsonOK(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) jsonError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": status, "error": msg})
}

func triggerWebhook(url string, data map[string]interface{}) {
	body, _ := json.Marshal(data)
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
