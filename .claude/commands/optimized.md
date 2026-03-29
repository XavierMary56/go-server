---
name: 优化审查
description: 面向当前 Go 仓库的代码优化审查规范
---
# 代码优化审查规范

## 一、优先检查项

1. **性能瓶颈** - 识别 O(n²) 操作、低效循环、重复分配
2. **并发问题** - 查找 goroutine 泄漏、锁竞争、channel 阻塞
3. **内存与 GC 压力** - 检查临时对象暴涨、字符串与 `[]byte` 反复转换
4. **接口链路效率** - 识别串行下游调用、过大响应体、日志/鉴权开销
5. **安全问题** - 查找 SQL 注入、XSS、硬编码密码或 API key

## 二、仓库约束
- 默认按 Go 仓库审查，不使用多语言通用模板。
- 优先参考：
  - `.claude/rules/repo-go-guardrails.md`
  - `.claude/rules/repo-architecture.md`
  - `.claude/rules/go-performance-guide.md`
  - `.claude/rules/go-concurrency-performance-guide.md`
  - `.claude/rules/go-pprof-troubleshooting-handbook.md`
- 不要虚构 `make test`、`make lint`、`npm test`、`pytest`、`golangci-lint` 等仓库中不存在的命令。

## 三、输出格式
每个问题按以下格式输出：
- 严重级别
- 代码位置
- 问题说明
- 建议修复方式
