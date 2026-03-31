---
description: Go pprof 性能排查手册
---

# Go pprof 性能排查手册

适用场景：用于线上或压测中的性能定位。若只是做常规代码级性能建议，优先参考 `go-performance-guide.md`；若问题集中在并发堆积、阻塞与锁竞争，结合 `go-concurrency-performance-guide.md` 一起使用。

## 一、适用场景
- Go 服务 CPU 飙高
- 内存增长异常
- goroutine 堆积
- 锁竞争或阻塞明显

## 二、排查顺序
1. 先确认异常是 CPU、内存、锁、阻塞还是 goroutine 数量问题
2. 用 pprof 抓对应 profile
3. 结合业务链路确认热点函数
4. 最后再改代码并复测

## 三、重点 profile
### 1. CPU profile
适合看：
- 哪些函数最耗 CPU
- 是否有热点循环、序列化、压缩、正则等问题

### 2. Heap / allocs
适合看：
- 哪些函数分配最多
- 是否有短生命周期对象暴涨
- 是否切片、map、字符串分配过多

### 3. Goroutine
适合看：
- 是否 goroutine 泄漏
- 是否某些调用长时间不退出

### 4. Mutex / block
适合看：
- 锁竞争
- channel 阻塞
- 下游调用拖慢上游协程

## 四、常见根因
- 无上限创建 goroutine
- 热点路径重复分配对象
- 持锁执行 IO
- channel 设计不合理
- 下游慢调用导致堆积

## 五、处理方向
- 给 goroutine 生命周期加约束
- 预分配 slice/map，减少临时对象
- 缩小锁粒度
- 为 IO 和下游依赖设置超时与并发限制

## 六、复盘模板
1. 异常指标
2. 采集的 profile 类型
3. Top hotspot / top alloc / top block
4. 根因定位
5. 优化动作
6. 复测结果
