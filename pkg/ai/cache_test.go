package ai

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestEnhancedCache_New 测试创建缓存
func TestEnhancedCache_New(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.items)
	assert.Equal(t, 5*time.Minute, cache.ttl)
	assert.Equal(t, 100, cache.maxSize)
}

// TestEnhancedCache_Get 测试获取缓存
func TestEnhancedCache_Get(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	t.Run("获取不存在的key", func(t *testing.T) {
		result, ok := cache.Get("notexists")
		assert.Nil(t, result)
		assert.False(t, ok)
	})

	t.Run("获取存在的key", func(t *testing.T) {
		cache.Set("key1", "value1")
		result, ok := cache.Get("key1")
		assert.Equal(t, "value1", result)
		assert.True(t, ok)
	})

	t.Run("获取过期的key", func(t *testing.T) {
		// 创建一个短ttl的缓存
		cache := NewEnhancedCache(1*time.Millisecond, 100)
		cache.Set("expiring", "value")

		// 等待过期
		time.Sleep(10 * time.Millisecond)

		result, ok := cache.Get("expiring")
		assert.Nil(t, result)
		assert.False(t, ok)
	})
}

// TestEnhancedCache_Set 测试设置缓存
func TestEnhancedCache_Set(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	t.Run("设置简单值", func(t *testing.T) {
		cache.Set("key", "value")
		result, ok := cache.Get("key")
		assert.Equal(t, "value", result)
		assert.True(t, ok)
	})

	t.Run("设置复杂类型", func(t *testing.T) {
		testData := map[string]int{"count": 42}
		cache.Set("complex", testData)

		result, ok := cache.Get("complex")
		assert.Equal(t, testData, result)
		assert.True(t, ok)
	})

	t.Run("覆盖已存在的key", func(t *testing.T) {
		cache.Set("key", "value1")
		cache.Set("key", "value2")

		result, ok := cache.Get("key")
		assert.Equal(t, "value2", result)
		assert.True(t, ok)
	})

	t.Run("多个key", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			cache.Set(fmt.Sprintf("key%d", i), i)
		}

		for i := 0; i < 10; i++ {
			result, ok := cache.Get(fmt.Sprintf("key%d", i))
			assert.Equal(t, i, result)
			assert.True(t, ok)
		}
	})
}

// TestEnhancedCache_SizeLimit 测试大小限制
func TestEnhancedCache_SizeLimit(t *testing.T) {
	// 创建小缓存
	cache := NewEnhancedCache(5*time.Minute, 5)

	// 添加超过maxSize的项目
	for i := 0; i < 10; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i)
	}

	// 验证至少有部分项目被保留
	found := 0
	for i := 0; i < 10; i++ {
		if _, ok := cache.Get(fmt.Sprintf("key%d", i)); ok {
			found++
		}
	}
	assert.Greater(t, found, 0, "应该有至少一个项目被保留")
}

// TestEnhancedCache_ConcurrentAccess 测试并发访问安全
func TestEnhancedCache_ConcurrentAccess(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	done := make(chan bool)

	for i := 0; i < 50; i++ {
		go func(idx int) {
			cache.Set(fmt.Sprintf("key%d", idx), idx)
			_, _ = cache.Get(fmt.Sprintf("key%d", idx))
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	// 不应该panic
	assert.NotPanics(t, func() {
		_, _ = cache.Get("anykey")
		cache.Set("anykey", "value")
	})
}

// TestEnhancedCache_Stats 测试统计信息
func TestEnhancedCache_Stats(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// 访问key1（命中）
	cache.Get("key1")

	// 访问不存在的键（未命中）
	cache.Get("nonexistent")

	stats := cache.GetStats()

	assert.Equal(t, int64(2), stats.ItemsAdded)
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

// TestEnhancedCache_HitRate 测试命中率
func TestEnhancedCache_HitRate(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	// 3次命中
	for i := 0; i < 3; i++ {
		cache.Set("key", "value")
		cache.Get("key")
	}

	// 1次未命中
	cache.Get("nonexistent")

	hitRate := cache.GetHitRate()
	assert.InDelta(t, 75.0, hitRate, 1.0) // 75% ± 1%
}

// TestEnhancedCache_Delete 测试删除
func TestEnhancedCache_Delete(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	assert.False(t, ok, "删除后键应该不存在")
}

// TestEnhancedCache_Clear 测试清空
func TestEnhancedCache_Clear(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	cache.Clear()

	assert.Equal(t, 0, cache.Size(), "清空后缓存大小应为0")
}

// TestEnhancedCache_LRU 测试LRU驱逐
func TestEnhancedCache_LRU(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 3)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 访问key1，使其成为最近使用
	cache.Get("key1")

	// 添加新键，key2应该被驱逐（因为key3是最新添加的）
	cache.Set("key4", "value4")

	_, ok := cache.Get("key2")
	assert.False(t, ok, "key2应该被LRU驱逐")

	_, ok = cache.Get("key1")
	assert.True(t, ok, "key1应该仍然存在")

	_, ok = cache.Get("key3")
	assert.True(t, ok, "key3应该仍然存在")
}

// TestEnhancedCache_GetLeastUsed 测试获取最久未使用
func TestEnhancedCache_GetLeastUsed(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 访问key1，使其成为最近使用
	cache.Get("key1")

	leastUsed := cache.GetLeastUsed(1)
	assert.Equal(t, 1, len(leastUsed), "期望获取1个最久未使用的条目")

	// 最久未使用的应该是key2或key3
	assert.Contains(t, []string{"key2", "key3"}, leastUsed[0].Key)
}

// TestEnhancedCache_GetMostUsed 测试获取最常使用
func TestEnhancedCache_GetMostUsed(t *testing.T) {
	cache := NewEnhancedCache(5*time.Minute, 100)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// 多次访问key1
	for i := 0; i < 5; i++ {
		cache.Get("key1")
	}

	// 访问key2一次
	cache.Get("key2")

	mostUsed := cache.GetMostUsed(1)
	assert.Equal(t, 1, len(mostUsed), "期望获取1个最常使用的条目")
	assert.Equal(t, "key1", mostUsed[0].Key, "最常使用的应该是key1")
}

// ============ 驱逐策略测试 ============

// TestEvictionPolicy_LRU 测试 LRU 驱逐策略
func TestEvictionPolicy_LRU(t *testing.T) {
	cache := NewEnhancedCacheWithPolicy(5*time.Minute, 3, EvictionPolicyLRU)

	// 添加3个条目
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 访问 key1 使其成为最近使用
	cache.Get("key1")

	// 添加第4个条目，应该驱逐 key2（最久未使用）
	cache.Set("key4", "value4")

	// 验证 key2 被驱逐
	_, ok := cache.Get("key2")
	assert.False(t, ok, "key2 应该被驱逐")

	// 验证其他条目仍然存在
	_, ok = cache.Get("key1")
	assert.True(t, ok, "key1 应该仍然存在")
	_, ok = cache.Get("key3")
	assert.True(t, ok, "key3 应该仍然存在")
	_, ok = cache.Get("key4")
	assert.True(t, ok, "key4 应该存在")
}

// TestEvictionPolicy_LFU 测试 LFU 驱逐策略
func TestEvictionPolicy_LFU(t *testing.T) {
	cache := NewEnhancedCacheWithPolicy(5*time.Minute, 3, EvictionPolicyLFU)

	// 添加3个条目
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 访问 key1 多次
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key1")

	// 访问 key2 一次
	cache.Get("key2")

	// 添加第4个条目，应该驱逐 key3（访问频率最低）
	cache.Set("key4", "value4")

	// 验证 key3 被驱逐
	_, ok := cache.Get("key3")
	assert.False(t, ok, "key3 应该被驱逐")

	// 验证其他条目仍然存在
	_, ok = cache.Get("key1")
	assert.True(t, ok, "key1 应该仍然存在")
	_, ok = cache.Get("key2")
	assert.True(t, ok, "key2 应该仍然存在")
	_, ok = cache.Get("key4")
	assert.True(t, ok, "key4 应该存在")
}

// TestEvictionPolicy_FIFO 测试 FIFO 驱逐策略
func TestEvictionPolicy_FIFO(t *testing.T) {
	cache := NewEnhancedCacheWithPolicy(5*time.Minute, 3, EvictionPolicyFIFO)

	// 添加3个条目
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 访问 key1（FIFO 不关心访问顺序）
	cache.Get("key1")
	cache.Get("key1")

	// 添加第4个条目，应该驱逐 key1（最先添加的）
	cache.Set("key4", "value4")

	// 验证 key1 被驱逐
	_, ok := cache.Get("key1")
	assert.False(t, ok, "key1 应该被驱逐")

	// 验证其他条目仍然存在
	_, ok = cache.Get("key2")
	assert.True(t, ok, "key2 应该仍然存在")
	_, ok = cache.Get("key3")
	assert.True(t, ok, "key3 应该仍然存在")
	_, ok = cache.Get("key4")
	assert.True(t, ok, "key4 应该存在")
}

// TestEvictionPolicy_TTL 测试 TTL 驱逐策略
func TestEvictionPolicy_TTL(t *testing.T) {
	cache := NewEnhancedCacheWithPolicy(10*time.Millisecond, 3, EvictionPolicyTTL)

	// 添加3个条目
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 添加第4个条目（TTL 策略不主动驱逐，但会驱逐最旧的）
	cache.Set("key4", "value4")

	// 验证缓存大小
	assert.Equal(t, 4, cache.Size(), "TTL 策略应该允许超过 maxSize")

	// 所有条目都应该存在（因为 TTL 策略不主动驱逐）
	_, ok := cache.Get("key1")
	assert.True(t, ok, "key1 应该存在")
	_, ok = cache.Get("key2")
	assert.True(t, ok, "key2 应该存在")
	_, ok = cache.Get("key3")
	assert.True(t, ok, "key3 应该存在")
	_, ok = cache.Get("key4")
	assert.True(t, ok, "key4 应该存在")

	// 等待过期
	time.Sleep(20 * time.Millisecond)

	// 所有条目都应该过期
	_, ok = cache.Get("key1")
	assert.False(t, ok, "key1 应该过期")
	_, ok = cache.Get("key2")
	assert.False(t, ok, "key2 应该过期")
}

// TestEvictionPolicy_Switching 测试策略切换
func TestEvictionPolicy_Switching(t *testing.T) {
	cache := NewEnhancedCacheWithPolicy(5*time.Minute, 3, EvictionPolicyLRU)

	// 使用 LRU 策略
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	cache.Get("key1") // 使 key1 成为最近使用

	// 切换到 FIFO 策略
	cache.SetEvictionPolicy(EvictionPolicyFIFO)
	assert.Equal(t, EvictionPolicyFIFO, cache.GetEvictionPolicy())

	// 添加第4个条目，应该驱逐 key1（最先添加的）
	cache.Set("key4", "value4")

	_, ok := cache.Get("key1")
	assert.False(t, ok, "key1 应该被驱逐（FIFO 策略）")

	// 切换回 LRU 策略
	cache.SetEvictionPolicy(EvictionPolicyLRU)
	assert.Equal(t, EvictionPolicyLRU, cache.GetEvictionPolicy())
}

// TestEvictionPolicy_Performance 测试策略性能
func TestEvictionPolicy_Performance(t *testing.T) {
	policies := []EvictionPolicy{
		EvictionPolicyLRU,
		EvictionPolicyLFU,
		EvictionPolicyFIFO,
		EvictionPolicyTTL,
	}

	for _, policy := range policies {
		t.Run(string(policy), func(t *testing.T) {
			cache := NewEnhancedCacheWithPolicy(5*time.Minute, 1000, policy)

			// 添加 2000 个条目
			for i := 0; i < 2000; i++ {
				cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
			}

			// 验证缓存大小（TTL 策略不限制大小）
			if policy != EvictionPolicyTTL {
				assert.Equal(t, 1000, cache.Size(), "缓存大小应该为 maxSize")
			}

			// 随机访问一些条目
			for i := 0; i < 100; i++ {
				key := fmt.Sprintf("key%d", i%1000)
				cache.Get(key)
			}

			// 获取统计信息
			stats := cache.GetStats()
			assert.GreaterOrEqual(t, stats.ItemsAdded, int64(2000), "应该添加了至少 2000 个条目")
		})
	}
}

// TestEvictionPolicy_Concurrency 测试并发安全性
func TestEvictionPolicy_Concurrency(t *testing.T) {
	policies := []EvictionPolicy{
		EvictionPolicyLRU,
		EvictionPolicyLFU,
		EvictionPolicyFIFO,
	}

	for _, policy := range policies {
		t.Run(string(policy), func(t *testing.T) {
			cache := NewEnhancedCacheWithPolicy(5*time.Minute, 100, policy)

			// 并发写入
			done := make(chan bool)
			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 100; j++ {
						key := fmt.Sprintf("goroutine%d_key%d", id, j)
						cache.Set(key, j)
					}
					done <- true
				}(i)
			}

			// 等待所有 goroutine 完成
			for i := 0; i < 10; i++ {
				<-done
			}

			// 并发读取
			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 100; j++ {
						key := fmt.Sprintf("goroutine%d_key%d", id, j)
						cache.Get(key)
					}
					done <- true
				}(i)
			}

			// 等待所有 goroutine 完成
			for i := 0; i < 10; i++ {
				<-done
			}

			// 验证缓存状态
			assert.Equal(t, 100, cache.Size(), "缓存大小应该为 maxSize")

			stats := cache.GetStats()
			assert.GreaterOrEqual(t, stats.ItemsAdded, int64(1000), "应该添加了至少 1000 个条目")
		})
	}
}
