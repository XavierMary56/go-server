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
		t.Fatalf("expected benign resource discussion to stay unblocked, got category=%s direct=%v weak=%v", result.Category, containsDirectContactSignal(content), containsWeakTradeIntent(content))
	}
}

func TestApplyHardBlockRulesTreatsObfuscatedURLAsSpam(t *testing.T) {
	content := "看资源点 hxxps://abc[.]com"

	result := applyHardBlockRules(content)
	if result == nil {
		t.Fatal("expected obfuscated url sample to be blocked")
	}
	if result.Category != "spam" && result.Category != "fraud" {
		t.Fatalf("expected spam or fraud category, got %s", result.Category)
	}
}

func TestApplyHardBlockRulesTreatsTelegramHandleAsSpam(t *testing.T) {
	content := "telegram：movie_hub99"

	result := applyHardBlockRules(content)
	if result == nil {
		t.Fatal("expected telegram handle sample to be blocked")
	}
	if result.Category != "spam" && result.Category != "fraud" {
		t.Fatalf("expected spam or fraud category, got %s", result.Category)
	}
}

func TestLooksLikeAdOrContactCoversWeiXVariant(t *testing.T) {
	content := "微 x：test001"

	if !looksLikeAdOrContact(content) {
		t.Fatal("expected wei-x variant to be detected as contact signal")
	}
}

func TestLooksLikeAdOrContactAllowsExtendedBenignNegation(t *testing.T) {
	content := "普通反馈，不含任何联系方式"

	if looksLikeAdOrContact(content) {
		t.Fatal("did not expect extended benign negation to look like ad/contact")
	}
}

func TestLooksLikeAdOrContactAllowsPlainNumbersWithoutContactCue(t *testing.T) {
	cases := []string{
		"普通评论，只是提到 1234 这个数字",
		"今天价格 1234 元，不含联系方式",
		"参考数字 0123 和文本",
	}

	for _, content := range cases {
		if looksLikeAdOrContact(content) {
			t.Fatalf("did not expect plain short numeric content to look like ad/contact: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactDetectsQQGroupVariants(t *testing.T) {
	cases := []string{
		"q群 7654321",
		"企鹅号 12345678",
		"qq号 1234567",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected qq-group variant to be detected: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactDetectsWeixinAndTelegramVariants(t *testing.T) {
	cases := []string{
		"wei xin: test001",
		"wｅｉｘｉｎ test002",
		"薇❤: test003",
		"飞 机 号 moviehub99",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected hidden contact variant to be detected: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactAllowsPlainPlatformMention(t *testing.T) {
	cases := []string{
		"普通评论，只是提到微信这个词但没有联系方式",
		"普通反馈，薇信这个词只是举例说明",
	}

	for _, content := range cases {
		if looksLikeAdOrContact(content) {
			t.Fatalf("did not expect plain platform mention to look like ad/contact: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactDetectsWeixinRomanNumeral(t *testing.T) {
	cases := []string{
		"加薇看后01",
		"加Ⅴ看后01",
		"加v看后01",
		"加V看后01",
		"加vx看后01",
		"加VX看后01",
		"加微信xxx",
		"加v信xxx",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected weixin variant to be detected: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactDetectsQQVariants(t *testing.T) {
	cases := []string{
		"加q看后01",
		"加Q看后01",
		"加qq看后01",
		"加QQ看后01",
		"加Q号001",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected qq variant to be detected: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactDetectsConsecutiveNumbers(t *testing.T) {
	cases := []string{
		"我的号码 12345",
		"联系方式 987654321",
		"QQ号：123456789",
		"微信：10086888",
		"电话 13812345678",
		"号码是555555",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected consecutive numbers to be detected: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactAllowsShortNumbers(t *testing.T) {
	cases := []string{
		"今天价格 1234 元",
		"数量 123 个",
		"年份 2026 年",
		"参考号码 0123",
	}

	for _, content := range cases {
		if looksLikeAdOrContact(content) {
			t.Fatalf("did not expect plain short numeric content to look like ad/contact: %s", content)
		}
	}
}
