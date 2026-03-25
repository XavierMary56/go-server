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
		Reason:     "contact-like wording",
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
	if !containsDirectContactSignal("add me on telegram and click link below") {
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

func TestApplyHardBlockRules(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		category string
	}{
		{name: "politics_cn", content: "讨论政变和颠覆政权", category: "politics"},
		{name: "politics_en", content: "coup and overthrow government", category: "politics"},
		{name: "politics_ru", content: "государственный переворот и протест", category: "politics"},
		{name: "adult", content: "rape and minor porn", category: "adult"},
		{name: "fraud", content: "scam gambling guaranteed profit", category: "fraud"},
		{name: "drugs", content: "buy drugs and heroin recipe", category: "abuse"},
		{name: "violence", content: "bomb making and gun for sale", category: "violence"},
		{name: "spam", content: "add me on telegram and click link below", category: "spam"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := applyHardBlockRules(tc.content)
			if result == nil {
				t.Fatal("expected hard block result")
			}
			if result.Verdict != "rejected" {
				t.Fatalf("unexpected verdict: %s", result.Verdict)
			}
			if result.Category != tc.category {
				t.Fatalf("unexpected category: %s", result.Category)
			}
		})
	}
}

func TestApplyHardBlockRulesVariantCoverage(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		category string
	}{
		{name: "politics_en_variant", content: "anti-government riot and regime change", category: "politics"},
		{name: "politics_ru_variant", content: "сепаратизм и независимость", category: "politics"},
		{name: "adult_ru_variant", content: "интим услуги и порно", category: "adult"},
		{name: "fraud_ru_variant", content: "финансовая пирамида и мошенничество", category: "fraud"},
		{name: "drugs_en_variant", content: "drug dealer with narcotics and meth", category: "abuse"},
		{name: "violence_ru_variant", content: "взрывчатка и оружие купить", category: "violence"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := applyHardBlockRules(tc.content)
			if result == nil {
				t.Fatal("expected hard block result")
			}
			if result.Category != tc.category {
				t.Fatalf("unexpected category: %s", result.Category)
			}
		})
	}
}

func TestApplyHardBlockRulesPoliticsStrictCoverage(t *testing.T) {
	cases := []string{
		"讨论某国总统选举和议会局势",
		"president election and parliament politics",
		"президент выборы и парламент",
	}

	for _, content := range cases {
		result := applyHardBlockRules(content)
		if result == nil {
			t.Fatalf("expected politics hard block result: %s", content)
		}
		if result.Category != "politics" {
			t.Fatalf("unexpected politics category for %s: %s", content, result.Category)
		}
		if result.Verdict != "rejected" {
			t.Fatalf("unexpected politics verdict for %s: %s", content, result.Verdict)
		}
	}
}

func TestNormalizeModelDecisionAlwaysRejectsPolitics(t *testing.T) {
	result := normalizeModelDecision(&aiResult{
		Verdict:    "flagged",
		Category:   "politics",
		Confidence: 0.88,
		Reason:     "political content",
	}, "ordinary political text")

	if result == nil {
		t.Fatal("expected normalized politics result")
	}
	if result.Verdict != "rejected" {
		t.Fatalf("expected rejected verdict, got %s", result.Verdict)
	}
	if result.Category != "politics" {
		t.Fatalf("expected politics category, got %s", result.Category)
	}
}

func TestApplyHardBlockRulesFraudCoverage(t *testing.T) {
	cases := []string{
		"兼职日结高薪，带你赚钱，稳赚不赔",
		"私服代充黑卡上分，博彩包赢",
		"杀猪盘资金盘，投资返利",
		"make money fast with guaranteed profit",
		"scam phishing fake account stolen card",
		"gambling betting casino online ponzi",
		"быстрый заработок и мошенничество",
		"ставки казино фишинг",
		"финансовая пирамида и афера",
	}

	for _, content := range cases {
		result := applyHardBlockRules(content)
		if result == nil {
			t.Fatalf("expected fraud hard block result: %s", content)
		}
		if result.Category != "fraud" {
			t.Fatalf("unexpected fraud category for %s: %s", content, result.Category)
		}
	}
}

func TestApplyHardBlockRulesDrugsCoverage(t *testing.T) {
	cases := []string{
		"冰毒海洛因摇头丸，买毒出货走货",
		"制毒配方，大麻和吸毒工具",
		"drugs cocaine heroin meth",
		"buy drugs sell drugs drug recipe",
		"how to make drugs with weed marijuana",
		"наркотики кокаин героин",
		"купить наркотики и изготовление наркотиков",
	}

	for _, content := range cases {
		result := applyHardBlockRules(content)
		if result == nil {
			t.Fatalf("expected drugs hard block result: %s", content)
		}
		if result.Category != "abuse" {
			t.Fatalf("unexpected drugs category for %s: %s", content, result.Category)
		}
	}
}

func TestApplyHardBlockRulesViolenceCoverage(t *testing.T) {
	cases := []string{
		"杀人暗杀爆炸，制作炸弹",
		"枪支购买，恐怖袭击，自制武器",
		"kill assassination bomb making explosives",
		"terrorist attack and gun for sale",
		"убийство и взрыв бомба",
		"террористическая атака и оружие купить",
	}

	for _, content := range cases {
		result := applyHardBlockRules(content)
		if result == nil {
			t.Fatalf("expected violence hard block result: %s", content)
		}
		if result.Category != "violence" {
			t.Fatalf("unexpected violence category for %s: %s", content, result.Category)
		}
	}
}

func TestApplyHardBlockRulesAdultCoverage(t *testing.T) {
	cases := []string{
		"约炮一夜情，上门服务迷药",
		"裸聊裸照，强奸幼女未成年儿童",
		"hookup one night stand nude sexvideo porn",
		"rape gang rape prostitution escort service",
		"underage sex and minor porn",
		"секс за деньги и порно",
		"секс видео и изнасилование",
		"интим услуги и несовершеннолетний секс",
	}

	for _, content := range cases {
		result := applyHardBlockRules(content)
		if result == nil {
			t.Fatalf("expected adult hard block result: %s", content)
		}
		if result.Category != "adult" {
			t.Fatalf("unexpected adult category for %s: %s", content, result.Category)
		}
	}
}

func TestLooksLikeAdOrContactCoversMultilingualDrainPhrases(t *testing.T) {
	cases := []string{
		"加我微信，私聊发你资源",
		"点击链接领取，福利群里见",
		"add me on WhatsApp and click link below",
		"free download and join group for free",
		"добавь в телеграм",
		"перейди по ссылке и вступить в группу",
	}

	for _, content := range cases {
		if !looksLikeAdOrContact(content) {
			t.Fatalf("expected multilingual drain phrase to be detected: %s", content)
		}
	}
}

func TestContainsDirectContactSignalCoversURLPatterns(t *testing.T) {
	cases := []string{
		"https://example.com/free-video",
		"http://abc.test/path",
		"www.example.com",
		"join at moviehub.com right now",
	}

	for _, content := range cases {
		if !containsDirectContactSignal(content) {
			t.Fatalf("expected URL-like signal to be detected: %s", content)
		}
	}
}
