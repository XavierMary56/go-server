---
description: Go 并发专项性能分析与优化建议
---

# Go 并发专项性能分析与优化建议

适用场景：当问题集中在 goroutine、channel、mutex/rwmutex、下游慢调用堆积时优先使用本规则；通用性能问题可参考 `go-performance-guide.md`，线上定位流程可参考 `go-pprof-troubleshooting-handbook.md`。

## 一、重点分析方向
- goroutine 数量是否失控
- channel 是否形成阻塞链
- mutex / rwmutex 是否热点竞争
- 下游服务慢导致上游 goroutine 堆积

## 二、优化建议
- 明确 goroutine 生命周期与退出机制
- 限制并发度，使用 worker pool
- 对 channel 使用场景做清晰约束
- 缩短锁持有时间
- 为下游调用设置超时、熔断、隔离

## 三、审查清单
- 是否有 goroutine 泄漏
- 是否无上限创建协程
- 是否在持锁期间执行 IO
- 是否因为慢消费导致生产堆积

## 四、验证工具
- pprof block / mutex
- trace
- goroutine profile
- Prometheus 指标
