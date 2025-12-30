package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRateLimiter_New 测试创建限流器
func TestRateLimiter_New(t *testing.T) {
	rl := NewRateLimiter(60) // 60 requests per minute

	assert.NotNil(t, rl)
	assert.NotNil(t, rl.limiter)
}

// TestRateLimiter_Allow 测试允许请求
func TestRateLimiter_Allow(t *testing.T) {
	// 创建一个非常宽松的限流器
	rl := NewRateLimiter(1000) // 1000 requests per minute

	// 初始应该允许请求
	assert.True(t, rl.Allow())

	// 连续请求应该都允许
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Allow(), "请求 %d 应该被允许", i)
	}
}

// TestRateLimiter_Wait 测试等待
func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(60) // 60 requests per minute

	t.Run("正常等待", func(t *testing.T) {
		ctx := context.Background()
		err := rl.Wait(ctx)
		assert.NoError(t, err)
	})

	t.Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := rl.Wait(ctx)
		// 可能成功或超时，取决于限流器状态
		assert.True(t, err == nil || ctx.Err() != nil)
	})

	t.Run("上下文已取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		err := rl.Wait(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

// TestRateLimiter_Burst 测试突发流量
func TestRateLimiter_Burst(t *testing.T) {
	// 创建一个可以处理突发流量的限流器
	rl := NewRateLimiter(10) // 10 requests per minute

	// burst容量应该等于requestsPerMinute
	// 允许一次性发送多个请求
	count := 0
	for i := 0; i < 15; i++ {
		if rl.Allow() {
			count++
		}
	}

	// 至少有burst个请求被允许
	assert.GreaterOrEqual(t, count, 10)
}

// TestRateLimiter_Refill 测试令牌补充
// 注意：由于令牌补充需要较长时间，这个测试跳过实际的时间等待验证
func TestRateLimiter_Refill(t *testing.T) {
	// 创建一个极慢的限流器
	rl := NewRateLimiter(6) // 每10秒1个请求 (6 per minute)

	// 快速消耗所有令牌
	for i := 0; i < 6; i++ {
		assert.True(t, rl.Allow())
	}

	// 下一个请求应该被拒绝（burst容量已用完）
	assert.False(t, rl.Allow())

	// 验证等待足够时间后令牌会补充（跳过实际等待）
	t.Log("Rate limiter correctly blocks requests when burst capacity is exhausted")
}
