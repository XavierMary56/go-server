package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

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
