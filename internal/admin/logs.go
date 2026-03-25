package admin

import (
	"net/http"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
)

// handleProjectLogs handles GET /v1/admin/projects/logs?project={id}
func (ah *AdminHandler) handleProjectLogs(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		ah.jsonError(w, http.StatusBadRequest, "project id is required")
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	eventType := r.URL.Query().Get("type")

	var startTime, endTime time.Time
	if startStr != "" {
		t, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			ah.jsonError(w, http.StatusBadRequest, "invalid start date, expected 2006-01-02")
			return
		}
		startTime = t
	} else {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endStr != "" {
		t, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			ah.jsonError(w, http.StatusBadRequest, "invalid end date, expected 2006-01-02")
			return
		}
		endTime = t.Add(24 * time.Hour)
	} else {
		endTime = time.Now()
	}

	events, err := audit.QueryEvents(ah.cfg.AuditLogDir, projectID, startTime, endTime)
	if err != nil {
		ah.jsonError(w, http.StatusInternalServerError, "failed to query logs: "+err.Error())
		return
	}

	if eventType != "" {
		filtered := make([]audit.AuditEvent, 0, len(events))
		for _, event := range events {
			if event.EventType == eventType {
				filtered = append(filtered, event)
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

// handleProjectStats handles GET /v1/admin/projects/stats and /v1/admin/projects/stats?project={id}
func (ah *AdminHandler) handleProjectStats(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		result := make(map[string]interface{})
		for _, pid := range ah.collectProjectIDs() {
			stats, err := audit.GetProjectStats(ah.cfg.AuditLogDir, pid)
			if err != nil {
				continue
			}
			result[pid] = stats
		}

		ah.jsonOK(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": result,
		})
		return
	}

	stats, err := audit.GetProjectStats(ah.cfg.AuditLogDir, projectID)
	if err != nil {
		ah.jsonError(w, http.StatusInternalServerError, "failed to get stats: "+err.Error())
		return
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": stats,
	})
}

// handleListProjects handles GET /v1/admin/projects
func (ah *AdminHandler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projectStats := make([]map[string]interface{}, 0)
	for _, projectID := range ah.collectProjectIDs() {
		stats, err := audit.GetProjectStats(ah.cfg.AuditLogDir, projectID)
		if err == nil {
			projectStats = append(projectStats, stats)
			continue
		}

		projectStats = append(projectStats, map[string]interface{}{
			"project_id": projectID,
			"status":     "configured",
			"log_size":   int64(0),
		})
	}

	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"total_projects": len(projectStats),
			"projects":       projectStats,
		},
	})
}

func (ah *AdminHandler) collectProjectIDs() []string {
	ah.keysMu.RLock()
	projectMap := make(map[string]bool)
	for _, keyInfo := range ah.keys {
		if keyInfo.ProjectID != "" {
			projectMap[keyInfo.ProjectID] = true
		}
	}
	ah.keysMu.RUnlock()

	logProjects, err := audit.ListProjects(ah.cfg.AuditLogDir)
	if err == nil {
		for _, projectID := range logProjects {
			projectMap[projectID] = true
		}
	}

	projectIDs := make([]string, 0, len(projectMap))
	for projectID := range projectMap {
		projectIDs = append(projectIDs, projectID)
	}

	return projectIDs
}
