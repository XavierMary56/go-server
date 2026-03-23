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
)

// Handler HTTP 处理器
type Handler struct {
	svc   *service.ModerationService
	log   *logger.Logger
	cfg   *config.Config
	tasks sync.Map // 存储异步任务结果
}

// New 创建 Handler 实例
func New(svc *service.ModerationService, log *logger.Logger, cfg *config.Config) *Handler {
	return &Handler{svc: svc, log: log, cfg: cfg}
}

// RegisterRoutes 注册所有路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/moderate", h.withMiddleware(h.handleModerate))
	mux.HandleFunc("/v1/moderate/async", h.withMiddleware(h.handleModerateAsync))
	mux.HandleFunc("/v1/task/", h.withMiddleware(h.handleTaskQuery))
	mux.HandleFunc("/v1/models", h.withMiddleware(h.handleModels))
	mux.HandleFunc("/v1/stats", h.withMiddleware(h.handleStats))
	mux.HandleFunc("/v1/health", h.handleHealth)
}

// ── 中间件 ─────────────────────────────────────────────────

func (h *Handler) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Project-Key")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 鉴权
		if h.cfg.EnableAuth {
			key := r.Header.Get("X-Project-Key")
			if !h.isValidKey(key) {
				h.jsonError(w, http.StatusUnauthorized, "无效的项目密钥")
				return
			}
		}

		next(w, r)
	}
}

func (h *Handler) isValidKey(key string) bool {
	for _, k := range h.cfg.AllowedKeys {
		if k == key {
			return true
		}
	}
	return false
}

// ── 路由处理器 ─────────────────────────────────────────────

// POST /v1/moderate  同步审核
func (h *Handler) handleModerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, http.StatusMethodNotAllowed, "仅支持 POST 请求")
		return
	}

	var req service.ModerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		h.jsonError(w, http.StatusBadRequest, "content 不能为空")
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

// POST /v1/moderate/async  异步审核
func (h *Handler) handleModerateAsync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, http.StatusMethodNotAllowed, "仅支持 POST 请求")
		return
	}

	var req service.ModerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		h.jsonError(w, http.StatusBadRequest, "content 不能为空")
		return
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	h.tasks.Store(taskID, map[string]interface{}{"status": "pending", "created_at": time.Now().Unix()})

	// 异步执行
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

		// Webhook 回调
		if req.WebhookURL != "" {
			go triggerWebhook(req.WebhookURL, taskData)
		}
	}()

	h.jsonOK(w, http.StatusAccepted, map[string]interface{}{
		"code":    202,
		"task_id": taskID,
		"message": "任务已接受，请通过 task_id 查询结果或等待 Webhook 回调",
	})
}

// GET /v1/task/{id}  查询异步任务
func (h *Handler) handleTaskQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, http.StatusMethodNotAllowed, "仅支持 GET 请求")
		return
	}

	re := regexp.MustCompile(`^/v1/task/(.+)$`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		h.jsonError(w, http.StatusBadRequest, "缺少 task_id")
		return
	}
	taskID := matches[1]

	val, ok := h.tasks.Load(taskID)
	if !ok {
		h.jsonError(w, http.StatusNotFound, "任务不存在: "+taskID)
		return
	}
	h.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": val})
}

// GET /v1/models  查询模型列表
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

// GET /v1/stats  统计数据
func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	h.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": h.svc.GetStats()})
}

// GET /v1/health  健康检查
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	h.jsonOK(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": "2.0.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// ── 工具函数 ───────────────────────────────────────────────

func (h *Handler) jsonOK(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) jsonError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": status, "error": msg})
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
