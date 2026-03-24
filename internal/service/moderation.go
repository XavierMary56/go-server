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
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
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
	Verdict    string  `json:"verdict"`    // approved | flagged | rejected
	Category   string  `json:"category"`   // none | spam | abuse | politics | adult | fraud | violence
	Confidence float64 `json:"confidence"` // 0.0 ~ 1.0
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

// openAIRequest OpenAI / Grok 请求体（兼容格式）
type openAIRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse OpenAI / Grok 响应体
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
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
	db     *storage.DB // 可选，用于从数据库获取 Provider Keys
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
func NewModerationService(cfg *config.Config, log *logger.Logger, db *storage.DB) *ModerationService {
	return &ModerationService{
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: newMemoryCache(cfg.CacheTTL),
		client: &http.Client{
			Timeout: time.Duration(cfg.APITimeout) * time.Second,
		},
		stats: newStats(),
	}
}

// providerOf 根据模型 ID 前缀判断所属 provider
func providerOf(modelID string) string {
	switch {
	case strings.HasPrefix(modelID, "gpt-"), strings.HasPrefix(modelID, "o1-"),
		strings.HasPrefix(modelID, "o3-"), strings.HasPrefix(modelID, "o4-"):
		return "openai"
	case strings.HasPrefix(modelID, "grok-"):
		return "grok"
	default:
		return "anthropic"
	}
}

// getActiveModels 从 DB 读取启用的模型列表，转为内部类型
func (s *ModerationService) getActiveModels() []config.ModelConfig {
	if s.db == nil {
		return nil
	}
	dbModels, err := s.db.ListModels()
	if err != nil || len(dbModels) == 0 {
		return nil
	}
	var models []config.ModelConfig
	for _, m := range dbModels {
		if !m.Enabled {
			continue
		}
		provider := m.Provider
		if provider == "" {
			provider = providerOf(m.ModelID)
		}
		models = append(models, config.ModelConfig{
			ID:       m.ModelID,
			Name:     m.Name,
			Weight:   m.Weight,
			Priority: m.Priority,
			Provider: provider,
		})
	}
	return models
}

// getProviderKey 获取指定 provider 的 API Key（优先 DB，回退 env var）
func (s *ModerationService) getProviderKey(provider string) (apiKey string, keyID int64) {
	if s.db != nil {
		switch provider {
		case "openai", "grok":
			keys, _ := s.db.GetEnabledProviderKeys(provider)
			if len(keys) > 0 {
				k := keys[0] // usage_count ASC，取最少使用的
				return k.Key, k.ID
			}
		case "anthropic":
			keys, _ := s.db.GetEnabledAnthropicKeys()
			if len(keys) > 0 {
				k := keys[0]
				return k.Key, k.ID
			}
		}
	}
	// fallback 到环境变量
	switch provider {
	case "openai":
		return s.cfg.OpenAIAPIKey, 0
	case "grok":
		return s.cfg.GrokAPIKey, 0
	default:
		return s.cfg.AnthropicAPIKey, 0
	}
}

// ── 审核入口 ───────────────────────────────────────────────

// Moderate 同步审核
func (s *ModerationService) Moderate(req *ModerateRequest) *ModerateResult {
	s.applyRequestDefaults(req)

	auditContent := s.buildAuditContent(req)
	if strings.TrimSpace(auditContent) == "" {
		auditContent = req.Content
	}

	// 命中缓存直接返回
	cacheKey := s.cacheKey(auditContent, req.Type, req.Strictness)
	if cached, ok := s.cache.Get(cacheKey); ok {
		result := cloneModerateResult(cached.(*ModerateResult))
		result.FromCache = true
		return result
	}

	// 构建模型调用队列（首选 + 故障转移备选）
	queue := s.buildModelQueue(req.Model)
	start := time.Now()
	if len(queue) == 0 {
		return s.safeFallback(auditContent, "none", 0)
	}

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
		return s.safeFallback(auditContent, queue[0].ID, time.Since(start).Milliseconds())
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
		"preview":    truncate(auditContent, 50),
	})

	return result
}

func (s *ModerationService) applyRequestDefaults(req *ModerateRequest) {
	if req.Type == "" {
		req.Type = "post"
	}
	if req.Strictness == "" {
		req.Strictness = "standard"
	}
}

// GetStats 获取统计数据
func (s *ModerationService) GetStats() map[string]interface{} {
	return s.stats.snapshot()
}

// GetModels 获取模型列表
func (s *ModerationService) GetModels() []config.ModelConfig {
	return s.getActiveModels()
}

// ── 模型调度 ───────────────────────────────────────────────

// buildModelQueue 构建模型调用队列
// auto = 加权随机选主力，其余按优先级排为 fallback
func (s *ModerationService) buildModelQueue(pref string) []config.ModelConfig {
	models := s.getActiveModels()
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
		primary = s.weightedRandom(models)
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
func (s *ModerationService) weightedRandom(models []config.ModelConfig) config.ModelConfig {
	total := 0
	for _, m := range models {
		total += m.Weight
	}
	if total == 0 {
		return models[0]
	}
	r := rand.Intn(total)
	sum := 0
	for _, m := range models {
		sum += m.Weight
		if r < sum {
			return m
		}
	}
	return models[0]
}

// ── API 调用路由 ───────────────────────────────────────────

var strictnessPrompts = map[string]string{
	"strict":   "严格模式：对轻微违规也要标记（flagged），宁可误判不可漏判。",
	"standard": "标准模式：按正常社区规范，明确违规才处理。",
	"loose":    "宽松模式：只对严重明显违规（rejected），模糊内容放行。",
}

func (s *ModerationService) callAPI(req *ModerateRequest, modelID string) (*ModerateResult, error) {
	provider := providerOf(modelID)
	switch provider {
	case "openai", "grok":
		return s.callOpenAICompatible(req, modelID, provider)
	default:
		return s.callAnthropic(req, modelID)
	}
}

// buildSystemPrompt 构建统一的审核系统提示词
func (s *ModerationService) buildSystemPrompt(req *ModerateRequest) string {
	strictHint := strictnessPrompts[req.Strictness]
	if strictHint == "" {
		strictHint = strictnessPrompts["standard"]
	}
	return fmt.Sprintf(`你是一个专业的中文社区内容审核 AI。%s

本次审核目标要非常聚焦：
1. 重点拦截广告、营销推广、站外引流、联系方式。
2. 除上述内容外，其他普通社区内容默认通过，不要因为语气、争议、擦边或其他无关因素误杀。
3. 标题、正文、附加上下文里只要出现联系方式或明显导流信息，都要优先判定。

重点识别内容包括但不限于：
- QQ、微信、vx、v、tg、telegram、飞机、群号、手机号、邮箱、二维码、网址、外链
- “加我”“联系我”“私聊我”“拉群”“代理加盟”“购买资源”“出售账号”“看片加群”“站外交易”等导流表达
- 变形写法、拆字、空格分隔、谐音、符号夹杂的联系方式

审核分类（category 字段）：
- none     : 正常内容
- spam     : 广告、营销推广、导流、联系方式、站外跳转
- fraud    : 欺诈诱导、虚假交易、诈骗式引流
- abuse    : 仅在极端辱骂骚扰时使用，否则默认放行
- politics : 默认不要使用，普通内容放行
- adult    : 默认不要使用，普通内容放行
- violence : 默认不要使用，普通内容放行

判断规则：
- approved : 不含广告、导流、联系方式，直接发布
- flagged  : 疑似广告或疑似联系方式，但证据不够明确，转人工复核
- rejected : 明确包含广告、导流、联系方式或明显站外交易信息，直接拒绝

输出要求：
- 优先保证“广告/联系方式不漏拦”
- 同时尽量减少误伤普通内容
- reason 用简短中文说明命中的广告或联系方式特征

只返回以下 JSON，不要任何其他文字：
{"verdict":"approved|flagged|rejected","category":"none|spam|abuse|politics|adult|fraud|violence","confidence":0.95,"reason":"简短原因15字内"}`, strictHint)
}

func (s *ModerationService) buildAuditContent(req *ModerateRequest) string {
	var parts []string

	if content := strings.TrimSpace(req.Content); content != "" {
		parts = append(parts, "主内容：\n"+content)
	}

	scene := ""
	if req.Context != nil {
		if value, ok := req.Context["scene"].(string); ok {
			scene = strings.TrimSpace(value)
		}
	}
	if scene != "" {
		parts = append(parts, "场景："+scene)
	}

	payload, ok := req.Context["payload"].(map[string]interface{})
	if !ok {
		return strings.Join(parts, "\n\n")
	}

	if title, ok := payload["title"].(string); ok && strings.TrimSpace(title) != "" {
		parts = append(parts, "标题：\n"+strings.TrimSpace(title))
	}
	if body, ok := payload["content"].(string); ok && strings.TrimSpace(body) != "" {
		parts = append(parts, "正文：\n"+strings.TrimSpace(body))
	}

	return strings.Join(parts, "\n\n")
}

// parseAIText 解析 AI 返回的文本为 aiResult
func parseAIText(text string) (*aiResult, error) {
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.TrimSpace(text)
	var ai aiResult
	if err := json.Unmarshal([]byte(text), &ai); err != nil {
		return nil, fmt.Errorf("解析 AI 结果失败: %w (原始: %s)", err, text)
	}
	return &ai, nil
}

func buildUserPrompt(contentType, auditContent string) string {
	return fmt.Sprintf("内容类型：%s\n\n待审内容：\n%s", contentType, auditContent)
}

func cloneModerateResult(result *ModerateResult) *ModerateResult {
	if result == nil {
		return nil
	}

	cloned := *result
	return &cloned
}

func newModerateResult(ai *aiResult, modelID string) *ModerateResult {
	return &ModerateResult{
		Verdict:    ai.Verdict,
		Category:   ai.Category,
		Confidence: ai.Confidence,
		Reason:     ai.Reason,
		ModelUsed:  modelID,
	}
}

// 成人内容不是当前项目的主拦截目标；未命中广告/联系方式时，避免被模型误判拦截。
func normalizeModelDecision(ai *aiResult, auditContent string) *aiResult {
	if ai == nil {
		return nil
	}

	normalized := *ai
	if normalized.Category == "adult" && !looksLikeAdOrContact(auditContent) {
		normalized.Verdict = "approved"
		normalized.Category = "none"
		normalized.Reason = "普通内容，未命中广告联系方式"
	}

	return &normalized
}

// callAnthropic 调用 Anthropic API
func (s *ModerationService) callAnthropic(req *ModerateRequest, modelID string) (*ModerateResult, error) {
	apiKey, keyID := s.getProviderKey("anthropic")
	if apiKey == "" {
		return nil, fmt.Errorf("未配置 Anthropic API Key")
	}

	auditContent := s.buildAuditContent(req)

	payload := anthropicRequest{
		Model:     modelID,
		MaxTokens: 200,
		System:    s.buildSystemPrompt(req),
		Messages: []anthropicMessage{
			{Role: "user", Content: buildUserPrompt(req.Type, auditContent)},
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
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", s.cfg.AnthropicVer)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		s.MarkAnthropicKeyUnhealthy(keyID)
		return nil, fmt.Errorf("Anthropic key 认证失败 (HTTP %d)，已标记为不可用", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API 错误 [%s]: %s", apiResp.Error.Type, apiResp.Error.Message)
	}
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("API 返回空内容")
	}

	if keyID > 0 && s.db != nil {
		s.db.IncrAnthropicKeyUsage(keyID)
	}

	ai, err := parseAIText(apiResp.Content[0].Text)
	if err != nil {
		return nil, err
	}
	ai = normalizeModelDecision(ai, auditContent)
	return newModerateResult(ai, modelID), nil
}

// callOpenAICompatible 调用 OpenAI / Grok API（兼容格式相同）
func (s *ModerationService) callOpenAICompatible(req *ModerateRequest, modelID, provider string) (*ModerateResult, error) {
	apiKey, keyID := s.getProviderKey(provider)
	if apiKey == "" {
		return nil, fmt.Errorf("未配置 %s API Key", provider)
	}

	auditContent := s.buildAuditContent(req)

	var apiURL string
	switch provider {
	case "grok":
		apiURL = s.cfg.GrokAPIURL
	default:
		apiURL = s.cfg.OpenAIAPIURL
	}

	payload := openAIRequest{
		Model:     modelID,
		MaxTokens: 200,
		Messages: []openAIMessage{
			{Role: "system", Content: s.buildSystemPrompt(req)},
			{Role: "user", Content: buildUserPrompt(req.Type, auditContent)},
		},
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.APITimeout)*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		s.MarkProviderKeyUnhealthy(keyID)
		return nil, fmt.Errorf("%s key 认证失败 (HTTP %d)，已标记为不可用", provider, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API 错误 [%s]: %s", apiResp.Error.Type, apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("API 返回空 choices")
	}

	if keyID > 0 && s.db != nil {
		s.db.IncrProviderKeyUsage(keyID)
	}

	ai, err := parseAIText(apiResp.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}
	ai = normalizeModelDecision(ai, auditContent)
	return newModerateResult(ai, modelID), nil
}

// ── 工具函数 ───────────────────────────────────────────────

func (s *ModerationService) cacheKey(content, typ, strictness string) string {
	raw := fmt.Sprintf("%s|%s|%s", content, typ, strictness)
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

func (s *ModerationService) safeFallback(auditContent, model string, ms int64) *ModerateResult {
	if looksLikeAdOrContact(auditContent) {
		return &ModerateResult{
			Verdict:    "rejected",
			Category:   "spam",
			Confidence: 0.99,
			Reason:     "包含联系方式或导流信息",
			ModelUsed:  "rule-fallback",
			LatencyMs:  ms,
			Fallback:   true,
		}
	}

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

func looksLikeAdOrContact(content string) bool {
	normalized := strings.ToLower(content)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.ReplaceAll(normalized, "\r", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")

	keywords := []string{
		"微信", "vx", "vx号", "qq", "telegram", "tg", "whatsapp", "line", "discord", "skype",
		"邮箱", "email", "加我", "联系我", "私聊", "拉群", "群号", "代理", "加盟", "引流",
		"外链", "网址", "链接", "下载地址", "扫码", "二维码",
	}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`[1-9][0-9]{5,}`),
		regexp.MustCompile(`[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`),
		regexp.MustCompile(`https?://[^\s]+`),
		regexp.MustCompile(`([a-z0-9\-]+\.)+[a-z]{2,}`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(normalized) {
			return true
		}
	}

	return false
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
