package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAIProvider Mock AI Provider实现
type MockAIProvider struct {
	mock.Mock
}

func (m *MockAIProvider) ParseReminder(ctx context.Context, text string) (*ProviderParseResult, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProviderParseResult), args.Error(1)
}

func (m *MockAIProvider) Chat(ctx context.Context, text string) (string, error) {
	args := m.Called(ctx, text)
	return args.String(0), args.Error(1)
}

func (m *MockAIProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAIProvider) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAIProvider) GetConfig() ProviderConfig {
	args := m.Called()
	return args.Get(0).(ProviderConfig)
}

// TestProviderManager_New 测试创建Provider管理器
func TestProviderManager_New(t *testing.T) {
	providers := make(map[string]AIProviderInterface)
	mockProvider := &MockAIProvider{}
	providers["openai"] = mockProvider

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.providers)
	assert.NotNil(t, manager.metrics)
	assert.NotNil(t, manager.cache)
	assert.NotNil(t, manager.breakers)
	assert.Equal(t, "openai", manager.primary)
}

// TestProviderManager_ParseWithFallback_CacheHit 测试缓存命中
func TestProviderManager_ParseWithFallback_CacheHit(t *testing.T) {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("openai")

	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	// 先设置缓存
	cachedResult := &ProviderParseResult{
		Content:    "测试提醒",
		Time:       time.Now(),
		Pattern:    "daily",
		Confidence: 0.9,
		TokensUsed: 100,
	}
	manager.cache.Set("test message", cachedResult)

	// 从缓存获取
	ctx := context.Background()
	result, err := manager.ParseWithFallback(ctx, "test message")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "测试提醒", result.Content)
}

// TestProviderManager_ParseWithFallback_Success 测试成功解析
func TestProviderManager_ParseWithFallback_Success(t *testing.T) {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("openai")
	mockProvider.On("ParseReminder", mock.Anything, "创建提醒").Return(&ProviderParseResult{
		Content:    "创建提醒",
		Time:       time.Now(),
		Pattern:    "daily",
		Confidence: 0.9,
		TokensUsed: 150,
	}, nil)

	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	ctx := context.Background()
	result, err := manager.ParseWithFallback(ctx, "创建提醒")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0.9, result.Confidence)
	mockProvider.AssertExpectations(t)
}

// TestProviderManager_ParseWithFallback_Fallback 测试备选Provider
func TestProviderManager_ParseWithFallback_Fallback(t *testing.T) {
	mockPrimary := &MockAIProvider{}
	mockPrimary.On("Name").Return("openai")
	mockPrimary.On("ParseReminder", mock.Anything, "测试").Return(nil, errors.New("primary failed"))

	mockFallback := &MockAIProvider{}
	mockFallback.On("Name").Return("claude")
	mockFallback.On("ParseReminder", mock.Anything, "测试").Return(&ProviderParseResult{
		Content:    "fallback结果",
		Confidence: 0.8,
		TokensUsed: 100,
	}, nil)

	providers := map[string]AIProviderInterface{
		"openai": mockPrimary,
		"claude": mockFallback,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{"claude"}, logger)

	ctx := context.Background()
	result, err := manager.ParseWithFallback(ctx, "测试")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "fallback结果", result.Content)
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertExpectations(t)
}

// TestProviderManager_ParseWithFallback_AllFailed 测试所有Provider都失败
func TestProviderManager_ParseWithFallback_AllFailed(t *testing.T) {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("openai")
	mockProvider.On("ParseReminder", mock.Anything, "测试").Return(nil, errors.New("all failed"))

	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	ctx := context.Background()
	result, err := manager.ParseWithFallback(ctx, "测试")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "all providers failed")
}

// TestProviderManager_ChatWithFallback_Success 测试聊天成功
func TestProviderManager_ChatWithFallback_Success(t *testing.T) {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("openai")
	mockProvider.On("Chat", mock.Anything, "你好").Return("你好！有什么可以帮助你的？", nil)

	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	ctx := context.Background()
	result, err := manager.ChatWithFallback(ctx, "你好")

	assert.NoError(t, err)
	assert.Equal(t, "你好！有什么可以帮助你的？", result)
}

// TestProviderManager_ChatWithFallback_Fallback 测试聊天降级
func TestProviderManager_ChatWithFallback_Fallback(t *testing.T) {
	mockPrimary := &MockAIProvider{}
	mockPrimary.On("Name").Return("openai")
	mockPrimary.On("Chat", mock.Anything, "测试").Return("", errors.New("primary failed"))

	mockFallback := &MockAIProvider{}
	mockFallback.On("Name").Return("claude")
	mockFallback.On("Chat", mock.Anything, "测试").Return("fallback回复", nil)

	providers := map[string]AIProviderInterface{
		"openai": mockPrimary,
		"claude": mockFallback,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{"claude"}, logger)

	ctx := context.Background()
	result, err := manager.ChatWithFallback(ctx, "测试")

	assert.NoError(t, err)
	assert.Equal(t, "fallback回复", result)
}

// TestProviderManager_GetMetrics 测试获取指标
func TestProviderManager_GetMetrics(t *testing.T) {
	mockProvider := &MockAIProvider{}
	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	metrics := manager.GetMetrics()
	assert.NotNil(t, metrics)
	assert.IsType(t, &ProviderMetrics{}, metrics)
}

// TestProviderManager_HealthCheck 测试健康检查
func TestProviderManager_HealthCheck(t *testing.T) {
	mockProvider1 := &MockAIProvider{}
	mockProvider1.On("Name").Return("openai")
	mockProvider1.On("HealthCheck", mock.Anything).Return(nil)

	mockProvider2 := &MockAIProvider{}
	mockProvider2.On("Name").Return("claude")
	mockProvider2.On("HealthCheck", mock.Anything).Return(errors.New("unhealthy"))

	providers := map[string]AIProviderInterface{
		"openai": mockProvider1,
		"claude": mockProvider2,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{"claude"}, logger)

	ctx := context.Background()
	results := manager.HealthCheck(ctx)

	assert.NoError(t, results["openai"])
	assert.Error(t, results["claude"])
}

// TestProviderManager_SelectProvider 测试选择Provider
func TestProviderManager_SelectProvider(t *testing.T) {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("openai")

	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	// 选择存在的Provider
	provider := manager.selectProvider("openai")
	assert.NotNil(t, provider)

	// 选择不存在的Provider
	provider = manager.selectProvider("nonexistent")
	assert.Nil(t, provider)
}

// TestProviderManager_UpdateCircuitBreakerStates 测试更新熔断器状态
func TestProviderManager_UpdateCircuitBreakerStates(t *testing.T) {
	mockProvider := &MockAIProvider{}
	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	// 初始状态应该是关闭的
	assert.Equal(t, StateClosed, manager.breakers["openai"].GetState())

	// 更新状态（这个测试主要是验证不会panic）
	assert.NotPanics(t, func() {
		manager.UpdateCircuitBreakerStates()
	})
}

// TestProviderManager_TryProvider_CircuitBreakerOpen 测试熔断器开启时
func TestProviderManager_TryProvider_CircuitBreakerOpen(t *testing.T) {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("openai")

	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	// 打开熔断器
	for i := 0; i < 5; i++ {
		manager.breakers["openai"].RecordFailure()
	}
	assert.Equal(t, StateOpen, manager.breakers["openai"].GetState())

	// 尝试调用（应该失败因为熔断器开启）
	ctx := context.Background()
	result, err := manager.tryProvider(ctx, mockProvider, "测试")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "circuit breaker open")
}

// TestProviderManager_ConcurrentAccess 测试并发访问安全
func TestProviderManager_ConcurrentAccess(t *testing.T) {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("openai")
	mockProvider.On("ParseReminder", mock.Anything, mock.Anything).Return(&ProviderParseResult{
		Content:    "结果",
		Confidence: 0.9,
		TokensUsed: 100,
	}, nil)

	providers := map[string]AIProviderInterface{
		"openai": mockProvider,
	}

	logger := logrus.New()
	manager := NewProviderManager(providers, "openai", []string{}, logger)

	ctx := context.Background()
	done := make(chan bool)

	for i := 0; i < 50; i++ {
		go func(idx int) {
			_, _ = manager.ParseWithFallback(ctx, "测试消息")
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	// 不应该panic
	assert.NotPanics(t, func() {
		manager.GetMetrics()
		manager.UpdateCircuitBreakerStates()
	})
}
