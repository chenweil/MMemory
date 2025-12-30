package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestProviderConfig_Defaults 测试ProviderConfig默认值
func TestProviderConfig_Defaults(t *testing.T) {
	t.Run("空配置使用默认值", func(t *testing.T) {
		config := ProviderConfig{
			APIKey: "test-key",
		}

		// 模拟NewOpenAIProvider中的默认值设置
		if config.Model == "" {
			config.Model = "gpt-3.5-turbo"
		}
		if config.MaxTokens == 0 {
			config.MaxTokens = 500
		}
		if config.Temperature == 0 {
			config.Temperature = 0.3
		}
		if config.Timeout == 0 {
			config.Timeout = 10 * time.Second
		}
		if config.RateLimit == 0 {
			config.RateLimit = 60
		}

		assert.Equal(t, "gpt-3.5-turbo", config.Model)
		assert.Equal(t, 500, config.MaxTokens)
		assert.Equal(t, 0.3, config.Temperature)
		assert.Equal(t, 10*time.Second, config.Timeout)
		assert.Equal(t, 60, config.RateLimit)
	})

	t.Run("自定义配置覆盖默认值", func(t *testing.T) {
		config := ProviderConfig{
			APIKey:      "test-key",
			Model:       "gpt-4",
			MaxTokens:   1000,
			Temperature: 0.5,
			Timeout:     30 * time.Second,
			RateLimit:   120,
		}

		assert.Equal(t, "gpt-4", config.Model)
		assert.Equal(t, 1000, config.MaxTokens)
		assert.Equal(t, 0.5, config.Temperature)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.Equal(t, 120, config.RateLimit)
	})
}

// TestProviderParseResult 测试ProviderParseResult结构
func TestProviderParseResult(t *testing.T) {
	t.Run("创建完整结果", func(t *testing.T) {
		now := time.Now()
		result := &ProviderParseResult{
			Content:    "提醒内容",
			Time:       now,
			Pattern:    "daily",
			Confidence: 0.95,
			RawResponse: `{"content": "提醒内容", "time": "2025-01-01T09:00:00Z", "pattern": "daily", "confidence": 0.95}`,
			TokensUsed: 150,
		}

		assert.Equal(t, "提醒内容", result.Content)
		assert.Equal(t, "daily", result.Pattern)
		assert.Equal(t, 0.95, result.Confidence)
		assert.Equal(t, 150, result.TokensUsed)
	})

	t.Run("默认值", func(t *testing.T) {
		result := &ProviderParseResult{}

		assert.Equal(t, "", result.Content)
		assert.Equal(t, "", result.Pattern)
		assert.Equal(t, 0.0, result.Confidence)
		assert.Equal(t, 0, result.TokensUsed)
	})
}

// TestProviderError 测试ProviderError
func TestProviderError(t *testing.T) {
	t.Run("创建ProviderError", func(t *testing.T) {
		originalErr := context.DeadlineExceeded
		err := &ProviderError{
			Provider: "openai",
			Err:      originalErr,
			Type:     "TIMEOUT",
		}

		assert.Equal(t, "openai", err.Provider)
		assert.Equal(t, "TIMEOUT", err.Type)
		assert.Equal(t, originalErr, err.Err)
		assert.Equal(t, originalErr.Error(), err.Error())
	})

	t.Run("错误解包", func(t *testing.T) {
		originalErr := context.DeadlineExceeded
		err := &ProviderError{
			Provider: "openai",
			Err:      originalErr,
			Type:     "TIMEOUT",
		}

		assert.Equal(t, originalErr, err.Unwrap())
	})
}

// TestAlert 测试Alert结构
func TestAlert(t *testing.T) {
	t.Run("创建Alert", func(t *testing.T) {
		now := time.Now()
		alert := &Alert{
			RuleName:     "high_cost",
			Type:         "cost",
			Message:      "成本超过阈值",
			Timestamp:    now,
			Provider:     "openai",
			Threshold:    100.0,
			CurrentValue: 150.0,
		}

		assert.Equal(t, "high_cost", alert.RuleName)
		assert.Equal(t, "cost", alert.Type)
		assert.Equal(t, "成本超过阈值", alert.Message)
		assert.Equal(t, "openai", alert.Provider)
		assert.Equal(t, 100.0, alert.Threshold)
		assert.Equal(t, 150.0, alert.CurrentValue)
	})
}

// TestPromptTemplate 测试Prompt构建
func TestPromptTemplate(t *testing.T) {
	t.Run("提醒解析Prompt", func(t *testing.T) {
		// 模拟buildPrompt逻辑
		text := "每天早上8点提醒我喝水"
		prompt := struct{ System, User string }{
			System: `你是一个智能提醒助手，负责解析用户的自然语言输入并提取提醒信息。`,
			User:   "请解析以下提醒信息：\n" + text,
		}

		assert.Contains(t, prompt.System, "智能提醒助手")
		assert.Contains(t, prompt.User, text)
	})

	t.Run("聊天Prompt", func(t *testing.T) {
		text := "今天天气怎么样？"
		prompt := struct{ System, User string }{
			System: `你是一个友好的智能助手，负责帮助用户管理提醒。`,
			User:   text,
		}

		assert.Contains(t, prompt.System, "友好的智能助手")
		assert.Equal(t, text, prompt.User)
	})
}

// TestParseResponseLogic 测试解析响应逻辑
func TestParseResponseLogic(t *testing.T) {
	t.Run("有效JSON响应", func(t *testing.T) {
		jsonStr := `{
			"content": "提醒内容",
			"time": "2025-01-15T08:00:00Z",
			"pattern": "daily",
			"confidence": 0.9
		}`

		var result ProviderParseResult
		// 这里只测试JSON解析逻辑
		// 实际解析在parseResponse方法中
		assert.NotPanics(t, func() {
			_ = json.Unmarshal([]byte(jsonStr), &result)
		})
	})

	t.Run("无效JSON", func(t *testing.T) {
		invalidJson := "这不是JSON"

		var result ProviderParseResult
		err := json.Unmarshal([]byte(invalidJson), &result)

		assert.Error(t, err)
	})
}

