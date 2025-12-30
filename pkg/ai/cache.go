package ai

import (
	"container/list"
	"sync"
	"time"

	"mmemory/pkg/metrics"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Key        string
	Value      interface{}
	Expiration time.Time
	AccessedAt time.Time
	AccessCount int64
}

// EnhancedCache 增强版内存缓存
type EnhancedCache struct {
	items     map[string]*list.Element
	lruList   *list.List // LRU 链表
	ttl       time.Duration
	maxSize   int
	maxMemory int64 // 最大内存使用（字节）
	currentMemory int64

	// 统计
	mu          sync.RWMutex
	stats       CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	ItemsAdded int64
	ItemsHit   int64
	CurrentSize int
}

// NewEnhancedCache 创建增强版缓存
func NewEnhancedCache(ttl time.Duration, maxSize int) *EnhancedCache {
	cache := &EnhancedCache{
		items:   make(map[string]*list.Element),
		lruList: list.New(),
		ttl:     ttl,
		maxSize: maxSize,
	}

	// 启动清理协程
	go cache.cleanup()

	return cache
}

// Get 获取缓存
func (c *EnhancedCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.items[key]
	if !exists {
		c.stats.Misses++
		metrics.RecordCacheMiss()
		return nil, false
	}

	entry := element.Value.(*CacheEntry)

	// 检查是否过期
	if time.Now().After(entry.Expiration) {
		delete(c.items, key)
		c.lruList.Remove(element)
		c.stats.Misses++
		metrics.RecordCacheMiss()
		return nil, false
	}

	// 更新访问信息（LRU）
	c.lruList.MoveToFront(element)
	entry.AccessedAt = time.Now()
	entry.AccessCount++

	c.stats.Hits++
	c.stats.ItemsHit++
	metrics.RecordCacheHit()

	return entry.Value, true
}

// Set 设置缓存
func (c *EnhancedCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiration := now.Add(c.ttl)

	// 如果键已存在，更新值
	if element, exists := c.items[key]; exists {
		entry := element.Value.(*CacheEntry)
		entry.Value = value
		entry.Expiration = expiration
		entry.AccessedAt = now
		entry.AccessCount++
		c.lruList.MoveToFront(element)
		return
	}

	// 检查是否需要驱逐
	if c.maxSize > 0 && len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	// 添加新条目
	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		Expiration:  expiration,
		AccessedAt:  now,
		AccessCount: 1,
	}

	element := c.lruList.PushFront(entry)
	c.items[key] = element
	c.stats.ItemsAdded++
}

// Delete 删除缓存条目
func (c *EnhancedCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		delete(c.items, key)
		c.lruList.Remove(element)
	}
}

// evictOldest 驱逐最久未使用的条目
func (c *EnhancedCache) evictOldest() {
	if element := c.lruList.Back(); element != nil {
		entry := element.Value.(*CacheEntry)
		delete(c.items, entry.Key)
		c.lruList.Remove(element)
		c.stats.Evictions++
		metrics.RecordCacheEviction()
	}
}

// GetStats 获取缓存统计
func (c *EnhancedCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:        c.stats.Hits,
		Misses:      c.stats.Misses,
		Evictions:   c.stats.Evictions,
		ItemsAdded:  c.stats.ItemsAdded,
		ItemsHit:    c.stats.ItemsHit,
		CurrentSize: len(c.items),
	}
}

// GetHitRate 获取缓存命中率
func (c *EnhancedCache) GetHitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total) * 100
}

// Size 返回当前缓存大小
func (c *EnhancedCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear 清空缓存
func (c *EnhancedCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lruList = list.New()
}

// cleanup 定期清理过期条目
func (c *EnhancedCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		evicted := 0

		for key, element := range c.items {
			entry := element.Value.(*CacheEntry)
			if now.After(entry.Expiration) {
				delete(c.items, key)
				c.lruList.Remove(element)
				evicted++
			}
		}

		if evicted > 0 {
			c.stats.Evictions += int64(evicted)
		}

		c.mu.Unlock()
	}
}

// GetLeastUsed 获取最久未使用的条目
func (c *EnhancedCache) GetLeastUsed(limit int) []CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]CacheEntry, 0, limit)

	// 从链表尾部开始获取
	element := c.lruList.Back()
	for element != nil && len(entries) < limit {
		entry := element.Value.(*CacheEntry)
		entries = append(entries, *entry)
		element = element.Prev()
	}

	return entries
}

// GetMostUsed 获取访问最频繁的条目
func (c *EnhancedCache) GetMostUsed(limit int) []CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]CacheEntry, 0, limit)

	// 遍历并排序
	type entryWithCount struct {
		entry  *CacheEntry
		access int64
	}

	allEntries := make([]entryWithCount, 0, len(c.items))
	for element := c.lruList.Front(); element != nil; element = element.Next() {
		entry := element.Value.(*CacheEntry)
		allEntries = append(allEntries, entryWithCount{entry, entry.AccessCount})
	}

	// 排序（按访问次数降序）
	for i := 0; i < len(allEntries); i++ {
		for j := i + 1; j < len(allEntries); j++ {
			if allEntries[j].access > allEntries[i].access {
				allEntries[i], allEntries[j] = allEntries[j], allEntries[i]
			}
		}
	}

	// 返回前N个
	for i := 0; i < len(allEntries) && i < limit; i++ {
		entries = append(entries, *allEntries[i].entry)
	}

	return entries
}
