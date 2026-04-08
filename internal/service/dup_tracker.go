package service

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// DupTracker 跟踪每个项目每天的重复内容提交次数
type DupTracker struct {
	mu      sync.Mutex
	entries map[string]int // key: "project:date:contentHash" -> count
	lastDay string         // 用于每日清理
}

// NewDupTracker 创建新的重复内容跟踪器
func NewDupTracker() *DupTracker {
	return &DupTracker{
		entries: make(map[string]int),
		lastDay: time.Now().Format("2006-01-02"),
	}
}

// CheckAndIncrement 检查内容是否超过项目的每日重复限制。
// 返回 true 表示内容已超限（应被拒绝）。
// maxDailyDup <= 0 表示不限制。
func (dt *DupTracker) CheckAndIncrement(projectName, content string, maxDailyDup int) bool {
	if maxDailyDup <= 0 {
		return false // 不限制
	}

	dt.mu.Lock()
	defer dt.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	// 每日清理
	if today != dt.lastDay {
		dt.entries = make(map[string]int)
		dt.lastDay = today
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	key := fmt.Sprintf("%s:%s:%s", projectName, today, hash)

	count := dt.entries[key]
	if count >= maxDailyDup {
		return true // 超限
	}

	dt.entries[key] = count + 1
	return false
}
