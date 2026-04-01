package service

type moderationRuleSpec struct {
	category string
	reason   string
	keywords []string
}

var politicsStrongKeywords = []string{
	"政变", "暴动", "起义", "颠覆政权", "独立运动", "分裂国家", "港独", "台独", "藏独",
	"非法集会", "示威游行", "政府腐败内幕", "高层黑料", "分裂主义", "反政府", "敏感政治",
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
	"政府", "总统", "总理", "议会", "选举", "政治",
	"government", "president", "primeminister", "parliament", "election", "politics",
	"правительство", "президент", "премьерминистр", "парламент", "выборы", "политика",
}

var hardBlockRules = []moderationRuleSpec{
	{
		category: "adult",
		reason:   "命中高风险成人内容",
		keywords: []string{
			// 移除单字"药"（会误拦"吃药"、"中药"等正常内容），仅保留"迷药"
			"约炮", "一夜情", "上门服务", "迷药", "裸聊", "裸照", "强奸", "轮奸",
			"幼女", "未成年儿童", "招嫖", "援交", "包养", "开房", "性服务", "迷奸",
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
			"刷单", "兼职", "日结", "高薪", "带你赚钱", "稳赚不赔", "私服", "代充", "黑卡",
			"赌博", "博彩", "上分", "bc", "BC", "下分", "外围", "足彩", "彩票", "盘口", "庄家",
			"杀猪盘", "资金盘", "投资返利", "套路贷", "套路盘",
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
			"weaponforsale", "homemadebomb", "massacre", "detonator", "firearm", "buildabomb",
			"убийство", "взрыв", "бомба", "террористическаяатака", "взрывчатка",
			"оружиекупить", "патроны",
		},
	},
}

var benignNegationPhrases = []string{
	"不含任何联系方式", "没有任何联系方式", "不含任何导流", "没有任何导流",
	"不带联系方式", "没有联系方式", "无联系方式", "不含联系方式", "未留联系方式",
	"没有导流内容", "不含引流", "无引流", "没有导流", "不带导流", "不是引流",
	"没有站外交易", "不含站外交易", "无站外交易", "不是广告", "非广告",
	// 弱否定：只是提及或举例
	"只是提到", "只是举例", "只是说", "只是提及", "只是提起",
	"nocontactinfo", "nocontactinformation", "nodiversion", "notanad", "noprivatecontact", "nooffplatformdeal",
	"безконтактов", "безрекламы", "безссылок",
}

var directContactKeywords = []string{
	"q群", "企鹅号", "qq号", "QQ号", "qq", "QQ", "tg", "TG", "bc", "BC",
	"wechat", "telegram", "wechat群",
	"微x", "微ｘ",
	"微信", "加我", "联系", "联系我", "私聊", "拉群", "群号", "加群", "入群",
	"Telegram", "whatsapp", "WhatsApp", "vx", "vx号", "加v", "加vx",
	"代理", "加盟", "引流", "外链", "网址", "链接", "下载地址", "扫码", "二维码",
	"主页联系", "看我头像", "站外继续聊", "邮箱",
	"t.me", "discord.gg", "bit.ly", "tinyurl",
	".com", ".cn", ".net", ".org", ".ru", ".cc", ".xyz", ".top", ".io", ".me",
	"http://", "https://", "www.",
}

var weakTradeDirectPhrases = []string{
	"去别处看", "主页找我", "看资料", "私下聊", "站外价更低", "外站价更低",
	// 更精准的词组，避免误拦
	"资源打包", "完整版资源", "有偿分享", "付费进群", "私聊发你资源", "点击链接领取",
	"招募代理", "招代理", "代理合作", "寻找代理商",
	"招盟商", "加盟我们",
	// 加入单纯的"加Q"、"加微信"等词组（避免单字但保留词组级检测）
	"加q", "加Q", "加qq", "加QQ", "加微信", "加薇", "加v",
	// 隐蔽的微信变体（空格分隔或其他形式）
	"wei xin", "wei xin:", "weixin", "weixinid", "weixinnumber",
	"飞机号", "飞机",  // Telegram 变体
	"免费看片", "福利群",
	"dmme", "contactmeprivately", "addmeonwhatsapp",
	"addmeontelegram", "clicklinkbelow", "freedownload", "freevideo", "joingroupforfree",
	"добавьвтелеграм", "напишивличку", "перейдипоссылке", "бесплатноскачать", "вступитьвгруппу", "частныйчат",
}

var weakTradeTokens = []string{
	"低价", "完整版", "打包", "有偿", "私下", "别处",
	// 移除单字"资源"和"代理"（会误拦"资源丰富"、"正规代理商"等正常内容）
	// 这些词的检测现在改为在 weakTradeDirectPhrases 的词组级别
	"free", "download", "video", "group", "telegram", "whatsapp",
	"скачать", "телеграм", "ссылке", "группу",
}
