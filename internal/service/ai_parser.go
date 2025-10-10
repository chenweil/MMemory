package service

import (
	"context"
	"fmt"

	aiInternal "mmemory/internal/ai"
	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
)

// AIParserService AI解析服务接口
type AIParserService interface {
	// ParseMessage 智能解析 - 支持多种意图
	ParseMessage(ctx context.Context, userID string, message string) (*ai.ParseResult, error)

	// Chat 对话功能
	Chat(ctx context.Context, userID string, message string) (*ai.ChatResponse, error)

	// SetFallbackParser 设置降级解析器（已通过FallbackChain实现）
	SetFallbackParser(parser aiInternal.Parser) error

	// GetStats 获取降级统计信息
	GetStats() *aiInternal.FallbackStats
}

// aiParserService AI解析服务实现
type aiParserService struct {
	fallbackChain *aiInternal.FallbackChain
	primaryAI     aiInternal.Parser
	backupAI      aiInternal.Parser
}

// NewAIParserService 创建AI解析服务
func NewAIParserService(config *ai.AIConfig) (AIParserService, error) {
	if config == nil || !config.Enabled {
		return nil, fmt.Errorf("AI is not enabled")
	}

	var parsers []aiInternal.Parser

	// 1. 主AI解析器 (OpenAI Primary Model)
	primaryAI := aiInternal.NewOpenAIClient(config)
	if primaryAI != nil && primaryAI.IsHealthy() {
		parsers = append(parsers, primaryAI)
		logger.Info("Primary AI parser (OpenAI) initialized")
	} else {
		logger.Warn("Primary AI parser failed to initialize")
	}

	// 2. 兜底AI解析器 (OpenAI Backup Model)
	// 注意: 这里使用相同的OpenAI client，只是在调用时会切换到backup model
	// 实际上OpenAI客户端内部已经有主备切换逻辑
	var backupAI aiInternal.Parser
	if primaryAI != nil {
		backupAI = primaryAI // 复用同一个客户端
	}

	// 3. 传统正则解析器
	regexParser := aiInternal.NewRegexParser()
	parsers = append(parsers, regexParser)
	logger.Info("Regex parser initialized")

	// 4. 兜底对话生成器
	fallbackChatParser := aiInternal.NewFallbackChatParser()
	parsers = append(parsers, fallbackChatParser)
	logger.Info("Fallback chat parser initialized")

	// 创建降级链
	fallbackChain := aiInternal.NewFallbackChain(parsers)

	logger.Infof("AI Parser Service initialized with %d parsers", len(parsers))

	return &aiParserService{
		fallbackChain: fallbackChain,
		primaryAI:     primaryAI,
		backupAI:      backupAI,
	}, nil
}

// ParseMessage 实现AIParserService接口
func (s *aiParserService) ParseMessage(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	logger.Infof("Parsing message for user %s: %s", userID, message)

	// 使用降级链解析
	result, err := s.fallbackChain.Parse(ctx, userID, message)
	if err != nil {
		logger.Errorf("All parsers failed for message: %s, error: %v", message, err)
		return nil, err
	}

	logger.Infof("Message parsed successfully by %s, intent: %s, confidence: %.2f",
		result.ParsedBy, result.Intent, result.Confidence)

	return result, nil
}

// Chat 实现AIParserService接口
func (s *aiParserService) Chat(ctx context.Context, userID string, message string) (*ai.ChatResponse, error) {
	logger.Infof("Chat request for user %s: %s", userID, message)

	// 如果有主AI，优先使用AI对话
	if s.primaryAI != nil && s.primaryAI.IsHealthy() {
		if openaiClient, ok := s.primaryAI.(*aiInternal.OpenAIClient); ok {
			response, err := openaiClient.Chat(ctx, userID, message)
			if err == nil {
				return response, nil
			}
			logger.Warnf("Primary AI chat failed: %v", err)
		}
	}

	// 降级到简单回复
	return &ai.ChatResponse{
		Response:    "我现在无法进行对话，请稍后再试。",
		ParsedBy:    "fallback",
		ProcessTime: 0,
	}, nil
}

// SetFallbackParser 实现AIParserService接口
func (s *aiParserService) SetFallbackParser(parser aiInternal.Parser) error {
	if parser == nil {
		return fmt.Errorf("parser cannot be nil")
	}

	s.fallbackChain.AddParser(parser)
	logger.Infof("Added fallback parser: %s", parser.GetName())

	return nil
}

// GetStats 实现AIParserService接口
func (s *aiParserService) GetStats() *aiInternal.FallbackStats {
	return s.fallbackChain.GetStats()
}
