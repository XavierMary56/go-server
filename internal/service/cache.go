package service

import (
	"sync"
	"time"
)

// Cache 缓存接口
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
}

// ── 内存缓存（默认，无需依赖）─────────────────────────────

type memEntry struct {
	value     interface{}
	expiresAt time.Time
}

type memoryCache struct {
	mu   sync.RWMutex
	data map[string]*memEntry
	ttl  time.Duration
}

func newMemoryCache(ttlSeconds int) Cache {
	c := &memoryCache{
		data: make(map[string]*memEntry),
		ttl:  time.Duration(ttlSeconds) * time.Second,
	}
	// 启动定期清理 goroutine
	go c.cleanup()
	return c
}

func (c *memoryCache) Get(key string) (interface{}, bool) {
	if c.ttl == 0 {
		return nil, false
	}
	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (c *memoryCache) Set(key string, value interface{}) {
	if c.ttl == 0 {
		return
	}
	c.mu.Lock()
	c.data[key] = &memEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *memoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

func (c *memoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.data {
			if now.After(v.expiresAt) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}
