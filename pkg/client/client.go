// Package client 提供内容审核服务的 Go 客户端 SDK
// 适用于需要与审核服务通信的其他 Go 服务
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ModerateRequest 审核请求
type ModerateRequest struct {
	Content    string                 `json:"content"`
	Type       string                 `json:"type"`       // post | comment
	Model      string                 `json:"model"`      // auto | 指定模型
	Strictness string                 `json:"strictness"` // standard | strict | loose
	WebhookURL string                 `json:"webhook_url,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// ModerateResult 审核结果
type ModerateResult struct {
	Code       int     `json:"code"`
	Verdict    string  `json:"verdict"`    // approved | flagged | rejected
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	ModelUsed  string  `json:"model_used"`
	LatencyMs  int64   `json:"latency_ms"`
	FromCache  bool    `json:"from_cache"`
	Error      string  `json:"error,omitempty"`
}

// Client 审核服务 HTTP 客户端
type Client struct {
	endpoint   string
	projectKey string
	timeout    time.Duration
	httpClient *http.Client
}

// Option 客户端配置项
type Option func(*Client)

// WithTimeout 设置请求超时
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithProjectKey 设置项目密钥
func WithProjectKey(key string) Option {
	return func(c *Client) { c.projectKey = key }
}

// New 创建审核客户端
//
//	client := client.New("http://mod.your-company.com", client.WithProjectKey("proj_xxx"))
func New(endpoint string, opts ...Option) *Client {
	c := &Client{
		endpoint: endpoint,
		timeout:  5 * time.Second,
	}
	for _, o := range opts {
		o(c)
	}
	c.httpClient = &http.Client{Timeout: c.timeout}
	return c
}

// Moderate 同步审核内容
//
//	result, err := client.Moderate("帖子内容", "post", nil)
func (c *Client) Moderate(content, contentType string, ctx map[string]interface{}) (*ModerateResult, error) {
	return c.call("/v1/moderate", &ModerateRequest{
		Content: content,
		Type:    contentType,
		Model:   "auto",
		Context: ctx,
	})
}

// ModerateStrict 严格模式审核
func (c *Client) ModerateStrict(content, contentType string) (*ModerateResult, error) {
	return c.call("/v1/moderate", &ModerateRequest{
		Content:    content,
		Type:       contentType,
		Model:      "auto",
		Strictness: "strict",
	})
}

// ModerateAsync 异步审核（立即返回 task_id）
func (c *Client) ModerateAsync(content, contentType, webhookURL string) (*ModerateResult, error) {
	return c.call("/v1/moderate/async", &ModerateRequest{
		Content:    content,
		Type:       contentType,
		Model:      "auto",
		WebhookURL: webhookURL,
	})
}

// IsApproved 快捷判断：内容是否直接通过
func (c *Client) IsApproved(content, contentType string) (bool, error) {
	result, err := c.Moderate(content, contentType, nil)
	if err != nil {
		return false, err
	}
	return result.Verdict == "approved", nil
}

// Health 健康检查
func (c *Client) Health() error {
	resp, err := c.httpClient.Get(c.endpoint + "/v1/health")
	if err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务不健康，HTTP %d", resp.StatusCode)
	}
	return nil
}

// ── 内部方法 ───────────────────────────────────────────────

func (c *Client) call(path string, req *ModerateRequest) (*ModerateResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.projectKey != "" {
		httpReq.Header.Set("X-Project-Key", c.projectKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// 网络故障：安全降级，返回 flagged
		return &ModerateResult{
			Verdict:  "flagged",
			Category: "none",
			Reason:   "审核服务不可达，已转人工",
			Error:    err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result ModerateResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &result, nil
}
