package service

import (
	"testing"
)

// TestAPIFailureCases 测试API返回被批准但应被拒绝的5个内容
func TestAPIFailureCases(t *testing.T) {
	testCases := []struct {
		content string
		expect  string // "rejected" or "approved"
	}{
		{"飞机号@test123", "rejected"},
		{"约炮交友", "rejected"},
		{"赌博彩票", "rejected"},
		{"吸毒害死人", "rejected"},
		{"港独分裂", "rejected"},
	}

	for _, tc := range testCases {
		result := applyHardBlockRules(tc.content)
		
		if result != nil && result.Verdict == "rejected" {
			t.Logf("✅ %q → rejected (Category: %s)", tc.content, result.Category)
		} else if result == nil {
			t.Logf("❌ %q → approved (should be rejected) - applyHardBlockRules returned nil", tc.content)
		} else {
			t.Logf("❌ %q → %s (should be rejected)", tc.content, result.Verdict)
		}
	}
}

// TestAuditContentBuilding 测试 buildAuditContent 加前缀后的检测
func TestAuditContentBuilding(t *testing.T) {
	testCases := []struct {
		content string
		withPrefix string
	}{
		{"飞机号@test123", "review body:\n飞机号@test123"},
		{"约炮交友", "review body:\n约炮交友"},
		{"赌博彩票", "review body:\n赌博彩票"},
		{"吸毒害死人", "review body:\n吸毒害死人"},
		{"港独分裂", "review body:\n港独分裂"},
	}

	for _, tc := range testCases {
		// 测试原始内容
		result1 := applyHardBlockRules(tc.content)
		// 测试带前缀的内容
		result2 := applyHardBlockRules(tc.withPrefix)
		
		t.Logf("原始: %q", tc.content)
		if result1 != nil {
			t.Logf("  原始 → %s", result1.Verdict)
		} else {
			t.Logf("  原始 → nil")
		}
		
		if result2 != nil {
			t.Logf("  带前缀 → %s", result2.Verdict)
		} else {
			t.Logf("  带前缀 → nil")
		}
	}
}
