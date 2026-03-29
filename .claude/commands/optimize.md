---
description: 面向当前 Go 仓库的代码性能审查与优化建议
---

# Go 仓库代码优化审查

对当前改动进行性能审查时，优先结合：
- `.claude/rules/repo-go-guardrails.md`
- `.claude/rules/repo-architecture.md`
- `.claude/rules/go-performance-guide.md`
- `.claude/rules/go-concurrency-performance-guide.md`
- `.claude/rules/go-pprof-troubleshooting-handbook.md`

## 审查优先级
1. **性能瓶颈**：识别 O(n²)、热点循环、重复分配、无效序列化
2. **并发问题**：检查 goroutine 泄漏、channel 阻塞、锁竞争、下游慢调用堆积
3. **内存与 GC 压力**：查找短生命周期对象暴涨、频繁转换、切片/map 未预分配
4. **接口链路效率**：检查多次串行下游调用、响应体过大、日志/鉴权开销过高
5. **安全与可维护性**：检查硬编码凭据、XSS、SQL 注入、命名与结构问题

## 输出要求
每个问题按以下结构输出：
- 严重级别（Critical / High / Medium / Low）
- 代码位置（文件与行号）
- 问题说明
- 建议修复方式

## 约束
- 不要虚构仓库中不存在的命令、Makefile、lint 或 coverage 流程。
- 若建议验证命令，优先使用 `CLAUDE.md` 中已有的 Go 命令。
- 结合当前仓库真实架构给建议，不按通用多语言模板输出。
