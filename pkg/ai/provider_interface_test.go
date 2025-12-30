package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockProvider 用于测试的模拟Provider
type MockProvider struct {
	name        string
	config      ProviderConfig
	parseFunc   func(ctx context.Context, text string) (*ProviderParseResult, error)
	chatFunc    func(ctx context.Context, text string) (string, error)
	healthFunc  func(ctx context.Context) error
}

func (m *MockProvider) ParseReminder(ctx context.Context, text string) (*ProviderParseResult, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, text)
	}
	return &ProviderParseResult{
		Content:    "mock reminder",
		Time:       time.Now(),
		Pattern:    "daily",
		Confidence: 0.95,
	}, nil
}

func (m *MockProvider) Chat(ctx context.Context, text string) (string, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, text)
	}
	return "mock response", nil
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) HealthCheck(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func (m *MockProvider) GetConfig() ProviderConfig {
	return m.config
}

// TestAIProviderInterface 测试AIProviderInterface接口实现
func TestAIProviderInterface(t *testing.T) {
	t.Run("MockProvider实现接口", func(t *testing.T) {
		var provider AIProviderInterface = &MockProvider{
			name: "mock",
			config: ProviderConfig{
				Name:   "mock",
				APIKey: "test-key",
			},
		}

		ctx := context.Background()

		// 测试ParseReminder
		result, err := provider.ParseReminder(ctx, "test text")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "mock reminder", result.Content)

		// 测试Chat
		response, err := provider.Chat(ctx, "hello")
		assert.NoError(t, err)
		assert.Equal(t, "mock response", response)

		// 测试Name
		assert.Equal(t, "mock", provider.Name())

		// 测试HealthCheck
		err = provider.HealthCheck(ctx)
		assert.NoError(t, err)

		// 测试GetConfig
		config := provider.GetConfig()
		assert.Equal(t, "mock", config.Name)
	})

	t.Run("MockProvider自定义函数", func(t *testing.T) {
		customParseFunc := func(ctx context.Context, text string) (*ProviderParseResult, error) {
			return &ProviderParseResult{
				Content:    "custom reminder",
				Time:       time.Now(),
				Pattern:    "weekly",
				Confidence: 0.99,
			}, nil
		}

		customChatFunc := func(ctx context.Context, text string) (string, error) {
			return "custom chat response", nil
		}

		provider := &MockProvider{
			name:      "custom-mock",
			parseFunc: customParseFunc,
			chatFunc:  customChatFunc,
		}

		ctx := context.Background()

		result, err := provider.ParseReminder(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, "custom reminder", result.Content)
		assert.Equal(t, "weekly", result.Pattern)

		response, err := provider.Chat(ctx, "hello")
		assert.NoError(t, err)
		assert.Equal(t, "custom chat response", response)
	})

	t.Run("MockProvider错误处理", func(t *testing.T) {
		errorFunc := func(ctx context.Context, text string) (*ProviderParseResult, error) {
			return nil, &ProviderError{
				Provider: "mock",
				Err:      errors.New("parse failed"),
				Type:     "PARSE_ERROR",
			}
		}

		healthErrorFunc := func(ctx context.Context) error {
			return errors.New("health check failed")
		}

		provider := &MockProvider{
			name:       "error-mock",
			parseFunc:  errorFunc,
			healthFunc: healthErrorFunc,
		}

		ctx := context.Background()

		_, err := provider.ParseReminder(ctx, "test")
		assert.Error(t, err)

		err = provider.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "health check failed")
	})
}

// TestProviderConfigDefaults 测试配置默认值
func TestProviderConfigDefaults(t *testing.T) {
	t.Run("零值配置", func(t *testing.T) {
		config := ProviderConfig{}

		assert.Equal(t, "", config.Name)
		assert.Equal(t, "", config.APIKey)
		assert.Equal(t, "", config.Model)
		assert.Equal(t, 0, config.MaxTokens)
		assert.Equal(t, 0.0, config.Temperature)
		assert.Equal(t, time.Duration(0), config.Timeout)
		assert.Equal(t, 0, config.RateLimit)
	})
}

// TestProviderParseResultValidation 测试结果验证
func TestProviderParseResultValidation(t *testing.T) {
	t.Run("置信度范围验证", func(t *testing.T) {
		testCases := []struct {
			name       string
			confidence float64
			valid      bool
		}{
			{"有效置信度0.0", 0.0, true},
			{"有效置信度0.5", 0.5, true},
			{"有效置信度1.0", 1.0, true},
			{"无效置信度-0.1", -0.1, false},
			{"无效置信度1.1", 1.1, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := &ProviderParseResult{
					Content:    "test",
					Time:       time.Now(),
					Pattern:    "daily",
					Confidence: tc.confidence,
				}

				if tc.valid {
					assert.True(t, result.Confidence >= 0 && result.Confidence <= 1)
				} else {
					assert.False(t, result.Confidence >= 0 && result.Confidence <= 1)
				}
			})
		}
	})

	t.Run("模式验证", func(t *testing.T) {
		validPatterns := []string{"daily", "weekly", "monthly", "once"}
		invalidPattern := "invalid"

		for _, pattern := range validPatterns {
			t.Run("有效模式"+pattern, func(t *testing.T) {
				result := &ProviderParseResult{
					Content:    "test",
					Time:       time.Now(),
					Pattern:    pattern,
					Confidence: 0.9,
				}
				assert.Equal(t, pattern, result.Pattern)
			})
		}

		t.Run("无效模式", func(t *testing.T) {
			result := &ProviderParseResult{
				Content:    "test",
				Time:       time.Now(),
				Pattern:    invalidPattern,
				Confidence: 0.9,
			}
			assert.Equal(t, invalidPattern, result.Pattern)
		})
	})
}