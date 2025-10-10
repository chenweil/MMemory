package ai

import (
	"context"

	"mmemory/pkg/ai"
)

// Parser 统一的解析器接口
// 用于实现主AI、兜底AI、正则解析和兜底对话的统一抽象
type Parser interface {
	// Parse 解析用户消息
	Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error)

	// GetName 获取解析器名称
	GetName() string

	// GetPriority 获取解析器优先级（数字越小优先级越高）
	GetPriority() int

	// IsHealthy 检查解析器是否健康
	IsHealthy() bool
}

// ParserResult 解析器结果（用于记录降级过程）
type ParserResult struct {
	Result    *ai.ParseResult
	Parser    string
	Attempted []string // 尝试过的解析器列表
	Error     error
}
