package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/time/rate"

	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
)

// OpenAIClient OpenAI客户端封装
type OpenAIClient struct {
	client      *openai.Client
	config      *ai.AIConfig
	rateLimiter *rate.Limiter
}

// NewOpenAIClient 创建OpenAI客户端
func NewOpenAIClient(config *ai.AIConfig) *OpenAIClient {
	if !config.Enabled || config.OpenAI.APIKey == "" {
		logger.Warn("OpenAI client created but AI is disabled or API key is missing")
		return nil
	}

	// 创建OpenAI客户端
	clientConfig := openai.DefaultConfig(config.OpenAI.APIKey)
	if config.OpenAI.BaseURL != "" {
		clientConfig.BaseURL = config.OpenAI.BaseURL
	}
	
	client := openai.NewClientWithConfig(clientConfig)

	// 创建限流器：每秒最多10个请求
	rateLimiter := rate.NewLimiter(rate.Limit(10), 1)

	return &OpenAIClient{
		client:      client,
		config:      config,
		rateLimiter: rateLimiter,
	}
}

// ParseMessage 解析消息
func (c *OpenAIClient) ParseMessage(ctx context.Context, userID, message string) (*ai.ParseResult, error) {
	start := time.Now()

	// 限流检查
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, ai.NewAIError(ai.ErrorTypeTimeout, "rate limit exceeded", err)
	}

	// 构建prompt
	prompt := c.buildReminderPrompt(message)

	// 调用OpenAI API
	result, err := c.callOpenAIWithRetry(ctx, prompt, c.config.OpenAI.PrimaryModel)
	if err != nil {
		// 如果主模型失败，尝试备用模型
		logger.Warnf("Primary model failed, trying backup model: %v", err)
		result, err = c.callOpenAIWithRetry(ctx, prompt, c.config.OpenAI.BackupModel)
		if err != nil {
			return nil, fmt.Errorf("both primary and backup models failed: %w", err)
		}
	}

	// 解析AI响应
	parseResult, err := c.parseAIResponse(result)
	if err != nil {
		return nil, ai.NewAIError(ai.ErrorTypeParsing, "failed to parse AI response", err)
	}

	// 设置元信息
	parseResult.ParsedBy = fmt.Sprintf("openai-%s", c.config.OpenAI.PrimaryModel)
	parseResult.ProcessTime = time.Since(start)
	parseResult.Timestamp = time.Now()

	logger.Infof("AI parsing completed in %v, intent: %s, confidence: %.2f", 
		parseResult.ProcessTime, parseResult.Intent, parseResult.Confidence)

	return parseResult, nil
}

// Chat 对话功能
func (c *OpenAIClient) Chat(ctx context.Context, userID, message string) (*ai.ChatResponse, error) {
	start := time.Now()

	// 限流检查
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, ai.NewAIError(ai.ErrorTypeTimeout, "rate limit exceeded", err)
	}

	// 构建对话prompt
	prompt := c.buildChatPrompt(message)

	// 调用OpenAI API
	response, err := c.callOpenAIWithRetry(ctx, prompt, c.config.OpenAI.PrimaryModel)
	if err != nil {
		return nil, fmt.Errorf("chat api call failed: %w", err)
	}

	return &ai.ChatResponse{
		Response:    strings.TrimSpace(response),
		ParsedBy:    fmt.Sprintf("openai-%s", c.config.OpenAI.PrimaryModel),
		ProcessTime: time.Since(start),
		Timestamp:   time.Now(),
	}, nil
}

// callOpenAIWithRetry 带重试的OpenAI API调用
func (c *OpenAIClient) callOpenAIWithRetry(ctx context.Context, prompt, model string) (string, error) {
	var lastErr error

	for i := 0; i < c.config.OpenAI.MaxRetries; i++ {
		if i > 0 {
			// 指数退避
			delay := time.Duration(i*i) * time.Second
			logger.Infof("Retrying OpenAI call in %v (attempt %d/%d)", delay, i+1, c.config.OpenAI.MaxRetries)
			
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		response, err := c.callOpenAI(ctx, prompt, model)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// 检查是否应该重试
		if !ai.IsRetryableError(err) {
			logger.Warnf("Non-retryable error occurred: %v", err)
			break
		}

		logger.Warnf("Retryable error occurred (attempt %d/%d): %v", i+1, c.config.OpenAI.MaxRetries, err)
	}

	return "", fmt.Errorf("max retries exceeded, last error: %w", lastErr)
}

// callOpenAI 调用OpenAI API
func (c *OpenAIClient) callOpenAI(ctx context.Context, prompt, model string) (string, error) {
	// 创建带超时的context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.OpenAI.Timeout)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model:       model,
		Temperature: c.config.OpenAI.Temperature,
		MaxTokens:   c.config.OpenAI.MaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	resp, err := c.client.CreateChatCompletion(timeoutCtx, req)
	if err != nil {
		return "", c.handleOpenAIError(err)
	}

	if len(resp.Choices) == 0 {
		return "", ai.NewAIError(ai.ErrorTypeAPI, "no response choices from OpenAI", nil)
	}

	return resp.Choices[0].Message.Content, nil
}

// handleOpenAIError 处理OpenAI错误
func (c *OpenAIClient) handleOpenAIError(err error) error {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "timeout"):
		return ai.NewAIError(ai.ErrorTypeTimeout, "OpenAI API timeout", err)
	case strings.Contains(errStr, "rate limit"):
		return ai.NewAIError(ai.ErrorTypeAPI, "OpenAI rate limit exceeded", ai.ErrAPIRateLimit)
	case strings.Contains(errStr, "insufficient_quota"):
		return ai.NewAIError(ai.ErrorTypeAPI, "OpenAI quota exceeded", ai.ErrAPIQuotaExceeded)
	case strings.Contains(errStr, "invalid_api_key"):
		return ai.NewAIError(ai.ErrorTypeAPI, "OpenAI API key invalid", ai.ErrAPIAuth)
	case strings.Contains(errStr, "model_not_found"):
		return ai.NewAIError(ai.ErrorTypeAPI, "OpenAI model not found", ai.ErrAPIModelNotFound)
	case strings.Contains(errStr, "connection"):
		return ai.NewAIError(ai.ErrorTypeNetwork, "OpenAI connection error", err)
	default:
		return ai.NewAIError(ai.ErrorTypeAPI, "OpenAI API error", err)
	}
}

// buildReminderPrompt 构建提醒解析prompt
func (c *OpenAIClient) buildReminderPrompt(message string) string {
	// 替换模板变量
	prompt := c.config.Prompts.ReminderParse
	prompt = strings.ReplaceAll(prompt, "{{.Message}}", message)
	prompt = strings.ReplaceAll(prompt, "{{.CurrentTime}}", time.Now().Format("2006-01-02 15:04:05"))
	
	// TODO: 后续添加对话历史支持
	prompt = strings.ReplaceAll(prompt, "{{.ConversationHistory}}", "")
	
	return prompt
}

// buildChatPrompt 构建对话prompt
func (c *OpenAIClient) buildChatPrompt(message string) string {
	prompt := c.config.Prompts.ChatResponse
	prompt = strings.ReplaceAll(prompt, "{{.Message}}", message)
	
	// TODO: 后续添加对话历史支持
	prompt = strings.ReplaceAll(prompt, "{{.ConversationHistory}}", "")
	
	return prompt
}

// parseAIResponse 解析AI响应
func (c *OpenAIClient) parseAIResponse(response string) (*ai.ParseResult, error) {
	// 清理响应内容
	response = strings.TrimSpace(response)
	
	// 移除可能的markdown代码块标记
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	}
	if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}
	
	response = strings.TrimSpace(response)

	// 解析JSON
	var result ai.ParseResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		logger.Errorf("Failed to parse AI response as JSON: %v\nResponse: %s", err, response)
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// 验证解析结果
	validation := result.Validate()
	if !validation.IsValid {
		logger.Errorf("AI response validation failed: %v", validation.Errors)
		return nil, fmt.Errorf("invalid AI response: %s", strings.Join(validation.Errors, ", "))
	}

	return &result, nil
}

// GetName 获取解析器名称
func (c *OpenAIClient) GetName() string {
	return fmt.Sprintf("openai-%s", c.config.OpenAI.PrimaryModel)
}

// GetPriority 获取解析器优先级
func (c *OpenAIClient) GetPriority() int {
	return ai.ParserTypePrimaryAI.Priority()
}

// IsHealthy 检查客户端是否健康
func (c *OpenAIClient) IsHealthy() bool {
	if c == nil || c.client == nil {
		return false
	}

	if !c.config.Enabled || c.config.OpenAI.APIKey == "" {
		return false
	}

	// TODO: 可以添加实际的健康检查调用
	return true
}

// Parse 实现Parser接口 - 统一的解析入口
func (c *OpenAIClient) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	return c.ParseMessage(ctx, userID, message)
}