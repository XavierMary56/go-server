package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// APIKey API 密钥配置
type APIKey struct {
	Key       string     // 密钥内容
	ProjectID string     // 项目 ID
	CreatedAt time.Time  // 创建时间
	ExpireAt  *time.Time // 过期时间（nil = 永不过期）
	RateLimit int        // 每分钟请求限制（0 = 无限制）
	Enabled   bool       // 是否启用
}

// KeyManager API 密钥管理器
type KeyManager struct {
	mu    sync.RWMutex
	keys  map[string]*APIKey
	usage map[string]*RateCounter // 密钥 -> 请求计数器
}

// RateCounter 速率计数器
type RateCounter struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
}

// New 创建密钥管理器
func New() *KeyManager {
	return &KeyManager{
		keys:  make(map[string]*APIKey),
		usage: make(map[string]*RateCounter),
	}
}

// RegisterKey 注册 API 密钥
func (km *KeyManager) RegisterKey(key *APIKey) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if key.Key == "" {
		return fmt.Errorf("密钥不能为空")
	}
	if key.ProjectID == "" {
		return fmt.Errorf("项目 ID 不能为空")
	}

	km.keys[key.Key] = key
	km.usage[key.Key] = &RateCounter{resetAt: time.Now().Add(1 * time.Minute)}
	return nil
}

// ValidateKey 验证密钥是否有效
func (km *KeyManager) ValidateKey(key string) (string, error) {
	km.mu.RLock()
	apiKey, exists := km.keys[key]
	km.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("密钥不存在")
	}

	// 检查是否启用
	if !apiKey.Enabled {
		return "", fmt.Errorf("密钥已禁用")
	}

	// 检查是否过期
	if apiKey.ExpireAt != nil && time.Now().After(*apiKey.ExpireAt) {
		return "", fmt.Errorf("密钥已过期")
	}

	return apiKey.ProjectID, nil
}

// CheckRateLimit 检查速率限制
func (km *KeyManager) CheckRateLimit(key string) error {
	km.mu.RLock()
	apiKey, exists := km.keys[key]
	counter, counterExists := km.usage[key]
	km.mu.RUnlock()

	if !exists || !counterExists {
		return fmt.Errorf("密钥不存在")
	}

	// 如果未设置限制，直接通过
	if apiKey.RateLimit == 0 {
		return nil
	}

	counter.mu.Lock()
	defer counter.mu.Unlock()

	// 重置计数器（每分钟）
	if time.Now().After(counter.resetAt) {
		counter.count = 0
		counter.resetAt = time.Now().Add(1 * time.Minute)
	}

	counter.count++
	if counter.count > apiKey.RateLimit {
		return fmt.Errorf("请求过于频繁: %d/%d", counter.count-1, apiKey.RateLimit)
	}

	return nil
}

// DisableKey 禁用密钥
func (km *KeyManager) DisableKey(key string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	apiKey, exists := km.keys[key]
	if !exists {
		return fmt.Errorf("密钥不存在")
	}

	apiKey.Enabled = false
	return nil
}

// ListKeys 列出所有密钥（仅展示 ProjectID 和状态）
func (km *KeyManager) ListKeys() map[string]interface{} {
	km.mu.RLock()
	defer km.mu.RUnlock()

	result := make(map[string]interface{})
	for key, apiKey := range km.keys {
		maskedKey := maskKey(key)
		status := "disabled"
		if apiKey.Enabled {
			if apiKey.ExpireAt != nil && time.Now().After(*apiKey.ExpireAt) {
				status = "expired"
			} else {
				status = "active"
			}
		}

		result[maskedKey] = map[string]interface{}{
			"project_id": apiKey.ProjectID,
			"status":     status,
			"created_at": apiKey.CreatedAt,
			"expire_at":  apiKey.ExpireAt,
			"rate_limit": apiKey.RateLimit,
		}
	}

	return result
}

// GenerateSignature 生成 HMAC-SHA256 签名（用于请求验证）
func GenerateSignature(secret string, timestamp string, method string, path string) string {
	message := fmt.Sprintf("%s|%s|%s", timestamp, method, path)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证签名，同时校验时间戳在 ±5 分钟窗口内以防重放攻击
func VerifySignature(secret string, timestamp string, method string, path string, signature string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	diff := time.Now().Unix() - ts
	if diff > 300 || diff < -300 {
		return false
	}
	expected := GenerateSignature(secret, timestamp, method, path)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// maskKey 隐藏密钥的中间部分
func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
