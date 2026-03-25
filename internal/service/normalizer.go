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

func normalizeForDetection(content string) string {
	normalized := strings.ToLower(content)

	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, r := range normalized {
		switch {
		case r == 12288:
			builder.WriteRune(' ')
		case r >= 65281 && r <= 65374:
			builder.WriteRune(r - 65248)
		default:
			builder.WriteRune(unicode.ToLower(r))
		}
	}

	normalized = builder.String()
	replacer := strings.NewReplacer(
		"围信", "微信",
		"薇信", "微信",
		"卫星", "微信",
		"v信", "微信",
		"微❤", "微信",
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
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		compact.WriteRune(r)
	}

	return compact.String()
}
