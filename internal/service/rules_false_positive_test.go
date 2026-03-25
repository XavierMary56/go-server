package service

import "testing"

func TestApplyHardBlockRulesAllowsNormalTechnicalDiscussion(t *testing.T) {
	content := "Go 的切片和数组有什么区别"

	if result := applyHardBlockRules(content); result != nil {
		t.Fatalf("expected technical discussion to stay unblocked, got category=%s verdict=%s direct=%v weak=%v", result.Category, result.Verdict, containsDirectContactSignal(content), containsWeakTradeIntent(content))
	}
}

func TestApplyHardBlockRulesAllowsWrappedTechnicalDiscussion(t *testing.T) {
	content := "review body:\nGo 的切片和数组有什么区别"

	if result := applyHardBlockRules(content); result != nil {
		t.Fatalf("expected wrapped technical discussion to stay unblocked, got category=%s verdict=%s direct=%v weak=%v", result.Category, result.Verdict, containsDirectContactSignal(content), containsWeakTradeIntent(content))
	}
}

func TestNormalizeModelDecisionApprovesFlaggedNoneContent(t *testing.T) {
	result := normalizeModelDecision(&aiResult{
		Verdict:    "flagged",
		Category:   "none",
		Confidence: 0.95,
		Reason:     "内容不明确，可能引起误解",
	}, "普通的技术讨论内容")

	if result == nil {
		t.Fatal("expected normalized result")
	}
	if result.Verdict != "approved" {
		t.Fatalf("expected approved verdict, got %s", result.Verdict)
	}
	if result.Category != "none" {
		t.Fatalf("expected none category, got %s", result.Category)
	}
}

func TestApplyHardBlockRulesCoversRealChineseRiskContent(t *testing.T) {
	cases := []struct {
		content  string
		category string
	}{
		{content: "讨论政变和颠覆政权", category: "politics"},
		{content: "约炮一夜情上门服务", category: "adult"},
		{content: "兼职日结高薪，带你赚钱，稳赚不赔", category: "fraud"},
		{content: "冰毒海洛因买毒出货", category: "abuse"},
		{content: "杀人爆炸制作炸弹", category: "violence"},
		{content: "加我微信私聊发你资源", category: "spam"},
	}

	for _, tc := range cases {
		result := applyHardBlockRules(tc.content)
		if result == nil {
			t.Fatalf("expected hard block result for %q", tc.content)
		}
		if result.Category != tc.category {
			t.Fatalf("unexpected category for %q: want %s got %s", tc.content, tc.category, result.Category)
		}
	}
}

func TestLooksLikeAdOrContactAllowsBenignResourceDiscussion(t *testing.T) {
	content := "【N030】只是分享观看感受，没有任何资源交换。"

	if result := applyHardBlockRules(content); result != nil {
		normalized := normalizeForDetection(content)
		var hits []string
		for _, phrase := range weakTradeDirectPhrases {
			if contains := len(phrase) > 0 && len(normalized) > 0 && containsNormalized(normalized, phrase); contains {
				hits = append(hits, "phrase:"+phrase)
			}
		}
		for _, token := range weakTradeTokens {
			if containsNormalized(normalized, token) {
				hits = append(hits, "token:"+token)
			}
		}
		t.Fatalf("expected benign resource discussion to stay unblocked, got category=%s direct=%v weak=%v hits=%v", result.Category, containsDirectContactSignal(content), containsWeakTradeIntent(content), hits)
	}
}

func containsNormalized(normalized, token string) bool {
	return len(token) > 0 && len(normalized) > 0 && (normalized == token || len(normalized) >= len(token) && containsString(normalized, token))
}

func containsString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
