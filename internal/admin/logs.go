package admin

import (
	"net/http"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
)

// 日志和审计相关的管理接口

// handleProjectLogs 处理 GET /v1/admin/projects/{id}/logs
func (ah *AdminHandler) handleProjectLogs(w http.ResponseWriter, r *http.Request) {
	// 从 URL 中提取项目ID
	// 格式：/v1/admin/projects/{id}/logs
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		ah.jsonError(w, http.StatusBadRequest, "项目ID不能为空")
		return
	}

	// 查询参数
	startStr := r.URL.Query().Get("start") // 2026-03-23
	endStr := r.URL.Query().Get("end")
	eventType := r.URL.Query().Get("type") // api_call, auth_attempt 等

	// 解析时间范围
	var startTime, endTime time.Time
	if startStr != "" {
		t, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			ah.jsonError(w, http.StatusBadRequest, "开始时间格式错误 (格式: 2006-01-02)")
			return
		}
		startTime = t
	} else {
		startTime = time.Now().Add(-24 * time.Hour) // 默认最近 24 小时
	}

	if endStr != "" {
		t, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			ah.jsonError(w, http.StatusBadRequest, "结束时间格式错误 (格式: 2006-01-02)")
			return
		}
		endTime = t.Add(24 * time.Hour) // 包含这一天
	} else {
		endTime = time.Now()
	}

	// 查询日志
	events, err := audit.QueryEvents(ah.cfg.AuditLogDir, projectID, startTime, endTime)
	if err != nil {
		ah.jsonError(w, http.StatusInternalServerError, "查询日志失败: "+err.Error())
		return
	}

	// 过滤事件类型（如果指定）
	if eventType != "" {
		var filtered []audit.AuditEvent
		for _, e := range events {
			if e.EventType == eventType {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"project_id":  projectID,
			"total_count": len(events),
			"start_time":  startTime.Format(time.RFC3339),
			"end_time":    endTime.Format(time.RFC3339),
			"event_type":  eventType,
			"logs":        events,
		},
	})
}

// handleProjectStats 处理 GET /v1/admin/projects/{id}/stats
func (ah *AdminHandler) handleProjectStats(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		ah.jsonError(w, http.StatusBadRequest, "项目ID不能为空")
		return
	}

	// 获取项目统计
	stats, err := audit.GetProjectStats(ah.cfg.AuditLogDir, projectID)
	if err != nil {
		ah.jsonError(w, http.StatusInternalServerError, "获取统计信息失败: "+err.Error())
		return
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": stats,
	})
}

// handleListProjects 处理 GET /v1/admin/projects
func (ah *AdminHandler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	// 从密钥表中获取所有已配置的项目
	ah.keysMu.RLock()
	configuredProjects := make(map[string]bool)
	for _, keyInfo := range ah.keys {
		if keyInfo.ProjectID != "" {
			configuredProjects[keyInfo.ProjectID] = true
		}
	}
	ah.keysMu.RUnlock()

	// 列出所有有日志的项目
	logProjects, err := audit.ListProjects(ah.cfg.AuditLogDir)
	if err != nil {
		// 如果日志目录不存在/不可读，继续使用配置的项目
		logProjects = []string{}
	}

	// 合并配置的项目和日志中的项目
	var allProjects []string
	projectMap := make(map[string]bool)

	// 先添加配置的项目
	for projectID := range configuredProjects {
		if !projectMap[projectID] {
			allProjects = append(allProjects, projectID)
			projectMap[projectID] = true
		}
	}

	// 再添加日志中有的项目
	for _, projectID := range logProjects {
		if !projectMap[projectID] {
			allProjects = append(allProjects, projectID)
			projectMap[projectID] = true
		}
	}

	// 获取每个项目的统计
	var projectStats []map[string]interface{}
	for _, projectID := range allProjects {
		stats, err := audit.GetProjectStats(ah.cfg.AuditLogDir, projectID)
		if err == nil {
			projectStats = append(projectStats, stats)
		} else {
			// 即使没有日志统计，也要返回项目信息
			projectStats = append(projectStats, map[string]interface{}{
				"project_id": projectID,
				"created_at": time.Now(),
				"status":     "configured",
			})
		}
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"total_projects": len(allProjects),
			"projects":       projectStats,
		},
	})
}
