package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
)

// ── 项目密钥管理 ────────────────────────────────────────

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

func (ah *AdminHandler) handleKeyDetail(w http.ResponseWriter, r *http.Request) {
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

func (ah *AdminHandler) listKeys(w http.ResponseWriter, r *http.Request) {
	ah.keysMu.RLock()
	defer ah.keysMu.RUnlock()

	result := make(map[string]*KeyInfo)
	for k, v := range ah.keys {
		maskedKey := ah.maskKey(k)
		keyInfo := *v
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

func (ah *AdminHandler) addKey(w http.ResponseWriter, r *http.Request) {
	var req AddKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.jsonError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
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
	if _, exists := ah.keys[req.Key]; exists {
		ah.jsonError(w, http.StatusConflict, "密钥已存在")
		return
	}

	keyInfo := &KeyInfo{
		ProjectID: req.ProjectID,
		Key:       req.Key,
		RateLimit: req.RateLimit,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Enabled:   true,
	}
	ah.keys[req.Key] = keyInfo
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

func (ah *AdminHandler) deleteKey(w http.ResponseWriter, r *http.Request, key string) {
	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	keyInfo, exists := ah.keys[key]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "密钥不存在")
		return
	}

	delete(ah.keys, key)
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

func (ah *AdminHandler) loadKeysFromEnv() {
	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	allowedKeys := ah.originalAllowedKeys
	if len(allowedKeys) == 0 {
		allowedKeys = ah.cfg.AllowedKeys
	}
	if len(allowedKeys) == 0 {
		return
	}

	for _, entry := range allowedKeys {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "|")
		var projectID, key string
		var rateLimit int
		if len(parts) >= 3 {
			projectID = strings.TrimSpace(parts[0])
			key = strings.TrimSpace(parts[1])
			fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &rateLimit)
		} else if len(parts) >= 2 {
			key = strings.TrimSpace(parts[0])
			fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &rateLimit)
			projectID = ah.extractProjectIDFromKey(key)
		} else {
			key = entry
			rateLimit = 1000
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

func (ah *AdminHandler) extractProjectIDFromKey(key string) string {
	if strings.HasPrefix(key, "proj_") {
		parts := strings.Split(key, "_")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return key
}

func (ah *AdminHandler) updateEnvFile() {
	var keys []string
	for _, keyInfo := range ah.keys {
		if keyInfo.Enabled {
			entry := fmt.Sprintf("%s|%s|%d", keyInfo.ProjectID, keyInfo.Key, keyInfo.RateLimit)
			keys = append(keys, entry)
		}
	}
	ah.cfg.AllowedKeys = keys
}

func (ah *AdminHandler) loadKeysFromDB() {
	keys, err := ah.db.ListProjectKeys()
	if err != nil {
		ah.log.Error("从数据库加载密钥失败: " + err.Error())
		ah.loadKeysFromEnv()
		return
	}

	if len(keys) == 0 {
		ah.log.Info("数据库中没有项目密钥，尝试从环境变量导入", nil)
		ah.loadKeysFromEnv()
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
