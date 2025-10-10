package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/pkg/ai"
)

// MockParser 模拟解析器
type MockParser struct {
	name       string
	priority   int
	shouldFail bool
	result     *ai.ParseResult
	isHealthy  bool
}

func NewMockParser(name string, priority int, shouldFail bool) *MockParser {
	return &MockParser{
		name:       name,
		priority:   priority,
		shouldFail: shouldFail,
		isHealthy:  true,
		result: &ai.ParseResult{
			Intent:     ai.IntentReminder,
			Confidence: 0.9,
			Reminder: &ai.ReminderInfo{
				Title: "test reminder",
			},
			ParsedBy:  name,
			Timestamp: time.Now(),
		},
	}
}

func (m *MockParser) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	if m.shouldFail {
		return nil, errors.New("mock parser error")
	}
	return m.result, nil
}

func (m *MockParser) GetName() string {
	return m.name
}

func (m *MockParser) GetPriority() int {
	return m.priority
}

func (m *MockParser) IsHealthy() bool {
	return m.isHealthy
}

// TestFallbackChain_BasicFlow 测试基本降级流程
func TestFallbackChain_BasicFlow(t *testing.T) {
	// 创建4个解析器: 主AI失败，兜底AI成功
	primaryAI := NewMockParser("primary-ai", 1, true)    // 失败
	backupAI := NewMockParser("backup-ai", 2, false)     // 成功
	regexParser := NewMockParser("regex", 3, false)      // 不会被调用
	fallbackChat := NewMockParser("fallback-chat", 4, false) // 不会被调用

	chain := NewFallbackChain([]Parser{
		primaryAI,
		backupAI,
		regexParser,
		fallbackChat,
	})

	ctx := context.Background()
	result, err := chain.Parse(ctx, "user1", "每天8点提醒我喝水")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backup-ai", result.ParsedBy)

	// 验证统计信息
	stats := chain.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.FailuresByParser["primary-ai"])
	assert.Equal(t, int64(1), stats.SuccessByParser["backup-ai"])
}

// TestFallbackChain_AllFail 测试所有解析器都失败
func TestFallbackChain_AllFail(t *testing.T) {
	primaryAI := NewMockParser("primary-ai", 1, true)
	backupAI := NewMockParser("backup-ai", 2, true)
	regexParser := NewMockParser("regex", 3, true)
	fallbackChat := NewMockParser("fallback-chat", 4, true)

	chain := NewFallbackChain([]Parser{
		primaryAI,
		backupAI,
		regexParser,
		fallbackChat,
	})

	ctx := context.Background()
	result, err := chain.Parse(ctx, "user1", "无法解析的消息")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "all parsers failed")

	// 验证所有解析器都被尝试了
	stats := chain.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.FailuresByParser["primary-ai"])
	assert.Equal(t, int64(1), stats.FailuresByParser["backup-ai"])
	assert.Equal(t, int64(1), stats.FailuresByParser["regex"])
	assert.Equal(t, int64(1), stats.FailuresByParser["fallback-chat"])
}

// TestFallbackChain_PriorityOrder 测试优先级排序
func TestFallbackChain_PriorityOrder(t *testing.T) {
	// 乱序创建解析器
	parser1 := NewMockParser("parser-low", 10, false)
	parser2 := NewMockParser("parser-high", 1, false)
	parser3 := NewMockParser("parser-mid", 5, false)

	chain := NewFallbackChain([]Parser{parser1, parser2, parser3})

	// 验证排序后的顺序
	parsers := chain.GetParsers()
	assert.Equal(t, "parser-high", parsers[0].GetName())
	assert.Equal(t, "parser-mid", parsers[1].GetName())
	assert.Equal(t, "parser-low", parsers[2].GetName())
}

// TestFallbackChain_SkipUnhealthyParser 测试跳过不健康的解析器
func TestFallbackChain_SkipUnhealthyParser(t *testing.T) {
	primaryAI := NewMockParser("primary-ai", 1, false)
	primaryAI.isHealthy = false // 设置为不健康

	backupAI := NewMockParser("backup-ai", 2, false)

	chain := NewFallbackChain([]Parser{primaryAI, backupAI})

	ctx := context.Background()
	result, err := chain.Parse(ctx, "user1", "test message")

	require.NoError(t, err)
	assert.Equal(t, "backup-ai", result.ParsedBy)

	// 验证不健康的解析器被记录为失败
	stats := chain.GetStats()
	assert.Equal(t, int64(1), stats.FailuresByParser["primary-ai"])
}

// TestFallbackChain_AddRemoveParser 测试动态添加/移除解析器
func TestFallbackChain_AddRemoveParser(t *testing.T) {
	parser1 := NewMockParser("parser1", 1, false)
	chain := NewFallbackChain([]Parser{parser1})

	// 添加解析器
	parser2 := NewMockParser("parser2", 2, false)
	chain.AddParser(parser2)

	parsers := chain.GetParsers()
	assert.Equal(t, 2, len(parsers))

	// 移除解析器
	removed := chain.RemoveParser("parser1")
	assert.True(t, removed)

	parsers = chain.GetParsers()
	assert.Equal(t, 1, len(parsers))
	assert.Equal(t, "parser2", parsers[0].GetName())

	// 移除不存在的解析器
	removed = chain.RemoveParser("non-existent")
	assert.False(t, removed)
}

// TestFallbackStats_SuccessRate 测试成功率计算
func TestFallbackStats_SuccessRate(t *testing.T) {
	parser := NewMockParser("test-parser", 1, false)
	chain := NewFallbackChain([]Parser{parser})

	ctx := context.Background()

	// 执行5次成功请求
	for i := 0; i < 5; i++ {
		_, _ = chain.Parse(ctx, "user1", "test")
	}

	// 执行2次失败请求
	parser.shouldFail = true
	for i := 0; i < 2; i++ {
		_, _ = chain.Parse(ctx, "user1", "test")
	}

	stats := chain.GetStats()
	assert.Equal(t, int64(7), stats.TotalRequests)

	// 总体成功率
	successRate := stats.GetSuccessRate()
	assert.InDelta(t, 5.0/7.0, successRate, 0.01)

	// 特定解析器成功率
	parserRate := stats.GetParserSuccessRate("test-parser")
	assert.InDelta(t, 5.0/7.0, parserRate, 0.01)
}
