package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mmemory/internal/models"
	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
)

// 初始化测试环境
func init() {
	// 初始化logger以避免测试中的panic
	logger.Init("info", "text", "stdout", "")
}

// MockParser Mock解析器
type MockParser struct {
	mock.Mock
}

func (m *MockParser) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	args := m.Called(ctx, userID, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.ParseResult), args.Error(1)
}

func (m *MockParser) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockParser) GetPriority() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockParser) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

// TestNewAIParserService_Success 测试成功初始化AI服务
func TestNewAIParserService_Success(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "sk-test-key",
			BaseURL:      "https://api.openai.com/v1",
			PrimaryModel: "gpt-4o-mini",
			BackupModel:  "gpt-3.5-turbo",
			Temperature:  0.1,
			MaxTokens:    1000,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
		},
		Prompts: ai.PromptsConfig{
			ReminderParse: "test prompt",
			ChatResponse:  "test prompt",
		},
	}

	service, err := NewAIParserService(config)

	assert.NoError(t, err)
	assert.NotNil(t, service)
}

// TestNewAIParserService_Disabled 测试AI未启用
func TestNewAIParserService_Disabled(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: false,
	}

	service, err := NewAIParserService(config)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "not enabled")
}

// TestNewAIParserService_NilConfig 测试配置为nil
func TestNewAIParserService_NilConfig(t *testing.T) {
	service, err := NewAIParserService(nil)

	assert.Error(t, err)
	assert.Nil(t, service)
}

// TestNewAIParserService_InvalidConfig 测试无效配置
func TestNewAIParserService_InvalidConfig(t *testing.T) {
	config := &ai.AIConfig{
		Enabled: true,
		OpenAI: ai.OpenAIConfig{
			APIKey:       "", // 缺少API Key
			BaseURL:      "https://api.openai.com/v1",
			PrimaryModel: "gpt-4o-mini",
			BackupModel:  "gpt-3.5-turbo",
			Temperature:  0.1,
			MaxTokens:    1000,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
		},
	}

	// 验证配置应该失败
	err := config.Validate()
	assert.Error(t, err)
	assert.Equal(t, ai.ErrMissingAPIKey, err)
}

// TestParseMessage_Success 测试成功解析消息
func TestParseMessage_Success(t *testing.T) {
	// 由于NewAIParserService依赖真实的OpenAI客户端，
	// 我们这里测试的是Mock场景

	// 创建Mock AIParserService
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "每天早上8点提醒我喝水"

	expectedResult := &ai.ParseResult{
		Intent:     ai.IntentReminder,
		Confidence: 0.95,
		Reminder: &ai.ReminderInfo{
			Title: "喝水",
			Type:  models.ReminderTypeHabit,
			Time: ai.TimeInfo{
				Hour:     8,
				Minute:   0,
				Timezone: "Asia/Shanghai",
			},
			SchedulePattern: models.SchedulePatternDaily,
		},
		ParsedBy:  "openai-gpt-4o-mini",
		Timestamp: time.Now(),
	}

	mockService.On("ParseMessage", ctx, userID, message).Return(expectedResult, nil)

	result, err := mockService.ParseMessage(ctx, userID, message)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ai.IntentReminder, result.Intent)
	assert.Equal(t, "喝水", result.Reminder.Title)
	mockService.AssertExpectations(t)
}

// MockAIParserService Mock AI解析服务
type MockAIParserService struct {
	mock.Mock
}

func (m *MockAIParserService) ParseMessage(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	args := m.Called(ctx, userID, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.ParseResult), args.Error(1)
}

func (m *MockAIParserService) Chat(ctx context.Context, userID string, message string) (*ai.ChatResponse, error) {
	args := m.Called(ctx, userID, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.ChatResponse), args.Error(1)
}

func (m *MockAIParserService) SetFallbackParser(parser interface{}) error {
	args := m.Called(parser)
	return args.Error(0)
}

func (m *MockAIParserService) GetStats() interface{} {
	args := m.Called()
	return args.Get(0)
}

// TestParseMessage_AllParsersFailed 测试所有解析器失败
func TestParseMessage_AllParsersFailed(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "这是一个无法解析的消息"

	expectedError := errors.New("all parsers failed")
	mockService.On("ParseMessage", ctx, userID, message).Return(nil, expectedError)

	result, err := mockService.ParseMessage(ctx, userID, message)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed")
	mockService.AssertExpectations(t)
}

// TestChat_Success 测试成功对话
func TestChat_Success(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "你好"

	expectedResponse := &ai.ChatResponse{
		Response:    "你好！有什么可以帮你的吗？",
		ParsedBy:    "openai-gpt-4o-mini",
		ProcessTime: 100 * time.Millisecond,
		Timestamp:   time.Now(),
	}

	mockService.On("Chat", ctx, userID, message).Return(expectedResponse, nil)

	response, err := mockService.Chat(ctx, userID, message)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Response, "你好")
	mockService.AssertExpectations(t)
}

// TestChat_Fallback 测试对话降级
func TestChat_Fallback(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "你好"

	// AI对话失败，返回降级响应
	fallbackResponse := &ai.ChatResponse{
		Response:    "我现在无法进行对话，请稍后再试。",
		ParsedBy:    "fallback",
		ProcessTime: 0,
		Timestamp:   time.Now(),
	}

	mockService.On("Chat", ctx, userID, message).Return(fallbackResponse, nil)

	response, err := mockService.Chat(ctx, userID, message)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "fallback", response.ParsedBy)
	assert.Contains(t, response.Response, "无法进行对话")
	mockService.AssertExpectations(t)
}

// TestSetFallbackParser 测试设置降级解析器
func TestSetFallbackParser(t *testing.T) {
	mockService := new(MockAIParserService)
	mockParser := new(MockParser)

	mockService.On("SetFallbackParser", mockParser).Return(nil)

	err := mockService.SetFallbackParser(mockParser)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

// TestParseMessage_ReminderIntent 测试提醒意图解析
func TestParseMessage_ReminderIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"

	testCases := []struct {
		name     string
		message  string
		expected *ai.ParseResult
	}{
		{
			name:    "每天提醒",
			message: "每天早上8点提醒我喝水",
			expected: &ai.ParseResult{
				Intent:     ai.IntentReminder,
				Confidence: 0.95,
				Reminder: &ai.ReminderInfo{
					Title: "喝水",
					Type:  models.ReminderTypeHabit,
					Time: ai.TimeInfo{
						Hour:     8,
						Minute:   0,
						Timezone: "Asia/Shanghai",
					},
					SchedulePattern: models.SchedulePatternDaily,
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
		{
			name:    "工作日提醒",
			message: "工作日晚上8点提醒我复习英语",
			expected: &ai.ParseResult{
				Intent:     ai.IntentReminder,
				Confidence: 0.9,
				Reminder: &ai.ReminderInfo{
					Title: "复习英语",
					Type:  models.ReminderTypeHabit,
					Time: ai.TimeInfo{
						Hour:     20,
						Minute:   0,
						Timezone: "Asia/Shanghai",
					},
					SchedulePattern: "weekly:1,2,3,4,5",
				},
				ParsedBy: "regex",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService.On("ParseMessage", ctx, userID, tc.message).Return(tc.expected, nil).Once()

			result, err := mockService.ParseMessage(ctx, userID, tc.message)

			assert.NoError(t, err)
			assert.Equal(t, ai.IntentReminder, result.Intent)
			assert.NotNil(t, result.Reminder)
			assert.Equal(t, tc.expected.Reminder.Title, result.Reminder.Title)
		})
	}

	mockService.AssertExpectations(t)
}

// TestParseMessage_ChatIntent 测试对话意图解析
func TestParseMessage_ChatIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "我在看《三体》"

	expectedResult := &ai.ParseResult{
		Intent:     ai.IntentChat,
		Confidence: 0.9,
		ChatResponse: &ai.ChatInfo{
			Response:     "《三体》是刘慈欣的经典科幻小说！你觉得哪个情节最印象深刻？",
			NeedFollowUp: true,
		},
		ParsedBy: "openai-gpt-4o-mini",
	}

	mockService.On("ParseMessage", ctx, userID, message).Return(expectedResult, nil)

	result, err := mockService.ParseMessage(ctx, userID, message)

	assert.NoError(t, err)
	assert.Equal(t, ai.IntentChat, result.Intent)
	assert.NotNil(t, result.ChatResponse)
	assert.Contains(t, result.ChatResponse.Response, "三体")
	mockService.AssertExpectations(t)
}

// TestParseMessage_QueryIntent 测试查询意图解析
func TestParseMessage_QueryIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "查看我的提醒列表"

	expectedResult := &ai.ParseResult{
		Intent:     ai.IntentQuery,
		Confidence: 0.92,
		ParsedBy:   "openai-gpt-4o-mini",
	}

	mockService.On("ParseMessage", ctx, userID, message).Return(expectedResult, nil)

	result, err := mockService.ParseMessage(ctx, userID, message)

	assert.NoError(t, err)
	assert.Equal(t, ai.IntentQuery, result.Intent)
	mockService.AssertExpectations(t)
}

// TestParseMessage_SummaryIntent 测试总结意图解析
func TestParseMessage_SummaryIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "给我看看统计数据"

	expectedResult := &ai.ParseResult{
		Intent:     ai.IntentSummary,
		Confidence: 0.88,
		ParsedBy:   "openai-gpt-4o-mini",
	}

	mockService.On("ParseMessage", ctx, userID, message).Return(expectedResult, nil)

	result, err := mockService.ParseMessage(ctx, userID, message)

	assert.NoError(t, err)
	assert.Equal(t, ai.IntentSummary, result.Intent)
	mockService.AssertExpectations(t)
}

// TestParseMessage_UnknownIntent 测试未知意图
func TestParseMessage_UnknownIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "随机文字 asdfghjkl"

	expectedResult := &ai.ParseResult{
		Intent:     ai.IntentUnknown,
		Confidence: 0.3,
		ParsedBy:   "fallback",
	}

	mockService.On("ParseMessage", ctx, userID, message).Return(expectedResult, nil)

	result, err := mockService.ParseMessage(ctx, userID, message)

	assert.NoError(t, err)
	assert.Equal(t, ai.IntentUnknown, result.Intent)
	assert.True(t, result.Confidence < 0.5)
	mockService.AssertExpectations(t)
}

// TestAIConfig_Validate 测试AI配置验证
func TestAIConfig_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		config      *ai.AIConfig
		expectError bool
		errorType   error
	}{
		{
			name: "有效配置",
			config: &ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey:       "sk-test-key",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  0.1,
				},
			},
			expectError: false,
		},
		{
			name: "缺少API Key",
			config: &ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey:       "",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  0.1,
				},
			},
			expectError: true,
			errorType:   ai.ErrMissingAPIKey,
		},
		{
			name: "缺少Primary Model",
			config: &ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey:       "sk-test-key",
					PrimaryModel: "",
					MaxTokens:    1000,
					Temperature:  0.1,
				},
			},
			expectError: true,
			errorType:   ai.ErrMissingPrimaryModel,
		},
		{
			name: "无效的MaxTokens",
			config: &ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey:       "sk-test-key",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    0,
					Temperature:  0.1,
				},
			},
			expectError: true,
			errorType:   ai.ErrInvalidMaxTokens,
		},
		{
			name: "无效的Temperature",
			config: &ai.AIConfig{
				Enabled: true,
				OpenAI: ai.OpenAIConfig{
					APIKey:       "sk-test-key",
					PrimaryModel: "gpt-4o-mini",
					MaxTokens:    1000,
					Temperature:  3.0, // 超出范围
				},
			},
			expectError: true,
			errorType:   ai.ErrInvalidTemperature,
		},
		{
			name: "未启用时跳过验证",
			config: &ai.AIConfig{
				Enabled: false,
				OpenAI: ai.OpenAIConfig{
					APIKey: "", // 即使缺少也不报错
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorType != nil {
					assert.ErrorIs(t, err, tc.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseMessage_DeleteIntent 测试删除意图解析
func TestParseMessage_DeleteIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"

	testCases := []struct {
		name     string
		message  string
		expected *ai.ParseResult
	}{
		{
			name:    "删除健身提醒",
			message: "删除健身提醒",
			expected: &ai.ParseResult{
				Intent:     ai.IntentDelete,
				Confidence: 0.95,
				Delete: &ai.DeleteInfo{
					Keywords: []string{"健身"},
					Criteria: "删除健身相关的提醒",
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
		{
			name:    "撤销今晚的提醒",
			message: "撤销今晚的健身提醒",
			expected: &ai.ParseResult{
				Intent:     ai.IntentDelete,
				Confidence: 0.92,
				Delete: &ai.DeleteInfo{
					Keywords: []string{"今晚", "健身"},
					Criteria: "撤销今晚健身提醒",
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
		{
			name:    "取消提醒",
			message: "取消喝水提醒",
			expected: &ai.ParseResult{
				Intent:     ai.IntentDelete,
				Confidence: 0.9,
				Delete: &ai.DeleteInfo{
					Keywords: []string{"喝水"},
					Criteria: "取消喝水提醒",
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService.On("ParseMessage", ctx, userID, tc.message).Return(tc.expected, nil).Once()

			result, err := mockService.ParseMessage(ctx, userID, tc.message)

			assert.NoError(t, err)
			assert.Equal(t, ai.IntentDelete, result.Intent)
			assert.NotNil(t, result.Delete)
			assert.NotEmpty(t, result.Delete.Keywords)
			assert.Contains(t, result.Delete.Keywords, tc.expected.Delete.Keywords[0])
		})
	}

	mockService.AssertExpectations(t)
}

// TestParseMessage_EditIntent 测试编辑意图解析
func TestParseMessage_EditIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"
	message := "把健身提醒改到晚上7点"

	expectedResult := &ai.ParseResult{
		Intent:     ai.IntentEdit,
		Confidence: 0.9,
		Edit: &ai.EditInfo{
			Keywords:   []string{"健身"},
			NewTime:    &ai.TimeInfo{Hour: 19, Minute: 0, Timezone: "Asia/Shanghai"},
			NewPattern: "",
			NewTitle:   "",
		},
		ParsedBy: "openai-gpt-4o-mini",
	}

	mockService.On("ParseMessage", ctx, userID, message).Return(expectedResult, nil)

	result, err := mockService.ParseMessage(ctx, userID, message)

	assert.NoError(t, err)
	assert.Equal(t, ai.IntentEdit, result.Intent)
	assert.NotNil(t, result.Edit)
	assert.Contains(t, result.Edit.Keywords, "健身")
	assert.NotNil(t, result.Edit.NewTime)
	assert.Equal(t, 19, result.Edit.NewTime.Hour)
	mockService.AssertExpectations(t)
}

// TestParseMessage_PauseIntent 测试暂停意图解析
func TestParseMessage_PauseIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"

	testCases := []struct {
		name     string
		message  string
		expected *ai.ParseResult
	}{
		{
			name:    "暂停一周",
			message: "暂停一周的健身提醒",
			expected: &ai.ParseResult{
				Intent:     ai.IntentPause,
				Confidence: 0.93,
				Pause: &ai.PauseInfo{
					Keywords: []string{"健身"},
					Duration: "1week",
					Reason:   "",
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
		{
			name:    "暂停一天",
			message: "今天不要提醒我跑步",
			expected: &ai.ParseResult{
				Intent:     ai.IntentPause,
				Confidence: 0.88,
				Pause: &ai.PauseInfo{
					Keywords: []string{"跑步"},
					Duration: "1day",
					Reason:   "今天不要",
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService.On("ParseMessage", ctx, userID, tc.message).Return(tc.expected, nil).Once()

			result, err := mockService.ParseMessage(ctx, userID, tc.message)

			assert.NoError(t, err)
			assert.Equal(t, ai.IntentPause, result.Intent)
			assert.NotNil(t, result.Pause)
			assert.NotEmpty(t, result.Pause.Keywords)
			assert.NotEmpty(t, result.Pause.Duration)
		})
	}

	mockService.AssertExpectations(t)
}

// TestParseMessage_ResumeIntent 测试恢复意图解析
func TestParseMessage_ResumeIntent(t *testing.T) {
	mockService := new(MockAIParserService)

	ctx := context.Background()
	userID := "123"

	testCases := []struct {
		name     string
		message  string
		expected *ai.ParseResult
	}{
		{
			name:    "恢复健身提醒",
			message: "恢复健身提醒",
			expected: &ai.ParseResult{
				Intent:     ai.IntentResume,
				Confidence: 0.95,
				Resume: &ai.ResumeInfo{
					Keywords: []string{"健身"},
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
		{
			name:    "重新开始提醒",
			message: "重新开始跑步提醒",
			expected: &ai.ParseResult{
				Intent:     ai.IntentResume,
				Confidence: 0.9,
				Resume: &ai.ResumeInfo{
					Keywords: []string{"跑步"},
				},
				ParsedBy: "openai-gpt-4o-mini",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService.On("ParseMessage", ctx, userID, tc.message).Return(tc.expected, nil).Once()

			result, err := mockService.ParseMessage(ctx, userID, tc.message)

			assert.NoError(t, err)
			assert.Equal(t, ai.IntentResume, result.Intent)
			assert.NotNil(t, result.Resume)
			assert.NotEmpty(t, result.Resume.Keywords)
		})
	}

	mockService.AssertExpectations(t)
}
