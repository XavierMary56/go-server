package service

import "testing"

func TestContainsBenignNegation(t *testing.T) {
	if !containsBenignNegation("【VN011】节奏还行，镜头切得挺顺。不带联系方式。") {
		t.Fatal("expected benign negation phrase to be detected")
	}
}

func TestNormalizeModelDecisionAllowsBenignNegation(t *testing.T) {
	result := normalizeModelDecision(&aiResult{
		Verdict:    "flagged",
		Category:   "spam",
		Confidence: 0.92,
		Reason:     "命中联系方式相关词",
	}, "【VN011】节奏还行，镜头切得挺顺。不带联系方式。")

	if result == nil {
		t.Fatal("expected normalized result")
	}
	if result.Verdict != "approved" {
		t.Fatalf("unexpected verdict: %s", result.Verdict)
	}
	if result.Category != "none" {
		t.Fatalf("unexpected category: %s", result.Category)
	}
}

func TestContainsDirectContactSignal(t *testing.T) {
	if !containsDirectContactSignal("加V test_video_01 领更多资源") {
		t.Fatal("expected direct contact signal to be detected")
	}
	if containsDirectContactSignal("【VN011】节奏还行，镜头切得挺顺。不带联系方式。") {
		t.Fatal("did not expect direct contact signal for benign negation sample")
	}
}

func TestLooksLikeAdOrContactIgnoresBenignNegation(t *testing.T) {
	if looksLikeAdOrContact("【VN011】节奏还行，镜头切得挺顺。不带联系方式。") {
		t.Fatal("did not expect benign negation sample to be treated as ad/contact")
	}
}

func TestNormalizeForDetectionCoversSplitAndVariantForms(t *testing.T) {
	cases := []string{
		"微-信 test001",
		"薇信 test002",
		"扣扣 1234567",
		"ＱＱ１２３４５６７",
		"T.G room001",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected variant form to be detected: %s", content)
		}
	}
}

func TestLooksLikeAdOrContactCoversWeakTradeIntent(t *testing.T) {
	cases := []string{
		"低价资源打包，有偿分享完整版",
		"主页找我，看资料，私下聊",
		"去别处看，外站价更低",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected weak trade intent to be detected: %s", content)
		}
	}
}
