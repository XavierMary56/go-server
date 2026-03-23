# ✅ 部署检查清单（完整版）

## 📊 当前项目状态

### ✓ 已完成
- [x] 创建了 **API 鉴权模块** (`internal/auth/auth.go`)
- [x] 创建了 **监控指标模块** (`internal/monitor/metrics.go`)
- [x] 创建了 **审计日志模块** (`internal/audit/audit.go`)
- [x] 创建了 **生产配置** (`.env.production`)
- [x] 创建了 **部署脚本** (`deploy.sh`)
- [x] 创建了 **监控脚本** (`monitor.sh`)
- [x] 创建了 **密钥管理脚本** (`manage-keys.sh`)
- [x] 创建了 **Nginx 配置** (`deploy/nginx.conf.production`)
- [x] 创建了 **完整文档** (DEPLOYMENT.md, AUTH_AND_MONITORING.md)
- [x] 更新了 **config.go** 以支持审计/监控配置

### ⏳ 待完成（需要代码集成）
- [ ] 修改 `internal/handler/handler.go`
  - 添加 audit 和 monitor 导入
  - 更新 Handler 结构体
  - 增强 withMiddleware 中间件
  - 添加 authenticateKey() 和 getClientIP() 函数
- [ ] 修改 `cmd/server/main.go`
  - 初始化 audit.Logger 和 monitor.Metrics
  - 传入新参数给 handler.New()
  - 注册 /metrics 端点

---

## 🚀 完成部署的 2 种方式

### 方式 A：手动集成（推荐，需要 5-10 分钟）

1. **按照 INTEGRATION_GUIDE.md 手动修改代码**
   - 编辑 `internal/handler/handler.go`
   - 编辑 `cmd/server/main.go`
   - 验证编译：`go build ./cmd/server`

2. **运行部署脚本**
   ```bash
   bash deploy.sh
   ```

3. **配置密钥和监控**
   ```bash
   # 编辑 .env（若还未创建）
   cp .env.production .env
   nano .env

   # 启动服务
   docker-compose up -d

   # 验证
   bash monitor.sh status
   ```

### 方式 B：快速部署（不集成新鉴权，使用现有鉴权）

如果你想跳过代码修改，直接部署现有功能：

```bash
# 1. 准备配置
cp .env.production .env
nano .env  # 填入 API Key

# 2. 启动服务（使用现有的简单鉴权）
docker-compose up -d

# 3. 验证
curl https://localhost:8080/v1/health
```

> **注意**：此方式无法使用新的竞速率限制和详细审计功能

---

## 📋 代码集成详细步骤

### Step 1：修改 internal/handler/handler.go

**需要修改的地方：**

1. 第 3-15 行：添加新导入
2. 第 17-23 行：更新 Handler 结构体（+3 个字段）
3. 第 25-28 行：更新 New() 函数签名
4. 第 42-66 行：重写 withMiddleware() 函数
5. 第 68-75 行：删除旧的 isValidKey() 函数
6. 添加新函数：authenticateKey()、getClientIP()

**预计修改量**：~150 行代码

**复杂度**：中等（主要是复制粘贴）

### Step 2：修改 cmd/server/main.go

**需要修改的地方：**

1. 第 3-17 行：添加新导入（audit, monitor）
2. 第 34-40 行：初始化 audit.Logger 和 monitor.Metrics
3. 第 39-42 行：更新 handler.New() 调用
4. 第 43-50 行：添加 /metrics 端点

**预计修改量**：~30 行代码

**复杂度**：低（只是初始化和注册）

### Step 3：验证编译

```bash
# 进入项目目录
cd /d/Users/Public/php20250819/2026www/go-server

# 检查是否有编译错误
go build ./cmd/server

# 如果成功，会在当前目录生成 server 可执行文件
ls -la server
```

---

## 🔍 集成前的清单

✅ 检查项

- [ ] 已阅读 INTEGRATION_GUIDE.md 并理解修改内容
- [ ] 已备份原始文件：`cp handler.go handler.go.backup`
- [ ] 已经安装 Go 1.21+（`go version`）
- [ ] 已经安装 Docker 和 Docker Compose

---

## 📝 代码修改简述

### handler.go 关键修改

```diff
- import (...)  // 旧的导入
+ import (
+   "github.com/.../internal/audit"
+   "github.com/.../internal/monitor"
+   ...
+ )

- type Handler struct {
-   svc   *service.ModerationService
-   log   *logger.Logger
-   cfg   *config.Config
-   tasks sync.Map
- }
+ type Handler struct {
+   svc         *service.ModerationService
+   log         *logger.Logger
+   cfg         *config.Config
+   tasks       sync.Map
+   metrics     *monitor.Metrics           // ✨ 新增
+   auditLogger *audit.AuditLogger         // ✨ 新增
+   rateLimits  map[string]*RateLimitInfo  // ✨ 新增
+   rateMu      sync.RWMutex
+ }

- func New(svc..., log..., cfg...) *Handler {
-   return &Handler{svc: svc, log: log, cfg: cfg}
- }
+ func New(svc..., log..., cfg..., metrics..., auditLogger...) *Handler {  // ✨ 新参数
+   return &Handler{
+     svc: svc, log: log, cfg: cfg,
+     metrics: metrics,        // ✨ 新增
+     auditLogger: auditLogger,// ✨ 新增
+     rateLimits: make(...),   // ✨ 新增
+   }
+ }
```

### main.go 关键修改

```diff
import (
+  "github.com/.../internal/audit"
+  "github.com/.../internal/monitor"
)

func main() {
  ...

+ auditLogger := audit.New(cfg.AuditLogDir, cfg.EnableAudit)  // ✨ 新增
+ metrics := monitor.NewMetrics()                             // ✨ 新增

- h := handler.New(svc, lg, cfg)
+ h := handler.New(svc, lg, cfg, metrics, auditLogger)       // ✨ 更新参数

  ...
}
```

---

## ⏱️ 时间估计

| 任务 | 时间 |
|-----|-----|
| 阅读和理解 INTEGRATION_GUIDE.md | 5 分钟 |
| 修改 handler.go | 5 分钟 |
| 修改 main.go | 3 分钟 |
| 编译验证 | 2 分钟 |
| 配置和部署 | 5 分钟 |
| **总计** | **20 分钟** |

---

## 🎯 下一步推荐

1. **现在就做？**
   ```bash
   # 按照 INTEGRATION_GUIDE.md 修改代码
   nano internal/handler/handler.go
   nano cmd/server/main.go
   go build ./cmd/server
   bash deploy.sh
   ```

2. **稍后再做？**
   - 保存好 INTEGRATION_GUIDE.md 和创建的文档
   - 新的模块文件已在 internal/auth、internal/monitor、internal/audit
   - 当准备好了，再按照清单修改代码

3. **需要帮助？**
   - 查看 INTEGRATION_GUIDE.md 中的"遇到问题？"部分
   - 所有新创建的模块已经是完整可用的

---

## 📚 相关文档

| 文档 | 用途 |
|-----|------|
| **INTEGRATION_GUIDE.md** | 💡 代码修改指南（详细步骤） |
| **DEPLOYMENT.md** | 🚀 完整部署指南 |
| **AUTH_AND_MONITORING.md** | 🔐 鉴权和监控快速参考 |
| **deploy.sh** | 🛠️ 自动部署脚本 |
| **monitor.sh** | 📊 监控和日志脚本 |
| **manage-keys.sh** | 🔑 密钥管理脚本 |

--

## ❓ 常见问题

**Q：能不能跳过代码集成，直接用现有鉴权部署？**
A：可以。新创建的模块是可选的，现有的 handler.go 中已经有基础的鉴权。但无法使用速率限制和详细审计功能。

**Q：集成后会不会破坏现有功能？**
A：不会。新的代码完全兼容现有功能，只是增强了鉴权和监控。

**Q：需要重新编译镜像吗？**
A：是的，修改代码后需要重新编译。`docker-compose up --build` 会自动处理。

**Q：我是否可以分步集成？**
A：可以。你可以先集成监控模块，再集成鉴权模块。模块之间相对独立。

