package config

import (
	"os"
	"strconv"
	"strings"
)

// ModelConfig 单个模型配置
type ModelConfig struct {
	ID       string // 模型 ID，如 claude-sonnet-4-20250514 / gpt-4o / grok-2
	Name     string // 显示名称
	Weight   int    // 调度权重 (0-100)
	Priority int    // 故障转移优先级（数字越小越优先）
	Provider string // anthropic | openai | grok（留空则按 ID 前缀自动识别）
}

// Config 全局配置
type Config struct {
	// 服务器配置
	Port int
	Env  string // production | development

	// Anthropic API
	AnthropicAPIKey string
	AnthropicAPIURL string
	AnthropicVer    string

	// OpenAI API
	OpenAIAPIKey string
	OpenAIAPIURL string

	// Grok (xAI) API
	GrokAPIKey string
	GrokAPIURL string

	// 模型池
	Models                    []ModelConfig
	EnableModelConfigFallback bool // 数据库无模型配置时是否回退到配置模型

	// 请求配置
	APITimeout int // 单次 API 超时（秒）
	MaxRetries int // 失败重试次数

	// 缓存
	CacheDriver string // memory | redis
	CacheTTL    int    // 缓存秒数（0=禁用）
	RedisAddr   string
	RedisPass   string
	RedisDB     int
	RedisPrefix string

	// 鉴权
	EnableAuth  bool
	AllowedKeys []string // 各项目的接入密钥

	// 管理员 API
	EnableAdminAPI bool
	AdminToken     string // 管理员令牌（逗号分隔多个）

	// 审计和监控
	EnableAudit   bool
	AuditLogDir   string
	EnableMetrics bool
	MetricsPort   int

	// 日志
	LogDir   string
	LogLevel string // debug | info | warn | error
}

// Load 加载配置，优先读取环境变量，其次使用默认值
func Load() (*Config, error) {
	// 尝试加载 .env 文件（开发环境）
	loadDotEnv()

	cfg := &Config{
		Port:            getEnvInt("PORT", 8080),
		Env:             getEnv("APP_ENV", "production"),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicAPIURL: getEnv("ANTHROPIC_API_URL", "https://api.anthropic.com/v1/messages"),
		AnthropicVer:    getEnv("ANTHROPIC_VERSION", "2023-06-01"),
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		OpenAIAPIURL:    getEnv("OPENAI_API_URL", "https://api.openai.com/v1/chat/completions"),
		GrokAPIKey:      getEnv("GROK_API_KEY", ""),
		GrokAPIURL:      getEnv("GROK_API_URL", "https://api.x.ai/v1/chat/completions"),
		Models: []ModelConfig{
			{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Weight: 100, Priority: 1, Provider: "anthropic"},
		},
		EnableModelConfigFallback: getEnvBool("ENABLE_MODEL_CONFIG_FALLBACK", true),
		APITimeout:                getEnvInt("API_TIMEOUT", 10),
		MaxRetries:      getEnvInt("MAX_RETRIES", 2),
		CacheDriver:     getEnv("CACHE_DRIVER", "memory"),
		CacheTTL:        getEnvInt("CACHE_TTL", 60),
		RedisAddr:       getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass:       getEnv("REDIS_PASS", ""),
		RedisDB:         getEnvInt("REDIS_DB", 0),
		RedisPrefix:     getEnv("REDIS_PREFIX", "mod:"),
		EnableAuth:      getEnvBool("ENABLE_AUTH", false),
		EnableAdminAPI:  getEnvBool("ENABLE_ADMIN_API", true),
		AdminToken:      getEnv("ADMIN_TOKEN", "admin-token-default"),
		EnableAudit:     getEnvBool("ENABLE_AUDIT", false),
		AuditLogDir:     getEnv("AUDIT_LOG_DIR", "./logs/audit"),
		EnableMetrics:   getEnvBool("ENABLE_METRICS", false),
		MetricsPort:     getEnvInt("METRICS_PORT", 9090),
		LogDir:          getEnv("LOG_DIR", "./logs"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}

	// 解析项目密钥列表
	if keys := getEnv("ALLOWED_KEYS", ""); keys != "" {
		cfg.AllowedKeys = strings.Split(keys, ",")
		for i, k := range cfg.AllowedKeys {
			cfg.AllowedKeys[i] = strings.TrimSpace(k)
		}
	}

	return cfg, nil
}

// ── 环境变量读取工具 ─────────────────────────────────────────

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

// loadDotEnv 简单解析 .env 文件（不依赖第三方库的备用实现）
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return // .env 不存在时静默忽略
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if os.Getenv(key) == "" { // 不覆盖已有的环境变量
			os.Setenv(key, val)
		}
	}
}
