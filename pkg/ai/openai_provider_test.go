package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewOpenAIProvider 测试创建OpenAI Provider
func TestNewOpenAIProvider(t *testing.T) {
	t.Run("创建成功 - 完整配置", func(t *testing.T) {
		config := ProviderConfig{
			Name:         "openai",
			APIKey:       "sk-test123",
			Model:        "gpt-4",
			MaxTokens:    2000,
			Temperature:  0.7,
			Timeout:      30 * time.Second,
			RateLimit:    100,
		}

		provider, err := NewOpenAIProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "openai", provider.Name())
		assert.Equal(t, "gpt-4", provider.config.Model)
		assert.Equal(t, 2000, provider.config.MaxTokens)
		assert.Equal(t, 0.7, provider.config.Temperature)
		assert.Equal(t, 30*time.Second, provider.config.Timeout)
		assert.Equal(t, 100, provider.config.RateLimit)
	})

	t.Run("创建成功 - 最小配置（使用默认值）", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "openai",
			APIKey: "sk-test123",
		}

		provider, err := NewOpenAIProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "openai", provider.Name())
		assert.Equal(t, "gpt-3.5-turbo", provider.config.Model) // 默认模型
		assert.Equal(t, 500, provider.config.MaxTokens)        // 默认最大token
		assert.Equal(t, 0.3, provider.config.Temperature)      // 默认温度
		assert.Equal(t, 10*time.Second, provider.config.Timeout) // 默认超时
		assert.Equal(t, 60, provider.config.RateLimit)         // 默认限流
	})

	t.Run("创建失败 - 缺少API Key", func(t *testing.T) {
		config := ProviderConfig{
			Name: "openai",
		}

		provider, err := NewOpenAIProvider(config)
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("创建失败 - 空API Key", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "openai",
			APIKey: "",
		}

		provider, err := NewOpenAIProvider(config)
		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}

// TestOpenAIProvider_Name 测试Name方法
func TestOpenAIProvider_Name(t *testing.T) {
	config := ProviderConfig{
		Name:   "openai",
		APIKey: "sk-test123",
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "openai", provider.Name())
}

// TestOpenAIProvider_GetConfig 测试GetConfig方法
func TestOpenAIProvider_GetConfig(t *testing.T) {
	config := ProviderConfig{
		Name:         "openai",
		APIKey:       "sk-test123",
		Model:        "gpt-4",
		MaxTokens:    2000,
		Temperature:  0.7,
		Timeout:      30 * time.Second,
		RateLimit:    100,
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	returnedConfig := provider.GetConfig()
	assert.Equal(t, config.Name, returnedConfig.Name)
	assert.Equal(t, config.APIKey, returnedConfig.APIKey)
	assert.Equal(t, config.Model, returnedConfig.Model)
	assert.Equal(t, config.MaxTokens, returnedConfig.MaxTokens)
	assert.Equal(t, config.Temperature, returnedConfig.Temperature)
	assert.Equal(t, config.Timeout, returnedConfig.Timeout)
	assert.Equal(t, config.RateLimit, returnedConfig.RateLimit)
}

// TestOpenAIProvider_buildPrompt 测试buildPrompt方法
func TestOpenAIProvider_buildPrompt(t *testing.T) {
	config := ProviderConfig{
		Name:   "openai",
		APIKey: "sk-test123",
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	t.Run("构建提醒解析Prompt", func(t *testing.T) {
		text := "明天早上9点提醒我开会"
		prompt := provider.buildPrompt(text)

		assert.NotEmpty(t, prompt.System)
		assert.Contains(t, prompt.System, "智能提醒助手")
		assert.Contains(t, prompt.System, "JSON格式")
		assert.Contains(t, prompt.System, "daily|weekly|monthly|once")

		assert.NotEmpty(t, prompt.User)
		assert.Contains(t, prompt.User, text)
		assert.Contains(t, prompt.User, "请解析以下提醒信息")
	})
}

// TestOpenAIProvider_buildChatPrompt 测试buildChatPrompt方法
func TestOpenAIProvider_buildChatPrompt(t *testing.T) {
	config := ProviderConfig{
		Name:   "openai",
		APIKey: "sk-test123",
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	t.Run("构建聊天Prompt", func(t *testing.T) {
		text := "你好，今天天气怎么样？"
		prompt := provider.buildChatPrompt(text)

		assert.NotEmpty(t, prompt.System)
		assert.Contains(t, prompt.System, "友好的智能助手")
		assert.Contains(t, prompt.System, "简洁、友好")

		assert.NotEmpty(t, prompt.User)
		assert.Equal(t, text, prompt.User)
	})
}

// TestOpenAIProvider_parseResponse 测试parseResponse方法
func TestOpenAIProvider_parseResponse(t *testing.T) {
	config := ProviderConfig{
		Name:   "openai",
		APIKey: "sk-test123",
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	t.Run("解析完整响应", func(t *testing.T) {
		jsonResponse := `{
			"content": "测试提醒",
			"time": "2025-10-01T09:00:00Z",
			"pattern": "daily",
			"confidence": 0.95
		}`

		result, err := provider.parseResponse(jsonResponse)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "测试提醒", result.Content)
		assert.Equal(t, "daily", result.Pattern)
		assert.Equal(t, 0.95, result.Confidence)
	})

	t.Run("解析缺少pattern的响应", func(t *testing.T) {
		jsonResponse := `{
			"content": "测试提醒",
			"time": "2025-10-01T09:00:00Z",
			"confidence": 0.9
		}`

		result, err := provider.parseResponse(jsonResponse)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "once", result.Pattern) // 应该使用默认值
	})

	t.Run("解析缺少confidence的响应", func(t *testing.T) {
		jsonResponse := `{
			"content": "测试提醒",
			"time": "2025-10-01T09:00:00Z",
			"pattern": "daily"
		}`

		result, err := provider.parseResponse(jsonResponse)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0.8, result.Confidence) // 应该使用默认值
	})

	t.Run("解析无效JSON", func(t *testing.T) {
		invalidJSON := `{"invalid": json}`

		result, err := provider.parseResponse(invalidJSON)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to parse response")
	})

	t.Run("解析缺少content的响应", func(t *testing.T) {
		jsonResponse := `{
			"time": "2025-10-01T09:00:00Z",
			"pattern": "daily",
			"confidence": 0.95
		}`

		result, err := provider.parseResponse(jsonResponse)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "missing content field")
	})

	t.Run("解析缺少time的响应", func(t *testing.T) {
		jsonResponse := `{
			"content": "测试提醒",
			"pattern": "daily",
			"confidence": 0.95
		}`

		result, err := provider.parseResponse(jsonResponse)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "missing or invalid time field")
	})

	t.Run("解析无效时间的响应", func(t *testing.T) {
		jsonResponse := `{
			"content": "测试提醒",
			"time": "invalid-time",
			"pattern": "daily",
			"confidence": 0.95
		}`

		result, err := provider.parseResponse(jsonResponse)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestOpenAIProvider_handleError 测试handleError方法
func TestOpenAIProvider_handleError(t *testing.T) {
	config := ProviderConfig{
		Name:   "openai",
		APIKey: "sk-test123",
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	t.Run("处理超时错误", func(t *testing.T) {
		timeoutErr := errors.New("context deadline exceeded")
		providerErr := provider.handleError(timeoutErr)

		assert.Error(t, providerErr)
		providerError, ok := providerErr.(*ProviderError)
		assert.True(t, ok)
		assert.Equal(t, "openai", providerError.Provider)
		assert.Equal(t, "TIMEOUT", providerError.Type)
	})

	t.Run("处理API错误", func(t *testing.T) {
		apiErr := errors.New("API request failed")
		providerErr := provider.handleError(apiErr)

		assert.Error(t, providerErr)
		providerError, ok := providerErr.(*ProviderError)
		assert.True(t, ok)
		assert.Equal(t, "openai", providerError.Provider)
		assert.Equal(t, "API_ERROR", providerError.Type)
	})

	t.Run("处理通用错误", func(t *testing.T) {
		genericErr := errors.New("some error")
		providerErr := provider.handleError(genericErr)

		assert.Error(t, providerErr)
		providerError, ok := providerErr.(*ProviderError)
		assert.True(t, ok)
		assert.Equal(t, "openai", providerError.Provider)
		assert.Equal(t, "API_ERROR", providerError.Type)
	})
}

// TestOpenAIProvider_HealthCheck 测试HealthCheck方法
func TestOpenAIProvider_HealthCheck(t *testing.T) {
	config := ProviderConfig{
		Name:   "openai",
		APIKey: "sk-test123",
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	t.Run("健康检查", func(t *testing.T) {
		ctx := context.Background()
		err := provider.HealthCheck(ctx)

		// 由于没有真实的API key，这里会失败，但至少应该返回错误而不是panic
		assert.Error(t, err)
	})

	t.Run("带超时的健康检查", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
	})
}

// TestOpenAIProvider_RateLimiter 测试限流器
func TestOpenAIProvider_RateLimiter(t *testing.T) {
	t.Run("创建带限流的Provider", func(t *testing.T) {
		config := ProviderConfig{
			Name:      "openai",
			APIKey:    "sk-test123",
			RateLimit: 10, // 10 req/min
		}

		provider, err := NewOpenAIProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.limiter)
	})

	t.Run("使用默认限流", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "openai",
			APIKey: "sk-test123",
		}

		provider, err := NewOpenAIProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.limiter)
	})
}

// TestOpenAIProvider_ParseReminder_ErrorCases 测试ParseReminder错误情况
func TestOpenAIProvider_ParseReminder_ErrorCases(t *testing.T) {
	config := ProviderConfig{
		Name:    "openai",
		APIKey:  "sk-test123",
		Timeout: 1 * time.Second, // 短超时
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	t.Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		_, err := provider.ParseReminder(ctx, "test")
		assert.Error(t, err)
	})

	t.Run("超时", func(t *testing.T) {
		ctx := context.Background()
		_, err := provider.ParseReminder(ctx, "test")
		// 由于没有真实的API key，会超时或返回API错误
		assert.Error(t, err)
	})
}

// TestOpenAIProvider_Chat_ErrorCases 测试Chat错误情况
func TestOpenAIProvider_Chat_ErrorCases(t *testing.T) {
	config := ProviderConfig{
		Name:    "openai",
		APIKey:  "sk-test123",
		Timeout: 1 * time.Second,
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	t.Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := provider.Chat(ctx, "hello")
		assert.Error(t, err)
	})

	t.Run("超时", func(t *testing.T) {
		ctx := context.Background()
		_, err := provider.Chat(ctx, "hello")
		assert.Error(t, err)
	})
}