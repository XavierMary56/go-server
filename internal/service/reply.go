package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReplyResult 自动回复生成结果
type ReplyResult struct {
	Content   string `json:"content"`
	ModelUsed string `json:"model_used"`
	LatencyMs int64  `json:"latency_ms"`
}

// GenerateReply 根据评论内容生成自动回复
// 复用现有模型调度和 provider 密钥，优先用轻量模型
func (s *ModerationService) GenerateReply(content string, replyContext map[string]string, style string) (*ReplyResult, error) {
	start := time.Now()

	if style == "" {
		style = "friendly"
	}

	prompt := buildReplyPrompt(content, replyContext, style)

	queue := s.getActiveModels()
	if len(queue) == 0 {
		return nil, fmt.Errorf("no active models available")
	}

	var lastErr error
	for _, model := range queue {
		provider := model.Provider
		if provider == "" {
			provider = providerOf(model.ID)
		}

		apiKey, _ := s.getProviderKey(provider)
		if apiKey == "" {
			continue
		}

		reply, err := s.callReplyAPI(provider, model.ID, apiKey, prompt)
		if err != nil {
			lastErr = err
			s.log.Warn(fmt.Sprintf("reply model %s failed: %v, trying next", model.ID, err))
			continue
		}

		return &ReplyResult{
			Content:   reply,
			ModelUsed: model.ID,
			LatencyMs: time.Since(start).Milliseconds(),
		}, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all models failed for reply: %w", lastErr)
	}
	return nil, fmt.Errorf("no models available for reply")
}

func buildReplyPrompt(content string, ctx map[string]string, style string) string {
	styleDesc := map[string]string{
		"friendly": "友好、亲切、简短、自然",
		"formal":   "正式、礼貌、得体",
		"humorous": "幽默、轻松、有趣",
	}

	desc := styleDesc[style]
	if desc == "" {
		desc = styleDesc["friendly"]
	}

	contentType := ""
	contentTitle := ""
	if ctx != nil {
		contentType = ctx["type"]
		contentTitle = ctx["content_title"]
	}

	return fmt.Sprintf(`你是一个社区互动助手。请根据用户评论生成一条简短的中文回复。

要求：
1. 风格：%s
2. 长度：10-30字，绝对不超过50字
3. 不要重复用户原话
4. 不要包含任何链接、联系方式、广告
5. 不要使用 emoji
6. 直接输出回复文字，不要加引号、前缀、解释

评论所属类型：%s
评论所属标题：%s

用户评论：%s`, desc, contentType, contentTitle, content)
}

// callReplyAPI 调用 AI 模型生成回复文本
func (s *ModerationService) callReplyAPI(provider, modelID, apiKey, prompt string) (string, error) {
	switch provider {
	case "openai", "grok":
		return s.callReplyOpenAI(provider, modelID, apiKey, prompt)
	default:
		return s.callReplyAnthropic(modelID, apiKey, prompt)
	}
}

func (s *ModerationService) callReplyAnthropic(modelID, apiKey, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     modelID,
		MaxTokens: 100,
		System:    "你是一个友好的社区互动助手，只输出回复文字，不加任何格式。",
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal error: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("api error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	reply := strings.TrimSpace(result.Content[0].Text)
	reply = trimQuotes(reply)
	return reply, nil
}

func (s *ModerationService) callReplyOpenAI(provider, modelID, apiKey, prompt string) (string, error) {
	reqBody := openAIRequest{
		Model:     modelID,
		MaxTokens: 100,
		Messages: []openAIMessage{
			{Role: "system", Content: "你是一个友好的社区互动助手，只输出回复文字，不加任何格式。"},
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	apiURL := "https://api.openai.com/v1/chat/completions"
	if provider == "grok" {
		apiURL = "https://api.x.ai/v1/chat/completions"
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal error: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("api error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	reply := strings.TrimSpace(result.Choices[0].Message.Content)
	reply = trimQuotes(reply)
	return reply, nil
}

// trimQuotes 去掉 AI 回复可能带的引号包裹
func trimQuotes(s string) string {
	cutset := "\"'\u201c\u201d\u2018\u2019\u300c\u300d"
	return strings.Trim(s, cutset)
}
