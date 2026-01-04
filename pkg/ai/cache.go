package ai

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"mmemory/pkg/logger"
	"mmemory/pkg/metrics"
)

// EvictionPolicy 驱逐策略类型
type EvictionPolicy string

const (
	EvictionPolicyLRU  EvictionPolicy = "lru"  // Least Recently Used
	EvictionPolicyLFU  EvictionPolicy = "lfu"  // Least Frequently Used
	EvictionPolicyFIFO EvictionPolicy = "fifo" // First In First Out
	EvictionPolicyTTL  EvictionPolicy = "ttl"  // Time To Live
)

// EvictionStrategy 驱逐策略接口
type EvictionStrategy interface {
	ShouldEvict(cache *EnhancedCache) bool
	SelectVictim(cache *EnhancedCache) string
	OnAccess(entry *CacheEntry)
	OnAdd(entry *CacheEntry)
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Key         string
	Value       interface{}
	Expiration  time.Time
	AccessedAt  time.Time
	AccessCount int64
	AddedAt     time.Time // 添加时间，用于 FIFO
}

// EnhancedCache 增强版内存缓存
type EnhancedCache struct {
	items     map[string]*list.Element
	lruList   *list.List // LRU 链表
	fifoQueue *list.List // FIFO 队列（用于 FIFO 策略）
	ttl       time.Duration
	maxSize   int
	maxMemory int64 // 最大内存使用（字节）
	currentMemory int64

	// 驱逐策略
	evictionStrategy EvictionStrategy
	policyType       EvictionPolicy

	// 统计
	mu    sync.RWMutex
	stats CacheStats
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

// NewEnhancedCache 创建增强版缓存（默认 LRU 策略）
func NewEnhancedCache(ttl time.Duration, maxSize int) *EnhancedCache {
	return NewEnhancedCacheWithPolicy(ttl, maxSize, EvictionPolicyLRU)
}

// NewEnhancedCacheWithPolicy 创建指定驱逐策略的缓存
func NewEnhancedCacheWithPolicy(ttl time.Duration, maxSize int, policy EvictionPolicy) *EnhancedCache {
	cache := &EnhancedCache{
		items:     make(map[string]*list.Element),
		lruList:   list.New(),
		fifoQueue: list.New(),
		ttl:       ttl,
		maxSize:   maxSize,
		policyType: policy,
	}

	// 根据策略初始化驱逐策略
	cache.evictionStrategy = cache.createStrategy(policy)

	// 启动清理协程
	go cache.cleanup()

	return cache
}

// createStrategy 根据策略类型创建驱逐策略实例
func (c *EnhancedCache) createStrategy(policy EvictionPolicy) EvictionStrategy {
	switch policy {
	case EvictionPolicyLFU:
		return &LFUStrategy{}
	case EvictionPolicyFIFO:
		return &FIFOStrategy{queue: c.fifoQueue}
	case EvictionPolicyTTL:
		return &TTLStrategy{}
	default:
		return &LRUStrategy{lruList: c.lruList}
	}
}

// SetEvictionPolicy 动态切换驱逐策略
func (c *EnhancedCache) SetEvictionPolicy(policy EvictionPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.policyType = policy
	c.evictionStrategy = c.createStrategy(policy)
}

// GetEvictionPolicy 获取当前驱逐策略
func (c *EnhancedCache) GetEvictionPolicy() EvictionPolicy {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.policyType
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
		c.fifoQueue.Remove(element) // 从 FIFO 队列中移除
		c.stats.Misses++
		metrics.RecordCacheMiss()
		return nil, false
	}

	// 更新访问信息
	entry.AccessedAt = time.Now()
	entry.AccessCount++

	// LRU 策略：将元素移动到链表前端
	if c.policyType == EvictionPolicyLRU {
		c.lruList.MoveToFront(element)
	}

	// 调用驱逐策略的 OnAccess 回调
	if c.evictionStrategy != nil {
		c.evictionStrategy.OnAccess(entry)
	}

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

		// 调用驱逐策略的 OnAccess 回调
		if c.evictionStrategy != nil {
			c.evictionStrategy.OnAccess(entry)
		}

		return
	}

	// 检查是否需要驱逐
	if c.maxSize > 0 && len(c.items) >= c.maxSize {
		if c.evictionStrategy != nil && c.evictionStrategy.ShouldEvict(c) {
			victimKey := c.evictionStrategy.SelectVictim(c)
			if victimKey != "" {
				c.deleteInternal(victimKey)
			}
		}
	}

	// 添加新条目
	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		Expiration:  expiration,
		AccessedAt:  now,
		AccessCount: 1,
		AddedAt:     now,
	}

	// 添加到 LRU 链表
	lruElement := c.lruList.PushFront(entry)
	c.items[key] = lruElement

	// 添加到 FIFO 队列
	fifoElement := c.fifoQueue.PushBack(key)

	// 将两个链表元素关联起来（使用相同的值）
	lruElement.Value = entry
	fifoElement.Value = key

	// 调用驱逐策略的 OnAdd 回调
	if c.evictionStrategy != nil {
		c.evictionStrategy.OnAdd(entry)
	}

	c.stats.ItemsAdded++
}

// deleteInternal 内部删除方法（不获取锁）
func (c *EnhancedCache) deleteInternal(key string) {
	if element, exists := c.items[key]; exists {
		delete(c.items, key)
		c.lruList.Remove(element)

		// 从 FIFO 队列中移除
		for e := c.fifoQueue.Front(); e != nil; e = e.Next() {
			if e.Value.(string) == key {
				c.fifoQueue.Remove(e)
				break
			}
		}

		c.stats.Evictions++
		metrics.RecordCacheEviction()
	}
}

// Delete 删除缓存条目
func (c *EnhancedCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleteInternal(key)
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
	c.fifoQueue = list.New()
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

		// 清理 FIFO 队列中的过期条目
		for e := c.fifoQueue.Front(); e != nil; {
			next := e.Next()
			key := e.Value.(string)
			if _, exists := c.items[key]; !exists {
				c.fifoQueue.Remove(e)
			}
			e = next
		}

		if evicted > 0 {
			c.stats.Evictions += int64(evicted)
		}

		// 记录缓存命中率指标
		hitRate := c.GetHitRate()
		metrics.RecordCacheHitRate("default", string(c.policyType), hitRate)

		// 记录缓存大小
		metrics.SetCacheSize(float64(c.Size()))

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

// ============ 驱逐策略实现 ============

// LRUStrategy Least Recently Used 策略
type LRUStrategy struct {
	lruList *list.List
}

func (s *LRUStrategy) ShouldEvict(cache *EnhancedCache) bool {
	return cache.maxSize > 0 && len(cache.items) >= cache.maxSize
}

func (s *LRUStrategy) SelectVictim(cache *EnhancedCache) string {
	if element := s.lruList.Back(); element != nil {
		entry := element.Value.(*CacheEntry)
		return entry.Key
	}
	return ""
}

func (s *LRUStrategy) OnAccess(entry *CacheEntry) {
	// LRU 策略需要将访问的元素移动到链表前端
	// 注意：这里无法直接访问 cache.lruList，需要在 cache.Get 中处理
}

func (s *LRUStrategy) OnAdd(entry *CacheEntry) {
	// LRU 策略在 Set 方法中已经处理了 PushFront
}

// LFUStrategy Least Frequently Used 策略
type LFUStrategy struct {
	frequencyMap map[string]int64
	mu           sync.RWMutex
}

func (s *LFUStrategy) ShouldEvict(cache *EnhancedCache) bool {
	return cache.maxSize > 0 && len(cache.items) >= cache.maxSize
}

func (s *LFUStrategy) SelectVictim(cache *EnhancedCache) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var minFreq int64 = math.MaxInt64
	var victimKey string

	// 遍历所有条目，找到访问频率最低的
	for key, element := range cache.items {
		entry := element.Value.(*CacheEntry)
		freq, exists := s.frequencyMap[key]
		if !exists {
			freq = entry.AccessCount
		}
		if freq < minFreq {
			minFreq = freq
			victimKey = key
		}
	}

	return victimKey
}

func (s *LFUStrategy) OnAccess(entry *CacheEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.frequencyMap == nil {
		s.frequencyMap = make(map[string]int64)
	}
	s.frequencyMap[entry.Key]++
}

func (s *LFUStrategy) OnAdd(entry *CacheEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.frequencyMap == nil {
		s.frequencyMap = make(map[string]int64)
	}
	s.frequencyMap[entry.Key] = 1
}

// FIFOStrategy First In First Out 策略
type FIFOStrategy struct {
	queue *list.List
}

func (s *FIFOStrategy) ShouldEvict(cache *EnhancedCache) bool {
	return cache.maxSize > 0 && len(cache.items) >= cache.maxSize
}

func (s *FIFOStrategy) SelectVictim(cache *EnhancedCache) string {
	if element := s.queue.Front(); element != nil {
		return element.Value.(string)
	}
	return ""
}

func (s *FIFOStrategy) OnAccess(entry *CacheEntry) {
	// FIFO 策略不关心访问顺序
}

func (s *FIFOStrategy) OnAdd(entry *CacheEntry) {
	// FIFO 策略在 Set 方法中已经处理了 PushBack
}

// TTLStrategy Time To Live 策略
type TTLStrategy struct{}

func (s *TTLStrategy) ShouldEvict(cache *EnhancedCache) bool {
	// TTL 策略总是依赖过期时间，不主动驱逐
	return false
}

func (s *TTLStrategy) SelectVictim(cache *EnhancedCache) string {
	// TTL 策略不主动选择受害者
	return ""
}

func (s *TTLStrategy) OnAccess(entry *CacheEntry) {
	// TTL 策略不关心访问顺序
}

func (s *TTLStrategy) OnAdd(entry *CacheEntry) {
	// TTL 策略不关心添加顺序
}

// ============ 缓存预热 ============

// Warmer 缓存预热器接口
type Warmer interface {
	WarmUp(ctx context.Context, cache *EnhancedCache) error
}

// WarmUpConfig 预热配置
type WarmUpConfig struct {
	Enabled   bool
	OnStartup bool
	OnDemand  bool
	Keys      []string // 预热的键列表
}

// DataSource 数据源接口（用于预热）
type DataSource interface {
	Get(ctx context.Context, key string) (interface{}, error)
}

// CacheWarmer 缓存预热器
type CacheWarmer struct {
	config     WarmUpConfig
	dataSource DataSource
}

// NewCacheWarmer 创建缓存预热器
func NewCacheWarmer(config WarmUpConfig, dataSource DataSource) *CacheWarmer {
	return &CacheWarmer{
		config:     config,
		dataSource: dataSource,
	}
}

// WarmUp 执行预热
func (w *CacheWarmer) WarmUp(ctx context.Context, cache *EnhancedCache) error {
	if !w.config.Enabled {
		return nil
	}

	successCount := 0
	failCount := 0

	for _, key := range w.config.Keys {
		value, err := w.dataSource.Get(ctx, key)
		if err != nil {
			failCount++
			continue
		}

		cache.Set(key, value)
		successCount++
	}

	if successCount > 0 {
		logger.Infof("缓存预热完成，成功加载 %d 个条目，失败 %d 个", successCount, failCount)
	}

	return nil
}

// StartWarmUp 启动预热（异步）
func (c *EnhancedCache) StartWarmUp(ctx context.Context, warmer Warmer) error {
	go func() {
		if err := warmer.WarmUp(ctx, c); err != nil {
			logger.Errorf("缓存预热失败: %v", err)
		}
	}()
	return nil
}

// WarmUpWithKeys 使用指定的键进行预热
func (c *EnhancedCache) WarmUpWithKeys(ctx context.Context, dataSource DataSource, keys []string) error {
	config := WarmUpConfig{
		Enabled: true,
		Keys:    keys,
	}
	warmer := NewCacheWarmer(config, dataSource)
	return warmer.WarmUp(ctx, c)
}

// ============ 测试辅助数据源 ============

// MockDataSource 模拟数据源（用于测试）
type MockDataSource struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewMockDataSource 创建模拟数据源
func NewMockDataSource() *MockDataSource {
	return &MockDataSource{
		data: make(map[string]interface{}),
	}
}

// Set 设置数据
func (m *MockDataSource) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Get 获取数据
func (m *MockDataSource) Get(ctx context.Context, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}
