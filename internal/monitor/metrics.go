package monitor

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics 系统监控指标
type Metrics struct {
	mu sync.RWMutex

	// 请求相关
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailedRequests  int64   `json:"failed_requests"`
	CachedRequests  int64   `json:"cached_requests"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	latencies       []int64 // 循环队列
	latenciesIdx    int     // 循环队列索引
	latencySum      int64   // 增量求和
	latencyCount    int64   // 已填充的槽位数（最大 len(latencies)）

	// API 调用
	APICallsTotal   int64   `json:"api_calls_total"`
	APICallsSuccess int64   `json:"api_calls_success"`
	APICallsFailed  int64   `json:"api_calls_failed"`
	AvgAPILatencyMs float64 `json:"avg_api_latency_ms"`

	// 模型使用
	ModelUsage   map[string]int64 `json:"model_usage"`
	ModelUsageMu sync.RWMutex

	// 错误统计
	ErrorCounts   map[string]int64 `json:"error_counts"`
	ErrorCountsMu sync.RWMutex

	// 鉴权
	AuthSuccessCount int64 `json:"auth_success_count"`
	AuthFailCount    int64 `json:"auth_fail_count"`

	// 时间
	StartTime     time.Time `json:"start_time"`
	LastResetTime time.Time `json:"last_reset_time"`
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	return &Metrics{
		latencies:     make([]int64, 1000), // 最近 1000 个请求的延迟
		ModelUsage:    make(map[string]int64),
		ErrorCounts:   make(map[string]int64),
		StartTime:     time.Now(),
		LastResetTime: time.Now(),
	}
}

// RecordRequest 记录请求
func (m *Metrics) RecordRequest(latencyMs int64, success bool, fromCache bool) {
	if success {
		atomic.AddInt64(&m.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&m.FailedRequests, 1)
	}
	atomic.AddInt64(&m.TotalRequests, 1)

	if fromCache {
		atomic.AddInt64(&m.CachedRequests, 1)
	}

	// 更新延迟信息（增量公式，O(1)）
	m.mu.Lock()
	oldVal := m.latencies[m.latenciesIdx]
	m.latencies[m.latenciesIdx] = latencyMs
	m.latenciesIdx = (m.latenciesIdx + 1) % len(m.latencies)
	m.latencySum = m.latencySum - oldVal + latencyMs
	if m.latencyCount < int64(len(m.latencies)) {
		m.latencyCount++
	}
	if m.latencyCount > 0 {
		m.AvgLatencyMs = float64(m.latencySum) / float64(m.latencyCount)
	}
	m.mu.Unlock()
}

// RecordAPICall 记录 API 调用
func (m *Metrics) RecordAPICall(model string, latencyMs int64, success bool) {
	if success {
		atomic.AddInt64(&m.APICallsSuccess, 1)
	} else {
		atomic.AddInt64(&m.APICallsFailed, 1)
	}
	atomic.AddInt64(&m.APICallsTotal, 1)

	// 更新模型使用统计
	m.ModelUsageMu.Lock()
	m.ModelUsage[model]++
	m.ModelUsageMu.Unlock()

	// 更新 API 平均延迟（简化计算）
	m.mu.Lock()
	m.AvgAPILatencyMs = (m.AvgAPILatencyMs*float64(m.APICallsTotal-1) + float64(latencyMs)) / float64(m.APICallsTotal)
	m.mu.Unlock()
}

// RecordError 记录错误
func (m *Metrics) RecordError(errType string) {
	m.ErrorCountsMu.Lock()
	m.ErrorCounts[errType]++
	m.ErrorCountsMu.Unlock()
}

// RecordAuth 记录认证
func (m *Metrics) RecordAuth(success bool) {
	if success {
		atomic.AddInt64(&m.AuthSuccessCount, 1)
	} else {
		atomic.AddInt64(&m.AuthFailCount, 1)
	}
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.ModelUsageMu.RLock()
	modelUsage := make(map[string]int64)
	for k, v := range m.ModelUsage {
		modelUsage[k] = v
	}
	m.ModelUsageMu.RUnlock()

	m.ErrorCountsMu.RLock()
	errorCounts := make(map[string]int64)
	for k, v := range m.ErrorCounts {
		errorCounts[k] = v
	}
	m.ErrorCountsMu.RUnlock()

	uptime := time.Since(m.StartTime)

	return map[string]interface{}{
		"uptime_seconds":       int64(uptime.Seconds()),
		"total_requests":       atomic.LoadInt64(&m.TotalRequests),
		"success_requests":     atomic.LoadInt64(&m.SuccessRequests),
		"failed_requests":      atomic.LoadInt64(&m.FailedRequests),
		"cached_requests":      atomic.LoadInt64(&m.CachedRequests),
		"avg_latency_ms":       fmt.Sprintf("%.2f", m.AvgLatencyMs),
		"success_rate_percent": fmt.Sprintf("%.2f", float64(atomic.LoadInt64(&m.SuccessRequests))*100/float64(atomic.LoadInt64(&m.TotalRequests))),
		"api_calls":            atomic.LoadInt64(&m.APICallsTotal),
		"api_calls_success":    atomic.LoadInt64(&m.APICallsSuccess),
		"api_calls_failed":     atomic.LoadInt64(&m.APICallsFailed),
		"avg_api_latency_ms":   fmt.Sprintf("%.2f", m.AvgAPILatencyMs),
		"model_usage":          modelUsage,
		"error_counts":         errorCounts,
		"auth_success":         atomic.LoadInt64(&m.AuthSuccessCount),
		"auth_fail":            atomic.LoadInt64(&m.AuthFailCount),
		"start_time":           m.StartTime.Format(time.RFC3339),
		"last_reset_time":      m.LastResetTime.Format(time.RFC3339),
	}
}

// Reset 重置指标
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt64(&m.TotalRequests, 0)
	atomic.StoreInt64(&m.SuccessRequests, 0)
	atomic.StoreInt64(&m.FailedRequests, 0)
	atomic.StoreInt64(&m.CachedRequests, 0)
	atomic.StoreInt64(&m.APICallsTotal, 0)
	atomic.StoreInt64(&m.APICallsSuccess, 0)
	atomic.StoreInt64(&m.APICallsFailed, 0)
	atomic.StoreInt64(&m.AuthSuccessCount, 0)
	atomic.StoreInt64(&m.AuthFailCount, 0)

	m.AvgLatencyMs = 0
	m.AvgAPILatencyMs = 0
	m.latencySum = 0
	m.latencyCount = 0
	for i := range m.latencies {
		m.latencies[i] = 0
	}

	m.ModelUsageMu.Lock()
	m.ModelUsage = make(map[string]int64)
	m.ModelUsageMu.Unlock()

	m.ErrorCountsMu.Lock()
	m.ErrorCounts = make(map[string]int64)
	m.ErrorCountsMu.Unlock()

	m.LastResetTime = time.Now()
}
