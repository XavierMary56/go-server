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
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

type ModerateRequest struct {
	Content    string                 `json:"content"`
	Type       string                 `json:"type"`
	Model      string                 `json:"model"`
	Strictness string                 `json:"strictness"`
	WebhookURL string                 `json:"webhook_url,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

type ModerateResult struct {
	Verdict    string  `json:"verdict"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	ModelUsed  string  `json:"model_used"`
	LatencyMs  int64   `json:"latency_ms"`
	FromCache  bool    `json:"from_cache"`
	Fallback   bool    `json:"fallback,omitempty"`
}

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

type aiResult struct {
	Verdict    string  `json:"verdict"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

type openAIRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

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

type ModerationService struct {
	cfg    *config.Config
	log    *logger.Logger
	cache  Cache
	client *http.Client
	stats  *Stats
	mu     sync.RWMutex
	db     *storage.DB
	calls  map[string]*inflightCall
}

type inflightCall struct {
	done   chan struct{}
	result *ModerateResult
}

type Stats struct {
	mu                sync.RWMutex
	Total             int64
	Approved          int64
	Flagged           int64
	Rejected          int64
	ModelCounts       map[string]int64
	FalsePositives    int64  // 用户反馈的误拦数
	FalseNegatives    int64  // 用户反馈的漏检数
	RecentFalseReports []string  // 最近的误拦反馈样本（用于调试）
}

func newStats() *Stats {
	return &Stats{
		ModelCounts:       make(map[string]int64),
		RecentFalseReports: make([]string, 0, 100),
	}
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

// RecordFalsePositive 记录用户报告的误拦案例
func (s *Stats) RecordFalsePositive(content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.FalsePositives++

	// 保存最近的 100 个误拦案例样本（用于分析改进）
	if len(s.RecentFalseReports) >= 100 {
		s.RecentFalseReports = s.RecentFalseReports[1:]
	}
	s.RecentFalseReports = append(s.RecentFalseReports, content)
}

// RecordFalseNegative 记录用户报告的漏检案例
func (s *Stats) RecordFalseNegative() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FalseNegatives++
}

func (s *Stats) snapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int64, len(s.ModelCounts))
	for k, v := range s.ModelCounts {
		counts[k] = v
	}

	return map[string]interface{}{
		"total":             s.Total,
		"approved":          s.Approved,
		"flagged":           s.Flagged,
		"rejected":          s.Rejected,
		"false_positives":   s.FalsePositives,
		"false_negatives":   s.FalseNegatives,
		"model_counts":      counts,
	}
}

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
		calls: make(map[string]*inflightCall),
	}
}

func providerOf(modelID string) string {
	switch {
	case strings.HasPrefix(modelID, "gpt-"),
		strings.HasPrefix(modelID, "o1-"),
		strings.HasPrefix(modelID, "o3-"),
		strings.HasPrefix(modelID, "o4-"):
		return "openai"
	case strings.HasPrefix(modelID, "grok-"):
		return "grok"
	default:
		return "anthropic"
	}
}

func (s *ModerationService) getActiveModels() []config.ModelConfig {
	fallbackModels := func() []config.ModelConfig {
		if !s.cfg.EnableModelConfigFallback {
			return nil
		}
		return append([]config.ModelConfig(nil), s.cfg.Models...)
	}

	if s.db == nil {
		return fallbackModels()
	}

	dbModels, err := s.db.ListModels()
	if err != nil || len(dbModels) == 0 {
		return fallbackModels()
	}

	models := make([]config.ModelConfig, 0, len(dbModels))
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

	if len(models) == 0 {
		return fallbackModels()
	}

	return models
}

func (s *ModerationService) getProviderKey(provider string) (string, int64) {
	if s.db != nil {
		switch provider {
		case "openai", "grok":
			keys, _ := s.db.GetEnabledProviderKeys(provider)
			if len(keys) > 0 {
				return keys[0].Key, keys[0].ID
			}
		case "anthropic":
			keys, _ := s.db.GetEnabledAnthropicKeys()
			if len(keys) > 0 {
				return keys[0].Key, keys[0].ID
			}
		}
	}

	switch provider {
	case "openai":
		return s.cfg.OpenAIAPIKey, 0
	case "grok":
		return s.cfg.GrokAPIKey, 0
	default:
		return s.cfg.AnthropicAPIKey, 0
	}
}

func (s *ModerationService) Moderate(req *ModerateRequest) *ModerateResult {
	s.applyRequestDefaults(req)

	auditContent := s.buildAuditContent(req)
	if strings.TrimSpace(auditContent) == "" {
		auditContent = req.Content
	}

	if hardResult := applyHardBlockRules(auditContent); hardResult != nil {
		hardResult.FromCache = false
		s.stats.record(hardResult.Verdict, hardResult.ModelUsed)
		return hardResult
	}

	cacheKey := s.cacheKey(auditContent, req.Type, req.Strictness)
	if cached, ok := s.cache.Get(cacheKey); ok {
		result := cloneModerateResult(cached.(*ModerateResult))
		result.FromCache = true
		return result
	}

	call, leader := s.beginInflight(cacheKey)
	if !leader {
		<-call.done
		return cloneModerateResult(call.result)
	}
	defer s.finishInflight(cacheKey, call)

	queue := s.buildModelQueue(req.Model)
	start := time.Now()
	if len(queue) == 0 {
		result := s.safeFallback(auditContent, "none", 0)
		call.result = cloneModerateResult(result)
		s.stats.record(result.Verdict, result.ModelUsed)
		return result
	}

	var (
		result  *ModerateResult
		lastErr error
	)

	for _, model := range queue {
		result, lastErr = s.callAPI(req, model.ID)
		if lastErr == nil {
			break
		}
		s.log.Warn(fmt.Sprintf("model %s failed: %v, trying next", model.ID, lastErr))
	}

	if lastErr != nil || result == nil {
		errMsg := "all models failed"
		if lastErr != nil {
			errMsg = "all models failed: " + lastErr.Error()
		}
		s.log.Error(errMsg)
		result = s.safeFallback(auditContent, queue[0].ID, time.Since(start).Milliseconds())
		call.result = cloneModerateResult(result)
		s.stats.record(result.Verdict, result.ModelUsed)
		return result
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	result.FromCache = false

	s.cache.Set(cacheKey, cloneModerateResult(result))
	s.stats.record(result.Verdict, result.ModelUsed)
	s.log.Info("moderation", map[string]interface{}{
		"verdict":    result.Verdict,
		"category":   result.Category,
		"confidence": result.Confidence,
		"model":      result.ModelUsed,
		"latency_ms": result.LatencyMs,
		"type":       req.Type,
		"preview":    truncate(auditContent, 80),
	})

	call.result = cloneModerateResult(result)
	return result
}

func (s *ModerationService) beginInflight(cacheKey string) (*inflightCall, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if call, ok := s.calls[cacheKey]; ok {
		return call, false
	}

	call := &inflightCall{done: make(chan struct{})}
	s.calls[cacheKey] = call
	return call, true
}

func (s *ModerationService) finishInflight(cacheKey string, call *inflightCall) {
	s.mu.Lock()
	delete(s.calls, cacheKey)
	s.mu.Unlock()
	close(call.done)
}

func (s *ModerationService) applyRequestDefaults(req *ModerateRequest) {
	if req.Type == "" {
		req.Type = "post"
	}
	if req.Strictness == "" {
		req.Strictness = "standard"
	}
	if req.Model == "" {
		req.Model = "auto"
	}
}

func (s *ModerationService) GetStats() map[string]interface{} {
	return s.stats.snapshot()
}

func (s *ModerationService) GetModels() []config.ModelConfig {
	return s.getActiveModels()
}

func (s *ModerationService) buildModelQueue(pref string) []config.ModelConfig {
	models := s.getActiveModels()
	if len(models) == 0 {
		return nil
	}

	var (
		primary config.ModelConfig
		found   bool
	)

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

	rest := make([]config.ModelConfig, 0, len(models)-1)
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

func (s *ModerationService) weightedRandom(models []config.ModelConfig) config.ModelConfig {
	total := 0
	for _, m := range models {
		total += m.Weight
	}
	if total <= 0 {
		return models[0]
	}

	target := rand.Intn(total)
	current := 0
	for _, m := range models {
		current += m.Weight
		if target < current {
			return m
		}
	}

	return models[0]
}

var strictnessPrompts = map[string]string{
	"strict":   "Apply the strictest standard. If there is obvious risk or strong suspicion, reject. Do not give risky content the benefit of the doubt.",
	"standard": "Apply the standard production policy. Clear violations must be rejected. Normal discussion should pass. Borderline risky content may be flagged only when you cannot determine clearly.",
	"loose":    "Apply the loosest standard. Only clear violations should be rejected. Normal discussion should pass.",
}

func (s *ModerationService) callAPI(req *ModerateRequest, modelID string) (*ModerateResult, error) {
	switch providerOf(modelID) {
	case "openai", "grok":
		return s.callOpenAICompatible(req, modelID, providerOf(modelID))
	default:
		return s.callAnthropic(req, modelID)
	}
}

func (s *ModerationService) buildSystemPrompt(req *ModerateRequest) string {
	strictHint := strictnessPrompts[req.Strictness]
	if strictHint == "" {
		strictHint = strictnessPrompts["standard"]
	}

	return fmt.Sprintf(`You are a multilingual content moderation model for user generated posts and comments.

Your job is to review the full content package and classify it with one final result.
The supported languages include Chinese, English, Russian, and mixed-language or obfuscated text.

Core moderation policy:
1. Reject any clear political-sensitive content, including coup, rebellion, separatism, independence movement, illegal protest, riot, anti-government agitation, political leaks, government scandal leaks, or other high-risk political mobilization or disruption content.
2. Reject any severe sexual exploitation, adult solicitation, prostitution, escort service, rape, gang rape, child sexual content, underage sexual content, drug-assisted sexual assault, or explicit off-platform adult diversion.
3. Reject any scam, fraud, gambling, phishing, black-market, stolen card, fake account, ponzi-like content, money laundering, carding, account selling, or other gray-market promotion.
4. Reject any drugs, narcotics, marijuana, meth, heroin, cocaine, ketamine, fentanyl, MDMA, LSD, drug trading, drug manufacturing, or prohibited-goods content.
5. Reject any killing, assassination, bomb making, explosives, terrorism, firearm trading, weapon sales, ammunition trading, detonators, or homemade weapon content.
6. Reject any ad, lead generation, contact information, off-platform diversion, URL, domain, email, QR-style diversion text, private chat diversion, group invitation, or gray-industry promotion.
7. Normal discussion should pass when it does not contain the violation classes above.

Important detection requirements:
- Detect direct keywords and also obfuscated variants, such as split words, inserted punctuation, spaces, symbols, mixed scripts, abbreviations, transliteration, or homophone-like substitutions.
- Treat URLs, domains, short links, email addresses, phone-like numbers, QQ, WeChat, VX, WX, TG, Telegram, WhatsApp, Line, Discord, Skype, group invitations, and off-platform contact requests as spam/diversion. Reject even if it's just a single keyword (e.g., 'QQ', 'TG', 'Telegram') or a single URL (e.g., 'xxx.cc', 'http://xxx') without any other text.
- Treat prefixes and domain fragments such as http://, https://, www., .com, .cn, .net, .org, .ru, .cc, .xyz, .top, .io, .me, t.me, discord.gg, bit.ly, and tinyurl as strong diversion signals.
- Treat phrases such as "加我", "联系我", "私聊", "主页联系", "看头像", "扫码", "二维码", "点击链接", "进群", "群号", "DM me", "contact me", "message me privately", "add me on Telegram", "add me on WhatsApp", "напиши в личку", and "перейди по ссылке" as spam/diversion signals when they indicate off-platform contact or transaction.
- If the content says there is no contact information or no diversion, do not punish it only because those words appear in a negated form.
- Pure adult discussion alone is not enough to reject unless it also contains solicitation, sexual transaction, exploitation, minors, rape, or diversion/contact information.
- Broad political words alone, such as government, president, parliament, or election, are not enough by themselves unless the content also expresses high-risk political sensitivity, mobilization, separatism, protest, leaks, or similar risky context.

Category mapping:
- politics: coup, rebellion, separatism, independence, illegal protest, riot, anti-government agitation, political leak, government scandal leak, or other high-risk political-sensitive content
- adult: adult solicitation, prostitution, escort service, child sexual content, underage sex, rape, sexual exploitation, explicit adult diversion
- fraud: scam, phishing, gambling, betting, casino, fake account, stolen card, ponzi, guaranteed profit, money laundering, carding, account selling
- abuse: drugs, narcotics, meth, heroin, cocaine, marijuana, ketamine, fentanyl, MDMA, LSD, drug trading, drug manufacturing
- violence: killing, assassination, bomb making, explosives, terror attack, gun sale, weapon sale, firearm sale, ammo sale
- spam: ads, lead generation, contact information, URLs, domains, off-platform diversion, private transaction, group invitation
- none: normal content with no policy hit

Verdict rules:
- rejected: clear hit of any category above
- flagged: suspicious but not conclusive
- approved: normal content with no meaningful violation signal

Reason rules:
- reason must be short, specific, and in Chinese
- reason should describe the matched violation type directly
- do not mention uncertainty when the content clearly hits a category

Output rules:
- return JSON only
- do not add markdown
- do not add explanation outside JSON
- use exactly this schema:
{"verdict":"approved|flagged|rejected","category":"none|spam|abuse|politics|adult|fraud|violence","confidence":0.95,"reason":"简短中文原因"}

%s`, strictHint)
}

func (s *ModerationService) buildAuditContent(req *ModerateRequest) string {
	parts := make([]string, 0, 4)

	if content := strings.TrimSpace(req.Content); content != "" {
		parts = append(parts, "review body:\n"+content)
	}

	if req.Context == nil {
		return strings.Join(parts, "\n\n")
	}

	if scene, ok := req.Context["scene"].(string); ok && strings.TrimSpace(scene) != "" {
		parts = append(parts, "scene: "+strings.TrimSpace(scene))
	}

	payload, ok := req.Context["payload"].(map[string]interface{})
	if !ok {
		return strings.Join(parts, "\n\n")
	}

	if title, ok := payload["title"].(string); ok && strings.TrimSpace(title) != "" {
		parts = append(parts, "title:\n"+strings.TrimSpace(title))
	}
	if body, ok := payload["content"].(string); ok && strings.TrimSpace(body) != "" {
		parts = append(parts, "body:\n"+strings.TrimSpace(body))
	}

	return strings.Join(parts, "\n\n")
}

func parseAIText(text string) (*aiResult, error) {
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.TrimSpace(text)

	var ai aiResult
	if err := json.Unmarshal([]byte(text), &ai); err != nil {
		return nil, fmt.Errorf("failed to parse AI result: %w (raw: %s)", err, text)
	}
	return &ai, nil
}

func buildUserPrompt(contentType, auditContent string) string {
	return fmt.Sprintf("content_type: %s\n\ncontent:\n%s", contentType, auditContent)
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

func (s *ModerationService) callAnthropic(req *ModerateRequest, modelID string) (*ModerateResult, error) {
	apiKey, keyID := s.getProviderKey("anthropic")
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is not configured")
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
		return nil, fmt.Errorf("failed to create Anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", s.cfg.AnthropicVer)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Anthropic HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		s.MarkAnthropicKeyUnhealthy(keyID)
		return nil, fmt.Errorf("Anthropic key auth failed with HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Anthropic response: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("Anthropic API error [%s]: %s", apiResp.Error.Type, apiResp.Error.Message)
	}
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("Anthropic API returned empty content")
	}

	if keyID > 0 && s.db != nil {
		_ = s.db.IncrAnthropicKeyUsage(keyID)
	}

	ai, err := parseAIText(apiResp.Content[0].Text)
	if err != nil {
		return nil, err
	}
	ai = normalizeModelDecision(ai, auditContent)
	return newModerateResult(ai, modelID), nil
}

func (s *ModerationService) callOpenAICompatible(req *ModerateRequest, modelID, provider string) (*ModerateResult, error) {
	apiKey, keyID := s.getProviderKey(provider)
	if apiKey == "" {
		return nil, fmt.Errorf("%s API key is not configured", provider)
	}

	auditContent := s.buildAuditContent(req)
	apiURL := s.cfg.OpenAIAPIURL
	if provider == "grok" {
		apiURL = s.cfg.GrokAPIURL
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
		return nil, fmt.Errorf("failed to create %s request: %w", provider, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s HTTP request failed: %w", provider, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		s.MarkProviderKeyUnhealthy(keyID)
		return nil, fmt.Errorf("%s key auth failed with HTTP %d", provider, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s API returned HTTP %d: %s", provider, resp.StatusCode, string(respBody))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", provider, err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("%s API error [%s]: %s", provider, apiResp.Error.Type, apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("%s API returned empty choices", provider)
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

func (s *ModerationService) cacheKey(content, typ, strictness string) string {
	raw := fmt.Sprintf("%s|%s|%s", content, typ, strictness)
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

func (s *ModerationService) safeFallback(auditContent, model string, ms int64) *ModerateResult {
	if hardResult := applyHardBlockRules(auditContent); hardResult != nil {
		hardResult.LatencyMs = ms
		hardResult.Fallback = true
		return hardResult
	}

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

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
