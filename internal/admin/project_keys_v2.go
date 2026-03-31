package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
)

func (ah *AdminHandler) handleProjectKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ah.listProjectKeys(w, r)
	case http.MethodPost:
		ah.addProjectKeyV2(w, r)
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

func (ah *AdminHandler) handleProjectKeyDetail(w http.ResponseWriter, r *http.Request) {
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
		ah.getProjectKey(w, r, key)
	case http.MethodPut:
		ah.updateProjectKeyV2(w, r, key)
	case http.MethodDelete:
		ah.deleteProjectKeyV2(w, r, key)
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

func (ah *AdminHandler) listProjectKeys(w http.ResponseWriter, r *http.Request) {
	ah.keysMu.RLock()
	defer ah.keysMu.RUnlock()

	result := make(map[string]*KeyInfo, len(ah.keys))
	for key, value := range ah.keys {
		item := *value
		result[key] = &item
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

func (ah *AdminHandler) addProjectKeyV2(w http.ResponseWriter, r *http.Request) {
	var req AddKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.jsonError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	req.ProjectName = strings.TrimSpace(req.ProjectName)
	req.Key = strings.TrimSpace(req.Key)
	if req.ProjectName == "" {
		ah.jsonError(w, http.StatusBadRequest, "项目名称不能为空")
		return
	}
	if req.RateLimit < 0 {
		ah.jsonError(w, http.StatusBadRequest, "速率限制不能为负数")
		return
	}
	if req.Key == "" {
		req.Key = ah.generateProjectKey(req.ProjectName)
	}

	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	if _, exists := ah.keys[req.Key]; exists {
		ah.jsonError(w, http.StatusConflict, "密钥已存在")
		return
	}
	for _, v := range ah.keys {
		if strings.EqualFold(v.ProjectName, req.ProjectName) {
			ah.jsonError(w, http.StatusConflict, "项目名称已存在，请使用其他名称")
			return
		}
	}

	keyInfo := &KeyInfo{
		ProjectName: req.ProjectName,
		Key:         req.Key,
		RateLimit:   req.RateLimit,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Enabled:     true,
	}

	if ah.db != nil {
		dbKey, err := ah.db.AddProjectKey(req.ProjectName, req.Key, req.RateLimit)
		if err != nil {
			ah.jsonError(w, http.StatusConflict, "密钥保存失败: "+err.Error())
			return
		}
		keyInfo.ID = dbKey.ID
		keyInfo.ProjectName = dbKey.ProjectName
		keyInfo.Key = dbKey.Key
		keyInfo.RateLimit = dbKey.RateLimit
		keyInfo.CreatedAt = dbKey.CreatedAt
		keyInfo.UpdatedAt = dbKey.UpdatedAt
		keyInfo.Enabled = dbKey.Enabled
	}

	ah.keys[keyInfo.Key] = keyInfo
	ah.updateEnvFile()

	ah.jsonOK(w, http.StatusCreated, map[string]interface{}{
		"code":    201,
		"message": "项目已添加",
		"data":    keyInfo,
	})

	ah.auditLogger.LogEvent(&audit.AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "admin_add_key",
		ProjectName:  keyInfo.ProjectName,
		Path:       r.RequestURI,
		Method:     r.Method,
		StatusCode: http.StatusCreated,
	})
}

func (ah *AdminHandler) getProjectKey(w http.ResponseWriter, r *http.Request, key string) {
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

func (ah *AdminHandler) updateProjectKeyV2(w http.ResponseWriter, r *http.Request, currentKey string) {
	var req UpdateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.jsonError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	keyInfo, exists := ah.keys[currentKey]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "密钥不存在")
		return
	}

	newProjectID := keyInfo.ProjectName
	newKey := keyInfo.Key
	newRateLimit := keyInfo.RateLimit
	newEnabled := keyInfo.Enabled

	if req.ProjectName != nil {
		value := strings.TrimSpace(*req.ProjectName)
		if value == "" {
			ah.jsonError(w, http.StatusBadRequest, "项目名称不能为空")
			return
		}
		if !strings.EqualFold(value, keyInfo.ProjectName) {
			for _, v := range ah.keys {
				if v.Key != currentKey && strings.EqualFold(v.ProjectName, value) {
					ah.jsonError(w, http.StatusConflict, "项目名称已存在，请使用其他名称")
					return
				}
			}
		}
		newProjectID = value
	}
	if req.Key != nil {
		value := strings.TrimSpace(*req.Key)
		if value == "" {
			ah.jsonError(w, http.StatusBadRequest, "密钥不能为空")
			return
		}
		if value != currentKey {
			if _, exists := ah.keys[value]; exists {
				ah.jsonError(w, http.StatusConflict, "密钥已存在")
				return
			}
		}
		newKey = value
	}
	if req.RateLimit != nil {
		if *req.RateLimit < 0 {
			ah.jsonError(w, http.StatusBadRequest, "速率限制不能为负数")
			return
		}
		newRateLimit = *req.RateLimit
	}
	if req.Enabled != nil {
		newEnabled = *req.Enabled
	}

	if ah.db != nil {
		if err := ah.db.UpdateProjectKey(currentKey, &newProjectID, &newKey, &newEnabled, &newRateLimit); err != nil {
			ah.jsonError(w, http.StatusConflict, "密钥更新失败: "+err.Error())
			return
		}
	}

	if newKey != currentKey {
		delete(ah.keys, currentKey)
	}

	keyInfo.ProjectName = newProjectID
	keyInfo.Key = newKey
	keyInfo.RateLimit = newRateLimit
	keyInfo.Enabled = newEnabled
	keyInfo.UpdatedAt = time.Now()
	ah.keys[newKey] = keyInfo
	ah.updateEnvFile()

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "密钥已更新",
		"data":    keyInfo,
	})

	ah.auditLogger.LogEvent(&audit.AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "admin_update_key",
		ProjectName:  keyInfo.ProjectName,
		Path:       r.RequestURI,
		Method:     r.Method,
		StatusCode: http.StatusOK,
	})
}

func (ah *AdminHandler) deleteProjectKeyV2(w http.ResponseWriter, r *http.Request, key string) {
	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	keyInfo, exists := ah.keys[key]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "密钥不存在")
		return
	}

	if ah.db != nil {
		if err := ah.db.DeleteProjectKey(key); err != nil {
			ah.jsonError(w, http.StatusInternalServerError, "删除密钥失败: "+err.Error())
			return
		}
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
		ProjectName:  keyInfo.ProjectName,
		Path:       r.RequestURI,
		Method:     r.Method,
		StatusCode: http.StatusOK,
	})
}

func (ah *AdminHandler) generateProjectKey(projectID string) string {
	projectPart := sanitizeProjectID(projectID)
	if projectPart == "" {
		projectPart = "project"
	}
	return fmt.Sprintf("sk-proj-%s-%s", projectPart, randomHex(8))
}

func sanitizeProjectID(projectID string) string {
	projectID = strings.ToLower(strings.TrimSpace(projectID))
	var builder strings.Builder
	for _, ch := range projectID {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
		default:
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	result = strings.ReplaceAll(result, "--", "-")
	return result
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
