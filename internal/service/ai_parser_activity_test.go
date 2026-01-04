package service

import (
	"context"
	"fmt"
	"testing"

	aiInternal "mmemory/internal/ai"
	"mmemory/pkg/ai"
)

// MockAIClient 用于测试
type MockAIClient struct {
	aiInternal.OpenAIClient // 嵌入真实的OpenAIClient以继承接口
	response                string
	err                     error
}

func (m *MockAIClient) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	return nil, nil
}

func (m *MockAIClient) ParseWithContext(ctx context.Context, userID string, message string, conversationHistory string) (*ai.ParseResult, error) {
	return nil, nil
}

func (m *MockAIClient) GetName() string {
	return "mock"
}

func (m *MockAIClient) GetPriority() int {
	return 1
}

func (m *MockAIClient) IsHealthy() bool {
	return true
}

// GenerateActivityReply 重写此方法以返回mock响应
func (m *MockAIClient) GenerateActivityReply(ctx context.Context, userMessage string, activityType string, details map[string]interface{}) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestGenerateActivityReply(t *testing.T) {
	tests := []struct {
		name         string
		userMessage  string
		activityType string
		details      map[string]interface{}
		mockResponse string
		mockError    error
		wantErr      bool
	}{
		{
			name:         "看书活动-成功",
			userMessage:  "我在看《时间简史》第二章",
			activityType: "read_book",
			details: map[string]interface{}{
				"book_name": "时间简史",
				"chapter":   "第二章",
			},
			mockResponse: "《时间简史》是霍金的经典科普著作。✅ 已记录:看书",
			mockError:    nil,
			wantErr:      false,
		},
		{
			name:         "喝水活动-成功",
			userMessage:  "我刚才喝了杯水",
			activityType: "drink_water",
			details: map[string]interface{}{
				"amount": "1杯",
			},
			mockResponse: "很好!保持适量饮水。✅ 已记录:喝水",
			mockError:    nil,
			wantErr:      false,
		},
		{
			name:         "AI服务失败-返回错误",
			userMessage:  "我在看书",
			activityType: "read_book",
			details:      map[string]interface{}{},
			mockResponse: "",
			mockError:    fmt.Errorf("AI service unavailable"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建mock AI客户端
			mockAI := &MockAIClient{
				response: tt.mockResponse,
				err:      tt.mockError,
			}

			// 创建fallback chain
			var parsers []aiInternal.Parser
			fallbackChain := aiInternal.NewFallbackChain(parsers)

			// 创建服务实例
			service := &aiParserService{
				fallbackChain: fallbackChain,
				primaryAI:     mockAI, // 使用mock替代真实的primaryAI
				backupAI:      nil,
			}

			// 调用方法
			reply, err := service.GenerateActivityReply(
				context.Background(),
				"123",
				tt.userMessage,
				tt.activityType,
				tt.details,
			)

			// 验证结果
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateActivityReply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && reply != tt.mockResponse {
				t.Errorf("GenerateActivityReply() = %v, want %v", reply, tt.mockResponse)
			}
		})
	}
}

// TestGenerateActivityReplyWithRealOpenAI 使用真实OpenAI客户端的集成测试
// 注意：此测试需要有效的API Key，在CI/CD中应跳过
func TestGenerateActivityReplyWithRealOpenAI(t *testing.T) {
	// 跳过此测试，除非显式启用
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// 此测试需要真实的环境变量设置
	// 在实际运行时，需要设置 MMEMORY_AI_OPENAI_API_KEY
	t.Skip("requres real API key, skipping in unit tests")
}
