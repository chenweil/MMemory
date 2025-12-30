package ai

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	aiParseRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_parse_requests_total",
			Help: "Total number of AI parse requests",
		},
		[]string{"provider", "status"},
	)

	aiParseDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_parse_duration_seconds",
			Help:    "Duration of AI parse requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider"},
	)

	aiTokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_parse_tokens_used",
			Help: "Total tokens used by AI providers",
		},
		[]string{"provider"},
	)

	aiCacheHitRatio = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_cache_hit_ratio",
			Help: "Cache hit ratio for AI parsing",
		},
		[]string{"provider"},
	)

	aiCircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"provider"},
	)

	aiRateLimitExceeded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_rate_limit_exceeded_total",
			Help: "Total number of rate limit exceeded events",
		},
		[]string{"provider"},
	)
)

// ProviderMetrics Provider指标
type ProviderMetrics struct {
	cacheHits   map[string]int64
	cacheMisses map[string]int64
	mu          sync.RWMutex
}

// NewProviderMetrics 创建指标
func NewProviderMetrics() *ProviderMetrics {
	return &ProviderMetrics{
		cacheHits:   make(map[string]int64),
		cacheMisses: make(map[string]int64),
	}
}

// RecordRequest 记录请求
func (m *ProviderMetrics) RecordRequest(provider string, success bool, duration float64) {
	status := "success"
	if !success {
		status = "failure"
	}

	aiParseRequestsTotal.WithLabelValues(provider, status).Inc()
	aiParseDuration.WithLabelValues(provider).Observe(duration)
}

// RecordTokens 记录Token使用
func (m *ProviderMetrics) RecordTokens(provider string, tokens int) {
	aiTokensUsed.WithLabelValues(provider).Add(float64(tokens))
}

// RecordCacheHit 记录缓存命中
func (m *ProviderMetrics) RecordCacheHit(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheHits[provider]++
	m.updateCacheHitRatio(provider)
}

// RecordCacheMiss 记录缓存未命中
func (m *ProviderMetrics) RecordCacheMiss(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheMisses[provider]++
	m.updateCacheHitRatio(provider)
}

// updateCacheHitRatio 更新缓存命中率
func (m *ProviderMetrics) updateCacheHitRatio(provider string) {
	hits := m.cacheHits[provider]
	misses := m.cacheMisses[provider]
	total := hits + misses

	if total > 0 {
		ratio := float64(hits) / float64(total)
		aiCacheHitRatio.WithLabelValues(provider).Set(ratio)
	}
}

// RecordCircuitBreakerState 记录熔断器状态
func RecordCircuitBreakerState(provider string, state CircuitBreakerState) {
	aiCircuitBreakerState.WithLabelValues(provider).Set(float64(state))
}

// RecordRateLimitExceeded 记录限流事件
func RecordRateLimitExceeded(provider string) {
	aiRateLimitExceeded.WithLabelValues(provider).Inc()
}