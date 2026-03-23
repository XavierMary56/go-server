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
		ah.jsonError(w, http.StatusMethodNotAllowed, "鏂规硶涓嶅厑璁?")
	}
}

func (ah *AdminHandler) handleProjectKeyDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		ah.jsonError(w, http.StatusBadRequest, "缂哄皯瀵嗛挜")
		return
	}

	key, err := url.PathUnescape(parts[4])
	if err != nil {
		ah.jsonError(w, http.StatusBadRequest, "瀵嗛挜鏍煎紡鏃犳晥")
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
		ah.jsonError(w, http.StatusMethodNotAllowed, "鏂规硶涓嶅厑璁?")
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
		ah.jsonError(w, http.StatusBadRequest, "璇锋眰浣撹В鏋愬け璐? "+err.Error())
		return
	}

	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.Key = strings.TrimSpace(req.Key)
	if req.ProjectID == "" {
		ah.jsonError(w, http.StatusBadRequest, "椤圭洰 ID 涓嶈兘涓虹┖")
		return
	}
	if req.RateLimit <= 0 {
		ah.jsonError(w, http.StatusBadRequest, "閫熺巼闄愬埗蹇呴』澶т簬 0")
		return
	}
	if req.Key == "" {
		req.Key = ah.generateProjectKey(req.ProjectID)
	}

	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	if _, exists := ah.keys[req.Key]; exists {
		ah.jsonError(w, http.StatusConflict, "瀵嗛挜宸插瓨鍦?")
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

	if ah.db != nil {
		dbKey, err := ah.db.AddProjectKey(req.ProjectID, req.Key, req.RateLimit)
		if err != nil {
			ah.jsonError(w, http.StatusConflict, "瀵嗛挜淇濆瓨澶辫触: "+err.Error())
			return
		}
		keyInfo.ProjectID = dbKey.ProjectID
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
		"message": "瀵嗛挜宸叉坊鍔?",
		"data":    keyInfo,
	})

	ah.auditLogger.LogEvent(&audit.AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "admin_add_key",
		ProjectID:  keyInfo.ProjectID,
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
		ah.jsonError(w, http.StatusNotFound, "瀵嗛挜涓嶅瓨鍦?")
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
		ah.jsonError(w, http.StatusBadRequest, "璇锋眰浣撹В鏋愬け璐? "+err.Error())
		return
	}

	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	keyInfo, exists := ah.keys[currentKey]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "瀵嗛挜涓嶅瓨鍦?")
		return
	}

	newProjectID := keyInfo.ProjectID
	newKey := keyInfo.Key
	newRateLimit := keyInfo.RateLimit
	newEnabled := keyInfo.Enabled

	if req.ProjectID != nil {
		value := strings.TrimSpace(*req.ProjectID)
		if value == "" {
			ah.jsonError(w, http.StatusBadRequest, "椤圭洰 ID 涓嶈兘涓虹┖")
			return
		}
		newProjectID = value
	}
	if req.Key != nil {
		value := strings.TrimSpace(*req.Key)
		if value == "" {
			ah.jsonError(w, http.StatusBadRequest, "瀵嗛挜涓嶈兘涓虹┖")
			return
		}
		if value != currentKey {
			if _, exists := ah.keys[value]; exists {
				ah.jsonError(w, http.StatusConflict, "瀵嗛挜宸插瓨鍦?")
				return
			}
		}
		newKey = value
	}
	if req.RateLimit != nil {
		if *req.RateLimit <= 0 {
			ah.jsonError(w, http.StatusBadRequest, "閫熺巼闄愬埗蹇呴』澶т簬 0")
			return
		}
		newRateLimit = *req.RateLimit
	}
	if req.Enabled != nil {
		newEnabled = *req.Enabled
	}

	if ah.db != nil {
		if err := ah.db.UpdateProjectKey(currentKey, &newProjectID, &newKey, &newEnabled, &newRateLimit); err != nil {
			ah.jsonError(w, http.StatusConflict, "瀵嗛挜鏇存柊澶辫触: "+err.Error())
			return
		}
	}

	if newKey != currentKey {
		delete(ah.keys, currentKey)
	}

	keyInfo.ProjectID = newProjectID
	keyInfo.Key = newKey
	keyInfo.RateLimit = newRateLimit
	keyInfo.Enabled = newEnabled
	keyInfo.UpdatedAt = time.Now()
	ah.keys[newKey] = keyInfo
	ah.updateEnvFile()

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "瀵嗛挜宸叉洿鏂?",
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

func (ah *AdminHandler) deleteProjectKeyV2(w http.ResponseWriter, r *http.Request, key string) {
	ah.keysMu.Lock()
	defer ah.keysMu.Unlock()

	keyInfo, exists := ah.keys[key]
	if !exists {
		ah.jsonError(w, http.StatusNotFound, "瀵嗛挜涓嶅瓨鍦?")
		return
	}

	if ah.db != nil {
		if err := ah.db.DeleteProjectKey(key); err != nil {
			ah.jsonError(w, http.StatusInternalServerError, "鍒犻櫎瀵嗛挜澶辫触: "+err.Error())
			return
		}
	}

	delete(ah.keys, key)
	ah.updateEnvFile()

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "瀵嗛挜宸插垹闄?",
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
