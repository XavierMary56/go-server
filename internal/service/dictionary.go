package service

type moderationRuleSpec struct {
	category string
	reason   string
	keywords []string
}

var politicsStrongKeywords = []string{
	"政变", "暴动", "起义", "颠覆政权", "独立运动", "分裂国家", "港独", "台独", "藏独",
	"非法集会", "示威游行", "政府腐败内幕", "高层黑料", "分裂主义", "反政府", "敏感政治",
	"coup", "rebellion", "overthrowgovernment", "separatism", "independencemovement",
	"illegalprotest", "riot", "governmentcorruptionscandal", "politicalleak", "classifiedinfo",
	"regimechange", "antigovernment",
	"государственныйпереворот", "мятеж", "восстание", "сепаратизм", "независимость",
	"незаконныймитинг", "протест", "политическийскандал", "утечка",
}

var politicsContextKeywords = []string{
	"政府", "总统", "总理", "议会", "选举", "政治",
	"government", "president", "primeminister", "parliament", "election", "politics",
	"правительство", "президент", "премьерминистр", "парламент", "выборы", "политика",
}

var hardBlockRules = []moderationRuleSpec{
	{
		category: "adult",
		reason:   "命中高风险成人内容",
		keywords: []string{
			"约炮", "一夜情", "上门服务", "迷药", "药", "裸聊", "裸照", "强奸", "轮奸",
			"幼女", "未成年", "儿童", "招嫖", "援交", "包养", "开房", "性服务", "迷奸",
			"hookup", "onenightstand", "nude", "nudes", "sexvideo", "porn", "rape", "gangrape",
			"prostitution", "escortservice", "escort", "sexualservice", "paidsex",
			"underagesex", "minorporn", "childporn",
			"сексзаденьги", "порно", "сексвидео", "изнасилование", "интимуслуги",
			"эскорт", "несовершеннолетнийсекс",
		},
	},
	{
		category: "fraud",
		reason:   "命中诈骗、赌博或黑产内容",
		keywords: []string{
			"刷单", "兼职日结高薪", "带你赚钱", "稳赚不赔", "私服", "代充", "黑卡",
			"赌博", "博彩", "上分", "bc", "杀猪盘", "资金盘", "投资返利", "套路盘",
			"跑分", "洗钱", "代收代付", "卡商", "接码", "出售账号",
			"makemoneyfast", "guaranteedprofit", "scam", "phishing", "gambling", "betting",
			"casinoonline", "fakeaccount", "stolencard", "investmentscheme", "ponzi",
			"ponzischeme", "moneylaundering", "carding", "accountselling",
			"быстрыйзаработок", "мошенничество", "ставки", "казино", "фишинг",
			"финансоваяпирамида", "отмываниеденег",
		},
	},
	{
		category: "abuse",
		reason:   "命中毒品或违禁品内容",
		keywords: []string{
			"冰毒", "海洛因", "摇头丸", "买毒", "出货", "走货", "制毒", "配方",
			"大麻", "吸毒工具", "k粉", "麻古",
			"drugs", "cocaine", "heroin", "meth", "buydrugs", "selldrugs", "drugrecipe",
			"howtomakedrugs", "weed", "marijuana", "narcotics", "drugdealer",
			"ketamine", "fentanyl", "mdma", "lsd",
			"наркотики", "кокаин", "героин", "купитьнаркотики", "изготовлениенаркотиков",
		},
	},
	{
		category: "violence",
		reason:   "命中暴力、恐怖或武器内容",
		keywords: []string{
			"杀人", "暗杀", "爆炸", "制作炸弹", "枪支购买", "恐怖袭击", "自制武器", "凶杀",
			"引爆", "炸药", "雷管", "制枪", "改枪", "子弹",
			"kill", "assassination", "bombmaking", "explosives", "terroristattack", "gunforsale",
			"weaponforsale", "homemadebomb", "massacre", "detonator", "ammo", "firearm", "buildabomb",
			"убийство", "взрыв", "бомба", "террористическаяатака", "взрывчатка",
			"оружиекупить", "патроны",
		},
	},
}

var benignNegationPhrases = []string{
	"不带联系方式", "没有联系方式", "无联系方式", "不含联系方式", "未留联系方式",
	"没有导流内容", "不含引流", "无引流", "没有导流", "不带导流", "不是引流",
	"没有站外交易", "不含站外交易", "无站外交易", "不是广告", "非广告",
	"nocontactinfo", "nocontactinformation", "nodiversion", "notanad", "noprivatecontact", "nooffplatformdeal",
	"безконтактов", "безрекламы", "безссылок",
}

var directContactKeywords = []string{
	"微信", "wechat", "wx", "vx", "vx号", "加v", "加vx", "加q", "qq",
	"telegram", "tg", "whatsapp", "line", "discord", "skype", "邮箱", "email",
	"加我", "联系", "联系我", "contactme", "dmme", "messageme", "私聊", "拉群", "群号",
	"代理", "加盟", "引流", "外链", "网址", "链接", "下载地址", "扫码", "二维码",
	"主页联系", "看我头像", "站外继续聊", "t.me", "discord.gg", "bit.ly", "tinyurl",
	".com", ".cn", ".net", ".org", ".ru", ".cc", ".xyz", ".top", ".io", ".me",
	"http://", "https://", "www.",
}

var weakTradeDirectPhrases = []string{
	"去别处看", "主页找我", "看资料", "私下聊", "站外价更低", "外站价更低",
	"资源打包", "完整版资源", "有偿分享", "付费进群", "私聊发你资源", "点击链接领取",
	"免费看片", "福利群", "dmme", "contactmeprivately", "addmeonwhatsapp",
	"addmeontelegram", "clicklinkbelow", "freedownload", "freevideo", "joingroupforfree",
	"добавьвтелеграм", "напишивличку", "перейдипоссылке", "бесплатноскачать", "вступитьвгруппу", "частныйчат",
}

var weakTradeTokens = []string{
	"低价", "资源", "完整版", "打包", "有偿", "私下", "别处", "看资料",
	"free", "download", "video", "group", "telegram", "whatsapp",
	"скачать", "телеграм", "ссылке", "группу",
}
