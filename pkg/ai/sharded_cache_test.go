package ai

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestShardedCache_New 测试创建分片缓存
func TestShardedCache_New(t *testing.T) {
	cache := NewShardedCache(5*time.Minute, 1000, 4)

	assert.NotNil(t, cache)
	assert.Equal(t, 4, cache.numShards)
	assert.Equal(t, 4, len(cache.shards))
}

// TestShardedCache_BasicOperations 测试基本操作
func TestShardedCache_BasicOperations(t *testing.T) {
	cache := NewShardedCache(5*time.Minute, 100, 4)

	// 测试 Set 和 Get
	cache.Set("key1", "value1")
	value, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", value)

	// 测试 Delete
	cache.Delete("key1")
	_, ok = cache.Get("key1")
	assert.False(t, ok)

	// 测试 Size
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	assert.Equal(t, 3, cache.Size())
}

// TestShardedCache_Distribution 测试数据分布
func TestShardedCache_Distribution(t *testing.T) {
	cache := NewShardedCache(5*time.Minute, 1000, 4)

	// 添加 100 个键
	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i)
	}

	// 检查分布
	distribution := cache.GetShardDistribution()
	assert.Equal(t, 4, len(distribution))

	// 验证总大小
	total := 0
	for _, size := range distribution {
		total += size
	}
	assert.Equal(t, 100, total)

	// 验证分布相对均匀（每个分片应该在 15-35 之间）
	for _, size := range distribution {
		assert.GreaterOrEqual(t, size, 15, "每个分片应该至少有 15 个条目")
		assert.LessOrEqual(t, size, 35, "每个分片应该最多有 35 个条目")
	}
}

// TestShardedCache_Concurrency 测试并发访问
func TestShardedCache_Concurrency(t *testing.T) {
	cache := NewShardedCache(5*time.Minute, 1000, 4)

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("goroutine%d_key%d", id, j)
				cache.Set(key, j)
			}
		}(i)
	}

	wg.Wait()

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("goroutine%d_key%d", id, j)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// 验证缓存状态（并发写入可能导致一些重复键被覆盖）
	assert.LessOrEqual(t, cache.Size(), numGoroutines*operationsPerGoroutine)

	stats := cache.GetTotalStats()
	assert.GreaterOrEqual(t, stats.ItemsAdded, int64(numGoroutines*operationsPerGoroutine))
}

// TestShardedCache_Stats 测试统计信息
func TestShardedCache_Stats(t *testing.T) {
	cache := NewShardedCache(5*time.Minute, 100, 4)

	// 添加一些数据
	for i := 0; i < 20; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i)
	}

	// 获取各分片统计
	stats := cache.GetStats()
	assert.Equal(t, 4, len(stats))

	// 获取总统计
	totalStats := cache.GetTotalStats()
	assert.Equal(t, 20, totalStats.CurrentSize)
	assert.Equal(t, int64(20), totalStats.ItemsAdded)

	// 测试命中率
	cache.Get("key1")
	cache.Get("key2")
	cache.Get("notexists")

	hitRate := cache.GetHitRate()
	assert.Greater(t, hitRate, 0.0)
	assert.Less(t, hitRate, 100.0)
}

// TestShardedCache_EvictionPolicy 测试驱逐策略
func TestShardedCache_EvictionPolicy(t *testing.T) {
	cache := NewShardedCache(5*time.Minute, 100, 4)

	// 设置 LFU 策略
	cache.SetEvictionPolicy(EvictionPolicyLFU)
	assert.Equal(t, EvictionPolicyLFU, cache.GetEvictionPolicy())

	// 添加超过容量的数据
	for i := 0; i < 500; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i)
	}

	// 验证大小限制
	assert.Equal(t, 100, cache.Size())

	// 切换到 FIFO 策略
	cache.SetEvictionPolicy(EvictionPolicyFIFO)
	assert.Equal(t, EvictionPolicyFIFO, cache.GetEvictionPolicy())
}

// TestShardedCache_Clear 测试清空缓存
func TestShardedCache_Clear(t *testing.T) {
	cache := NewShardedCache(5*time.Minute, 100, 4)

	// 添加数据
	for i := 0; i < 50; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i)
	}

	assert.Equal(t, 50, cache.Size())

	// 清空
	cache.Clear()

	assert.Equal(t, 0, cache.Size())

	// 验证所有分片都被清空
	stats := cache.GetStats()
	for _, stat := range stats {
		assert.Equal(t, 0, stat.CurrentSize)
	}
}

// TestShardedCache_WarmUp 测试预热功能
func TestShardedCache_WarmUp(t *testing.T) {
	dataSource := NewMockDataSource()
	for i := 0; i < 20; i++ {
		dataSource.Set(fmt.Sprintf("key%d", i), i)
	}

	cache := NewShardedCache(5*time.Minute, 100, 4)

	ctx := context.Background()
	keys := make([]string, 20)
	for i := 0; i < 20; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	err := cache.WarmUpWithKeys(ctx, dataSource, keys)
	assert.NoError(t, err)

	// 验证数据已加载
	assert.Equal(t, 20, cache.Size())

	for i := 0; i < 20; i++ {
		value, ok := cache.Get(fmt.Sprintf("key%d", i))
		assert.True(t, ok)
		assert.Equal(t, i, value)
	}
}

// TestShardedCache_Performance 测试性能
func TestShardedCache_Performance(t *testing.T) {
	// 对比单缓存和分片缓存的性能
	singleCache := NewEnhancedCache(5*time.Minute, 10000)
	shardedCache := NewShardedCache(5*time.Minute, 10000, 4)

	// 预填充
	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("key%d", i)
		singleCache.Set(key, i)
		shardedCache.Set(key, i)
	}

	// 测试单缓存读取性能
	start := time.Now()
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i%5000)
		singleCache.Get(key)
	}
	singleDuration := time.Since(start)

	// 测试分片缓存读取性能
	start = time.Now()
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i%5000)
		shardedCache.Get(key)
	}
	shardedDuration := time.Since(start)

	// 分片缓存应该更快（或者至少不慢太多）
	t.Logf("单缓存读取时间: %v", singleDuration)
	t.Logf("分片缓存读取时间: %v", shardedDuration)
	t.Logf("性能提升: %.2f%%", float64(singleDuration-shardedDuration)/float64(singleDuration)*100)
}

// TestShardedCache_InvalidShardCount 测试无效的分片数
func TestShardedCache_InvalidShardCount(t *testing.T) {
	// 测试 0 个分片
	cache := NewShardedCache(5*time.Minute, 100, 0)
	assert.Equal(t, 4, cache.numShards, "应该使用默认值 4")

	// 测试负数
	cache = NewShardedCache(5*time.Minute, 100, -1)
	assert.Equal(t, 4, cache.numShards, "应该使用默认值 4")
}

// TestShardedCache_Expiration 测试过期
func TestShardedCache_Expiration(t *testing.T) {
	cache := NewShardedCache(10*time.Millisecond, 100, 4)

	cache.Set("key1", "value1")

	// 等待过期
	time.Sleep(20 * time.Millisecond)

	_, ok := cache.Get("key1")
	assert.False(t, ok, "key1 应该过期")
}