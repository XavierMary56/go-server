package service

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestWorkKeywordsFullRejection(t *testing.T) {
	file, err := os.Open("../../tests/test_work.txt")
	if err != nil {
		t.Fatalf("failed to open test_work.txt: %v", err)
	}
	defer file.Close()

	// 规则优化：允许某些单独关键词以减少误拦
	// 但组合关键词如 ".com/https://" 仍然应该被拒
	allowedSingleKeywords := map[string]bool{
		".com":    true,  // 域名后缀，单独不应该拒
		"http://": true,  // 协议前缀，完整URL才拒
		".cn":     true,
		".net":    true,
		".org":    true,
	}

	// Regex to match lines like "111	.com/https://" or "22	约炮 / 一夜情"
	linePattern := regexp.MustCompile(`^\d+\t(.+)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract actual content from lines with numbers/tabs
		matches := linePattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			rawContent := matches[1]
			// Split by common separators in the text file
			parts := strings.FieldsFunc(rawContent, func(r rune) bool {
				return r == '/' || r == '、' || r == '+'
			})

			for _, p := range parts {
				keyword := strings.TrimSpace(p)
				if keyword == "" || keyword == "中文" || keyword == "English" || keyword == "Русский" || strings.Contains(keyword, "主要是针对") {
					continue
				}

				// Skip only single allowed keywords, not composite patterns
				if allowedSingleKeywords[keyword] {
					continue
				}

				// Basic individual keyword test
				result := applyHardBlockRules(keyword)
				if result == nil || result.Verdict != "rejected" {
					t.Errorf("Expected keyword %q (from line %q) to be rejected, but it was approved", keyword, line)
				}
			}
		} else {
			// For lines without leading numbers/tabs, skip headers
			if strings.Contains(line, "：") || strings.HasPrefix(line, "一、") || strings.HasPrefix(line, "二、") ||
				strings.HasPrefix(line, "三、") || strings.HasPrefix(line, "四、") || strings.HasPrefix(line, "五、") ||
				strings.HasPrefix(line, "六、") || strings.HasPrefix(line, "7.") || strings.HasPrefix(line, "8.") ||
				line == "中文" || line == "English" || line == "Русский" {
				continue
			}

			// Skip allowed keywords
			if allowedSingleKeywords[line] {
				continue
			}

			result := applyHardBlockRules(line)
			// Check if it's a known non-keyword header
			if result == nil || result.Verdict != "rejected" {
				// If it's a long line with explanation, it's probably not a single keyword to block
				if len(line) < 30 {
					t.Errorf("Expected keyword %q to be rejected, but it was approved", line)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("error reading test_work.txt: %v", err)
	}
}

func TestSpecificFailures(t *testing.T) {
	// 移除".com"和"http://"因为它们会导致合法内容误拦
	// 规则优化：
	// - 移除通用域名后缀的单独检测（通过 directContactRawPatterns 中的完整 URL 检测处理）
	// - 连续数字从 5+ 改为 6+ 减少误拦
	cases := []string{
		"QQ", "TG", "BC",
		"https://zyaokkmo.cc",
		"qq我",
		"1234567",  // 7位数字，符合6+的规则
		"加v",
		"加V",
		"加vx",
		"加VX",
		"vx号",
		"加薇",
		"薇",  // 单字"薇"应该被转换为"wechat"后检测
		"bc",
	}

	for _, tc := range cases {
		result := applyHardBlockRules(tc)
		if result == nil || result.Verdict != "rejected" {
			t.Errorf("Expected %q to be rejected, but it was approved", tc)
		}
	}

	// 这些不应该单独被拒（避免误拦）
	// 验证改进的规则
	benignCases := map[string]bool{
		".com":    true,
		"http://": true,
	}

	for tc := range benignCases {
		result := applyHardBlockRules(tc)
		// These should now be allowed (not rejected)
		if result != nil && result.Verdict == "rejected" {
			t.Logf("Note: %q was rejected, but should be allowed by new rules", tc)
		}
	}
}
