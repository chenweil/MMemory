package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider OpenAI服务提供商实现
type OpenAIProvider struct {
	client *openai.Client
	config ProviderConfig
	limiter *RateLimiter
}

// NewOpenAIProvider 创建OpenAI Provider
func NewOpenAIProvider(config ProviderConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	client := openai.NewClient(config.APIKey)

	// 设置默认值
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
		config.RateLimit = 60 // 默认60 req/min
	}

	return &OpenAIProvider{
		client:  client,
		config:  config,
		limiter: NewRateLimiter(config.RateLimit),
	}, nil
}

// ParseReminder 实现AI解析
func (p *OpenAIProvider) ParseReminder(ctx context.Context, text string) (*ProviderParseResult, error) {
	// 限流检查
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, &ProviderError{
			Provider: p.Name(),
			Err:      err,
			Type:     "RATE_LIMIT",
		}
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// 构建Prompt
	prompt := p.buildPrompt(text)

	// 调用OpenAI API
	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.config.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompt.System,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt.User,
				},
			},
			MaxTokens:   p.config.MaxTokens,
			Temperature: float32(p.config.Temperature),
		},
	)

	if err != nil {
		return nil, p.handleError(err)
	}

	// 解析响应
	if len(resp.Choices) == 0 {
		return nil, &ProviderError{
			Provider: p.Name(),
			Err:      fmt.Errorf("empty response"),
			Type:     "INVALID_RESPONSE",
		}
	}

	result, err := p.parseResponse(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, &ProviderError{
			Provider: p.Name(),
			Err:      err,
			Type:     "INVALID_RESPONSE",
		}
	}

	result.RawResponse = resp.Choices[0].Message.Content
	result.TokensUsed = resp.Usage.TotalTokens

	return result, nil
}

// Chat 聊天对话
func (p *OpenAIProvider) Chat(ctx context.Context, text string) (string, error) {
	// 限流检查
	if err := p.limiter.Wait(ctx); err != nil {
		return "", &ProviderError{
			Provider: p.Name(),
			Err:      err,
			Type:     "RATE_LIMIT",
		}
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// 构建聊天Prompt
	prompt := p.buildChatPrompt(text)

	// 调用OpenAI API
	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.config.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompt.System,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt.User,
				},
			},
			MaxTokens:   p.config.MaxTokens,
			Temperature: float32(p.config.Temperature),
		},
	)

	if err != nil {
		return "", p.handleError(err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// buildPrompt 构建提醒解析Prompt
func (p *OpenAIProvider) buildPrompt(text string) struct{ System, User string } {
	return struct{ System, User string }{
		System: `你是一个智能提醒助手，负责解析用户的自然语言输入并提取提醒信息。

请严格按照以下JSON格式返回结果：
{
  "content": "提醒的具体内容",
  "time": "2025-10-01T09:00:00Z",
  "pattern": "daily|weekly|monthly|once",
  "confidence": 0.95
}

规则：
1. time必须是ISO 8601格式的完整时间
2. pattern只能是: daily(每天)、weekly(每周)、monthly(每月)、once(一次性)
3. confidence是0-1之间的浮点数，表示解析置信度
4. 如果无法确定时间，返回当前时间+1小时
5. 如果无法确定模式，默认为once`,
		User: fmt.Sprintf("请解析以下提醒信息：\n%s", text),
	}
}

// buildChatPrompt 构建聊天Prompt
func (p *OpenAIProvider) buildChatPrompt(text string) struct{ System, User string } {
	return struct{ System, User string }{
		System: `你是一个友好的智能助手，负责帮助用户管理提醒。请用简洁、友好的语调回答用户的问题。`,
		User:  text,
	}
}

// parseResponse 解析API响应
func (p *OpenAIProvider) parseResponse(content string) (*ProviderParseResult, error) {
	var result ProviderParseResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 验证必填字段
	if result.Content == "" {
		return nil, fmt.Errorf("missing content field")
	}
	if result.Time.IsZero() {
		return nil, fmt.Errorf("missing or invalid time field")
	}
	if result.Pattern == "" {
		result.Pattern = "once"
	}
	if result.Confidence == 0 {
		result.Confidence = 0.8
	}

	return &result, nil
}

// handleError 错误处理和分类
func (p *OpenAIProvider) handleError(err error) error {
	// 判断错误类型
	if err.Error() == "context deadline exceeded" {
		return &ProviderError{
			Provider: p.Name(),
			Err:      err,
			Type:     "TIMEOUT",
		}
	}

	// API错误处理
	return &ProviderError{
		Provider: p.Name(),
		Err:      err,
		Type:     "API_ERROR",
	}
}

// Name 实现接口
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// HealthCheck 实现接口
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := p.client.ListModels(ctx)
	return err
}

// GetConfig 实现接口
func (p *OpenAIProvider) GetConfig() ProviderConfig {
	return p.config
}