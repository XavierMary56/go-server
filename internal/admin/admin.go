package admin

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

// AdminHandler 管理 API 处理器
type AdminHandler struct {
	cfg         *config.Config
	log         *logger.Logger
	auditLogger *audit.AuditLogger
	db          *storage.DB
	svc         *service.ModerationService
	keysMu      sync.RWMutex
	keys        map[string]*KeyInfo // 内存中的密钥管理
	// 保存原始的 AllowedKeys，用于数据库为空时的初始化
	originalAllowedKeys []string
}

// KeyInfo 密钥信息
type KeyInfo struct {
	ProjectName string    `json:"project_name"`
	Key         string    `json:"key"`
	RateLimit   int       `json:"rate_limit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Enabled     bool      `json:"enabled"`
}

// 请求/响应结构体

// AddKeyRequest 添加密钥请求
type AddKeyRequest struct {
	ProjectName string `json:"project_name"`
	Key         string `json:"key"`
	RateLimit   int    `json:"rate_limit"` // 每分钟请求限制
}

// UpdateKeyRequest 更新密钥请求
type UpdateKeyRequest struct {
	ProjectName *string `json:"project_name,omitempty"`
	Key         *string `json:"key,omitempty"`
	RateLimit   *int    `json:"rate_limit,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

// ListKeysResponse 密钥列表响应
type ListKeysResponse struct {
	Code int                 `json:"code"`
	Data map[string]*KeyInfo `json:"data"`
}

// New 创建管理处理器
func New(cfg *config.Config, log *logger.Logger, auditLogger *audit.AuditLogger, db *storage.DB, svc *service.ModerationService) *AdminHandler {
	handler := &AdminHandler{
		cfg:         cfg,
		log:         log,
		auditLogger: auditLogger,
		db:          db,
		svc:         svc,
		keys:        make(map[string]*KeyInfo),
	}

	// 优先从数据库加载密钥，否则从环境变量导入
	if db != nil {
		handler.loadKeysFromDB()
	} else {
		handler.loadKeysFromEnv()
	}

	return handler
}

// SetOriginalAllowedKeys 设置原始的 AllowedKeys（用于数据库初始化）
func (ah *AdminHandler) SetOriginalAllowedKeys(keys []string) {
	ah.originalAllowedKeys = keys
}

// maskKey 隐藏密钥中间部分
func (ah *AdminHandler) maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// getClientIP 获取客户端 IP
func (ah *AdminHandler) getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// GetAllowedKeys 获取当前允许的所有密钥（供 handler 使用）
func (ah *AdminHandler) GetAllowedKeys() []string {
	ah.keysMu.RLock()
	defer ah.keysMu.RUnlock()

	var keys []string
	for _, keyInfo := range ah.keys {
		entry := fmt.Sprintf("%s|%s|%d",
			keyInfo.ProjectName,
			keyInfo.Key,
			keyInfo.RateLimit,
		)
		keys = append(keys, entry)
	}
	return keys
}
