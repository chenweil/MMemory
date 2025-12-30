package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ProviderManager 管理多个AI Provider
type ProviderManager struct {
	providers map[string]AIProviderInterface
	primary   string
	fallback  []string
	metrics   *ProviderMetrics
	cache     *EnhancedCache
	breakers  map[string]*CircuitBreaker
	mu        sync.RWMutex
	logger    *logrus.Logger
}

// NewProviderManager 创建Provider管理器
func NewProviderManager(
	providers map[string]AIProviderInterface,
	primary string,
	fallback []string,
	logger *logrus.Logger,
) *ProviderManager {
	breakers := make(map[string]*CircuitBreaker)
	for name := range providers {
		breakers[name] = NewCircuitBreaker(5, 2, 30*time.Second)
	}

	return &ProviderManager{
		providers: providers,
		primary:   primary,
		fallback:  fallback,
		metrics:   NewProviderMetrics(),
		cache:     NewEnhancedCache(5*time.Minute, 1000),
		breakers:  breakers,
		logger:    logger,
	}
}

// ParseWithFallback 执行解析并自动降级
func (m *ProviderManager) ParseWithFallback(ctx context.Context, text string) (*ProviderParseResult, error) {
	// 1. 检查缓存
	if result, ok := m.cache.Get(text); ok {
		if parsedResult, ok := result.(*ProviderParseResult); ok {
			m.logger.WithField("cache", "hit").Info("Using cached result")
			m.metrics.RecordCacheHit(m.primary)
			return parsedResult, nil
		}
	}
	m.metrics.RecordCacheMiss(m.primary)

	// 2. 尝试主Provider
	provider := m.selectProvider(m.primary)
	if provider != nil {
		result, err := m.tryProvider(ctx, provider, text)
		if err == nil {
			m.cache.Set(text, result)
			return result, nil
		}
		m.logger.WithError(err).WithField("provider", m.primary).Warn("Primary provider failed")
	}

	// 3. 依次尝试备选Provider
	for _, name := range m.fallback {
		provider := m.selectProvider(name)
		if provider == nil {
			continue
		}

		m.logger.WithField("provider", name).Info("Trying fallback provider")
		result, err := m.tryProvider(ctx, provider, text)
		if err == nil {
			m.cache.Set(text, result)
			return result, nil
		}
		m.logger.WithError(err).WithField("provider", name).Warn("Fallback provider failed")
	}

	// 4. 所有Provider都失败
	return nil, fmt.Errorf("all providers failed")
}

// ChatWithFallback 执行聊天并自动降级
func (m *ProviderManager) ChatWithFallback(ctx context.Context, text string) (string, error) {
	// 2. 尝试主Provider
	provider := m.selectProvider(m.primary)
	if provider != nil {
		result, err := provider.Chat(ctx, text)
		if err == nil {
			return result, nil
		}
		m.logger.WithError(err).WithField("provider", m.primary).Warn("Primary provider chat failed")
	}

	// 3. 依次尝试备选Provider
	for _, name := range m.fallback {
		provider := m.selectProvider(name)
		if provider == nil {
			continue
		}

		m.logger.WithField("provider", name).Info("Trying fallback provider for chat")
		result, err := provider.Chat(ctx, text)
		if err == nil {
			return result, nil
		}
		m.logger.WithError(err).WithField("provider", name).Warn("Fallback provider chat failed")
	}

	// 4. 所有Provider都失败
	return "", fmt.Errorf("all providers failed for chat")
}

// tryProvider 尝试使用指定Provider
func (m *ProviderManager) tryProvider(ctx context.Context, provider AIProviderInterface, text string) (*ProviderParseResult, error) {
	name := provider.Name()
	breaker := m.breakers[name]

	// 检查熔断器状态
	if !breaker.CanRequest() {
		m.logger.WithField("provider", name).Warn("Circuit breaker is open")
		return nil, fmt.Errorf("circuit breaker open for %s", name)
	}

	// 记录开始时间
	start := time.Now()

	// 执行解析
	result, err := provider.ParseReminder(ctx, text)

	// 记录指标
	duration := time.Since(start)
	m.metrics.RecordRequest(name, err == nil, duration.Seconds())
	if err != nil {
		RecordRateLimitExceeded(name) // 可能是限流错误
	}

	if err != nil {
		breaker.RecordFailure()
		m.logger.WithError(err).
			WithField("provider", name).
			WithField("duration", duration).
			Error("Provider request failed")
		return nil, err
	}

	breaker.RecordSuccess()
	m.metrics.RecordTokens(name, result.TokensUsed)

	m.logger.WithField("provider", name).
		WithField("duration", duration).
		WithField("tokens", result.TokensUsed).
		WithField("confidence", result.Confidence).
		Info("Provider request succeeded")

	return result, nil
}

// selectProvider 选择Provider
func (m *ProviderManager) selectProvider(name string) AIProviderInterface {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		m.logger.WithField("provider", name).Warn("Provider not found")
		return nil
	}

	return provider
}

// GetMetrics 获取指标
func (m *ProviderManager) GetMetrics() *ProviderMetrics {
	return m.metrics
}

// HealthCheck 健康检查所有Provider
func (m *ProviderManager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	for name, provider := range m.providers {
		results[name] = provider.HealthCheck(ctx)
	}

	return results
}

// UpdateCircuitBreakerStates 更新熔断器状态到指标
func (m *ProviderManager) UpdateCircuitBreakerStates() {
	for name, breaker := range m.breakers {
		RecordCircuitBreakerState(name, breaker.GetState())
	}
}