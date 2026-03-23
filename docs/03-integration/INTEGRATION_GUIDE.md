# 🔧 鉴权和监控集成方案

在部署前，需要将新创建的 **鉴权、审计、监控** 模块整合到应用代码中。

## 集成步骤

### 步骤1：修改 handler.go

**位置**：`internal/handler/handler.go`

**修改内容**：

1️⃣ **添加新的导入**（第 12-15 行）：
```go
import (
	...
	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/monitor"
	// ... 其他导入
)
```

2️⃣ **更新 Handler 结构体**（第 17-23 行）：
```go
type Handler struct {
	svc         *service.ModerationService
	log         *logger.Logger
	cfg         *config.Config
	tasks       sync.Map                   // 存储异步任务结果
	metrics     *monitor.Metrics           // 新增：性能监控
	auditLogger *audit.AuditLogger         // 新增：审计日志
	rateLimits  map[string]*RateLimitInfo  // 新增：速率限制
	rateMu      sync.RWMutex               // 新增：速率限制锁
}

type RateLimitInfo struct {
	Count     int
	ResetTime time.Time
	Limit     int
}
```

3️⃣ **更新 New() 函数**（第 25-28 行）：
```go
func New(svc *service.ModerationService, log *logger.Logger, cfg *config.Config,
	metrics *monitor.Metrics, auditLogger *audit.AuditLogger) *Handler {
	return &Handler{
		svc:         svc,
		log:         log,
		cfg:         cfg,
		metrics:     metrics,
		auditLogger: auditLogger,
		rateLimits:  make(map[string]*RateLimitInfo),
	}
}
```

4️⃣ **增强 withMiddleware 函数**（第 42-66 行，替换整个函数）：

```go
func (h *Handler) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Project-Key")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 获取客户端 IP
		clientIP := getClientIP(r)

		// 鉴权
		var projectID string
		var apiKey string
		var statusCode int
		var errorMsg string

		if h.cfg.EnableAuth {
			apiKey = r.Header.Get("X-Project-Key")
			projectID, statusCode, errorMsg = h.authenticateKey(apiKey)
			if statusCode != http.StatusOK {
				latency := time.Since(startTime).Milliseconds()
				h.auditLogger.LogAPICall(projectID, apiKey, r.Method, r.RequestURI, statusCode, latency, clientIP, errorMsg)
				h.metrics.RecordAuth(false)
				h.jsonError(w, statusCode, errorMsg)
				return
			}
			h.metrics.RecordAuth(true)
		}

		// 执行处理函数
		next(w, r)

		// 记录审计日志
		latency := time.Since(startTime).Milliseconds()
		if h.cfg.EnableAudit && h.auditLogger != nil {
			h.auditLogger.LogAPICall(projectID, apiKey, r.Method, r.RequestURI, 200, latency, clientIP, "")
		}

		// 记录监控指标
		if h.metrics != nil {
			h.metrics.RecordRequest(latency, true, false)
		}
	}
}

// authenticateKey 验证 API 密钥并检查速率限制
func (h *Handler) authenticateKey(apiKey string) (string, int, string) {
	if apiKey == "" {
		return "", http.StatusUnauthorized, "缺少 API 密钥"
	}

	// 验证密钥是否在允许列表中
	var projectID string
	var keyLimit int

	for _, entry := range h.cfg.AllowedKeys {
		parts := strings.Split(entry, "|")
		if len(parts) >= 2 && parts[1] == apiKey {
			projectID = parts[0]
			if len(parts) >= 3 {
				// 解析速率限制
				fmt.Sscanf(parts[2], "%d", &keyLimit)
			}
			break
		}
	}

	if projectID == "" {
		return "", http.StatusUnauthorized, "无效的项目密钥"
	}

	// 检查速率限制
	if keyLimit > 0 {
		h.rateMu.Lock()
		defer h.rateMu.Unlock()

		info, exists := h.rateLimits[apiKey]
		if !exists || time.Now().After(info.ResetTime) {
			// 新的计数周期
			info = &RateLimitInfo{
				Count:     1,
				ResetTime: time.Now().Add(1 * time.Minute),
				Limit:     keyLimit,
			}
			h.rateLimits[apiKey] = info
		} else {
			info.Count++
			if info.Count > info.Limit {
				h.auditLogger.LogRateLimitExceeded(projectID, apiKey, "")
				return projectID, http.StatusTooManyRequests, fmt.Sprintf("请求过于频繁: %d/%d", info.Count-1, info.Limit)
			}
		}
	}

	return projectID, http.StatusOK, ""
}

// getClientIP 获取客户端真实 IP
func getClientIP(r *http.Request) string {
	// 优先取 X-Forwarded-For（Nginx 代理）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// 取 X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 取直接连接 IP
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}
```

5️⃣ **移除旧的 isValidKey() 函数**（第 68-75 行）
```go
// 删除此函数，已被 authenticateKey() 替代
```

### 步骤2：修改 main.go

**位置**：`cmd/server/main.go`

**修改内容**：

1️⃣ **添加新的导入**：
```go
import (
	...
	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/monitor"
)
```

2️⃣ **在 main() 中初始化监控和审计**（在创建 Handler 前）：
```go
func main() {
	// ... 现有代码 ...

	// 初始化审计服务
	auditLogger := audit.New(cfg.AuditLogDir, cfg.EnableAudit)
	defer auditLogger.Close()

	// 初始化监控指标
	metrics := monitor.NewMetrics()

	// 初始化审核服务
	svc := service.NewModerationService(cfg, lg)

	// 创建 Handler（传入新的参数）
	h := handler.New(svc, lg, cfg, metrics, auditLogger)

	// 注册路由
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 注册监控端点（可选）
	if cfg.EnableMetrics {
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(metrics.GetSnapshot())
		})
	}

	// ... 其他代码 ...
}
```

### 步骤3：编译和测试

```bash
# 检查是否有编译错误
go build ./cmd/server

# 或者完整编译
go build -o moderation-server ./cmd/server

# 测试
./moderation-server
```

## 集成检查清单

- [ ] 添加了新的导入（audit, monitor）
- [ ] 更新了 Handler 结构体
- [ ] 更新了 New() 函数签名
- [ ] 增强了 withMiddleware 函数
- [ ] 添加了 authenticateKey() 函数
- [ ] 添加了 getClientIP() 函数
- [ ] 删除了旧的 isValidKey() 函数
- [ ] 修改了 main.go 初始化代码
- [ ] 编译通过（`go build`）
- [ ] 本地测试通过

## 快速集成命令

如果你想快速完整的集成，可以用这个流程：

```bash
# 1. 备份原文件
cp internal/handler/handler.go internal/handler/handler.go.backup
cp cmd/server/main.go cmd/server/main.go.backup

# 2. 应用集成修改
# （手动编辑上述文件，按照上面的步骤修改）

# 3. 验证编译
go build ./cmd/server

# 4. 如果有错，恢复备份
cp internal/handler/handler.go.backup internal/handler/handler.go
```

## 遇到问题？

| 问题 | 解决方案 |
|-----|--------|
| `undefined: audit` | 确认 import 中有 `"github.com/.../internal/audit"` |
| `undefined: monitor` | 确认 import 中有 `"github.com/.../internal/monitor"` |
| `too many arguments to function` | 检查 `handler.New()` 和 `handler.RegisterRoutes()` 调用 |
| 编译错误 | 检查文件中是否有语法错误或遗漏的分号 |

