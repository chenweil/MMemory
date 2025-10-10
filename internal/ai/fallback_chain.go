package ai

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
)

// FallbackChain 四层降级链
// 降级顺序: 主AI → 兜底AI → 正则 → 兜底对话
type FallbackChain struct {
	parsers []Parser
	mu      sync.RWMutex
	stats   *FallbackStats
}

// FallbackStats 降级统计信息
type FallbackStats struct {
	TotalRequests   int64
	SuccessByParser map[string]int64
	FailuresByParser map[string]int64
	mu              sync.RWMutex
}

// NewFallbackChain 创建降级链
func NewFallbackChain(parsers []Parser) *FallbackChain {
	// 按优先级排序（数字越小优先级越高）
	sortedParsers := make([]Parser, len(parsers))
	copy(sortedParsers, parsers)

	sort.Slice(sortedParsers, func(i, j int) bool {
		return sortedParsers[i].GetPriority() < sortedParsers[j].GetPriority()
	})

	return &FallbackChain{
		parsers: sortedParsers,
		stats: &FallbackStats{
			SuccessByParser:  make(map[string]int64),
			FailuresByParser: make(map[string]int64),
		},
	}
}

// Parse 执行降级解析
func (f *FallbackChain) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	f.stats.mu.Lock()
	f.stats.TotalRequests++
	f.stats.mu.Unlock()

	var lastErr error
	attemptedParsers := make([]string, 0, len(f.parsers))

	// 依次尝试每个解析器
	for _, parser := range f.parsers {
		parserName := parser.GetName()
		attemptedParsers = append(attemptedParsers, parserName)

		// 检查解析器是否健康
		if !parser.IsHealthy() {
			logger.Warnf("Parser %s is unhealthy, skipping", parserName)
			f.recordFailure(parserName)
			continue
		}

		logger.Infof("Attempting to parse with %s (priority: %d)", parserName, parser.GetPriority())

		// 设置超时
		parseCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// 执行解析
		start := time.Now()
		result, err := parser.Parse(parseCtx, userID, message)
		duration := time.Since(start)

		if err == nil && result != nil {
			// 解析成功
			logger.Infof("Parser %s succeeded in %v, confidence: %.2f", parserName, duration, result.Confidence)
			f.recordSuccess(parserName)

			// 记录尝试过的解析器
			result.ParsedBy = parserName
			return result, nil
		}

		// 解析失败，记录错误
		logger.Warnf("Parser %s failed in %v: %v", parserName, duration, err)
		f.recordFailure(parserName)
		lastErr = err
	}

	// 所有解析器都失败了
	logger.Errorf("All parsers failed. Attempted: %v", attemptedParsers)
	if lastErr != nil {
		return nil, fmt.Errorf("all parsers failed, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("all parsers failed with unknown errors")
}

// recordSuccess 记录成功
func (f *FallbackChain) recordSuccess(parserName string) {
	f.stats.mu.Lock()
	defer f.stats.mu.Unlock()
	f.stats.SuccessByParser[parserName]++
}

// recordFailure 记录失败
func (f *FallbackChain) recordFailure(parserName string) {
	f.stats.mu.Lock()
	defer f.stats.mu.Unlock()
	f.stats.FailuresByParser[parserName]++
}

// GetStats 获取统计信息
func (f *FallbackChain) GetStats() *FallbackStats {
	f.stats.mu.RLock()
	defer f.stats.mu.RUnlock()

	// 创建副本
	statsCopy := &FallbackStats{
		TotalRequests:    f.stats.TotalRequests,
		SuccessByParser:  make(map[string]int64),
		FailuresByParser: make(map[string]int64),
	}

	for k, v := range f.stats.SuccessByParser {
		statsCopy.SuccessByParser[k] = v
	}
	for k, v := range f.stats.FailuresByParser {
		statsCopy.FailuresByParser[k] = v
	}

	return statsCopy
}

// GetParsers 获取解析器列表
func (f *FallbackChain) GetParsers() []Parser {
	f.mu.RLock()
	defer f.mu.RUnlock()

	parsersCopy := make([]Parser, len(f.parsers))
	copy(parsersCopy, f.parsers)
	return parsersCopy
}

// AddParser 添加解析器
func (f *FallbackChain) AddParser(parser Parser) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.parsers = append(f.parsers, parser)

	// 重新排序
	sort.Slice(f.parsers, func(i, j int) bool {
		return f.parsers[i].GetPriority() < f.parsers[j].GetPriority()
	})

	logger.Infof("Added parser %s with priority %d", parser.GetName(), parser.GetPriority())
}

// RemoveParser 移除解析器
func (f *FallbackChain) RemoveParser(parserName string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i, parser := range f.parsers {
		if parser.GetName() == parserName {
			f.parsers = append(f.parsers[:i], f.parsers[i+1:]...)
			logger.Infof("Removed parser %s", parserName)
			return true
		}
	}

	return false
}

// GetSuccessRate 获取总体成功率
func (s *FallbackStats) GetSuccessRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalRequests == 0 {
		return 0.0
	}

	totalSuccess := int64(0)
	for _, count := range s.SuccessByParser {
		totalSuccess += count
	}

	return float64(totalSuccess) / float64(s.TotalRequests)
}

// GetParserSuccessRate 获取特定解析器的成功率
func (s *FallbackStats) GetParserSuccessRate(parserName string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	success := s.SuccessByParser[parserName]
	failure := s.FailuresByParser[parserName]
	total := success + failure

	if total == 0 {
		return 0.0
	}

	return float64(success) / float64(total)
}
