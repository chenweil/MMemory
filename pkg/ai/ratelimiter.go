package ai

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiter 限流器 (Token Bucket算法)
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter 创建限流器
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	// 转换为每秒速率
	rps := float64(requestsPerMinute) / 60.0

	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), requestsPerMinute),
	}
}

// Wait 等待令牌可用
func (r *RateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow() bool {
	return r.limiter.Allow()
}