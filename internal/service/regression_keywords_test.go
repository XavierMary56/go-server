package service

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// TestRegressionWithKeywordsFromExcel 使用Excel测试数据中的关键字进行回归测试
func TestRegressionWithKeywordsFromExcel(t *testing.T) {
	file, err := os.Open("../../tests/test_regression_keywords.txt")
	if err != nil {
		t.Fatalf("failed to open test_regression_keywords.txt: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	passCount := 0
	failCount := 0
	skipCount := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			skipCount++
			continue
		}

		// 解析格式: 内容|预期结果|说明
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			t.Logf("Warning: line %d invalid format: %s", lineNum, line)
			skipCount++
			continue
		}

		content := strings.TrimSpace(parts[0])
		expectedVerdict := strings.TrimSpace(parts[1])
		description := ""
		if len(parts) >= 3 {
			description = strings.TrimSpace(parts[2])
		}

		if content == "" {
			skipCount++
			continue
		}

		// 运行检测
		result := applyHardBlockRules(content)

		var actualVerdict string
		if result == nil {
			actualVerdict = "approved"
		} else {
			actualVerdict = result.Verdict
		}

		// 检查结果
		if actualVerdict != expectedVerdict {
			failCount++
			t.Errorf("❌ Line %d FAILED: %q\n   Expected: %s, Got: %s\n   Description: %s\n   Result: %+v",
				lineNum, content, expectedVerdict, actualVerdict, description, result)
		} else {
			passCount++
			if testing.Verbose() {
				t.Logf("✅ Line %d PASSED: %q → %s (%s)",
					lineNum, content, actualVerdict, description)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("error reading file: %v", err)
	}

	// 总结报告
	totalTests := passCount + failCount
	passRate := 0.0
	if totalTests > 0 {
		passRate = float64(passCount) * 100.0 / float64(totalTests)
	}

	t.Logf("\n========== 回归测试总结 ==========")
	t.Logf("总测试数: %d", totalTests)
	t.Logf("通过: %d", passCount)
	t.Logf("失败: %d", failCount)
	t.Logf("跳过: %d", skipCount)
	t.Logf("通过率: %.1f%%", passRate)
	t.Logf("==================================")

	if failCount > 0 {
		t.Errorf("Regression test failed: %d test(s) failed out of %d", failCount, totalTests)
	}
}

// TestKeywordCoverageFromExcel 验证覆盖所有测试关键字
func TestKeywordCoverageFromExcel(t *testing.T) {
	testCases := []struct {
		content        string
		shouldReject   bool
		category       string
		description    string
	}{
		// 应该通过的内容
		{"产品ID是12345很不错", false, "", "纯数字ID"},
		{"这个订单号是98765，谢谢", false, "", "订单号5位数字"},
		{"我吃了点中药来调理身体", false, "", "医药讨论"},
		{"可以在wikipedia.org查看更多信息", false, "", "网站引用"},
		{"官方网站是example.com", false, "", "品牌网址"},
		{"我是资源很丰富的供应商", false, "", "正常业务描述"},
		{"只是提到一下这个产品", false, "", "弱否定用语"},
		{"我没有任何联系方式在这里", false, "", "明确否定"},
		{"9999", false, "", "4位数字"},

		// 应该被拒绝的内容
		{"加微信：test123", true, "spam", "直接微信引流"},
		{"qq群：123456789", true, "spam", "QQ群信息"},
		{"联系电话：13800138000", true, "spam", "电话号码"},
		{"点击链接www.xxx.com获得折扣", true, "spam", "明确导流"},
		{"飞机号@test123", true, "spam", "Telegram变体"},
		{"约炮交友", true, "adult", "成人内容"},
		{"赌博彩票", true, "fraud", "违法内容"},
		{"吸毒害死人", true, "abuse", "毒品内容"},
		{"港独分裂", true, "politics", "政治敏感"},
		{"杀人放火", true, "violence", "暴力内容"},
		{"12345678901", true, "spam", "11位连续数字"},
	}

	passCount := 0
	failCount := 0

	for i, tc := range testCases {
		result := applyHardBlockRules(tc.content)
		isRejected := result != nil && result.Verdict == "rejected"

		if isRejected == tc.shouldReject {
			passCount++
			if testing.Verbose() {
				status := "✅"
				if tc.shouldReject {
					status = "✅ 拦截"
				} else {
					status = "✅ 通过"
				}
				t.Logf("%s Case %d: %q - %s", status, i+1, tc.content, tc.description)
			}
		} else {
			failCount++
			expected := "approved"
			if tc.shouldReject {
				expected = "rejected"
			}
			actual := "approved"
			if isRejected {
				actual = "rejected"
			}
			t.Errorf("❌ Case %d: %q\n   Expected: %s, Got: %s\n   Description: %s\n   Result: %+v",
				i+1, tc.content, expected, actual, tc.description, result)
		}
	}

	passRate := 0.0
	if len(testCases) > 0 {
		passRate = float64(passCount) * 100.0 / float64(len(testCases))
	}

	t.Logf("\n========== 关键字覆盖测试 ==========")
	t.Logf("总测试数: %d", len(testCases))
	t.Logf("通过: %d", passCount)
	t.Logf("失败: %d", failCount)
	t.Logf("通过率: %.1f%%", passRate)
	t.Logf("====================================")

	if failCount > 0 {
		t.Errorf("Keyword coverage test failed: %d test(s) failed", failCount)
	}
}
