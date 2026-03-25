package service

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

// KeyCheckResult 单个 key 的检测结果
type KeyCheckResult struct {
	ID       int64  `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Status   string `json:"status"` // healthy | unhealthy
	Error    string `json:"error,omitempty"`
}

// StartHealthChecker 启动后台定时健康检测
func (s *ModerationService) StartHealthChecker(ctx context.Context, interval time.Duration) {
	go func() {
		// 启动时先检测一次
		s.CheckAllKeys()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.CheckAllKeys()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// CheckAllKeys 检测所有启用 key 的可用性，返回结果列表
func (s *ModerationService) CheckAllKeys() []KeyCheckResult {
	if s.db == nil {
		return nil
	}
	var results []KeyCheckResult

	// Anthropic keys
	aKeys, _ := s.db.ListAnthropicKeys()
	for _, k := range aKeys {
		if !k.Enabled {
			continue
		}
		ok, errMsg := s.pingAnthropicKey(k.Key)
		status := statusOf(ok)
		s.db.UpdateAnthropicKeyStatus(k.ID, status)
		results = append(results, KeyCheckResult{
			ID: k.ID, Provider: "anthropic", Name: k.Name, Status: status, Error: errMsg,
		})
	}

	// Provider keys (openai / grok)
	for _, provider := range []string{"openai", "grok"} {
		pKeys, _ := s.db.ListProviderKeys(provider)
		for _, k := range pKeys {
			if !k.Enabled {
				continue
			}
			ok, errMsg := s.pingProviderKey(k.Key, provider)
			status := statusOf(ok)
			s.db.UpdateProviderKeyStatus(k.ID, status)
			results = append(results, KeyCheckResult{
				ID: k.ID, Provider: provider, Name: k.Name, Status: status, Error: errMsg,
			})
		}
	}

	s.log.Info(fmt.Sprintf("health check completed: %d keys checked", len(results)), nil)
	return results
}

// CheckAnthropicKeyByID 检测单个 Anthropic key
func (s *ModerationService) CheckAnthropicKeyByID(id int64) KeyCheckResult {
	if s.db == nil {
		return KeyCheckResult{ID: id, Provider: "anthropic", Status: "unhealthy", Error: "db not available"}
	}
	k, err := s.db.GetAnthropicKeyByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return KeyCheckResult{ID: id, Provider: "anthropic", Status: "unhealthy", Error: "key not found"}
		}
		return KeyCheckResult{ID: id, Provider: "anthropic", Status: "unhealthy", Error: err.Error()}
	}
	ok, errMsg := s.pingAnthropicKey(k.Key)
	status := statusOf(ok)
	s.db.UpdateAnthropicKeyStatus(id, status)
	return KeyCheckResult{ID: id, Provider: "anthropic", Name: k.Name, Status: status, Error: errMsg}
}

// CheckProviderKeyByID 检测单个 OpenAI/Grok key
func (s *ModerationService) CheckProviderKeyByID(id int64) KeyCheckResult {
	if s.db == nil {
		return KeyCheckResult{ID: id, Status: "unhealthy", Error: "db not available"}
	}
	k, err := s.db.GetProviderKeyByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return KeyCheckResult{ID: id, Status: "unhealthy", Error: "key not found"}
		}
		return KeyCheckResult{ID: id, Status: "unhealthy", Error: err.Error()}
	}
	ok, errMsg := s.pingProviderKey(k.Key, k.Provider)
	status := statusOf(ok)
	s.db.UpdateProviderKeyStatus(id, status)
	return KeyCheckResult{ID: id, Provider: k.Provider, Name: k.Name, Status: status, Error: errMsg}
}

// MarkAnthropicKeyUnhealthy 在请求中遇到 auth 错误时立即标记
func (s *ModerationService) MarkAnthropicKeyUnhealthy(id int64) {
	if s.db != nil && id > 0 {
		s.db.UpdateAnthropicKeyStatus(id, "unhealthy")
	}
}

// MarkProviderKeyUnhealthy 在请求中遇到 auth 错误时立即标记
func (s *ModerationService) MarkProviderKeyUnhealthy(id int64) {
	if s.db != nil && id > 0 {
		s.db.UpdateProviderKeyStatus(id, "unhealthy")
	}
}

// ── 内部 ping 实现 ────────────────────────────────────────

var pingClient = &http.Client{Timeout: 12 * time.Second}

// pingAnthropicKey 用 GET /v1/models 检测 Key 是否有效（不消耗 token）
func (s *ModerationService) pingAnthropicKey(apiKey string) (ok bool, errMsg string) {
	baseURL := apiBaseURL(s.cfg.AnthropicAPIURL)
	req, err := http.NewRequest("GET", baseURL+"/v1/models", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", s.cfg.AnthropicVer)

	resp, err := pingClient.Do(req)
	if err != nil {
		return false, "network: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return false, fmt.Sprintf("auth failed (HTTP %d)", resp.StatusCode)
	}
	return true, ""
}

// pingProviderKey 用 GET /v1/models 检测 OpenAI/Grok Key
func (s *ModerationService) pingProviderKey(apiKey, provider string) (ok bool, errMsg string) {
	var apiURL string
	switch provider {
	case "grok":
		apiURL = s.cfg.GrokAPIURL
	default:
		apiURL = s.cfg.OpenAIAPIURL
	}
	baseURL := apiBaseURL(apiURL)
	req, err := http.NewRequest("GET", baseURL+"/v1/models", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := pingClient.Do(req)
	if err != nil {
		return false, "network: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return false, fmt.Sprintf("auth failed (HTTP %d)", resp.StatusCode)
	}
	return true, ""
}

// apiBaseURL 从完整 API URL 提取基础地址
// "https://api.openai.com/v1/chat/completions" → "https://api.openai.com"
func apiBaseURL(apiURL string) string {
	if idx := strings.Index(apiURL, "/v1/"); idx != -1 {
		return apiURL[:idx]
	}
	return apiURL
}

func statusOf(ok bool) string {
	if ok {
		return "healthy"
	}
	return "unhealthy"
}

// GetKeyStatus 提供给外部查询当前所有 key 的状态（供 admin 接口使用）
func (s *ModerationService) GetKeyStatus() ([]storage.AnthropicKey, []storage.ProviderKey) {
	if s.db == nil {
		return nil, nil
	}
	aKeys, _ := s.db.ListAnthropicKeys()
	var ak []storage.AnthropicKey
	for _, k := range aKeys {
		ak = append(ak, *k)
	}

	var pk []storage.ProviderKey
	for _, prov := range []string{"openai", "grok"} {
		pKeys, _ := s.db.ListProviderKeys(prov)
		for _, k := range pKeys {
			pk = append(pk, *k)
		}
	}
	return ak, pk
}
