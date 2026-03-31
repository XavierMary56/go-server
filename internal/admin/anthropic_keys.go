package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
		type safeKey struct {
			ID         int64      `json:"id"`
			Name       string     `json:"name"`
			KeyMasked  string     `json:"key_masked"`
			Enabled    bool       `json:"enabled"`
			Status     string     `json:"status"`
			UsageCount int64      `json:"usage_count"`
			LastUsedAt *time.Time `json:"last_used_at"`
			CheckedAt  *time.Time `json:"checked_at"`
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
			Enabled *bool  `json:"enabled"`
			Name    string `json:"name"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if req.Name != "" {
			if err := ah.db.UpdateAnthropicKeyName(id, req.Name); err != nil {
				ah.jsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if req.Enabled != nil {
			if err := ah.db.UpdateAnthropicKey(id, *req.Enabled); err != nil {
				ah.jsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
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
