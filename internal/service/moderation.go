package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
)

// ── 请求 / 响应结构体 ──────────────────────────────────────

// ModerateRequest 审核请求
type ModerateRequest struct {
	Content    string                 `json:"content"`
	Type       string                 `json:"type"`       // post | comment
	Model      string                 `json:"model"`      // auto | 指定模型 ID
	Strictness string                 `json:"strictness"` // standard | strict | loose
	WebhookURL string                 `json:"webhook_url,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// ModerateResult 审核结果
type ModerateResult struct {
	Verdict    string  `json:"verdict"`     // approved | flagged | rejected
	Category   string  `json:"category"`    // none | spam | abuse | politics | adult | fraud | violence
	Confidence float64 `json:"confidence"`  // 0.0 ~ 1.0
	Reason     string  `json:"reason"`
	ModelUsed  string  `json:"model_used"`
	LatencyMs  int64   `json:"latency_ms"`
	FromCache  bool    `json:"from_cache"`
	Fallback   bool    `json:"fallback,omitempty"`
}

// anthropicRequest Anthropic API 请求体
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse Anthropic API 响应体
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// aiResult AI 返回的 JSON 结果
type aiResult struct {
	Verdict    string  `json:"verdict"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// ── 服务结构体 ─────────────────────────────────────────────

// ModerationService 核心审核服务
type ModerationService struct {
	cfg    *config.Config
	log    *logger.Logger
	cache  Cache
	client *http.Client
	stats  *Stats
	mu     sync.RWMutex
}

// Stats 运行时统计
type Stats struct {
	mu          sync.RWMutex
	Total       int64
	Approved    int64
	Flagged     int64
	Rejected    int64
	ModelCounts map[string]int64
}

func newStats() *Stats {
	return &Stats{ModelCounts: make(map[string]int64)}
}

func (s *Stats) record(verdict, model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Total++
	switch verdict {
	case "approved":
		s.Approved++
	case "flagged":
		s.Flagged++
	case "rejected":
		s.Rejected++
	}
	s.ModelCounts[model]++
}

func (s *Stats) snapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := make(map[string]int64, len(s.ModelCounts))
	for k, v := range s.ModelCounts {
		counts[k] = v
	}
	return map[string]interface{}{
		"total":        s.Total,
		"approved":     s.Approved,
		"flagged":      s.Flagged,
		"rejected":     s.Rejected,
		"model_counts": counts,
	}
}

// NewModerationService 创建服务实例
func NewModerationService(cfg *config.Config, log *logger.Logger) *ModerationService {
	return &ModerationService{
		cfg:   cfg,
		log:   log,
		cache: newMemoryCache(cfg.CacheTTL),
		client: &http.Client{
			Timeout: time.Duration(cfg.APITimeout) * time.Second,
		},
		stats: newStats(),
	}
}

// ── 审核入口 ───────────────────────────────────────────────

// Moderate 同步审核
func (s *ModerationService) Moderate(req *ModerateRequest) *ModerateResult {
	if req.Type == "" {
		req.Type = "post"
	}
	if req.Strictness == "" {
		req.Strictness = "standard"
	}

	// 命中缓存直接返回
	cacheKey := s.cacheKey(req.Content, req.Type, req.Strictness)
	if cached, ok := s.cache.Get(cacheKey); ok {
		result := cached.(*ModerateResult)
		result.FromCache = true
		return result
	}

	// 构建模型调用队列（首选 + 故障转移备选）
	queue := s.buildModelQueue(req.Model)
	start := time.Now()

	var result *ModerateResult
	var lastErr error

	for _, model := range queue {
		result, lastErr = s.callAPI(req, model.ID)
		if lastErr == nil {
			break
		}
		s.log.Warn(fmt.Sprintf("模型 %s 调用失败: %v，尝试下一个...", model.ID, lastErr))
	}

	if lastErr != nil || result == nil {
		s.log.Error("所有模型均失败: " + lastErr.Error())
		return s.safeFallback(queue[0].ID, time.Since(start).Milliseconds())
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	result.FromCache = false

	// 写入缓存 & 统计
	s.cache.Set(cacheKey, result)
	s.stats.record(result.Verdict, result.ModelUsed)
	s.log.Info("moderation", map[string]interface{}{
		"verdict":    result.Verdict,
		"category":   result.Category,
		"confidence": result.Confidence,
		"model":      result.ModelUsed,
		"latency_ms": result.LatencyMs,
		"type":       req.Type,
		"preview":    truncate(req.Content, 50),
	})

	return result
}

// GetStats 获取统计数据
func (s *ModerationService) GetStats() map[string]interface{} {
	return s.stats.snapshot()
}

// GetModels 获取模型列表
func (s *ModerationService) GetModels() []config.ModelConfig {
	return s.cfg.Models
}

// ── 模型调度 ───────────────────────────────────────────────

// buildModelQueue 构建模型调用队列
// auto = 加权随机选主力，其余按优先级排为 fallback
func (s *ModerationService) buildModelQueue(pref string) []config.ModelConfig {
	models := s.cfg.Models
	if len(models) == 0 {
		return nil
	}

	var primary config.ModelConfig
	found := false

	if pref != "" && pref != "auto" {
		for _, m := range models {
			if m.ID == pref {
				primary = m
				found = true
				break
			}
		}
	}

	if !found {
		primary = s.weightedRandom()
	}

	// 其余模型按优先级升序排列作为 fallback
	var rest []config.ModelConfig
	for _, m := range models {
		if m.ID != primary.ID {
			rest = append(rest, m)
		}
	}
	sort.Slice(rest, func(i, j int) bool {
		return rest[i].Priority < rest[j].Priority
	})

	return append([]config.ModelConfig{primary}, rest...)
}

// weightedRandom 按权重随机选模型
func (s *ModerationService) weightedRandom() config.ModelConfig {
	total := 0
	for _, m := range s.cfg.Models {
		total += m.Weight
	}
	r := rand.Intn(total)
	sum := 0
	for _, m := range s.cfg.Models {
		sum += m.Weight
		if r < sum {
			return m
		}
	}
	return s.cfg.Models[0]
}

// ── Anthropic API 调用 ─────────────────────────────────────

var strictnessPrompts = map[string]string{
	"strict":   "严格模式：对轻微违规也要标记（flagged），宁可误判不可漏判。",
	"standard": "标准模式：按正常社区规范，明确违规才处理。",
	"loose":    "宽松模式：只对严重明显违规（rejected），模糊内容放行。",
}

func (s *ModerationService) callAPI(req *ModerateRequest, modelID string) (*ModerateResult, error) {
	strictHint := strictnessPrompts[req.Strictness]
	if strictHint == "" {
		strictHint = strictnessPrompts["standard"]
	}

	systemPrompt := fmt.Sprintf(`你是一个专业的中文社区内容审核 AI。%s

审核分类（category 字段）：
- none     : 正常内容
- spam     : 垃圾广告、营销推广、外部链接刷量
- abuse    : 侮辱谩骂、人身攻击、骚扰恐吓
- politics : 政治敏感、违禁话题、煽动性内容
- adult    : 色情低俗内容
- fraud    : 欺诈诱导、虚假信息
- violence : 暴力血腥

判断规则：
- approved : 内容正常，直接发布
- flagged  : 存在疑虑，转人工复核
- rejected : 明确违规，直接拒绝

只返回以下 JSON，不要任何其他文字：
{"verdict":"approved|flagged|rejected","category":"none|spam|abuse|politics|adult|fraud|violence","confidence":0.95,"reason":"简短原因15字内"}`, strictHint)

	payload := anthropicRequest{
		Model:     modelID,
		MaxTokens: 200,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("内容类型：%s\n\n待审内容：\n%s", req.Type, req.Content),
			},
		},
	}

	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.APITimeout)*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.AnthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", s.cfg.AnthropicAPIKey)
	httpReq.Header.Set("anthropic-version", s.cfg.AnthropicVer)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应 JSON 失败: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API 错误 [%s]: %s", apiResp.Error.Type, apiResp.Error.Message)
	}
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("API 返回空内容")
	}

	// 解析 AI 返回的 JSON
	text := apiResp.Content[0].Text
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.TrimSpace(text)

	var ai aiResult
	if err := json.Unmarshal([]byte(text), &ai); err != nil {
		return nil, fmt.Errorf("解析 AI 结果失败: %w (原始: %s)", err, text)
	}

	return &ModerateResult{
		Verdict:    ai.Verdict,
		Category:   ai.Category,
		Confidence: ai.Confidence,
		Reason:     ai.Reason,
		ModelUsed:  modelID,
	}, nil
}

// ── 工具函数 ───────────────────────────────────────────────

func (s *ModerationService) cacheKey(content, typ, strictness string) string {
	raw := fmt.Sprintf("%s|%s|%s", content, typ, strictness)
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

func (s *ModerationService) safeFallback(model string, ms int64) *ModerateResult {
	return &ModerateResult{
		Verdict:    "flagged",
		Category:   "none",
		Confidence: 0,
		Reason:     "审核服务异常，已转人工队列",
		ModelUsed:  "fallback",
		LatencyMs:  ms,
		Fallback:   true,
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
