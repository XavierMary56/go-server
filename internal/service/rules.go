package service

import (
	"regexp"
	"strings"
)

var (
	directContactRawPatterns = []*regexp.Regexp{
		regexp.MustCompile(`https?://[^\s]+`),
		regexp.MustCompile(`www\.[^\s]+`),
		regexp.MustCompile(`t\.me/[^\s]+`),
		regexp.MustCompile(`discord\.gg/[^\s]+`),
		regexp.MustCompile(`bit\.ly/[^\s]+`),
		regexp.MustCompile(`tinyurl\.com/[^\s]+`),
		regexp.MustCompile(`(?:^|[\s(])@[a-z0-9_]{5,}\b`),
		regexp.MustCompile(`([a-z0-9\-]+\.)+(com|cn|net|org|ru|cc|xyz|top|info|io|co|tv|me|biz|vip|app|link|shop|live|site|fun|pro|club|online|store|cloud|test|gg|ly)\b`),
	}
	directContactCompactPatterns = []*regexp.Regexp{
		regexp.MustCompile(`[1-9][0-9]{5,}`),
		regexp.MustCompile(`[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`),
		regexp.MustCompile(`([a-z0-9\-]+\.)+[a-z]{2,}`),
	}
)

func applyHardBlockRules(content string) *ModerateResult {
	normalized := normalizeForDetection(content)

	if containsKeyword(normalized, politicsStrongKeywords) || countKeywordHits(normalized, politicsContextKeywords) >= 2 {
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

func countKeywordHits(normalized string, keywords []string) int {
	hits := 0
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			hits++
		}
	}

	return hits
}
