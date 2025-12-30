package ai

import (
	"context"
	"time"
)

// AIProviderInterface 定义AI服务提供商的统一接口
// 注意：这个接口用于新的多Provider架构，与现有types.go分开
type AIProviderInterface interface {
	// ParseReminder 解析自然语言为提醒信息
	ParseReminder(ctx context.Context, text string) (*ProviderParseResult, error)

	// Chat 聊天对话
	Chat(ctx context.Context, text string) (string, error)

	// Name 获取提供商名称
	Name() string

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// GetConfig 获取配置信息
	GetConfig() ProviderConfig
}

// ProviderParseResult Provider解析结果（简化版，用于新架构）
type ProviderParseResult struct {
	Content     string    `json:"content"`      // 提醒内容
	Time        time.Time `json:"time"`         // 提醒时间
	Pattern     string    `json:"pattern"`      // 重复模式 (daily/weekly/once)
	Confidence  float64   `json:"confidence"`   // 置信度 0-1
	RawResponse string    `json:"raw_response"` // 原始响应
	TokensUsed  int       `json:"tokens_used"`  // Token使用量
}

// ProviderConfig Provider配置
type ProviderConfig struct {
	Name         string        `yaml:"name"`
	Endpoint     string        `yaml:"endpoint"`
	APIKey       string        `yaml:"api_key"`
	Model        string        `yaml:"model"`
	MaxTokens    int           `yaml:"max_tokens"`
	Temperature  float64       `yaml:"temperature"`
	Timeout      time.Duration `yaml:"timeout"`
	RateLimit    int           `yaml:"rate_limit"` // 每分钟请求数
}

// ProviderError 统一错误类型
type ProviderError struct {
	Provider string
	Err      error
	Type     string
}

func (e *ProviderError) Error() string {
	return e.Err.Error()
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// Alert 告警结构
type Alert struct {
	RuleName     string     `json:"rule_name"`
	Type         string     `json:"type"`
	Message      string     `json:"message"`
	Timestamp    time.Time  `json:"timestamp"`
	Provider     string     `json:"provider"`
	Threshold    float64    `json:"threshold"`
	CurrentValue float64    `json:"current_value"`
}