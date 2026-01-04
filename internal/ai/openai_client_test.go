package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/internal/models"
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
		{
			name:             "超时错误",
			inputError:       &ai.AIError{Type: ai.ErrorTypeTimeout, Message: "timeout"},
			expectedContains: "timeout",
		},
		{
			name:             "配额错误（预定义错误，走默认分支）",
			inputError:       ai.ErrAPIQuotaExceeded,
			expectedContains: "API error",
		},
		{
			name:             "认证错误（预定义错误，走默认分支）",
			inputError:       ai.ErrAPIAuth,
			expectedContains: "API error",
		},
		{
			name:             "模型错误（预定义错误，走默认分支）",
			inputError:       ai.ErrAPIModelNotFound,
			expectedContains: "API error",
		},
		{
			name:             "连接错误",
			inputError:       &ai.AIError{Type: ai.ErrorTypeNetwork, Message: "connection refused"},
			expectedContains: "connection",
		},
		{
			name:             "通用API错误（走默认分支）",
			inputError:       &ai.AIError{Type: ai.ErrorTypeAPI, Message: "api error"},
			expectedContains: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.handleOpenAIError(tt.inputError)

			// 验证返回的是AIError
			var aiErr *ai.AIError
			require.ErrorAs(t, err, &aiErr)
			assert.Contains(t, aiErr.Message, tt.expectedContains)
		})
	}
}

// TestOpenAIClient_formatConversationHistory 测试格式化对话历史
func TestOpenAIClient_formatConversationHistory(t *testing.T) {
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
		name     string
		history  string
		validate func(t *testing.T, result string)
	}{
		{
			name:    "空历史",
			history: "",
			validate: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
		{
			name:    "单行历史",
			history: "用户说：你好",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "最近对话历史")
				assert.Contains(t, result, "用户说：你好")
				assert.Contains(t, result, "•")
			},
		},
		{
			name: "多行历史",
			history: "用户说：你好\n用户说：今天天气如何\n用户说：提醒我喝水",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "最近对话历史")
				assert.Contains(t, result, "用户说：你好")
				assert.Contains(t, result, "用户说：今天天气如何")
				assert.Contains(t, result, "用户说：提醒我喝水")
				assert.Contains(t, result, "•")
			},
		},
		{
			name: "超过20行历史",
			history: func() string {
				var lines string
				for i := 1; i <= 25; i++ {
					lines += fmt.Sprintf("用户说：消息%d\n", i)
				}
				return lines
			}(),
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "最近对话历史")
				// 验证包含最后几条消息
				assert.Contains(t, result, "消息20")
				assert.Contains(t, result, "消息25")
				// 验证历史被截断（不应该包含早期的消息）
				// 注意：使用完整行匹配，避免子字符串匹配问题
				assert.NotContains(t, result, "• 用户说：消息1\n")
				assert.NotContains(t, result, "• 用户说：消息5\n")
			},
		},
		{
			name:    "带空行的历史",
			history: "用户说：你好\n\n用户说：天气如何",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "最近对话历史")
				assert.Contains(t, result, "用户说：你好")
				assert.Contains(t, result, "用户说：天气如何")
				// 空行应该被忽略
				lines := strings.Split(result, "•")
				assert.Equal(t, 3, len(lines)) // 标题 + 2条消息
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.formatConversationHistory(tt.history)
			tt.validate(t, result)
		})
	}
}

// TestOpenAIClient_parseAIResponse 测试解析AI响应
func TestOpenAIClient_parseAIResponse(t *testing.T) {
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
		name        string
		response    string
		expectError bool
		validate    func(t *testing.T, result *ai.ParseResult)
	}{
		{
			name: "有效的JSON响应",
			response: `{
				"intent": "reminder",
				"confidence": 0.95,
				"reminder": {
					"title": "喝水",
					"type": "habit",
					"time": {
						"hour": 8,
						"minute": 0,
						"timezone": "Asia/Shanghai"
					},
					"schedule_pattern": "daily"
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.ParseIntent("reminder"), result.Intent)
				assert.Equal(t, float32(0.95), result.Confidence)
				assert.Equal(t, "喝水", result.Reminder.Title)
				assert.Equal(t, 8, result.Reminder.Time.Hour)
				assert.Equal(t, models.SchedulePatternDaily, result.Reminder.SchedulePattern)
			},
		},
		{
			name: "带markdown代码块的JSON",
			response: "```json\n{\n\t\"intent\": \"reminder\",\n\t\"confidence\": 0.9,\n\t\"reminder\": {\n\t\t\"title\": \"测试\",\n\t\t\"type\": \"habit\",\n\t\t\"time\": {\n\t\t\t\"hour\": 8,\n\t\t\t\"minute\": 0,\n\t\t\t\"timezone\": \"Asia/Shanghai\"\n\t\t},\n\t\t\"schedule_pattern\": \"daily\"\n\t}\n}\n```",
			expectError: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.ParseIntent("reminder"), result.Intent)
				assert.Equal(t, float32(0.9), result.Confidence)
			},
		},
		{
			name:        "无效的JSON",
			response:    "这不是有效的JSON",
			expectError: true,
		},
		{
			name: "空响应",
			response:    "",
			expectError: true,
		},
		{
			name: "缺少必需字段的JSON",
			response: `{
				"intent": "reminder"
			}`,
			expectError: true, // confidence是必需的
		},
		{
			name: "带前后空格的JSON",
			response: `  {
				"intent": "reminder",
				"confidence": 0.8,
				"reminder": {
					"title": "测试",
					"type": "habit",
					"time": {
						"hour": 8,
						"minute": 0,
						"timezone": "Asia/Shanghai"
					},
					"schedule_pattern": "daily"
				}
			}  `,
			expectError: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.ParseIntent("reminder"), result.Intent)
			},
		},
		{
			name: "chat意图响应",
			response: `{
				"intent": "chat",
				"confidence": 0.7,
				"chat_response": {
					"response": "好的，我明白了"
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.ParseIntent("chat"), result.Intent)
				assert.Equal(t, "好的，我明白了", result.ChatResponse.Response)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseAIResponse(tt.response)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				tt.validate(t, result)
			}
		})
	}
}

// TestOpenAIClient_buildReminderPromptWithContext 测试构建带上下文的提醒prompt
func TestOpenAIClient_buildReminderPromptWithContext(t *testing.T) {
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

	tests := []struct {
		name               string
		message            string
		conversationHistory string
		validate           func(t *testing.T, prompt string)
	}{
		{
			name:               "无对话历史",
			message:            "每天早上8点提醒我喝水",
			conversationHistory: "",
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "每天早上8点提醒我喝水")
				assert.NotContains(t, prompt, "{{.Message}}")
				assert.NotContains(t, prompt, "{{.ConversationHistory}}")
				assert.NotContains(t, prompt, "最近对话历史")
			},
		},
		{
			name:               "带对话历史",
			message:            "提醒我喝水",
			conversationHistory: "用户说：我最近口渴\n用户说：需要多喝水",
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "提醒我喝水")
				assert.Contains(t, prompt, "最近对话历史")
				assert.Contains(t, prompt, "用户说：我最近口渴")
				assert.Contains(t, prompt, "用户说：需要多喝水")
				assert.NotContains(t, prompt, "{{.ConversationHistory}}")
			},
		},
		{
			name:               "长对话历史",
			message:            "提醒我吃药",
			conversationHistory: strings.Repeat("用户说：消息\n", 25),
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "提醒我吃药")
				assert.Contains(t, prompt, "最近对话历史")
				// 历史应该被截断
				assert.NotContains(t, prompt, "{{.ConversationHistory}}")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := client.buildReminderPromptWithContext(tt.message, tt.conversationHistory)
			tt.validate(t, prompt)
		})
	}
}

// TestOpenAIClient_ParseWithContext 测试带上下文的解析
func TestOpenAIClient_ParseWithContext(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-invalid-key-for-test",
			BaseURL:      "https://api.openai.com/v1",
			PrimaryModel: "gpt-4o-mini",
			BackupModel:  "gpt-4o-mini",
			Temperature:  0.1,
			MaxTokens:    1000,
			MaxRetries:   1,
		},
		Prompts: ai.PromptsConfig{
			ReminderParse: "Parse: {{.Message}} History: {{.ConversationHistory}}",
			ChatResponse:  "Reply: {{.Message}}",
		},
	}
	client := NewOpenAIClient(config)
	require.NotNil(t, client)

	ctx := context.Background()

	tests := []struct {
		name               string
		userID             string
		message            string
		conversationHistory string
		expectError        bool
	}{
		{
			name:               "无历史记录",
			userID:             "test-user",
			message:            "每天早上8点提醒我喝水",
			conversationHistory: "",
			expectError:        true, // API key无效
		},
		{
			name:               "带历史记录",
			userID:             "test-user",
			message:            "提醒我喝水",
			conversationHistory: "用户说：我最近口渴",
			expectError:        true, // API key无效
		},
		{
			name:               "长历史记录",
			userID:             "test-user",
			message:            "提醒我吃药",
			conversationHistory: strings.Repeat("用户说：消息\n", 30),
			expectError:        true, // API key无效
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.ParseWithContext(ctx, tt.userID, tt.message, tt.conversationHistory)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

// TestOpenAIClient_Chat 测试对话功能
func TestOpenAIClient_Chat(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-invalid-key-for-test",
			BaseURL:      "https://api.openai.com/v1",
			PrimaryModel: "gpt-4o-mini",
			Temperature:  0.1,
			MaxTokens:    1000,
			MaxRetries:   1,
		},
		Prompts: ai.PromptsConfig{
			ChatResponse: "Reply to: {{.Message}}",
		},
	}
	client := NewOpenAIClient(config)
	require.NotNil(t, client)

	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		message     string
		expectError bool
	}{
		{
			name:        "简单对话",
			userID:      "test-user",
			message:     "你好",
			expectError: true, // API key无效
		},
		{
			name:        "复杂对话",
			userID:      "test-user",
			message:     "今天天气如何？我应该穿什么衣服？",
			expectError: true, // API key无效
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Chat(ctx, tt.userID, tt.message)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.Response)
				assert.NotEmpty(t, result.ParsedBy)
				assert.Greater(t, result.ProcessTime, time.Duration(0))
			}
		})
	}
}

// TestOpenAIClient_callOpenAIWithRetry 测试带重试的API调用
func TestOpenAIClient_callOpenAIWithRetry(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-invalid-key-for-test",
			BaseURL:      "https://api.openai.com/v1",
			PrimaryModel: "gpt-4o-mini",
			Temperature:  0.1,
			MaxTokens:    1000,
			MaxRetries:   2,
		},
	}
	client := NewOpenAIClient(config)
	require.NotNil(t, client)

	ctx := context.Background()

	t.Run("API调用失败", func(t *testing.T) {
		// 使用无效的API key，应该会失败
		result, err := client.callOpenAIWithRetry(ctx, "test prompt", "gpt-4o-mini")
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "max retries exceeded")
	})

	t.Run("上下文取消", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // 立即取消

		result, err := client.callOpenAIWithRetry(cancelCtx, "test prompt", "gpt-4o-mini")
		assert.Error(t, err)
		assert.Empty(t, result)
	})
}
