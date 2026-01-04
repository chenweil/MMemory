package ai

import (
	"context"
	"sync"
	"time"
)

// ShardedCache 分片缓存，用于提高并发性能
type ShardedCache struct {
	shards    []*EnhancedCache
	numShards int
	hashFunc  func(string) uint32
	mu        sync.RWMutex
}

// NewShardedCache 创建分片缓存
func NewShardedCache(ttl time.Duration, maxSize int, numShards int) *ShardedCache {
	if numShards < 1 {
		numShards = 4 // 默认 4 个分片
	}

	shards := make([]*EnhancedCache, numShards)
	shardSize := maxSize / numShards
	if shardSize < 10 {
		shardSize = 10 // 最小每个分片 10 个条目
	}

	for i := 0; i < numShards; i++ {
		shards[i] = NewEnhancedCache(ttl, shardSize)
	}

	return &ShardedCache{
		shards:    shards,
		numShards: numShards,
		hashFunc:  fnv32aHash, // FNV-1a 32-bit 哈希
	}
}

// getShard 获取键所在的分片
func (sc *ShardedCache) getShard(key string) *EnhancedCache {
	hash := sc.hashFunc(key)
	return sc.shards[hash%uint32(sc.numShards)]
}

// Get 获取缓存值
func (sc *ShardedCache) Get(key string) (interface{}, bool) {
	return sc.getShard(key).Get(key)
}

// Set 设置缓存值
func (sc *ShardedCache) Set(key string, value interface{}) {
	sc.getShard(key).Set(key, value)
}

// Delete 删除缓存值
func (sc *ShardedCache) Delete(key string) {
	sc.getShard(key).Delete(key)
}

// Size 获取缓存总大小
func (sc *ShardedCache) Size() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	total := 0
	for _, shard := range sc.shards {
		total += shard.Size()
	}
	return total
}

// Clear 清空所有分片
func (sc *ShardedCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, shard := range sc.shards {
		shard.Clear()
	}
}

// GetStats 获取所有分片的统计信息
func (sc *ShardedCache) GetStats() []CacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := make([]CacheStats, sc.numShards)
	for i, shard := range sc.shards {
		stats[i] = shard.GetStats()
	}
	return stats
}

// GetTotalStats 获取合并后的统计信息
func (sc *ShardedCache) GetTotalStats() CacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	total := CacheStats{}
	for _, shard := range sc.shards {
		shardStats := shard.GetStats()
		total.Hits += shardStats.Hits
		total.Misses += shardStats.Misses
		total.Evictions += shardStats.Evictions
		total.ItemsAdded += shardStats.ItemsAdded
		total.ItemsHit += shardStats.ItemsHit
		total.CurrentSize += shardStats.CurrentSize
	}

	return total
}

// GetHitRate 获取整体缓存命中率
func (sc *ShardedCache) GetHitRate() float64 {
	stats := sc.GetTotalStats()
	total := stats.Hits + stats.Misses
	if total == 0 {
		return 0
	}
	return float64(stats.Hits) / float64(total) * 100
}

// SetEvictionPolicy 设置所有分片的驱逐策略
func (sc *ShardedCache) SetEvictionPolicy(policy EvictionPolicy) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, shard := range sc.shards {
		shard.SetEvictionPolicy(policy)
	}
}

// GetEvictionPolicy 获取驱逐策略（所有分片应该相同）
func (sc *ShardedCache) GetEvictionPolicy() EvictionPolicy {
	if sc.numShards == 0 {
		return EvictionPolicyLRU
	}
	return sc.shards[0].GetEvictionPolicy()
}

// WarmUpWithKeys 预热所有分片
func (sc *ShardedCache) WarmUpWithKeys(ctx context.Context, dataSource DataSource, keys []string) error {
	// 按分片分组键
	shardKeys := make([][]string, sc.numShards)
	for _, key := range keys {
		hash := sc.hashFunc(key)
		shardIndex := hash % uint32(sc.numShards)
		shardKeys[shardIndex] = append(shardKeys[shardIndex], key)
	}

	// 并行预热各分片
	var wg sync.WaitGroup
	errChan := make(chan error, sc.numShards)

	for i, keys := range shardKeys {
		if len(keys) == 0 {
			continue
		}

		wg.Add(1)
		go func(shardIndex int, keys []string) {
			defer wg.Done()
			err := sc.shards[shardIndex].WarmUpWithKeys(ctx, dataSource, keys)
			if err != nil {
				errChan <- err
			}
		}(i, keys)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// GetShardDistribution 获取各分片的数据分布
func (sc *ShardedCache) GetShardDistribution() []int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	distribution := make([]int, sc.numShards)
	for i, shard := range sc.shards {
		distribution[i] = shard.Size()
	}

	return distribution
}

// fnv32aHash FNV-1a 32-bit 哈希函数
func fnv32aHash(key string) uint32 {
	const prime32 = uint32(16777619)
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= prime32
	}
	return hash
}