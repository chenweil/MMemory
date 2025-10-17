package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDefaultAIConfig 测试获取默认配置
func TestGetDefaultAIConfig(t *testing.T) {
	config := GetDefaultAIConfig()

	require.NotNil(t, config, "默认配置不应为nil")

	// 验证默认值
	assert.False(t, config.Enabled, "默认应该关闭AI")
	assert.Equal(t, "https://api.openai.com/v1", config.OpenAI.BaseURL)
	assert.Equal(t, "gpt-4o-mini", config.OpenAI.PrimaryModel)
	assert.Equal(t, "gpt-3.5-turbo", config.OpenAI.BackupModel)
	assert.Equal(t, float32(0.1), config.OpenAI.Temperature)
	assert.Equal(t, 1000, config.OpenAI.MaxTokens)
	assert.Equal(t, 30*time.Second, config.OpenAI.Timeout)
	assert.Equal(t, 3, config.OpenAI.MaxRetries)

	// 验证Prompt模板不为空
	assert.NotEmpty(t, config.Prompts.ReminderParse, "ReminderParse模板不应为空")
	assert.NotEmpty(t, config.Prompts.ChatResponse, "ChatResponse模板不应为空")

	// 验证Prompt模板包含必要的占位符
	assert.Contains(t, config.Prompts.ReminderParse, "{{.Message}}", "应包含消息占位符")
	assert.Contains(t, config.Prompts.ReminderParse, "{{.CurrentTime}}", "应包含时间占位符")
	assert.Contains(t, config.Prompts.ChatResponse, "{{.Message}}", "对话模板应包含消息占位符")
}

// TestAIConfig_Validate 测试配置验证
func TestAIConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *AIConfig
		expectErr error
	}{
		{
			name: "有效配置",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  0.5,
				},
			},
			expectErr: nil,
		},
		{
			name: "AI未启用时跳过验证",
			config: &AIConfig{
				Enabled: false,
				OpenAI: OpenAIConfig{
					APIKey: "", // 即使缺少API Key也应该通过
				},
			},
			expectErr: nil,
		},
		{
			name: "缺少API Key",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  0.5,
				},
			},
			expectErr: ErrMissingAPIKey,
		},
		{
			name: "缺少Primary Model",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "",
					MaxTokens:    1000,
					Temperature:  0.5,
				},
			},
			expectErr: ErrMissingPrimaryModel,
		},
		{
			name: "无效的MaxTokens (0)",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    0,
					Temperature:  0.5,
				},
			},
			expectErr: ErrInvalidMaxTokens,
		},
		{
			name: "无效的MaxTokens (负数)",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    -100,
					Temperature:  0.5,
				},
			},
			expectErr: ErrInvalidMaxTokens,
		},
		{
			name: "无效的Temperature (负数)",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  -0.1,
				},
			},
			expectErr: ErrInvalidTemperature,
		},
		{
			name: "无效的Temperature (>2)",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  2.1,
				},
			},
			expectErr: ErrInvalidTemperature,
		},
		{
			name: "Temperature边界值 (0)",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  0.0,
				},
			},
			expectErr: nil,
		},
		{
			name: "Temperature边界值 (2)",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test123",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  2.0,
				},
			},
			expectErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetDefaultReminderPrompt 测试默认提醒Prompt模板
func TestGetDefaultReminderPrompt(t *testing.T) {
	prompt := getDefaultReminderPrompt()

	assert.NotEmpty(t, prompt, "默认提醒Prompt不应为空")

	// 验证包含关键元素
	expectedElements := []string{
		"{{.Message}}",           // 消息占位符
		"{{.CurrentTime}}",       // 时间占位符
		"{{.ConversationHistory}}", // 对话历史占位符
		"intent",                 // Intent字段
		"confidence",             // 置信度字段
		"reminder",               // 提醒对象
		"delete",                 // 删除意图
		"edit",                   // 编辑意图
		"pause",                  // 暂停意图
		"resume",                 // 恢复意图
		"chat",                   // 对话意图
		"query",                  // 查询意图
		"summary",                // 总结意图
		"JSON",                   // 要求返回JSON格式
	}

	for _, element := range expectedElements {
		assert.Contains(t, prompt, element, "Prompt应该包含: "+element)
	}

	// 验证包含示例
	assert.Contains(t, prompt, "示例", "应该包含示例说明")
	assert.Contains(t, prompt, "每天早上8点提醒我喝水", "应该包含提醒示例")
	assert.Contains(t, prompt, "撤销今晚的健身提醒", "应该包含删除示例")
	assert.Contains(t, prompt, "把健身提醒改到晚上7点", "应该包含编辑示例")
	assert.Contains(t, prompt, "暂停一周的健身提醒", "应该包含暂停示例")
}

// TestGetDefaultChatPrompt 测试默认对话Prompt模板
func TestGetDefaultChatPrompt(t *testing.T) {
	prompt := getDefaultChatPrompt()

	assert.NotEmpty(t, prompt, "默认对话Prompt不应为空")

	// 验证包含关键元素
	expectedElements := []string{
		"{{.Message}}",           // 消息占位符
		"{{.ConversationHistory}}", // 对话历史占位符
		"MMemory",                // 助手名称
		"智能助手",                  // 角色定位
	}

	for _, element := range expectedElements {
		assert.Contains(t, prompt, element, "对话Prompt应该包含: "+element)
	}

	// 验证包含直接回复说明
	assert.Contains(t, prompt, "直接回复", "应该要求直接回复")
}

// TestPromptTemplateVariables 测试Prompt模板变量一致性
func TestPromptTemplateVariables(t *testing.T) {
	reminderPrompt := getDefaultReminderPrompt()
	chatPrompt := getDefaultChatPrompt()

	// 两个模板都应该支持相同的核心变量
	commonVariables := []string{
		"{{.Message}}",
		"{{.ConversationHistory}}",
	}

	for _, variable := range commonVariables {
		assert.Contains(t, reminderPrompt, variable, "ReminderPrompt应该包含: "+variable)
		assert.Contains(t, chatPrompt, variable, "ChatPrompt应该包含: "+variable)
	}

	// ReminderPrompt特有的变量
	assert.Contains(t, reminderPrompt, "{{.CurrentTime}}", "ReminderPrompt应该包含CurrentTime变量")
}

// TestAIConfigDefaults 测试默认配置的合理性
func TestAIConfigDefaults(t *testing.T) {
	config := GetDefaultAIConfig()

	// Temperature应该在合理范围内
	assert.GreaterOrEqual(t, config.OpenAI.Temperature, float32(0.0))
	assert.LessOrEqual(t, config.OpenAI.Temperature, float32(2.0))

	// MaxTokens应该合理
	assert.Greater(t, config.OpenAI.MaxTokens, 0)
	assert.LessOrEqual(t, config.OpenAI.MaxTokens, 10000)

	// Timeout应该合理
	assert.Greater(t, config.OpenAI.Timeout, time.Duration(0))
	assert.LessOrEqual(t, config.OpenAI.Timeout, 5*time.Minute)

	// MaxRetries应该合理
	assert.Greater(t, config.OpenAI.MaxRetries, 0)
	assert.LessOrEqual(t, config.OpenAI.MaxRetries, 10)

	// BaseURL应该是有效的URL格式
	assert.Contains(t, config.OpenAI.BaseURL, "http")
	assert.Contains(t, config.OpenAI.BaseURL, "://")

	// Model名称不应为空
	assert.NotEmpty(t, config.OpenAI.PrimaryModel)
	assert.NotEmpty(t, config.OpenAI.BackupModel)
}

// TestAIConfigValidate_EdgeCases 测试配置验证的边界情况
func TestAIConfigValidate_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		config    *AIConfig
		expectErr bool
		errType   error
	}{
		{
			name: "Temperature=0有效",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test",
					PrimaryModel: "gpt-4",
					MaxTokens:    100,
					Temperature:  0.0,
				},
			},
			expectErr: false,
		},
		{
			name: "Temperature=2有效",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test",
					PrimaryModel: "gpt-4",
					MaxTokens:    100,
					Temperature:  2.0,
				},
			},
			expectErr: false,
		},
		{
			name: "MaxTokens=1有效",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test",
					PrimaryModel: "gpt-4",
					MaxTokens:    1,
					Temperature:  0.5,
				},
			},
			expectErr: false,
		},
		{
			name: "极大的MaxTokens有效",
			config: &AIConfig{
				Enabled: true,
				OpenAI: OpenAIConfig{
					APIKey:       "sk-test",
					PrimaryModel: "gpt-4",
					MaxTokens:    100000,
					Temperature:  0.5,
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
