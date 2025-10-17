package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/pkg/ai"
)

// TestNewOpenAIClient 测试创建OpenAI客户端
func TestNewOpenAIClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *ai.AIConfig
		expectNil bool
	}{
		{
			name: "有效配置",
			config: &ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey:       "sk-test-key",
					BaseURL:      "https://api.openai.com/v1",
					PrimaryModel: "gpt-4o-mini",
					BackupModel:  "gpt-4o-mini",
					Temperature:  0.1,
					MaxTokens:    1000,
					MaxRetries:   3,
				},
			},
			expectNil: false,
		},
		{
			name: "AI未启用",
			config: &ai.AIConfig{
				Enabled: false,
				OpenAI: ai.OpenAIConfig{
					APIKey: "sk-test-key",
				},
			},
			expectNil: true,
		},
		{
			name: "缺少API Key",
			config: &ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey: "",
				},
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOpenAIClient(tt.config)

			if tt.expectNil {
				assert.Nil(t, client)
			} else {
				require.NotNil(t, client)
				assert.NotNil(t, client.client)
				assert.NotNil(t, client.rateLimiter)
				assert.Equal(t, tt.config, client.config)
			}
		})
	}
}

// TestOpenAIClient_GetName 测试获取名称
func TestOpenAIClient_GetName(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-test",
			PrimaryModel: "gpt-4o-mini",
		},
	}
	client := NewOpenAIClient(config)

	require.NotNil(t, client)
	name := client.GetName()
	assert.Contains(t, name, "openai")
	assert.Contains(t, name, "gpt-4o-mini")
}

// TestOpenAIClient_GetPriority 测试获取优先级
func TestOpenAIClient_GetPriority(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-test",
			PrimaryModel: "gpt-4o-mini",
		},
	}
	client := NewOpenAIClient(config)

	require.NotNil(t, client)
	priority := client.GetPriority()
	assert.Equal(t, 1, priority, "OpenAI客户端应该是最高优先级")
}

// TestOpenAIClient_IsHealthy 测试健康检查
func TestOpenAIClient_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		client   *OpenAIClient
		expected bool
	}{
		{
			name: "健康的客户端",
			client: NewOpenAIClient(&ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey:       "sk-test",
					PrimaryModel: "gpt-4o-mini",
				},
			}),
			expected: true,
		},
		{
			name:     "nil客户端",
			client:   nil,
			expected: false,
		},
		{
			name: "AI未启用",
			client: &OpenAIClient{
				config: &ai.AIConfig{
					Enabled: false,
					OpenAI: ai.OpenAIConfig{
						APIKey: "sk-test",
					},
				},
			},
			expected: false,
		},
		{
			name: "缺少API Key",
			client: &OpenAIClient{
				config: &ai.AIConfig{
					Enabled: true,
					OpenAI: ai.OpenAIConfig{
						APIKey: "",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsHealthy()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestOpenAIClient_Parse 测试Parse接口（转发到ParseMessage）
func TestOpenAIClient_Parse(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-invalid-key-for-test",
			BaseURL:      "https://api.openai.com/v1",
			PrimaryModel: "gpt-4o-mini",
			BackupModel:  "gpt-4o-mini",
			Temperature:  0.1,
			MaxTokens:    1000,
			MaxRetries:   1, // 只重试一次避免测试太慢
		},
		Prompts: ai.PromptsConfig{
			ReminderParse: "Parse this: {{.Message}}",
			ChatResponse:  "Reply to: {{.Message}}",
		},
	}
	client := NewOpenAIClient(config)
	require.NotNil(t, client)

	ctx := context.Background()

	// 注意：这个测试会失败因为API key无效，但能测试代码路径
	_, err := client.Parse(ctx, "test-user", "每天早上8点提醒我喝水")

	// 应该返回错误（因为API key无效）
	assert.Error(t, err)
}

// TestOpenAIClient_buildReminderPrompt 测试构建提醒prompt
func TestOpenAIClient_buildReminderPrompt(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-test",
			PrimaryModel: "gpt-4o-mini",
		},
		Prompts: ai.PromptsConfig{
			ReminderParse: "用户消息: {{.Message}}\n当前时间: {{.CurrentTime}}\n历史: {{.ConversationHistory}}",
		},
	}
	client := NewOpenAIClient(config)
	require.NotNil(t, client)

	prompt := client.buildReminderPrompt("每天早上8点提醒我喝水")

	// 验证prompt包含预期元素
	assert.Contains(t, prompt, "每天早上8点提醒我喝水", "应包含用户消息")
	assert.Contains(t, prompt, "用户消息", "应包含模板文本")
	assert.NotContains(t, prompt, "{{.Message}}", "不应包含未替换的占位符")
	assert.NotContains(t, prompt, "{{.CurrentTime}}", "不应包含未替换的占位符")
	assert.NotContains(t, prompt, "{{.ConversationHistory}}", "不应包含未替换的占位符")
}

// TestOpenAIClient_buildChatPrompt 测试构建对话prompt
func TestOpenAIClient_buildChatPrompt(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-test",
			PrimaryModel: "gpt-4o-mini",
		},
		Prompts: ai.PromptsConfig{
			ChatResponse: "回复: {{.Message}}\n历史: {{.ConversationHistory}}",
		},
	}
	client := NewOpenAIClient(config)
	require.NotNil(t, client)

	prompt := client.buildChatPrompt("你好")

	// 验证prompt包含预期元素
	assert.Contains(t, prompt, "你好", "应包含用户消息")
	assert.Contains(t, prompt, "回复", "应包含模板文本")
	assert.NotContains(t, prompt, "{{.Message}}", "不应包含未替换的占位符")
	assert.NotContains(t, prompt, "{{.ConversationHistory}}", "不应包含未替换的占位符")
}

// TestOpenAIClient_handleOpenAIError 测试错误处理
func TestOpenAIClient_handleOpenAIError(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-test",
			PrimaryModel: "gpt-4o-mini",
		},
	}
	client := NewOpenAIClient(config)
	require.NotNil(t, client)

	tests := []struct {
		name             string
		inputError       error
		expectedContains string
	}{
		{
			name:             "限流错误",
			inputError:       ai.ErrAPIRateLimit,
			expectedContains: "rate limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.handleOpenAIError(tt.inputError)

			// 验证返回的是AIError
			var aiErr *ai.AIError
			assert.ErrorAs(t, err, &aiErr)
		})
	}
}
