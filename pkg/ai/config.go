package ai

import (
	"time"
)

// AIConfig AI配置结构
type AIConfig struct {
	Enabled bool           `mapstructure:"enabled" yaml:"enabled"`
	OpenAI  OpenAIConfig   `mapstructure:"openai" yaml:"openai"`
	Prompts PromptsConfig  `mapstructure:"prompts" yaml:"prompts"`
}

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
	APIKey       string        `mapstructure:"api_key" yaml:"api_key"`
	BaseURL      string        `mapstructure:"base_url" yaml:"base_url"`
	PrimaryModel string        `mapstructure:"primary_model" yaml:"primary_model"`
	BackupModel  string        `mapstructure:"backup_model" yaml:"backup_model"`
	Temperature  float32       `mapstructure:"temperature" yaml:"temperature"`
	MaxTokens    int           `mapstructure:"max_tokens" yaml:"max_tokens"`
	Timeout      time.Duration `mapstructure:"timeout" yaml:"timeout"`
	MaxRetries   int           `mapstructure:"max_retries" yaml:"max_retries"`
}

// PromptsConfig Prompt模板配置
type PromptsConfig struct {
	ReminderParse string `mapstructure:"reminder_parse" yaml:"reminder_parse"`
	ChatResponse  string `mapstructure:"chat_response" yaml:"chat_response"`
}

// GetDefaultAIConfig 获取默认AI配置
func GetDefaultAIConfig() *AIConfig {
	return &AIConfig{
		Enabled: false, // 默认关闭，需要手动启用
		OpenAI: OpenAIConfig{
			BaseURL:      "https://api.openai.com/v1",
			PrimaryModel: "gpt-4o-mini",
			BackupModel:  "gpt-3.5-turbo",
			Temperature:  0.1,
			MaxTokens:    1000,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
		},
		Prompts: PromptsConfig{
			ReminderParse: getDefaultReminderPrompt(),
			ChatResponse:  getDefaultChatPrompt(),
		},
	}
}

// Validate 验证AI配置
func (c *AIConfig) Validate() error {
	if !c.Enabled {
		return nil // 如果未启用，跳过验证
	}

	if c.OpenAI.APIKey == "" {
		return ErrMissingAPIKey
	}

	if c.OpenAI.PrimaryModel == "" {
		return ErrMissingPrimaryModel
	}

	if c.OpenAI.MaxTokens <= 0 {
		return ErrInvalidMaxTokens
	}

	if c.OpenAI.Temperature < 0 || c.OpenAI.Temperature > 2 {
		return ErrInvalidTemperature
	}

	return nil
}

// getDefaultReminderPrompt 默认提醒解析Prompt
func getDefaultReminderPrompt() string {
	return `你是MMemory的智能助手。请分析用户消息，识别意图并返回JSON格式结果。

当前时间: {{.CurrentTime}}
用户消息: "{{.Message}}"
{{if .ConversationHistory}}对话历史: {{.ConversationHistory}}{{end}}

支持的功能:
1. 创建提醒 - 用户想要设置提醒、待办、日程
2. 普通对话 - 用户想要聊天、询问信息  
3. 查询总结 - 用户想要查看或总结某些内容

时间格式说明:
- 支持绝对时间: "明天8点", "下周一9点"
- 支持相对时间: "1小时后", "明天"
- 支持重复模式: "每天", "每周一三五", "工作日"

请返回以下JSON格式(不要包含markdown代码块标记):
{
  "intent": "reminder|chat|summary|query",
  "confidence": 0.95,
  "reminder": {
    "title": "具体要做的事情",
    "type": "habit|task",
    "time": {
      "hour": 8,
      "minute": 0,
      "timezone": "Asia/Shanghai",
      "is_relative_time": false,
      "relative_desc": ""
    },
    "schedule_pattern": "daily|weekly:1,3,5|monthly:1,15|once",
    "description": "详细描述"
  },
  "chat_response": {
    "response": "如果是对话意图的回复内容",
    "need_follow_up": false
  }
}

示例:
用户: "每天早上8点提醒我喝水"
返回: {"intent":"reminder","confidence":0.95,"reminder":{"title":"喝水","type":"habit","time":{"hour":8,"minute":0,"timezone":"Asia/Shanghai"},"schedule_pattern":"daily"}}

用户: "我在看《三体》"
返回: {"intent":"chat","confidence":0.9,"chat_response":{"response":"《三体》是刘慈欣的经典科幻小说，讲述了人类文明与三体文明的接触。你觉得哪个情节最印象深刻？","need_follow_up":true}}`
}

// getDefaultChatPrompt 默认对话Prompt
func getDefaultChatPrompt() string {
	return `你是MMemory智能助手。用户正在与你对话。请自然、友好地回应。
如果用户提到书籍、电影等，可以简单介绍。保持回复简洁（50字以内）。

{{if .ConversationHistory}}对话历史: {{.ConversationHistory}}{{end}}
用户消息: "{{.Message}}"

请直接回复，不需要JSON格式。`
}