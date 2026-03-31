package admin

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const staticVersionSettingKey = "static_version"

var printableAdminTokenPattern = regexp.MustCompile(`^[!-~]{12,128}$`)

func (ah *AdminHandler) handleAdminTokenSettings(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}

	switch r.Method {
	case http.MethodGet:
		ah.getAdminTokenSettings(w, r)
	case http.MethodPut:
		ah.updateAdminTokenSettings(w, r)
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

func (ah *AdminHandler) getAdminTokenSettings(w http.ResponseWriter, r *http.Request) {
	setting, err := ah.db.GetAdminSetting(adminTokenSettingKey)
	if err != nil {
		ah.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	source := "env"
	configured := strings.TrimSpace(ah.cfg.AdminToken) != ""
	var updatedAt *time.Time
	if setting != nil && strings.TrimSpace(setting.Value) != "" {
		source = "database"
		configured = true
		updatedAt = setting.UpdatedAt
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"source":     source,
			"configured": configured,
			"updated_at": updatedAt,
		},
	})
}

func (ah *AdminHandler) updateAdminTokenSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewToken     string `json:"new_token"`
		ConfirmToken string `json:"confirm_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.jsonError(w, http.StatusBadRequest, "请求体解析失败")
		return
	}

	req.NewToken = strings.TrimSpace(req.NewToken)
	req.ConfirmToken = strings.TrimSpace(req.ConfirmToken)
	if req.NewToken == "" || req.ConfirmToken == "" {
		ah.jsonError(w, http.StatusBadRequest, "新令牌和确认令牌不能为空")
		return
	}
	if req.NewToken != req.ConfirmToken {
		ah.jsonError(w, http.StatusBadRequest, "两次输入的令牌不一致")
		return
	}
	if !printableAdminTokenPattern.MatchString(req.NewToken) {
		ah.jsonError(w, http.StatusBadRequest, "管理员令牌必须为 12-128 位可打印 ASCII 字符，且不能包含空格")
		return
	}

	if err := ah.db.SetAdminSetting(adminTokenSettingKey, hashAdminToken(req.NewToken)); err != nil {
		ah.jsonError(w, http.StatusInternalServerError, "保存管理员令牌失败")
		return
	}

	if ah.auditLogger != nil {
		ah.auditLogger.LogConfigChange("admin", "update", map[string]interface{}{
			"config_type": "admin_token",
		})
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "管理员令牌已更新",
		"data": map[string]interface{}{
			"source": "database",
		},
	})
}

func (ah *AdminHandler) handleStaticVersionSettings(w http.ResponseWriter, r *http.Request) {
	if ah.db == nil {
		ah.jsonError(w, http.StatusServiceUnavailable, "数据库未初始化")
		return
	}

	switch r.Method {
	case http.MethodGet:
		ah.getStaticVersionSettings(w)
	case http.MethodPut:
		ah.updateStaticVersionSettings(w, r)
	default:
		ah.jsonError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

func (ah *AdminHandler) getStaticVersionSettings(w http.ResponseWriter) {
	setting, err := ah.db.GetAdminSetting(staticVersionSettingKey)
	if err != nil {
		ah.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	version := ""
	var updatedAt *time.Time
	if setting != nil && strings.TrimSpace(setting.Value) != "" {
		version = setting.Value
		updatedAt = setting.UpdatedAt
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"version":    version,
			"updated_at": updatedAt,
		},
	})
}

func (ah *AdminHandler) updateStaticVersionSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.jsonError(w, http.StatusBadRequest, "请求体解析失败")
		return
	}

	req.Version = strings.TrimSpace(req.Version)
	if req.Version == "" {
		ah.jsonError(w, http.StatusBadRequest, "版本号不能为空")
		return
	}

	if err := ah.db.SetAdminSetting(staticVersionSettingKey, req.Version); err != nil {
		ah.jsonError(w, http.StatusInternalServerError, "保存版本号失败")
		return
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "静态资源版本号已更新",
		"data": map[string]interface{}{
			"version": req.Version,
		},
	})
}
