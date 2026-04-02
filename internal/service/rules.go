package service

import (
	"regexp"
	"strings"
)

var (
	// 合并多个正则为单个模式，一次匹配完成所有检测
	directContactRawPattern = regexp.MustCompile(
		`https?://[^\s]+` +
			`|hxxps?://[^\s]+` +
			`|www\.[^\s]+` +
			`|t\.me/[^\s]+` +
			`|discord\.gg/[^\s]+` +
			`|bit\.ly/[^\s]+` +
			`|tinyurl\.com/[^\s]+` +
			`|(?:^|[\s(])@[a-z0-9_]{5,}\b`,
	)
	directContactCompactPattern = regexp.MustCompile(`[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	// 检查可能的QQ号或电话号码
	// 6位数字可能是产品ID、验证码等合法内容，7位起拦更合理
	consecutiveNumbersPattern = regexp.MustCompile(`\d{7,}`)
	// 内容去掉空格后几乎全是数字（纯数字灌水评论）
	// 改为检测多个分开的数字块（如"12345 67890 11111"），而不是单个数字
	pureNumberContentPattern = regexp.MustCompile(`^\d+[\s]+\d+`)
)

func applyHardBlockRules(content string) *ModerateResult {
	normalized := normalizeForDetection(content)

	// 政治敏感内容：最高优先级，直接拒绝（置信度 0.99）
	if containsKeyword(normalized, politicsStrongKeywords) {
		return &ModerateResult{
			Verdict:    "rejected",
			Category:   "politics",
			Confidence: 0.99,
			Reason:     "命中涉政敏感内容，严格拒绝",
			ModelUsed:  "hard-rule",
		}
	}

	// 核心规则检测（保留原有的直接拒绝逻辑，确保测试兼容性）
	for _, rule := range hardBlockRules {
		for _, keyword := range rule.keywords {
			if strings.Contains(normalized, keyword) {
				// 阶段 2 改进：添加置信度而不是都使用 0.99
				confidence := 0.90
				switch rule.category {
				case "adult":
					confidence = 0.85
				case "fraud":
					confidence = 0.80
				case "abuse":
					confidence = 0.80
				case "violence":
					confidence = 0.85
				}

				return &ModerateResult{
					Verdict:    "rejected",
					Category:   rule.category,
					Confidence: confidence,
					Reason:     rule.reason,
					ModelUsed:  "hard-rule",
				}
			}
		}
	}

	// 纯数字内容检测：直接拒绝（这确实是垃圾/灌水）
	if pureNumberContentPattern.MatchString(strings.TrimSpace(content)) {
		return &ModerateResult{
			Verdict:    "rejected",
			Category:   "spam",
			Confidence: 0.90,
			Reason:     "纯数字内容，疑似号码或无意义灌水",
			ModelUsed:  "hard-rule",
		}
	}

	if looksLikeAdOrContact(content) {
		return &ModerateResult{
			Verdict:    "rejected",
			Category:   "spam",
			Confidence: 0.85,
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

	if directContactRawPattern.MatchString(rawLower) {
		return true
	}

	// email 检测在 rawLower 上运行（归一化会删除 @ 导致无法匹配）
	if directContactCompactPattern.MatchString(rawLower) {
		return true
	}

	// 如果有benign negation（如"没有联系方式"），则不认为包含直接联系信号
	if containsBenignNegation(content) {
		return false
	}

	for _, keyword := range directContactKeywords {
		if strings.Contains(sanitized, keyword) {
			return true
		}
	}

	// 检查7位或以上连续数字（QQ号、电话号码等）
	if containsConsecutiveNumbers(rawLower) {
		return true
	}

	return false
}

func containsWeakTradeIntent(content string) bool {
	normalized := normalizeForDetection(content)

	// 先检测明确的导流词组（单个匹配即拦）
	for _, phrase := range weakTradeDirectPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}

	// 弱导流令牌需要更多匹配（从 2 提升到 3）以减少误拦
	// 因为 "资源" 和 "代理" 已被移除，减少了单词数量
	hitCount := 0
	for _, token := range weakTradeTokens {
		if strings.Contains(normalized, token) {
			hitCount++
		}
	}

	return hitCount >= 3
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
