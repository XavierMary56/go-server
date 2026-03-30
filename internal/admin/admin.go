package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

// AdminHandler 管理 API 处理器
type AdminHandler struct {
	cfg         *config.Config
	log         *logger.Logger
	auditLogger *audit.AuditLogger
	db          *storage.DB
	svc         *service.ModerationService
	keysMu      sync.RWMutex
	keys        map[string]*KeyInfo // 内存中的密钥管理
	// 保存原始的 AllowedKeys，用于数据库为空时的初始化
	originalAllowedKeys []string
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
	ProjectID *string `json:"project_id,omitempty"`
	Key       *string `json:"key,omitempty"`
	RateLimit *int    `json:"rate_limit,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

// ListKeysResponse 密钥列表响应
type ListKeysResponse struct {
	Code int                 `json:"code"`
	Data map[string]*KeyInfo `json:"data"`
}

// New 创建管理处理器
func New(cfg *config.Config, log *logger.Logger, auditLogger *audit.AuditLogger, db *storage.DB, svc *service.ModerationService) *AdminHandler {
	handler := &AdminHandler{
		cfg:         cfg,
		log:         log,
		auditLogger: auditLogger,
		db:          db,
		svc:         svc,
		keys:        make(map[string]*KeyInfo),
	}

	// 优先从数据库加载密钥，否则从环境变量导入
	if db != nil {
		handler.loadKeysFromDB()
	} else {
		handler.loadKeysFromEnv()
	}

	return handler
}

// SetOriginalAllowedKeys 设置原始的 AllowedKeys（用于数据库初始化）
func (ah *AdminHandler) SetOriginalAllowedKeys(keys []string) {
	ah.originalAllowedKeys = keys
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

	key, err := url.PathUnescape(parts[4])
	if err != nil {
		ah.jsonError(w, http.StatusBadRequest, "密钥格式无效")
		return
	}
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

	if req.RateLimit < 0 {
		ah.jsonError(w, http.StatusBadRequest, "速率限制不能为负数")
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
		if *req.RateLimit < 0 {
			ah.jsonError(w, http.StatusBadRequest, "速率限制不能为负数")
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

// ── 工具函数 ────────────────────────────────────────

// loadKeysFromEnv 从环境变量加载密钥
func (ah *AdminHandler) loadKeysFromEnv() {
	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	// 优先使用 originalAllowedKeys（如果设置了），否则使用 cfg.AllowedKeys
	allowedKeys := ah.originalAllowedKeys
	if allowedKeys == nil || len(allowedKeys) == 0 {
		allowedKeys = ah.cfg.AllowedKeys
	}

	if allowedKeys == nil || len(allowedKeys) == 0 {
		return
	}

	for _, entry := range allowedKeys {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// 支持两种格式：
		// 1. 新格式：projectID|key|rateLimit
		// 2. 简化格式：key（从密钥本身提取项目ID）
		parts := strings.Split(entry, "|")

		var projectID, key string
		var rateLimit int

		if len(parts) >= 3 {
			// 新格式：projectID|key|rateLimit
			projectID = strings.TrimSpace(parts[0])
			key = strings.TrimSpace(parts[1])
			fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &rateLimit)
		} else if len(parts) >= 2 {
			// 过渡格式：key|rateLimit
			key = strings.TrimSpace(parts[0])
			fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &rateLimit)
			projectID = ah.extractProjectIDFromKey(key)
		} else {
			// 简化格式：仅有密钥
			key = entry
			rateLimit = 1000 // 默认速率限制
			projectID = ah.extractProjectIDFromKey(key)
		}

		if key != "" {
			ah.keys[key] = &KeyInfo{
				ProjectID: projectID,
				Key:       key,
				RateLimit: rateLimit,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Enabled:   true,
			}
			ah.log.Info(fmt.Sprintf("已加载密钥: %s (项目: %s, 限制: %d/分钟)", key, projectID, rateLimit), nil)
		}
	}
}

// extractProjectIDFromKey 从密钥中提取项目ID
func (ah *AdminHandler) extractProjectIDFromKey(key string) string {
	// 格式可能是 proj_forum_xxx，提取 forum
	if strings.HasPrefix(key, "proj_") {
		parts := strings.Split(key, "_")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return key
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

// loadKeysFromDB 从数据库加载密钥到内存
func (ah *AdminHandler) loadKeysFromDB() {
	keys, err := ah.db.ListProjectKeys()
	if err != nil {
		ah.log.Error("从数据库加载密钥失败: " + err.Error())
		ah.loadKeysFromEnv()
		return
	}

	// 如果数据库为空，从环境变量导入
	if len(keys) == 0 {
		ah.log.Info("数据库中没有项目密钥，尝试从环境变量导入", nil)
		ah.loadKeysFromEnv()
		// 将 env 里的 key 写入数据库
		ah.keysMu.RLock()
		defer ah.keysMu.RUnlock()
		for _, k := range ah.keys {
			if _, err := ah.db.AddProjectKey(k.ProjectID, k.Key, k.RateLimit); err != nil {
				ah.log.Error(fmt.Sprintf("添加项目密钥到数据库失败 [%s|%s]: %v", k.ProjectID, k.Key, err))
			} else {
				ah.log.Info(fmt.Sprintf("项目密钥已导入数据库: %s|%s", k.ProjectID, k.Key), nil)
			}
		}
		return
	}

	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()
	for _, k := range keys {
		ah.keys[k.Key] = &KeyInfo{
			ProjectID: k.ProjectID,
			Key:       k.Key,
			RateLimit: k.RateLimit,
			Enabled:   k.Enabled,
			CreatedAt: k.CreatedAt,
			UpdatedAt: k.UpdatedAt,
		}
	}
	ah.log.Info(fmt.Sprintf("从数据库加载了 %d 个项目密钥", len(keys)), nil)
}

// ── Anthropic 密钥管理 ───────────────────────────────────

func (ah *AdminHandler) handleAnthropicKeys(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}
	switch r.Method {
	case http.MethodGet:
		keys, err := ah.db.ListAnthropicKeys()
		if err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// 返回脱敏后的密钥信息
		type safeKey struct {
			ID         int64      `json:"id"`
			Name       string     `json:"name"`
			KeyMasked  string     `json:"key_masked"`
			Enabled    bool       `json:"enabled"`
			Status     string     `json:"status"` // healthy / unhealthy / unknown
			UsageCount int64      `json:"usage_count"`
			LastUsedAt *time.Time `json:"last_used_at"`
			CheckedAt  *time.Time `json:"checked_at"` // 最近检测时间
			CreatedAt  time.Time  `json:"created_at"`
		}
		var safe []safeKey
		for _, k := range keys {
			safe = append(safe, safeKey{
				ID:         k.ID,
				Name:       k.Name,
				KeyMasked:  ah.maskKey(k.Key),
				Enabled:    k.Enabled,
				Status:     k.Status,
				UsageCount: k.UsageCount,
				LastUsedAt: k.LastUsedAt,
				CheckedAt:  k.CheckedAt,
				CreatedAt:  k.CreatedAt,
			})
		}
		if safe == nil {
			safe = []safeKey{}
		}
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": safe})

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Key  string `json:"key"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if req.Name == "" || req.Key == "" {
			ah.jsonError(w, http.StatusBadRequest, "名称和密钥不能为空")
			return
		}
		k, err := ah.db.AddAnthropicKey(req.Name, req.Key)
		if err != nil {
			ah.jsonError(w, http.StatusConflict, "密钥已存在或保存失败: "+err.Error())
			return
		}
		ah.jsonOK(w, http.StatusCreated, map[string]interface{}{"code": 201, "data": k})
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

func (ah *AdminHandler) handleAnthropicKeyDetail(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}
	// 提取 ID
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/verify"), "/")
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ah.jsonError(w, http.StatusBadRequest, "无效的 ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Enabled bool `json:"enabled"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if err := ah.db.UpdateAnthropicKey(id, req.Enabled); err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已更新"})
	case http.MethodDelete:
		if err := ah.db.DeleteAnthropicKey(id); err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已删除"})
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

// handleVerifyAnthropicKey 验证 Anthropic 密钥是否有效（通过 ID 从 DB 取真实密钥）
func (ah *AdminHandler) handleVerifyAnthropicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	var req struct {
		ID int64 `json:"id"`
	}
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &req)
	if req.ID == 0 {
		ah.jsonError(w, http.StatusBadRequest, "ID 不能为空")
		return
	}

	keys, err := ah.db.ListAnthropicKeys()
	if err != nil {
		ah.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var actualKey string
	for _, k := range keys {
		if k.ID == req.ID {
			actualKey = k.Key
			break
		}
	}
	if actualKey == "" {
		ah.jsonError(w, http.StatusNotFound, "密钥不存在")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	payload := `{"model":"claude-haiku-4-5-20251001","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
	httpReq, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(payload))
	httpReq.Header.Set("x-api-key", actualKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "valid": false, "reason": "网络错误: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	valid := resp.StatusCode != 401 && resp.StatusCode != 403
	reason := "验证通过"
	if !valid {
		reason = "密钥无效或已过期"
	}
	ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "valid": valid, "reason": reason})
}

// ── 模型管理 ────────────────────────────────────────────

func (ah *AdminHandler) handleModels(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}
	switch r.Method {
	case http.MethodGet:
		models, err := ah.db.ListModels()
		if err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if len(models) == 0 && len(ah.cfg.Models) > 0 {
			type fallbackModel struct {
				ID       int64  `json:"id"`
				ModelID  string `json:"model_id"`
				Name     string `json:"name"`
				Provider string `json:"provider"`
				Weight   int    `json:"weight"`
				Priority int    `json:"priority"`
				Enabled  bool   `json:"enabled"`
				Source   string `json:"source"`
			}
			fallbacks := make([]fallbackModel, 0, len(ah.cfg.Models))
			for i, m := range ah.cfg.Models {
				provider := m.Provider
				if provider == "" {
					switch {
					case strings.HasPrefix(m.ID, "gpt-"), strings.HasPrefix(m.ID, "o1-"), strings.HasPrefix(m.ID, "o3-"), strings.HasPrefix(m.ID, "o4-"):
						provider = "openai"
					case strings.HasPrefix(m.ID, "grok-"):
						provider = "grok"
					default:
						provider = "anthropic"
					}
				}
				fallbacks = append(fallbacks, fallbackModel{
					ID:       int64(-(i + 1)),
					ModelID:  m.ID,
					Name:     m.Name,
					Provider: provider,
					Weight:   m.Weight,
					Priority: m.Priority,
					Enabled:  true,
					Source:   "config-fallback",
				})
			}
			ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": fallbacks})
			return
		}
		if models == nil {
			models = []*storage.ModelConfig{}
		}
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": models})

	case http.MethodPost:
		var req struct {
			ModelID  string `json:"model_id"`
			Name     string `json:"name"`
			Provider string `json:"provider"`
			Weight   int    `json:"weight"`
			Priority int    `json:"priority"`
			Enabled  bool   `json:"enabled"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if req.ModelID == "" || req.Name == "" {
			ah.jsonError(w, http.StatusBadRequest, "模型 ID 和名称不能为空")
			return
		}
		if err := validateModelID(req.ModelID, req.Provider); err != nil {
			ah.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.Weight <= 0 {
			req.Weight = 50
		}
		if req.Priority <= 0 {
			req.Priority = 1
		}
		if err := ah.db.UpsertModel(req.ModelID, req.Name, req.Provider, req.Weight, req.Priority, req.Enabled); err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ah.jsonOK(w, http.StatusCreated, map[string]interface{}{"code": 201, "message": "模型已保存"})
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

func validateModelID(modelID, provider string) error {
	modelID = strings.TrimSpace(modelID)
	provider = strings.TrimSpace(provider)

	if len(modelID) < 3 || strings.ContainsAny(modelID, " \t\r\n") {
		return fmt.Errorf("模型 ID 格式无效，请填写真实模型名")
	}

	switch provider {
	case "openai":
		if !(strings.HasPrefix(modelID, "gpt-") ||
			strings.HasPrefix(modelID, "o1-") ||
			strings.HasPrefix(modelID, "o3-") ||
			strings.HasPrefix(modelID, "o4-")) {
			return fmt.Errorf("OpenAI 模型 ID 格式无效")
		}
	case "grok":
		if !strings.HasPrefix(modelID, "grok-") {
			return fmt.Errorf("Grok 模型 ID 格式无效")
		}
	case "anthropic":
		if !strings.HasPrefix(modelID, "claude-") {
			return fmt.Errorf("Anthropic 模型 ID 格式无效")
		}
	}

	return nil
}

func (ah *AdminHandler) handleModelDetail(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ah.jsonError(w, http.StatusBadRequest, "无效的 ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Weight   *int  `json:"weight,omitempty"`
			Priority *int  `json:"priority,omitempty"`
			Enabled  *bool `json:"enabled,omitempty"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		ah.db.UpdateModel(id, req.Weight, req.Priority, req.Enabled)
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已更新"})
	case http.MethodDelete:
		ah.db.DeleteModel(id)
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已删除"})
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

// ── Provider 密钥管理（OpenAI / Grok） ─────────────────────

func (ah *AdminHandler) handleProviderKeys(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}
	// 从查询参数读取 provider；为空时返回全部提供方密钥
	provider := r.URL.Query().Get("provider")

	switch r.Method {
	case http.MethodGet:
		var keys []*storage.ProviderKey
		var err error
		if provider != "" {
			keys, err = ah.db.ListProviderKeys(provider)
		} else {
			oaiKeys, e1 := ah.db.ListProviderKeys("openai")
			grokKeys, e2 := ah.db.ListProviderKeys("grok")
			if e1 != nil {
				err = e1
			} else if e2 != nil {
				err = e2
			} else {
				keys = append(oaiKeys, grokKeys...)
			}
		}
		if err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		type safeKey struct {
			ID         int64      `json:"id"`
			Provider   string     `json:"provider"`
			Name       string     `json:"name"`
			KeyMasked  string     `json:"key_masked"`
			Enabled    bool       `json:"enabled"`
			UsageCount int64      `json:"usage_count"`
			LastUsedAt *time.Time `json:"last_used_at"`
			CreatedAt  time.Time  `json:"created_at"`
		}
		var safe []safeKey
		for _, k := range keys {
			safe = append(safe, safeKey{
				ID:         k.ID,
				Provider:   k.Provider,
				Name:       k.Name,
				KeyMasked:  ah.maskKey(k.Key),
				Enabled:    k.Enabled,
				UsageCount: k.UsageCount,
				LastUsedAt: k.LastUsedAt,
				CreatedAt:  k.CreatedAt,
			})
		}
		if safe == nil {
			safe = []safeKey{}
		}
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": safe})

	case http.MethodPost:
		var req struct {
			Provider string `json:"provider"`
			Name     string `json:"name"`
			Key      string `json:"key"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if req.Provider != "openai" && req.Provider != "grok" {
			ah.jsonError(w, http.StatusBadRequest, "provider 必须是 openai 或 grok")
			return
		}
		if req.Name == "" || req.Key == "" {
			ah.jsonError(w, http.StatusBadRequest, "名称和密钥不能为空")
			return
		}
		k, err := ah.db.AddProviderKey(req.Provider, req.Name, req.Key)
		if err != nil {
			ah.jsonError(w, http.StatusConflict, "密钥已存在或保存失败: "+err.Error())
			return
		}
		ah.jsonOK(w, http.StatusCreated, map[string]interface{}{"code": 201, "data": k})
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

// ── 健康检测 ─────────────────────────────────────────────

// handleCheckAllKeys POST /v1/admin/keys/check-all 检测所有密钥
func (ah *AdminHandler) handleCheckAllKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	if ah.svc == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "服务未初始化")
		return
	}
	results := ah.svc.CheckAllKeys()
	ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": results})
}

// handleCheckAnthropicKey POST /v1/admin/anthropic-keys/check 检测单个 Anthropic 密钥
func (ah *AdminHandler) handleCheckAnthropicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	if ah.svc == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "服务未初始化")
		return
	}
	var req struct {
		ID int64 `json:"id"`
	}
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &req)
	if req.ID == 0 {
		ah.jsonError(w, http.StatusBadRequest, "id 不能为空")
		return
	}
	result := ah.svc.CheckAnthropicKeyByID(req.ID)
	ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": result})
}

// handleCheckProviderKey POST /v1/admin/provider-keys/check 检测单个 Provider 密钥
func (ah *AdminHandler) handleCheckProviderKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	if ah.svc == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "服务未初始化")
		return
	}
	var req struct {
		ID int64 `json:"id"`
	}
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &req)
	if req.ID == 0 {
		ah.jsonError(w, http.StatusBadRequest, "id 不能为空")
		return
	}
	result := ah.svc.CheckProviderKeyByID(req.ID)
	ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "data": result})
}

func (ah *AdminHandler) handleProviderKeyDetail(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ah.jsonError(w, http.StatusBadRequest, "无效的 ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Enabled bool `json:"enabled"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if err := ah.db.UpdateProviderKey(id, req.Enabled); err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已更新"})
	case http.MethodDelete:
		if err := ah.db.DeleteProviderKey(id); err != nil {
			ah.jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ah.jsonOK(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已删除"})
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}
