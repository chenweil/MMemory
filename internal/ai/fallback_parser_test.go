package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/pkg/ai"
)

// TestNewFallbackChatParser 测试创建兜底解析器
func TestNewFallbackChatParser(t *testing.T) {
	parser := NewFallbackChatParser()

	require.NotNil(t, parser, "解析器不应为nil")
	assert.Len(t, parser.responses, 3, "应该有3个默认响应")

	// 验证响应内容包含关键元素
	for _, resp := range parser.responses {
		assert.NotEmpty(t, resp, "响应不应为空")
		assert.Contains(t, resp, "提醒", "响应应该包含'提醒'关键词")
	}
}

// TestFallbackChatParser_Parse 测试基本解析功能
func TestFallbackChatParser_Parse(t *testing.T) {
	parser := NewFallbackChatParser()
	ctx := context.Background()

	tests := []struct {
		name     string
		message  string
		wantErr  bool
		validate func(*testing.T, *ai.ParseResult)
	}{
		{
			name:    "短消息",
			message: "你好",
			wantErr: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.IntentChat, result.Intent)
				assert.Equal(t, float32(0.5), result.Confidence)
				assert.NotNil(t, result.ChatResponse)
				assert.NotEmpty(t, result.ChatResponse.Response)
				assert.False(t, result.ChatResponse.NeedFollowUp)
			},
		},
		{
			name:    "中等长度消息",
			message: "我想设置一个提醒但是不知道怎么说",
			wantErr: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.IntentChat, result.Intent)
				assert.Equal(t, float32(0.5), result.Confidence)
				assert.NotNil(t, result.ChatResponse)
				assert.NotEmpty(t, result.ChatResponse.Response)
			},
		},
		{
			name:    "较长消息",
			message: "这是一条很长很长的消息，用来测试兜底解析器对不同长度消息的响应处理机制",
			wantErr: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.IntentChat, result.Intent)
				assert.Equal(t, float32(0.5), result.Confidence)
				assert.NotNil(t, result.ChatResponse)
			},
		},
		{
			name:    "带前后空格的消息",
			message: "  hello world  ",
			wantErr: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.IntentChat, result.Intent)
				assert.NotNil(t, result.ChatResponse)
			},
		},
		{
			name:    "空消息（只有空格）",
			message: "   ",
			wantErr: false,
			validate: func(t *testing.T, result *ai.ParseResult) {
				assert.Equal(t, ai.IntentChat, result.Intent)
				assert.NotNil(t, result.ChatResponse)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(ctx, "test-user", tt.message)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// 通用验证
			assert.Equal(t, "fallback-chat", result.ParsedBy)
			assert.False(t, result.Timestamp.IsZero())

			// 特定验证
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestFallbackChatParser_Parse_HelpMessages 测试帮助消息触发
func TestFallbackChatParser_Parse_HelpMessages(t *testing.T) {
	parser := NewFallbackChatParser()
	ctx := context.Background()

	tests := []struct {
		name            string
		message         string
		expectHelpMsg   bool
		helpMsgContains []string
	}{
		{
			name:          "包含'帮助'关键词",
			message:       "帮助",
			expectHelpMsg: true,
			helpMsgContains: []string{
				"MMemory 提醒助手使用指南",
				"每日提醒",
				"每周提醒",
				"一次性提醒",
			},
		},
		{
			name:          "包含'怎么用'关键词",
			message:       "怎么用",
			expectHelpMsg: true,
			helpMsgContains: []string{
				"MMemory",
				"提醒助手",
				"使用指南",
			},
		},
		{
			name:          "同时包含'帮助'和其他内容",
			message:       "请给我一些帮助",
			expectHelpMsg: true,
			helpMsgContains: []string{
				"MMemory",
				"📅",
				"📆",
				"⏰",
			},
		},
		{
			name:          "不包含触发词",
			message:       "你好啊",
			expectHelpMsg: false,
			helpMsgContains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(ctx, "test-user", tt.message)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.ChatResponse)

			response := result.ChatResponse.Response

			if tt.expectHelpMsg {
				// 验证是帮助消息
				for _, keyword := range tt.helpMsgContains {
					assert.Contains(t, response, keyword, "帮助消息应该包含: "+keyword)
				}
			} else {
				// 验证不是帮助消息（应该是默认响应之一）
				isDefaultResponse := false
				for _, defaultResp := range parser.responses {
					if response == defaultResp {
						isDefaultResponse = true
						break
					}
				}
				assert.True(t, isDefaultResponse, "应该返回默认响应")
			}
		})
	}
}

// TestFallbackChatParser_getHelpMessage 测试帮助消息内容
func TestFallbackChatParser_getHelpMessage(t *testing.T) {
	parser := NewFallbackChatParser()
	helpMsg := parser.getHelpMessage()

	assert.NotEmpty(t, helpMsg)

	// 验证帮助消息包含必要的元素
	expectedElements := []string{
		"MMemory",
		"提醒助手",
		"使用指南",
		"每日提醒",
		"每周提醒",
		"一次性提醒",
		"每天早上8点提醒我喝水",
		"每周一下午3点提醒我开会",
		"明天下午2点提醒我取快递",
		"工作日晚上8点提醒我复习英语",
	}

	for _, element := range expectedElements {
		assert.Contains(t, helpMsg, element, "帮助消息应该包含: "+element)
	}

	// 验证帮助消息包含emoji
	assert.Contains(t, helpMsg, "📅")
	assert.Contains(t, helpMsg, "📆")
	assert.Contains(t, helpMsg, "⏰")
}

// TestFallbackChatParser_GetName 测试获取解析器名称
func TestFallbackChatParser_GetName(t *testing.T) {
	parser := NewFallbackChatParser()
	name := parser.GetName()

	assert.Equal(t, "fallback-chat", name)
}

// TestFallbackChatParser_GetPriority 测试获取优先级
func TestFallbackChatParser_GetPriority(t *testing.T) {
	parser := NewFallbackChatParser()
	priority := parser.GetPriority()

	// 兜底解析器应该是最低优先级
	assert.Equal(t, ai.ParserTypeFallback.Priority(), priority)
	assert.Equal(t, 4, priority, "兜底解析器优先级应该是4（最低）")
}

// TestFallbackChatParser_IsHealthy 测试健康检查
func TestFallbackChatParser_IsHealthy(t *testing.T) {
	parser := NewFallbackChatParser()
	isHealthy := parser.IsHealthy()

	// 兜底解析器总是健康的
	assert.True(t, isHealthy, "兜底解析器应该总是健康的")
}

// TestFallbackChatParser_ResponseRotation 测试响应轮换
func TestFallbackChatParser_ResponseRotation(t *testing.T) {
	parser := NewFallbackChatParser()
	ctx := context.Background()

	// 创建不同长度的消息，触发不同的响应
	messages := []string{
		"a",       // 长度1 -> 1 % 3 = 1
		"ab",      // 长度2 -> 2 % 3 = 2
		"abc",     // 长度3 -> 3 % 3 = 0
		"abcd",    // 长度4 -> 4 % 3 = 1
		"abcde",   // 长度5 -> 5 % 3 = 2
		"abcdef",  // 长度6 -> 6 % 3 = 0
	}

	responses := make(map[string]int)

	for _, msg := range messages {
		result, err := parser.Parse(ctx, "test-user", msg)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ChatResponse)

		response := result.ChatResponse.Response
		responses[response]++
	}

	// 应该使用了所有3个默认响应
	assert.Equal(t, 3, len(responses), "应该使用了所有3个不同的响应")

	// 每个响应应该被使用了2次
	for response, count := range responses {
		assert.Equal(t, 2, count, "响应 '%s...' 应该被使用2次", response[:20])
	}
}

// TestFallbackChatParser_ContextCancellation 测试上下文取消
func TestFallbackChatParser_ContextCancellation(t *testing.T) {
	parser := NewFallbackChatParser()

	// 创建一个已经取消的context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// 即使context被取消，兜底解析器也应该能正常返回
	// （因为它不做任何耗时操作）
	result, err := parser.Parse(ctx, "test-user", "测试消息")

	// 应该成功返回（兜底解析器不检查context）
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestFallbackChatParser_DifferentUserIDs 测试不同用户ID
func TestFallbackChatParser_DifferentUserIDs(t *testing.T) {
	parser := NewFallbackChatParser()
	ctx := context.Background()

	userIDs := []string{"user1", "user2", "user3", ""}

	for _, userID := range userIDs {
		t.Run("UserID_"+userID, func(t *testing.T) {
			result, err := parser.Parse(ctx, userID, "测试消息")

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, ai.IntentChat, result.Intent)
		})
	}
}

// TestFallbackChatParser_SpecialCharacters 测试特殊字符
func TestFallbackChatParser_SpecialCharacters(t *testing.T) {
	parser := NewFallbackChatParser()
	ctx := context.Background()

	specialMessages := []string{
		"@#$%^&*()",
		"😀😃😄😁",
		"こんにちは",
		"Привет",
		"<script>alert('xss')</script>",
		"SELECT * FROM users;",
		strings.Repeat("a", 1000), // 很长的消息
	}

	for _, msg := range specialMessages {
		t.Run("Message_"+msg[:min(len(msg), 20)], func(t *testing.T) {
			result, err := parser.Parse(ctx, "test-user", msg)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, ai.IntentChat, result.Intent)
			assert.NotNil(t, result.ChatResponse)
		})
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
