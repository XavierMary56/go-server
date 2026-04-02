package service

import (
	"strings"
	"unicode"
)

func containsBenignNegation(content string) bool {
	normalized := normalizeForDetection(content)
	for _, phrase := range benignNegationPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func stripBenignNegation(content string) string {
	sanitized := normalizeForDetection(content)
	if !containsBenignNegation(content) {
		return sanitized
	}

	for _, phrase := range benignNegationPhrases {
		sanitized = strings.ReplaceAll(sanitized, phrase, "")
	}

	return sanitized
}

// detectLanguage 简单的语言检测：检查是否包含特定语言的字符
func detectLanguage(content string) string {
	for _, r := range content {
		// 韩文范围（Hangul Syllables）
		if r >= 0xAC00 && r <= 0xD7AF {
			return "korean"
		}
		// 日文范围（Hiragana + Katakana）
		if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
			return "japanese"
		}
		// 西里尔字母（俄文等）
		if r >= 0x0400 && r <= 0x04FF {
			return "russian"
		}
	}
	// 默认为中文/英文混合
	return "default"
}

func normalizeForDetection(content string) string {
	lang := detectLanguage(content)

	// 韩文和日文：保留原文结构，避免过度规范化导致字符丢失
	if lang == "korean" || lang == "japanese" {
		return normalizeMultilingualDefault(content)
	}

	// 默认规范化路径（中文/英文）：合并 ToLower + 全角转半角为一次遍历
	var builder strings.Builder
	builder.Grow(len(content))
	for _, r := range content {
		switch {
		case r == 12288: // 全角空格
			builder.WriteRune(' ')
		case r >= 65281 && r <= 65374: // 全角字符转半角
			builder.WriteRune(unicode.ToLower(r - 65248))
		default:
			builder.WriteRune(unicode.ToLower(r))
		}
	}

	normalized := builder.String()
	replacer := strings.NewReplacer(
		"微信", "wechat",
		"微x", "wechat",
		"微 x", "wechat",
		"微ｘ", "wechat",
		"围信", "wechat",
		"薇信", "wechat",
		"薇❤", "wechat",
		"薇心", "wechat",
		"薇", "wechat",
		"卫星", "wechat",
		"v信", "wechat",
		"加薇", "wechat",
		"ⅴ", "v",
		"扣扣", "qq",
		"球球", "qq",
		"电报", "telegram",
		"飞机", "telegram",
		"油箱", "邮箱",
	)
	normalized = replacer.Replace(normalized)

	var compact strings.Builder
	compact.Grow(len(normalized))
	for _, r := range normalized {
		if unicode.IsSpace(r) || (unicode.IsPunct(r) && r != '.' && r != '/' && r != ':') || unicode.IsSymbol(r) {
			continue
		}
		compact.WriteRune(r)
	}

	return compact.String()
}

// normalizeMultilingualDefault 用于韩文、日文等特殊处理，避免过度规范化
func normalizeMultilingualDefault(content string) string {
	// 只做基本的小写转换，不删除字符
	// 避免删除可能是有意义的多语言字符
	normalized := strings.ToLower(content)

	// 只替换中文谐音词
	replacer := strings.NewReplacer(
		"微信", "wechat",
		"微x", "wechat",
		"微ｘ", "wechat",
		"薇", "wechat",
		"电报", "telegram",
		"飞机", "telegram",
	)
	normalized = replacer.Replace(normalized)

	// 删除空格但保留字符结构
	var compact strings.Builder
	compact.Grow(len(normalized))
	for _, r := range normalized {
		if !unicode.IsSpace(r) {
			compact.WriteRune(r)
		}
	}

	return compact.String()
}
