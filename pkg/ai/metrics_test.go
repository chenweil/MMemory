package ai

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// 初始化prometheus注册表用于测试
func init() {
	// 重置指标以避免测试间污染
	aiParseRequestsTotal.Reset()
	aiParseDuration.Reset()
	aiTokensUsed.Reset()
	aiCacheHitRatio.Reset()
	aiCircuitBreakerState.Reset()
	aiRateLimitExceeded.Reset()
}

// TestProviderMetrics_New 测试创建指标
func TestProviderMetrics_New(t *testing.T) {
	metrics := NewProviderMetrics()

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.cacheHits)
	assert.NotNil(t, metrics.cacheMisses)
}

// TestProviderMetrics_RecordRequest 测试记录请求
func TestProviderMetrics_RecordRequest(t *testing.T) {
	metrics := NewProviderMetrics()

	// 重置指标
	aiParseRequestsTotal.Reset()

	t.Run("记录成功请求", func(t *testing.T) {
		metrics.RecordRequest("openai", true, 0.5)

		// 验证计数器增加
		count := testutil.ToFloat64(aiParseRequestsTotal.WithLabelValues("openai", "success"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("记录失败请求", func(t *testing.T) {
		metrics.RecordRequest("claude", false, 1.0)

		count := testutil.ToFloat64(aiParseRequestsTotal.WithLabelValues("claude", "failure"))
		assert.Equal(t, float64(1), count)
	})
}

// TestProviderMetrics_RecordTokens 测试记录token使用
func TestProviderMetrics_RecordTokens(t *testing.T) {
	metrics := NewProviderMetrics()

	// 重置指标
	aiTokensUsed.Reset()

	t.Run("记录token", func(t *testing.T) {
		metrics.RecordTokens("openai", 100)

		count := testutil.ToFloat64(aiTokensUsed.WithLabelValues("openai"))
		assert.Equal(t, float64(100), count)
	})

	t.Run("累加token", func(t *testing.T) {
		metrics.RecordTokens("openai", 50)

		count := testutil.ToFloat64(aiTokensUsed.WithLabelValues("openai"))
		assert.Equal(t, float64(150), count)
	})
}

// TestProviderMetrics_CacheHitRatio 测试缓存命中率
func TestProviderMetrics_CacheHitRatio(t *testing.T) {
	metrics := NewProviderMetrics()

	t.Run("初始命中率", func(t *testing.T) {
		// 初始状态无数据，命中率应该为0
		hits := metrics.getCacheHits("openai")
		misses := metrics.getCacheMisses("openai")
		assert.Equal(t, int64(0), hits)
		assert.Equal(t, int64(0), misses)
	})

	t.Run("记录命中", func(t *testing.T) {
		metrics.RecordCacheHit("openai")
		metrics.RecordCacheHit("openai")

		hits := metrics.getCacheHits("openai")
		assert.Equal(t, int64(2), hits)
	})

	t.Run("记录未命中", func(t *testing.T) {
		metrics.RecordCacheMiss("openai")

		misses := metrics.getCacheMisses("openai")
		assert.Equal(t, int64(1), misses)
	})

	t.Run("多Provider隔离", func(t *testing.T) {
		metrics.RecordCacheHit("openai")
		metrics.RecordCacheMiss("claude")

		assert.Equal(t, int64(3), metrics.getCacheHits("openai"))
		assert.Equal(t, int64(0), metrics.getCacheHits("claude"))
		assert.Equal(t, int64(1), metrics.getCacheMisses("claude"))
	})
}

// TestRecordCircuitBreakerState 测试记录熔断器状态
func TestRecordCircuitBreakerState(t *testing.T) {
	// 重置指标
	aiCircuitBreakerState.Reset()

	t.Run("记录关闭状态", func(t *testing.T) {
		RecordCircuitBreakerState("openai", StateClosed)

		value := testutil.ToFloat64(aiCircuitBreakerState.WithLabelValues("openai"))
		assert.Equal(t, float64(StateClosed), value)
	})

	t.Run("记录开启状态", func(t *testing.T) {
		RecordCircuitBreakerState("claude", StateOpen)

		value := testutil.ToFloat64(aiCircuitBreakerState.WithLabelValues("claude"))
		assert.Equal(t, float64(StateOpen), value)
	})

	t.Run("记录半开状态", func(t *testing.T) {
		RecordCircuitBreakerState("openai", StateHalfOpen)

		value := testutil.ToFloat64(aiCircuitBreakerState.WithLabelValues("openai"))
		assert.Equal(t, float64(StateHalfOpen), value)
	})
}

// TestRecordRateLimitExceeded 测试记录限流事件
func TestRecordRateLimitExceeded(t *testing.T) {
	// 重置指标
	aiRateLimitExceeded.Reset()

	t.Run("记录限流事件", func(t *testing.T) {
		RecordRateLimitExceeded("openai")

		count := testutil.ToFloat64(aiRateLimitExceeded.WithLabelValues("openai"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("累加限流事件", func(t *testing.T) {
		RecordRateLimitExceeded("openai")
		RecordRateLimitExceeded("openai")

		count := testutil.ToFloat64(aiRateLimitExceeded.WithLabelValues("openai"))
		assert.Equal(t, float64(3), count)
	})

	t.Run("多Provider隔离", func(t *testing.T) {
		RecordRateLimitExceeded("claude")

		openaiCount := testutil.ToFloat64(aiRateLimitExceeded.WithLabelValues("openai"))
		claudeCount := testutil.ToFloat64(aiRateLimitExceeded.WithLabelValues("claude"))

		assert.Equal(t, float64(3), openaiCount)
		assert.Equal(t, float64(1), claudeCount)
	})
}

// TestProviderMetrics_ConcurrentAccess 测试并发访问安全
func TestProviderMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewProviderMetrics()

	done := make(chan bool)

	for i := 0; i < 50; i++ {
		go func(idx int) {
			if idx%2 == 0 {
				metrics.RecordCacheHit("provider")
			} else {
				metrics.RecordCacheMiss("provider")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	// 不应该panic，验证数据一致性
	assert.NotPanics(t, func() {
		metrics.RecordCacheHit("provider")
		metrics.RecordCacheMiss("provider")
	})

	// 验证数据
	hits := metrics.getCacheHits("provider")
	misses := metrics.getCacheMisses("provider")

	// 25 hits + 1 from verification
	assert.Equal(t, int64(26), hits)
	// 25 misses + 1 from verification
	assert.Equal(t, int64(26), misses)
}

// 辅助方法用于测试
func (m *ProviderMetrics) getCacheHits(provider string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cacheHits[provider]
}

func (m *ProviderMetrics) getCacheMisses(provider string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cacheMisses[provider]
}

// TestMetrics_Integration 测试指标集成
func TestMetrics_Integration(t *testing.T) {
	// 重置所有指标
	aiParseRequestsTotal.Reset()
	aiTokensUsed.Reset()
	aiCacheHitRatio.Reset()
	aiCircuitBreakerState.Reset()
	aiRateLimitExceeded.Reset()

	metrics := NewProviderMetrics()

	// 模拟一个完整的请求周期
	metrics.RecordRequest("openai", true, 0.3)
	metrics.RecordTokens("openai", 150)
	metrics.RecordCacheHit("openai")
	RecordCircuitBreakerState("openai", StateClosed)

	// 验证所有指标都被正确记录
	assert.Equal(t, float64(1), testutil.ToFloat64(aiParseRequestsTotal.WithLabelValues("openai", "success")))
	assert.Equal(t, float64(150), testutil.ToFloat64(aiTokensUsed.WithLabelValues("openai")))
	assert.Equal(t, float64(1), testutil.ToFloat64(aiCacheHitRatio.WithLabelValues("openai")))
	assert.Equal(t, float64(StateClosed), testutil.ToFloat64(aiCircuitBreakerState.WithLabelValues("openai")))
}
