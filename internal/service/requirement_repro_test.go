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
	cases := []string{
		"QQ", "TG", "BC",
		"https://zyaokkmo.cc",
		"qq我",
		"1234567",
		"加v",
		"加V",
		"加vx",
		"加VX",
		"vx号",
		"加薇",
		"薇",
		"bc",
		".com",
		"http://",
	}

	for _, tc := range cases {
		result := applyHardBlockRules(tc)
		if result == nil || result.Verdict != "rejected" {
			t.Errorf("Expected %q to be rejected, but it was approved", tc)
		}
	}
}
