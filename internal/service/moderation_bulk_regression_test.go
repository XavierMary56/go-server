package service

import (
	"fmt"
	"strings"
	"testing"
)

func TestBulkViolationCoverageByCategory(t *testing.T) {
	categoryCases := map[string][]string{
		"politics": buildCategorySamples(
			[]string{
				"政变", "暴动", "起义", "颠覆政权", "分裂国家",
				"coup", "rebellion", "overthrow government", "illegal protest", "political leak",
				"государственный переворот", "мятеж", "сепаратизм", "незаконный митинг", "политический скандал",
			},
			[]string{
				"这里在讨论%s的具体方案",
				"有人传播%s相关内容",
				"群里正在聊%s的话题",
				"帖子反复提到%s",
				"这段文本包含%s信息",
				"this content promotes %s directly",
				"the post is about %s today",
			},
			100,
		),
		"adult": buildCategorySamples(
			[]string{
				"约炮", "一夜情", "上门服务", "迷药", "裸聊",
				"hookup", "one night stand", "porn", "rape", "escort service",
				"секс за деньги", "порно", "секс видео", "изнасилование", "интим услуги",
			},
			[]string{
				"这条内容明确出现%s",
				"有人在传播%s相关描述",
				"这里反复提到%s",
				"this text explicitly mentions %s",
				"the message contains %s",
				"в тексте есть %s",
			},
			100,
		),
		"fraud": buildCategorySamples(
			[]string{
				"刷单", "兼职日结高薪", "带你赚钱", "稳赚不赔", "资金盘",
				"make money fast", "guaranteed profit", "scam", "phishing", "casino online",
				"быстрый заработок", "мошенничество", "ставки", "фишинг", "финансовая пирамида",
			},
			[]string{
				"这段话正在宣传%s",
				"有人发布%s的内容",
				"这里存在%s风险",
				"this post advertises %s",
				"the chat contains %s",
				"сообщение содержит %s",
			},
			100,
		),
		"abuse": buildCategorySamples(
			[]string{
				"冰毒", "海洛因", "摇头丸", "买毒", "制毒",
				"drugs", "cocaine", "heroin", "buy drugs", "marijuana",
				"наркотики", "кокаин", "героин", "купить наркотики", "изготовление наркотиков",
			},
			[]string{
				"文本里直接写了%s",
				"这里出现了%s交易",
				"这段内容包含%s",
				"this content includes %s",
				"the message promotes %s",
				"в сообщении есть %s",
			},
			100,
		),
		"violence": buildCategorySamples(
			[]string{
				"杀人", "暗杀", "爆炸", "制作炸弹", "恐怖袭击",
				"kill", "assassination", "bomb making", "terrorist attack", "gun for sale",
				"убийство", "взрыв", "бомба", "террористическая атака", "оружие купить",
			},
			[]string{
				"文本在描述%s",
				"有人讨论%s的做法",
				"这里含有%s相关信息",
				"this post is about %s",
				"the content contains %s",
				"текст содержит %s",
			},
			100,
		),
		"spam": buildCategorySamples(
			[]string{
				"加我微信", "QQ联系我", "私聊发你资源", "点击链接领取", "福利群",
				"add me on WhatsApp", "add me on Telegram", "click link below", "free download", "join group for free",
				"добавь в телеграм", "перейди по ссылке", "бесплатно скачать", "вступить в группу", "частный чат",
			},
			[]string{
				"这条评论在引流：%s",
				"这里明显包含%s",
				"帖子正在用%s导流",
				"this content contains %s",
				"the message uses %s for diversion",
				"сообщение содержит %s",
			},
			100,
		),
	}

	for category, cases := range categoryCases {
		if len(cases) != 100 {
			t.Fatalf("expected 100 cases for %s, got %d", category, len(cases))
		}
		for i, content := range cases {
			result := applyHardBlockRules(content)
			if result == nil {
				t.Fatalf("%s case %d expected hard block: %s", category, i, content)
			}
			if result.Category != category {
				t.Fatalf("%s case %d expected %s, got %s: %s", category, i, category, result.Category, content)
			}
		}
	}
}

func TestBulkNormalScenarioCoverage(t *testing.T) {
	normalCases := []string{
		"【N001】节奏还行，镜头切得挺顺。不带联系方式。",
		"【N002】剧情不复杂，但看着不累。没有导流内容。",
		"【N003】普通讨论观感，没有站外交易。",
		"【N004】有点暧昧，不过只是看片感受。",
		"【N005】尺度不小，但就是正常讨论。",
		"【N006】前半段铺垫稍慢，后面发力明显。",
		"【N007】我更在意拍法，近景稍微有点多。",
		"【N008】情绪拉得挺满，后半段更上头。",
		"【N009】说句粗话，这段确实挺猛，就是普通吐槽。",
		"【N010】分享下感受，整体属于耐看型。",
		"【N011】成人向描述不少，不过就是讨论内容。",
		"【N012】口语化挺重，但读起来不累。",
		"【N013】只聊观感，这条吃瓜文节奏很快。",
		"【N014】站在路人视角看，信息量不小，不含引流。",
		"【N015】说实话这展开有点狗血，没有联系方式。",
		"【N016】骂归骂，单看内容还挺抓马。",
		"【N017】文案有点夸张，不过讨论度确实高。",
		"【N018】普通吃瓜，感觉当故事看就行。",
		"【N019】this is a normal review with no contact info.",
		"【N020】just talking about the plot, no links or groups.",
		"【N021】the scene is intense but it is only a discussion.",
		"【N022】ordinary feedback, not an ad and no private contact.",
		"【N023】casual chat about the episode, nothing else.",
		"【N024】это обычное обсуждение сюжета без ссылок.",
		"【N025】просто отзыв о видео, без контактов и рекламы.",
		"【N026】обычная беседа, нет перехода во внешний чат.",
		"【N027】没有联系方式这几个字只是说明没有，不是引流。",
		"【N028】不带联系方式，不含引流，不含站外交易。",
		"【N029】普通评论，提到成人内容但没有广告。",
		"【N030】只是分享观看感受，没有任何资源交换。",
	}

	if len(normalCases) != 30 {
		t.Fatalf("expected 30 normal cases, got %d", len(normalCases))
	}

	for i, content := range normalCases {
		if result := applyHardBlockRules(content); result != nil {
			t.Fatalf("normal case %d unexpectedly blocked as %s: %s", i, result.Category, content)
		}
		if looksLikeAdOrContact(content) {
			t.Fatalf("normal case %d unexpectedly looked like ad/contact: %s", i, content)
		}
	}
}

func buildCategorySamples(keywords []string, templates []string, target int) []string {
	cases := make([]string, 0, target)
	seen := make(map[string]struct{}, target)

	for _, keyword := range keywords {
		for _, template := range templates {
			candidate := fmt.Sprintf(template, keyword)
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			cases = append(cases, candidate)
			if len(cases) == target {
				return cases
			}
		}
	}

	for len(cases) < target {
		idx := len(cases) % len(keywords)
		templateIdx := len(cases) % len(templates)
		candidate := fmt.Sprintf("%s #%03d", fmt.Sprintf(templates[templateIdx], keywords[idx]), len(cases)+1)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		cases = append(cases, candidate)
	}

	return cases
}
