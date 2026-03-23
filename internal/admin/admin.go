package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
)

// AdminHandler 管理 API 处理器
type AdminHandler struct {
	cfg         *config.Config
	log         *logger.Logger
	auditLogger *audit.AuditLogger
	keysMu      sync.RWMutex
	keys        map[string]*KeyInfo // 内存中的密钥管理
}

// KeyInfo 密钥信息
type KeyInfo struct {
	ProjectID string    `json:"project_id"`
	Key       string    `json:"key"`
	RateLimit int       `json:"rate_limit"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Enabled   bool      `json:"enabled"`
}

// 请求/响应结构体

// AddKeyRequest 添加密钥请求
type AddKeyRequest struct {
	ProjectID string `json:"project_id"`
	Key       string `json:"key"`
	RateLimit int    `json:"rate_limit"` // 每分钟请求限制
}

// UpdateKeyRequest 更新密钥请求
type UpdateKeyRequest struct {
	RateLimit *int  `json:"rate_limit,omitempty"`
	Enabled   *bool `json:"enabled,omitempty"`
}

// ListKeysResponse 密钥列表响应
type ListKeysResponse struct {
	Code int                 `json:"code"`
	Data map[string]*KeyInfo `json:"data"`
}

// New 创建管理处理器
func New(cfg *config.Config, log *logger.Logger, auditLogger *audit.AuditLogger) *AdminHandler {
	handler := &AdminHandler{
		cfg:         cfg,
		log:         log,
		auditLogger: auditLogger,
		keys:        make(map[string]*KeyInfo),
	}

	// 从环境变量加载初始密钥
	handler.loadKeysFromEnv()

	return handler
}

// RegisterRoutes 注册管理路由
func (ah *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	// 所有管理端点都需要身份验证
	mux.HandleFunc("/v1/admin/keys", ah.withAdminAuth(ah.handleKeys))
	mux.HandleFunc("/v1/admin/keys/", ah.withAdminAuth(ah.handleKeyDetail))
	mux.HandleFunc("/v1/admin/health", ah.handleAdminHealth)

	// 日志和审计相关的管理端点
	mux.HandleFunc("/v1/admin/projects", ah.withAdminAuth(ah.handleListProjects))
	mux.HandleFunc("/v1/admin/projects/logs", ah.withAdminAuth(ah.handleProjectLogs))
	mux.HandleFunc("/v1/admin/projects/stats", ah.withAdminAuth(ah.handleProjectStats))

	// Web UI
	ah.registerWebUI(mux)
}

// ── 中间件 ────────────────────────────────────────

// withAdminAuth 管理员鉴权中间件
func (ah *AdminHandler) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 检查授权头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			ah.jsonError(w, http.StatusUnauthorized, "缺少 Authorization 头")
			return
		}

		// 验证 Bearer token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader || !ah.isValidAdminToken(token) {
			ah.jsonError(w, http.StatusUnauthorized, "无效的管理员令牌")
			ah.auditLogger.LogEvent(&audit.AuditEvent{
				Timestamp: time.Now(),
				EventType: "admin_auth_failed",
				ErrorMsg:  "无效的管理员令牌",
				IPAddress: ah.getClientIP(r),
				Path:      r.RequestURI,
			})
			return
		}

		next(w, r)
	}
}

// ── 路由处理器 ────────────────────────────────────────

// handleKeys 处理 GET/POST /v1/admin/keys
func (ah *AdminHandler) handleKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ah.listKeys(w, r)
	case http.MethodPost:
		ah.addKey(w, r)
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

// handleKeyDetail 处理 GET/PUT/DELETE /v1/admin/keys/{key}
func (ah *AdminHandler) handleKeyDetail(w http.ResponseWriter, r *http.Request) {
	// 提取密钥
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		ah.jsonError(w, http.StatusBadRequest, "缺少密钥")
		return
	}

	key := parts[4]
	switch r.Method {
	case http.MethodGet:
		ah.getKey(w, r, key)
	case http.MethodPut:
		ah.updateKey(w, r, key)
	case http.MethodDelete:
		ah.deleteKey(w, r, key)
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

// listKeys 列出所有密钥
func (ah *AdminHandler) listKeys(w http.ResponseWriter, r *http.Request) {
	ah.keysMu.RLock()
	defer ah.keysMu.RUnlock()

	// 复制密钥映射（隐藏敏感部分）
	result := make(map[string]*KeyInfo)
	for k, v := range ah.keys {
		maskedKey := ah.maskKey(k)
		keyInfo := *v // 复制
		result[maskedKey] = &keyInfo
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": result,
	})

	ah.auditLogger.LogEvent(&audit.AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "admin_list_keys",
		Path:       r.RequestURI,
		Method:     r.Method,
		StatusCode: http.StatusOK,
	})
}

// addKey 添加新密钥
func (ah *AdminHandler) addKey(w http.ResponseWriter, r *http.Request) {
	var req AddKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.jsonError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	// 验证请求
	if req.ProjectID == "" || req.Key == "" {
		ah.jsonError(w, http.StatusBadRequest, "项目 ID 和密钥不能为空")
		return
	}

	if req.RateLimit <= 0 {
		ah.jsonError(w, http.StatusBadRequest, "速率限制必须大于 0")
		return
	}

	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	// 检查密钥是否已存在
	if _, exists := ah.keys[req.Key]; exists {
		ah.jsonError(w, http.StatusConflict, "密钥已存在")
		return
	}

	// 添加密钥
	keyInfo := &KeyInfo{
		ProjectID: req.ProjectID,
		Key:       req.Key,
		RateLimit: req.RateLimit,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Enabled:   true,
	}

	ah.keys[req.Key] = keyInfo

	// 更新 .env 文件（可选）
	ah.updateEnvFile()

	ah.jsonOK(w, http.StatusCreated, map[string]interface{}{
		"code":    201,
		"message": "密钥已添加",
		"data":    keyInfo,
	})

	ah.auditLogger.LogEvent(&audit.AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "admin_add_key",
		ProjectID:  req.ProjectID,
		Path:       r.RequestURI,
		Method:     r.Method,
		StatusCode: http.StatusCreated,
	})
}

// getKey 获取单个密钥详情
func (ah *AdminHandler) getKey(w http.ResponseWriter, r *http.Request, key string) {
	ah.keysMu.RLock()
	defer ah.keysMu.RUnlock()

	keyInfo, exists := ah.keys[key]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "密钥不存在")
		return
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": keyInfo,
	})
}

// updateKey 更新密钥配置
func (ah *AdminHandler) updateKey(w http.ResponseWriter, r *http.Request, key string) {
	var req UpdateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.jsonError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	keyInfo, exists := ah.keys[key]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "密钥不存在")
		return
	}

	// 更新字段
	if req.RateLimit != nil {
		if *req.RateLimit <= 0 {
			ah.jsonError(w, http.StatusBadRequest, "速率限制必须大于 0")
			return
		}
		keyInfo.RateLimit = *req.RateLimit
	}

	if req.Enabled != nil {
		keyInfo.Enabled = *req.Enabled
	}

	keyInfo.UpdatedAt = time.Now()

	// 更新 .env 文件
	ah.updateEnvFile()

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "密钥已更新",
		"data":    keyInfo,
	})

	ah.auditLogger.LogEvent(&audit.AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "admin_update_key",
		ProjectID:  keyInfo.ProjectID,
		Path:       r.RequestURI,
		Method:     r.Method,
		StatusCode: http.StatusOK,
	})
}

// deleteKey 删除密钥
func (ah *AdminHandler) deleteKey(w http.ResponseWriter, r *http.Request, key string) {
	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	keyInfo, exists := ah.keys[key]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "密钥不存在")
		return
	}

	delete(ah.keys, key)

	// 更新 .env 文件
	ah.updateEnvFile()

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "密钥已删除",
		"data": map[string]interface{}{
			"key": key,
		},
	})

	ah.auditLogger.LogEvent(&audit.AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "admin_delete_key",
		ProjectID:  keyInfo.ProjectID,
		Path:       r.RequestURI,
		Method:     r.Method,
		StatusCode: http.StatusOK,
	})
}

// handleAdminHealth 管理界面健康检查（无需认证）
func (ah *AdminHandler) handleAdminHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"status":              "ok",
		"admin_api_available": true,
	})
}

// ── 工具函数 ────────────────────────────────────────

// isValidAdminToken 验证管理员令牌
func (ah *AdminHandler) isValidAdminToken(token string) bool {
	// 从配置中读取管理员令牌
	// 支持逗号分隔的多个令牌
	adminTokens := strings.Split(ah.cfg.AdminToken, ",")
	for _, t := range adminTokens {
		if strings.TrimSpace(t) == token {
			return true
		}
	}
	return false
}

// loadKeysFromEnv 从环境变量加载密钥
func (ah *AdminHandler) loadKeysFromEnv() {
	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	if ah.cfg.AllowedKeys == nil {
		return
	}

	for _, entry := range ah.cfg.AllowedKeys {
		parts := strings.Split(entry, "|")
		if len(parts) >= 3 {
			projectID := strings.TrimSpace(parts[0])
			key := strings.TrimSpace(parts[1])
			rateLimit := 0
			fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &rateLimit)

			ah.keys[key] = &KeyInfo{
				ProjectID: projectID,
				Key:       key,
				RateLimit: rateLimit,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Enabled:   true,
			}
		}
	}
}

// updateEnvFile 更新 .env 文件中的 ALLOWED_KEYS
func (ah *AdminHandler) updateEnvFile() {
	// 构建新的 ALLOWED_KEYS 字符串
	var keys []string
	for _, keyInfo := range ah.keys {
		if keyInfo.Enabled {
			entry := fmt.Sprintf("%s|%s|%d",
				keyInfo.ProjectID,
				keyInfo.Key,
				keyInfo.RateLimit,
			)
			keys = append(keys, entry)
		}
	}

	// 更新内存中的配置
	ah.cfg.AllowedKeys = keys

	// 注意：实际的 .env 文件更新需要文件操作
	// 这里只是更新内存配置，使 API 立即生效
}

// maskKey 隐藏密钥中间部分
func (ah *AdminHandler) maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// getClientIP 获取客户端 IP
func (ah *AdminHandler) getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// jsonOK 返回成功响应
func (ah *AdminHandler) jsonOK(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// jsonError 返回错误响应
func (ah *AdminHandler) jsonError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":  status,
		"error": msg,
	})
}

// GetAllowedKeys 获取当前允许的所有密钥（供 handler 使用）
func (ah *AdminHandler) GetAllowedKeys() []string {
	ah.keysMu.RLock()
	defer ah.keysMu.RUnlock()

	var keys []string
	for _, keyInfo := range ah.keys {
		entry := fmt.Sprintf("%s|%s|%d",
			keyInfo.ProjectID,
			keyInfo.Key,
			keyInfo.RateLimit,
		)
		keys = append(keys, entry)
	}
	return keys
}
