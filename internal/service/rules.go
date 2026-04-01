package service

import (
	"regexp"
	"strings"
)

var (
	directContactRawPatterns = []*regexp.Regexp{
		regexp.MustCompile(`https?://[^\s]+`),
		regexp.MustCompile(`hxxps?://[^\s]+`),
		regexp.MustCompile(`www\.[^\s]+`),
		regexp.MustCompile(`t\.me/[^\s]+`),
		regexp.MustCompile(`discord\.gg/[^\s]+`),
		regexp.MustCompile(`bit\.ly/[^\s]+`),
		regexp.MustCompile(`tinyurl\.com/[^\s]+`),
		regexp.MustCompile(`(?:^|[\s(])@[a-z0-9_]{5,}\b`),
		regexp.MustCompile(`([a-z0-9\-]+)(?:\.|\[\.\])+(com|cn|net|org|ru|cc|xyz|top|info|io|co|tv|me|biz|vip|app|link|shop|live|site|fun|pro|club|online|store|cloud|test|gg|ly)\b`),
		regexp.MustCompile(`([a-z0-9\-]+\.)+(com|cn|net|org|ru|cc|xyz|top|info|io|co|tv|me|biz|vip|app|link|shop|live|site|fun|pro|club|online|store|cloud|test|gg|ly)\b`),
	}
	directContactCompactPatterns = []*regexp.Regexp{
		regexp.MustCompile(`[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`),
		regexp.MustCompile(`([a-z0-9\-]+\.)+[a-z]{2,}`),
	}
	// 检查5个或以上连续数字（QQ号、微信号、电话号码等）
	consecutiveNumbersPattern = regexp.MustCompile(`\d{5,}`)
	// 内容去掉空格后几乎全是数字（纯数字灌水评论）
	pureNumberContentPattern = regexp.MustCompile(`^[\d\s]{5,}$`)
)

func applyHardBlockRules(content string) *ModerateResult {
	normalized := normalizeForDetection(content)

	if containsKeyword(normalized, politicsStrongKeywords) {
		return &ModerateResult{
			Verdict:    "rejected",
			Category:   "politics",
			Confidence: 0.99,
			Reason:     "命中涉政敏感内容，严格拒绝",
			ModelUsed:  "hard-rule",
		}
	}

	for _, rule := range hardBlockRules {
		for _, keyword := range rule.keywords {
			if strings.Contains(normalized, keyword) {
				return &ModerateResult{
					Verdict:    "rejected",
					Category:   rule.category,
					Confidence: 0.99,
					Reason:     rule.reason,
					ModelUsed:  "hard-rule",
				}
			}
		}
	}

	if pureNumberContentPattern.MatchString(strings.TrimSpace(content)) {
		return &ModerateResult{
			Verdict:    "rejected",
			Category:   "spam",
			Confidence: 0.99,
			Reason:     "纯数字内容，疑似号码或无意义灌水",
			ModelUsed:  "hard-rule",
		}
	}

	if looksLikeAdOrContact(content) {
		return &ModerateResult{
			Verdict:    "rejected",
			Category:   "spam",
			Confidence: 0.99,
			Reason:     "命中广告导流或联系方式",
			ModelUsed:  "hard-rule",
		}
	}

	return nil
}

func normalizeModelDecision(ai *aiResult, auditContent string) *aiResult {
	if ai == nil {
		return nil
	}

	if hardResult := applyHardBlockRules(auditContent); hardResult != nil {
		return &aiResult{
			Verdict:    hardResult.Verdict,
			Category:   hardResult.Category,
			Confidence: hardResult.Confidence,
			Reason:     hardResult.Reason,
		}
	}

	normalized := *ai
	if normalized.Category == "politics" {
		normalized.Verdict = "rejected"
		if strings.TrimSpace(normalized.Reason) == "" {
		normalized.Reason = "命中政治类内容，严格拒绝"
		}
		return &normalized
	}

	if containsBenignNegation(auditContent) && !containsDirectContactSignal(auditContent) {
		normalized.Verdict = "approved"
		normalized.Category = "none"
		normalized.Reason = "明确说明不含联系方式或导流"
		return &normalized
	}

	if normalized.Verdict == "flagged" && normalized.Category == "none" {
		normalized.Verdict = "approved"
		normalized.Reason = "正常内容，无明显违规"
		return &normalized
	}

	if normalized.Category == "adult" && !looksLikeAdOrContact(auditContent) {
		normalized.Verdict = "approved"
		normalized.Category = "none"
		normalized.Reason = "普通成人讨论，未命中导流"
	}

	return &normalized
}

func looksLikeAdOrContact(content string) bool {
	if containsDirectContactSignal(content) {
		return true
	}

	return containsWeakTradeIntent(content)
}

func containsDirectContactSignal(content string) bool {
	rawLower := strings.ToLower(content)
	sanitized := stripBenignNegation(content)

	for _, pattern := range directContactRawPatterns {
		if pattern.MatchString(rawLower) {
			return true
		}
	}

	for _, keyword := range directContactKeywords {
		if strings.Contains(sanitized, keyword) {
			return true
		}
	}

	for _, pattern := range directContactCompactPatterns {
		if pattern.MatchString(sanitized) {
			return true
		}
	}

	// 检查5个或以上连续数字（QQ号、微信号、电话号码等）
	if containsConsecutiveNumbers(rawLower) {
		return true
	}

	return false
}

func containsWeakTradeIntent(content string) bool {
	normalized := normalizeForDetection(content)

	for _, phrase := range weakTradeDirectPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}

	hitCount := 0
	for _, token := range weakTradeTokens {
		if strings.Contains(normalized, token) {
		hitCount++
		}
	}

	return hitCount >= 2
}

func containsKeyword(normalized string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}

	return false
}

func containsConsecutiveNumbers(content string) bool {
	return consecutiveNumbersPattern.MatchString(content)
}

func TestLooksLikeAdOrContactExternal(content string) bool {
	return looksLikeAdOrContact(content)
}
