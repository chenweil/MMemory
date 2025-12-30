package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClaudeProvider 测试创建Claude Provider
func TestNewClaudeProvider(t *testing.T) {
	t.Run("创建成功 - 完整配置", func(t *testing.T) {
		config := ProviderConfig{
			Name:         "claude",
			APIKey:       "sk-ant-test123",
			Model:        "claude-3-opus-20240229",
			MaxTokens:    2000,
			Temperature:  0.7,
			Timeout:      30 * time.Second,
			RateLimit:    50,
		}

		provider, err := NewClaudeProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "claude", provider.Name())
		assert.Equal(t, "claude-3-opus-20240229", provider.config.Model)
		assert.Equal(t, 2000, provider.config.MaxTokens)
		assert.Equal(t, 0.7, provider.config.Temperature)
		assert.Equal(t, 30*time.Second, provider.config.Timeout)
		assert.Equal(t, 50, provider.config.RateLimit)
	})

	t.Run("创建成功 - 最小配置（使用默认值）", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
		}

		provider, err := NewClaudeProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "claude", provider.Name())
		assert.Equal(t, "claude-3-haiku-20240307", provider.config.Model) // 默认模型
		assert.Equal(t, 500, provider.config.MaxTokens)                // 默认最大token
		assert.Equal(t, 0.3, provider.config.Temperature)              // 默认温度
		assert.Equal(t, 10*time.Second, provider.config.Timeout)       // 默认超时
		assert.Equal(t, 50, provider.config.RateLimit)                 // 默认限流
	})

	t.Run("创建失败 - 缺少API Key", func(t *testing.T) {
		config := ProviderConfig{
			Name: "claude",
		}

		provider, err := NewClaudeProvider(config)
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("创建失败 - 空API Key", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "",
		}

		provider, err := NewClaudeProvider(config)
		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}

// TestClaudeProvider_Name 测试Name方法
func TestClaudeProvider_Name(t *testing.T) {
	config := ProviderConfig{
		Name:   "claude",
		APIKey: "sk-ant-test123",
	}

	provider, err := NewClaudeProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "claude", provider.Name())
}

// TestClaudeProvider_GetConfig 测试GetConfig方法
func TestClaudeProvider_GetConfig(t *testing.T) {
	config := ProviderConfig{
		Name:         "claude",
		APIKey:       "sk-ant-test123",
		Model:        "claude-3-opus-20240229",
		MaxTokens:    2000,
		Temperature:  0.7,
		Timeout:      30 * time.Second,
		RateLimit:    50,
	}

	provider, err := NewClaudeProvider(config)
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

// TestClaudeProvider_ParseReminder 测试ParseReminder方法
func TestClaudeProvider_ParseReminder(t *testing.T) {
	config := ProviderConfig{
		Name:   "claude",
		APIKey: "sk-ant-test123",
	}

	provider, err := NewClaudeProvider(config)
	require.NoError(t, err)

	t.Run("ParseReminder - 未实现", func(t *testing.T) {
		ctx := context.Background()
		result, err := provider.ParseReminder(ctx, "明天提醒我开会")

		assert.Error(t, err)
		assert.Nil(t, result)

		providerError, ok := err.(*ProviderError)
		assert.True(t, ok)
		assert.Equal(t, "claude", providerError.Provider)
		assert.Equal(t, "NOT_IMPLEMENTED", providerError.Type)
		assert.Contains(t, providerError.Err.Error(), "not fully implemented")
	})

	t.Run("ParseReminder - 上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := provider.ParseReminder(ctx, "test")
		assert.Error(t, err)

		providerError, ok := err.(*ProviderError)
		assert.True(t, ok)
		assert.Equal(t, "claude", providerError.Provider)
	})
}

// TestClaudeProvider_Chat 测试Chat方法
func TestClaudeProvider_Chat(t *testing.T) {
	config := ProviderConfig{
		Name:   "claude",
		APIKey: "sk-ant-test123",
	}

	provider, err := NewClaudeProvider(config)
	require.NoError(t, err)

	t.Run("Chat - 未实现", func(t *testing.T) {
		ctx := context.Background()
		response, err := provider.Chat(ctx, "你好")

		assert.Error(t, err)
		assert.Empty(t, response)
		assert.Contains(t, err.Error(), "not fully implemented")
	})

	t.Run("Chat - 上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := provider.Chat(ctx, "hello")
		assert.Error(t, err)
	})
}

// TestClaudeProvider_HealthCheck 测试HealthCheck方法
func TestClaudeProvider_HealthCheck(t *testing.T) {
	config := ProviderConfig{
		Name:   "claude",
		APIKey: "sk-ant-test123",
	}

	provider, err := NewClaudeProvider(config)
	require.NoError(t, err)

	t.Run("HealthCheck - 未实现", func(t *testing.T) {
		ctx := context.Background()
		err := provider.HealthCheck(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not fully implemented")
	})

	t.Run("HealthCheck - 上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
	})
}

// TestClaudeProvider_RateLimiter 测试限流器
func TestClaudeProvider_RateLimiter(t *testing.T) {
	t.Run("创建带限流的Provider", func(t *testing.T) {
		config := ProviderConfig{
			Name:      "claude",
			APIKey:    "sk-ant-test123",
			RateLimit: 20, // 20 req/min
		}

		provider, err := NewClaudeProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.limiter)
	})

	t.Run("使用默认限流", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
		}

		provider, err := NewClaudeProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.limiter)
	})
}

// TestClaudeProvider_ConfigDefaults 测试配置默认值
func TestClaudeProvider_ConfigDefaults(t *testing.T) {
	t.Run("验证模型默认值", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
		}

		provider, err := NewClaudeProvider(config)
		require.NoError(t, err)

		assert.Equal(t, "claude-3-haiku-20240307", provider.config.Model)
	})

	t.Run("验证MaxTokens默认值", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
		}

		provider, err := NewClaudeProvider(config)
		require.NoError(t, err)

		assert.Equal(t, 500, provider.config.MaxTokens)
	})

	t.Run("验证Temperature默认值", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
		}

		provider, err := NewClaudeProvider(config)
		require.NoError(t, err)

		assert.Equal(t, 0.3, provider.config.Temperature)
	})

	t.Run("验证Timeout默认值", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
		}

		provider, err := NewClaudeProvider(config)
		require.NoError(t, err)

		assert.Equal(t, 10*time.Second, provider.config.Timeout)
	})

	t.Run("验证RateLimit默认值", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
		}

		provider, err := NewClaudeProvider(config)
		require.NoError(t, err)

		assert.Equal(t, 50, provider.config.RateLimit)
	})
}

// TestClaudeProvider_ErrorHandling 测试错误处理
func TestClaudeProvider_ErrorHandling(t *testing.T) {
	config := ProviderConfig{
		Name:   "claude",
		APIKey: "sk-ant-test123",
	}

	provider, err := NewClaudeProvider(config)
	require.NoError(t, err)

	t.Run("ParseReminder返回ProviderError", func(t *testing.T) {
		ctx := context.Background()
		_, err := provider.ParseReminder(ctx, "test")

		assert.Error(t, err)
		providerError, ok := err.(*ProviderError)
		assert.True(t, ok, "错误应该是ProviderError类型")
		assert.Equal(t, "claude", providerError.Provider)
		assert.Equal(t, "NOT_IMPLEMENTED", providerError.Type)
		assert.NotNil(t, providerError.Err)
	})

	t.Run("Chat返回错误", func(t *testing.T) {
		ctx := context.Background()
		_, err := provider.Chat(ctx, "test")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not fully implemented")
	})

	t.Run("HealthCheck返回错误", func(t *testing.T) {
		ctx := context.Background()
		err := provider.HealthCheck(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not fully implemented")
	})
}

// TestClaudeProvider_InterfaceCompliance 测试接口实现
func TestClaudeProvider_InterfaceCompliance(t *testing.T) {
	config := ProviderConfig{
		Name:   "claude",
		APIKey: "sk-ant-test123",
	}

	provider, err := NewClaudeProvider(config)
	require.NoError(t, err)

	t.Run("实现AIProviderInterface接口", func(t *testing.T) {
		var _ AIProviderInterface = provider // 编译时检查接口实现
		assert.NotNil(t, provider)
	})
}

// TestClaudeProvider_MultipleInstances 测试多个实例
func TestClaudeProvider_MultipleInstances(t *testing.T) {
	t.Run("创建多个独立实例", func(t *testing.T) {
		config1 := ProviderConfig{
			Name:    "claude",
			APIKey:  "sk-ant-test1",
			Model:   "claude-3-opus-20240229",
			Timeout: 20 * time.Second,
		}

		config2 := ProviderConfig{
			Name:    "claude",
			APIKey:  "sk-ant-test2",
			Model:   "claude-3-haiku-20240307",
			Timeout: 30 * time.Second,
		}

		provider1, err := NewClaudeProvider(config1)
		assert.NoError(t, err)
		assert.NotNil(t, provider1)

		provider2, err := NewClaudeProvider(config2)
		assert.NoError(t, err)
		assert.NotNil(t, provider2)

		// 验证两个实例的配置不同
		assert.NotEqual(t, provider1.config.APIKey, provider2.config.APIKey)
		assert.NotEqual(t, provider1.config.Model, provider2.config.Model)
		assert.NotEqual(t, provider1.config.Timeout, provider2.config.Timeout)
	})
}

// TestClaudeProvider_ZeroValueConfig 测试零值配置
func TestClaudeProvider_ZeroValueConfig(t *testing.T) {
	t.Run("部分字段为零值", func(t *testing.T) {
		config := ProviderConfig{
			Name:   "claude",
			APIKey: "sk-ant-test123",
			// 其他字段为零值
		}

		provider, err := NewClaudeProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)

		// 验证默认值被设置
		assert.NotZero(t, provider.config.Model)
		assert.NotZero(t, provider.config.MaxTokens)
		assert.NotZero(t, provider.config.Temperature)
		assert.NotZero(t, provider.config.Timeout)
		assert.NotZero(t, provider.config.RateLimit)
	})
}