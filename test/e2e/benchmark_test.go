package e2e

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"mmemory/pkg/ai"
	sqliterepo "mmemory/internal/repository/sqlite"
)

// BenchmarkCachePerformance 缓存性能基准测试
func BenchmarkCachePerformance(b *testing.B) {
	cache := ai.NewEnhancedCache(5*time.Minute, 10000)

	b.Run("SetOperation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Set(
				string(rune('a'+i%26))+string(rune('0'+i%10)),
				map[string]interface{}{"value": i, "data": "test"},
			)
		}
	})

	b.Run("GetOperation", func(b *testing.B) {
		// 预填充缓存
		for i := 0; i < 1000; i++ {
			cache.Set(string(rune('a'+i%26)), i)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Get(string(rune('a' + i%26)))
		}
	})

	b.Run("ConcurrentAccess", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				key := string(rune('a' + counter%26))
				cache.Set(key, counter)
				cache.Get(key)
				counter++
			}
		})
	})
}

// BenchmarkCacheHitRate 缓存命中率基准测试
func BenchmarkCacheHitRate(b *testing.B) {
	cache := ai.NewEnhancedCache(5*time.Minute, 1000)

	// 填充缓存
	for i := 0; i < 100; i++ {
		cache.Set("key", "value")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

// BenchmarkQueryOptimizerPerformance 查询优化器性能基准测试
func BenchmarkQueryOptimizerPerformance(b *testing.B) {
	optimizer := sqliterepo.NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	b.Run("RecordQuery", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			optimizer.RecordQuery(
				"users",
				"SELECT",
				"SELECT * FROM users WHERE id = 1",
				5*time.Millisecond,
				1,
				false,
			)
		}
	})

	b.Run("GetMetrics", func(b *testing.B) {
		// 预填充一些查询
		for i := 0; i < 100; i++ {
			optimizer.RecordQuery("users", "SELECT", "SELECT * FROM users", 5*time.Millisecond, 1, false)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			optimizer.GetMetrics()
		}
	})

	b.Run("GetSlowQueryStats", func(b *testing.B) {
		// 预填充一些慢查询
		for i := 0; i < 50; i++ {
			optimizer.RecordQuery("reminders", "INSERT", "INSERT INTO reminders", 50*time.Millisecond, 1, false)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			optimizer.GetSlowQueryStats()
		}
	})
}

// BenchmarkLRUPerformance LRU驱逐性能基准测试
func BenchmarkLRUPerformance(b *testing.B) {
	b.Run("LRUEviction", func(b *testing.B) {
		// 创建小缓存以触发驱逐
		cache := ai.NewEnhancedCache(5*time.Minute, 100)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Set(string(rune('a'+i%26)), i)
		}
	})

	b.Run("LRUWithAccess", func(b *testing.B) {
		cache := ai.NewEnhancedCache(5*time.Minute, 100)

		// 预填充
		for i := 0; i < 100; i++ {
			cache.Set(string(rune('a'+i%26)), i)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 交替访问和写入以触发LRU
			cache.Get(string(rune('a' + i%26)))
			cache.Set(string(rune('b'+(i%26))), i)
		}
	})
}

// BenchmarkCostControllerPerformance 成本控制器性能基准测试
func BenchmarkCostControllerPerformance(b *testing.B) {
	// 创建测试用的 logger
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.ErrorLevel)

	providers := map[string]*ai.ProviderCost{
		"openai": {
			CostPer1KTokens: 0.01,
			Model:            "gpt-4o-mini",
			Provider:        "openai",
			Enabled:         true,
			Priority:        1,
		},
		"claude": {
			CostPer1KTokens: 0.015,
			Model:            "claude-3-5-sonnet",
			Provider:        "claude",
			Enabled:         true,
			Priority:        2,
		},
	}
	budget := ai.BudgetConfig{
		MonthlyBudget: 100.0,
		DailyBudget:   5.0,
		UserBudget:    1.0,
	}

	costCtrl := ai.NewCostController(providers, budget, testLogger)
	defer costCtrl.Stop()

	b.Run("SetMonthlyCost", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			costCtrl.SetMonthlyCost("openai", float64(i))
		}
	})

	b.Run("GetMonthlyReport", func(b *testing.B) {
		// 预填充数据
		costCtrl.SetMonthlyCost("openai", 10.0)
		costCtrl.SetMonthlyCost("claude", 5.0)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			costCtrl.GetMonthlyReport()
		}
	})

	b.Run("PredictCosts", func(b *testing.B) {
		// 预填充数据
		costCtrl.SetMonthlyCost("openai", 10.0)
		costCtrl.SetMonthlyCost("claude", 5.0)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			costCtrl.PredictCosts()
		}
	})

	b.Run("IsOverBudget", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			costCtrl.IsOverBudget()
		}
	})

	b.Run("IsNearBudgetLimit", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			costCtrl.IsNearBudgetLimit(0.8)
		}
	})
}

// BenchmarkCacheStatsPerformance 缓存统计性能基准测试
func BenchmarkCacheStatsPerformance(b *testing.B) {
	cache := ai.NewEnhancedCache(5*time.Minute, 1000)

	// 预填充
	for i := 0; i < 100; i++ {
		cache.Set(string(rune('a'+i%26)), i)
	}

	b.Run("GetStats", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.GetStats()
		}
	})

	b.Run("GetHitRate", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.GetHitRate()
		}
	})

	b.Run("Size", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Size()
		}
	})

	b.Run("GetLeastUsed", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.GetLeastUsed(10)
		}
	})

	b.Run("GetMostUsed", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.GetMostUsed(10)
		}
	})
}

// BenchmarkDeleteAndClearPerformance 删除和清空性能基准测试
func BenchmarkDeleteAndClearPerformance(b *testing.B) {
	b.Run("Delete", func(b *testing.B) {
		cache := ai.NewEnhancedCache(5*time.Minute, 1000)

		// 预填充
		for i := 0; i < 100; i++ {
			cache.Set(string(rune('a'+i%26)), i)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Delete(string(rune('a' + i%26)))
			cache.Set(string(rune('a' + i%26)), i) // 重新添加以保持测试一致
		}
	})

	b.Run("Clear", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache := ai.NewEnhancedCache(5*time.Minute, 1000)
			// 预填充
			for j := 0; j < 100; j++ {
				cache.Set(string(rune('a'+j%26)), j)
			}
			cache.Clear()
		}
	})
}

// BenchmarkCacheThroughput 缓存吞吐量基准测试
func BenchmarkCacheThroughput(b *testing.B) {
	cache := ai.NewEnhancedCache(5*time.Minute, 100000)

	b.Run("HighVolumeWrites", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Set(string(rune('a'+i%26)), i)
		}
	})

	b.Run("HighVolumeReads", func(b *testing.B) {
		// 预填充大量数据
		for i := 0; i < 10000; i++ {
			cache.Set(string(rune('a'+i%10000)), i)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Get(string(rune('a' + i%10000)))
		}
	})

	b.Run("MixedReadWrite", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cache.Set(string(rune('a'+i%26)), i)
			cache.Get(string(rune('a' + (i-1)%26)))
		}
	})
}
