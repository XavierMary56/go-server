package admin

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
)

const adminTokenSettingKey = "admin_token_hash"

// withAdminAuth validates the admin bearer token before serving a request.
func (ah *AdminHandler) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			ah.jsonError(w, http.StatusUnauthorized, "缺少 Authorization 头")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader || !ah.isValidAdminToken(token) {
			ah.jsonError(w, http.StatusUnauthorized, "无效的管理员令牌")
			ah.auditLogger.LogEvent(&audit.AuditEvent{
				Timestamp: time.Now(),
				EventType: "admin_auth_failed",
				ErrorMsg:  "无效的管理员令牌",
				IPAddress: ah.getClientIP(r),
				Path:      r.RequestURI,
			})
			return
		}

		next(w, r)
	}
}

// isValidAdminToken checks whether the provided token matches config.
func (ah *AdminHandler) isValidAdminToken(token string) bool {
	if ah.db != nil {
		setting, err := ah.db.GetAdminSetting(adminTokenSettingKey)
		if err == nil && setting != nil && strings.TrimSpace(setting.Value) != "" {
			expected := hashAdminToken(token)
			return subtle.ConstantTimeCompare([]byte(setting.Value), []byte(expected)) == 1
		}
	}

	adminTokens := strings.Split(ah.cfg.AdminToken, ",")
	for _, t := range adminTokens {
		if strings.TrimSpace(t) == token {
			return true
		}
	}
	return false
}

func hashAdminToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
