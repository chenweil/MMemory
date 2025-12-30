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
