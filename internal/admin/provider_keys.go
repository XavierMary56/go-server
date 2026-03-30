package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

// ── Provider 密钥管理（OpenAI / Grok） ─────────────────────

func (ah *AdminHandler) handleProviderKeys(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}
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

// ── 健康检测 ─────────────────────────────────────────────

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
