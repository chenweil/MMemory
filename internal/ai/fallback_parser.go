package ai

import (
	"context"
	"strings"
	"time"

	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
)

// FallbackChatParser 兜底对话解析器
// 当所有解析器都失败时，返回友好的提示消息
type FallbackChatParser struct {
	responses []string
}

// NewFallbackChatParser 创建兜底对话解析器
func NewFallbackChatParser() *FallbackChatParser {
	return &FallbackChatParser{
		responses: []string{
			"抱歉，我没有理解你说的内容。可以尝试这样说：\n• 每天早上8点提醒我喝水\n• 明天下午3点提醒我开会\n• 每周一上午9点提醒我写周报",
			"我还在学习中，没能理解你的意思。你可以试试：\n• 工作日晚上8点提醒我复习英语\n• 2025年10月15日上午10点提醒我体检",
			"我不太明白你的意思呢。提醒功能支持以下格式：\n• 每天/每周/工作日 + 时间 + 提醒我 + 内容\n• 明天/具体日期 + 时间 + 提醒我 + 内容",
		},
	}
}

// Parse 实现Parser接口
func (p *FallbackChatParser) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	message = strings.TrimSpace(message)

	logger.Warnf("FallbackChatParser triggered for message: %s", message)

	// 根据消息长度选择不同的回复
	responseIdx := len(message) % len(p.responses)
	response := p.responses[responseIdx]

	// 如果消息包含特定关键词，提供更具体的帮助
	if strings.Contains(message, "帮助") || strings.Contains(message, "怎么用") {
		response = p.getHelpMessage()
	}

	return &ai.ParseResult{
		Intent:     ai.IntentChat,
		Confidence: 0.5, // 兜底响应置信度较低
		ChatResponse: &ai.ChatInfo{
			Response:       response,
			NeedFollowUp:   false,
			FollowUpPrompt: "",
		},
		ParsedBy:    p.GetName(),
		ProcessTime: 0,
		Timestamp:   time.Now(),
	}, nil
}

// getHelpMessage 获取帮助消息
func (p *FallbackChatParser) getHelpMessage() string {
	return `MMemory 提醒助手使用指南：

📅 每日提醒：
• 每天早上8点提醒我喝水
• 每天9点30分提醒我吃药

📆 每周提醒：
• 每周一下午3点提醒我开会
• 工作日晚上8点提醒我复习英语

⏰ 一次性提醒：
• 明天下午2点提醒我取快递
• 2025年10月15日上午10点提醒我体检

如果有任何问题，请联系开发者。`
}

// GetName 实现Parser接口
func (p *FallbackChatParser) GetName() string {
	return "fallback-chat"
}

// GetPriority 实现Parser接口
func (p *FallbackChatParser) GetPriority() int {
	return ai.ParserTypeFallback.Priority()
}

// IsHealthy 实现Parser接口
func (p *FallbackChatParser) IsHealthy() bool {
	return true // 兜底解析器总是健康的
}
