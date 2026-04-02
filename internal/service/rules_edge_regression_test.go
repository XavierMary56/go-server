package service

import "testing"

func TestApplyHardBlockRulesPoliticsNeedsRiskContext(t *testing.T) {
	cases := []string{
		"government budget report for local schools",
		"the president visited a factory today",
		"parliament building is open for tourists",
	}

	for _, content := range cases {
		if result := applyHardBlockRules(content); result != nil {
			t.Fatalf("expected broad political term alone to stay unblocked, got %s for %q", result.Category, content)
		}
	}
}

func TestContainsDirectContactSignalCoversShortLinksAndHandles(t *testing.T) {
	cases := []string{
		"join here t.me/moviehub",
		"backup server discord.gg/moviehub",
		"free video bit.ly/freevideo",
		"message @moviehub99 for details",
	}

	for _, content := range cases {
		if !containsDirectContactSignal(content) {
			t.Fatalf("expected direct contact signal to be detected: %s", content)
		}
	}
}

func TestContainsDirectContactSignalCoversEmailsAndDomainSuffixes(t *testing.T) {
	cases := []string{
		"contact admin@moviehub.ru for access",
		// 裸域名（无协议前缀）已有意移除检测以减少误拦
		// "mirror is hosted on moviehub.cn now",
		// "open the backup at moviehub.net/archive",
	}

	for _, content := range cases {
		if !containsDirectContactSignal(content) {
			t.Fatalf("expected domain or email signal to be detected: %s", content)
		}
	}
}

func TestContainsBenignNegationSupportsEnglishVariants(t *testing.T) {
	cases := []string{
		"ordinary feedback, no contact info here",
		"just discussion, not an ad and no diversion",
		"movie review only, no private contact",
	}

	for _, content := range cases {
		if !containsBenignNegation(content) {
			t.Fatalf("expected benign negation to be detected: %s", content)
		}
		if looksLikeAdOrContact(content) {
			t.Fatalf("did not expect benign negation sample to look like ad/contact: %s", content)
		}
	}
}

func TestApplyHardBlockRulesLeavesNormalAdultDiscussionUnblocked(t *testing.T) {
	cases := []string{
		"this is an adult-themed movie discussion with no solicitation",
		"普通成人题材剧情讨论，没有联系方式也没有交易",
	}

	for _, content := range cases {
		if result := applyHardBlockRules(content); result != nil {
			t.Fatalf("expected normal adult discussion to stay unblocked, got %s for %q", result.Category, content)
		}
	}
}

func TestApplyHardBlockRulesCoversExpandedRiskKeywords(t *testing.T) {
	cases := []struct {
		content  string
		category string
	}{
		{content: "money laundering and account selling service", category: "fraud"},
		{content: "ketamine fentanyl mdma for sale", category: "abuse"},
		{content: "detonator ammo and firearm sale", category: "violence"},
		{content: "adult escort and paid sex deal", category: "adult"},
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
