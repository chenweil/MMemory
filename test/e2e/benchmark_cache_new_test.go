package e2e

import (
	"context"
	"testing"
	"time"

	"mmemory/pkg/ai"
)

// BenchmarkEvictionPolicyComparison 驱逐策略性能对比
func BenchmarkEvictionPolicyComparison(b *testing.B) {
	policies := []ai.EvictionPolicy{
		ai.EvictionPolicyLRU,
		ai.EvictionPolicyLFU,
		ai.EvictionPolicyFIFO,
		ai.EvictionPolicyTTL,
	}

	for _, policy := range policies {
		b.Run(string(policy), func(b *testing.B) {
			cache := ai.NewEnhancedCacheWithPolicy(5*time.Minute, 1000, policy)
			
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				key := string(rune('a' + i%100))
				cache.Set(key, i)
				cache.Get(string(rune('a' + (i-1)%100)))
			}
		})
	}
}

// BenchmarkCacheWarmUp 缓存预热性能
func BenchmarkCacheWarmUp(b *testing.B) {
	cache := ai.NewEnhancedCache(5*time.Minute, 10000)

	// 创建模拟数据源
	dataSource := ai.NewMockDataSource()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		config := ai.WarmUpConfig{
			Enabled:   true,
			OnStartup: true,
			OnDemand:  false,
			Keys:      []string{"key1", "key2", "key3"},
		}
		warmer := ai.NewCacheWarmer(config, dataSource)
		warmer.WarmUp(context.Background(), cache)
	}
}

// BenchmarkShardedCacheVsSingle 分片缓存与单缓存性能对比
func BenchmarkShardedCacheVsSingle(b *testing.B) {
	b.Run("SingleCache", func(b *testing.B) {
		cache := ai.NewEnhancedCache(5*time.Minute, 10000)
		
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := string(rune('a' + i%1000))
			cache.Set(key, i)
			cache.Get(key)
		}
	})

	b.Run("ShardedCache", func(b *testing.B) {
		cache := ai.NewShardedCache(5*time.Minute, 10000, 8)
		
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := string(rune('a' + i%1000))
			cache.Set(key, i)
			cache.Get(key)
		}
	})
}
