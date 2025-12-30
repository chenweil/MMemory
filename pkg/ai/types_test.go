package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mmemory/internal/models"
)

func TestParseResultValidate_NewIntents(t *testing.T) {
	tests := []struct {
		name       string
		result     *ParseResult
		wantValid  bool
		wantErrors []string
	}{
		{
			name: "delete intent with keywords",
			result: &ParseResult{
				Intent:     IntentDelete,
				Confidence: 0.9,
				Delete: &DeleteInfo{
					Keywords: []string{"健身", "晚上"},
				},
			},
			wantValid: true,
		},
		{
			name: "delete intent missing details",
			result: &ParseResult{
				Intent:     IntentDelete,
				Confidence: 0.9,
				Delete: &DeleteInfo{
					Keywords: []string{},
					Criteria: "",
				},
			},
			wantValid:  false,
			wantErrors: []string{"delete keywords or criteria required"},
		},
		{
			name: "edit intent requires update fields",
			result: &ParseResult{
				Intent:     IntentEdit,
				Confidence: 0.9,
				Edit: &EditInfo{
					Keywords: []string{"健身"},
				},
			},
			wantValid:  false,
			wantErrors: []string{"edit requires at least one field to update"},
		},
		{
			name: "pause intent valid",
			result: &ParseResult{
				Intent:     IntentPause,
				Confidence: 0.8,
				Pause: &PauseInfo{
					Keywords: []string{"健身"},
					Duration: "P1W",
				},
			},
			wantValid: true,
		},
		{
			name: "resume intent missing keywords",
			result: &ParseResult{
				Intent:     IntentResume,
				Confidence: 0.8,
				Resume: &ResumeInfo{
					Keywords: []string{""},
				},
			},
			wantValid:  false,
			wantErrors: []string{"resume keywords required"},
		},
		{
			name: "reminder intent still valid",
			result: &ParseResult{
				Intent:     IntentReminder,
				Confidence: 0.95,
				Reminder: &ReminderInfo{
					Title: "喝水",
					Type:  models.ReminderTypeHabit,
					Time: TimeInfo{
						Hour:     8,
						Minute:   0,
						Timezone: "Asia/Shanghai",
					},
					SchedulePattern: models.SchedulePatternDaily,
				},
				Timestamp: time.Now(),
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.Validate()
			if result.IsValid != tt.wantValid {
				t.Fatalf("Validate().IsValid = %v, want %v (errors: %v)", result.IsValid, tt.wantValid, result.Errors)
			}
			if !tt.wantValid {
				for _, want := range tt.wantErrors {
					found := false
					for _, got := range result.Errors {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected error %q not found in %v", want, result.Errors)
					}
				}
			}
		})
	}
}

// TestParseIntent_String 测试Intent类型的String方法
func TestParseIntent_String(t *testing.T) {
	tests := []struct {
		intent   ParseIntent
		expected string
	}{
		{IntentReminder, "reminder"},
		{IntentChat, "chat"},
		{IntentQuery, "query"},
		{IntentSummary, "summary"},
		{IntentDelete, "delete"},
		{IntentEdit, "edit"},
		{IntentPause, "pause"},
		{IntentResume, "resume"},
		{IntentUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.intent), func(t *testing.T) {
			result := tt.intent.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParserType_String 测试ParserType的String方法
func TestParserType_String(t *testing.T) {
	tests := []struct {
		parserType ParserType
		expected   string
	}{
		{ParserTypePrimaryAI, "primary_ai"},
		{ParserTypeBackupAI, "backup_ai"},
		{ParserTypeRegex, "regex"},
		{ParserTypeFallback, "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.parserType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParserType_Priority 测试ParserType的Priority方法
func TestParserType_Priority(t *testing.T) {
	tests := []struct {
		parserType       ParserType
		expectedPriority int
	}{
		{ParserTypePrimaryAI, 1},
		{ParserTypeBackupAI, 2},
		{ParserTypeRegex, 3},
		{ParserTypeFallback, 4},
	}

	for _, tt := range tests {
		t.Run(tt.parserType.String(), func(t *testing.T) {
			priority := tt.parserType.Priority()
			assert.Equal(t, tt.expectedPriority, priority)
		})
	}

	// 验证优先级顺序（数字越小优先级越高）
	assert.Less(t, ParserTypePrimaryAI.Priority(), ParserTypeBackupAI.Priority(), "Primary AI应该比Backup AI优先级高")
	assert.Less(t, ParserTypeBackupAI.Priority(), ParserTypeRegex.Priority(), "Backup AI应该比Regex优先级高")
	assert.Less(t, ParserTypeRegex.Priority(), ParserTypeFallback.Priority(), "Regex应该比Fallback优先级高")
}

// TestParseResult_IsHighConfidence 测试高置信度判断
func TestParseResult_IsHighConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float32
		expected   bool
	}{
		{"置信度0.9", 0.9, true},
		{"置信度0.8", 0.8, true},
		{"置信度0.79", 0.79, false},
		{"置信度0.7", 0.7, false},
		{"置信度1.0", 1.0, true},
		{"置信度0.0", 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ParseResult{
				Intent:     IntentReminder,
				Confidence: tt.confidence,
			}
			assert.Equal(t, tt.expected, result.IsHighConfidence())
		})
	}
}

// TestParseResult_IsMediumConfidence 测试中等置信度判断
func TestParseResult_IsMediumConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float32
		expected   bool
	}{
		{"置信度0.7", 0.7, true},
		{"置信度0.6", 0.6, true},
		{"置信度0.5", 0.5, true},
		{"置信度0.79", 0.79, true},
		{"置信度0.8", 0.8, false},  // 高置信度
		{"置信度0.49", 0.49, false}, // 低置信度
		{"置信度0.0", 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ParseResult{
				Intent:     IntentReminder,
				Confidence: tt.confidence,
			}
			assert.Equal(t, tt.expected, result.IsMediumConfidence())
		})
	}
}

// TestParseResult_IsLowConfidence 测试低置信度判断
func TestParseResult_IsLowConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float32
		expected   bool
	}{
		{"置信度0.4", 0.4, true},
		{"置信度0.3", 0.3, true},
		{"置信度0.1", 0.1, true},
		{"置信度0.0", 0.0, true},
		{"置信度0.49", 0.49, true},
		{"置信度0.5", 0.5, false},  // 中等置信度
		{"置信度0.8", 0.8, false},  // 高置信度
		{"置信度1.0", 1.0, false},  // 高置信度
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ParseResult{
				Intent:     IntentReminder,
				Confidence: tt.confidence,
			}
			assert.Equal(t, tt.expected, result.IsLowConfidence())
		})
	}
}

// TestConfidenceThresholds 测试置信度阈值边界
func TestConfidenceThresholds(t *testing.T) {
	// 测试边界值
	highThreshold := &ParseResult{Intent: IntentChat, Confidence: 0.8}
	assert.True(t, highThreshold.IsHighConfidence())
	assert.False(t, highThreshold.IsMediumConfidence())
	assert.False(t, highThreshold.IsLowConfidence())

	mediumLowBoundary := &ParseResult{Intent: IntentChat, Confidence: 0.5}
	assert.False(t, mediumLowBoundary.IsHighConfidence())
	assert.True(t, mediumLowBoundary.IsMediumConfidence())
	assert.False(t, mediumLowBoundary.IsLowConfidence())

	mediumHighBoundary := &ParseResult{Intent: IntentChat, Confidence: 0.79}
	assert.False(t, mediumHighBoundary.IsHighConfidence())
	assert.True(t, mediumHighBoundary.IsMediumConfidence())
	assert.False(t, mediumHighBoundary.IsLowConfidence())

	lowBoundary := &ParseResult{Intent: IntentChat, Confidence: 0.49}
	assert.False(t, lowBoundary.IsHighConfidence())
	assert.False(t, lowBoundary.IsMediumConfidence())
	assert.True(t, lowBoundary.IsLowConfidence())
}

// TestParseIntent_IsValid 测试Intent类型有效性
func TestParseIntent_IsValid(t *testing.T) {
	validIntents := []ParseIntent{
		IntentReminder,
		IntentChat,
		IntentQuery,
		IntentSummary,
		IntentDelete,
		IntentEdit,
		IntentPause,
		IntentResume,
		IntentWeather,
		IntentUnknown,
	}

	for _, intent := range validIntents {
		t.Run(string(intent), func(t *testing.T) {
			assert.True(t, intent.IsValid(), "Intent %s应该是有效的", intent)
		})
	}

	// 测试无效的Intent
	invalidIntent := ParseIntent("invalid_intent")
	assert.False(t, invalidIntent.IsValid(), "无效的Intent应该返回false")
}

// TestParserTypePriorities 测试解析器优先级的合理性
func TestParserTypePriorities(t *testing.T) {
	// 所有优先级应该大于0
	assert.Greater(t, ParserTypePrimaryAI.Priority(), 0)
	assert.Greater(t, ParserTypeBackupAI.Priority(), 0)
	assert.Greater(t, ParserTypeRegex.Priority(), 0)
	assert.Greater(t, ParserTypeFallback.Priority(), 0)

	// 所有优先级应该不同
	priorities := []int{
		ParserTypePrimaryAI.Priority(),
		ParserTypeBackupAI.Priority(),
		ParserTypeRegex.Priority(),
		ParserTypeFallback.Priority(),
	}

	seen := make(map[int]bool)
	for _, p := range priorities {
		assert.False(t, seen[p], "优先级%d重复", p)
		seen[p] = true
	}
}

// TestTimeInfo 测试TimeInfo结构
func TestTimeInfo(t *testing.T) {
	timeInfo := TimeInfo{
		Hour:     14,
		Minute:   30,
		Timezone: "Asia/Shanghai",
	}

	assert.Equal(t, 14, timeInfo.Hour)
	assert.Equal(t, 30, timeInfo.Minute)
	assert.Equal(t, "Asia/Shanghai", timeInfo.Timezone)
}

// TestReminderInfo 测试ReminderInfo结构
func TestReminderInfo(t *testing.T) {
	reminder := ReminderInfo{
		Title:           "测试提醒",
		Type:            models.ReminderTypeTask,
		Description:     "测试描述",
		SchedulePattern: models.SchedulePatternDaily,
		Time: TimeInfo{
			Hour:   9,
			Minute: 0,
		},
	}

	assert.Equal(t, "测试提醒", reminder.Title)
	assert.Equal(t, models.ReminderTypeTask, reminder.Type)
	assert.Equal(t, "测试描述", reminder.Description)
	assert.Equal(t, models.SchedulePatternDaily, reminder.SchedulePattern)
	assert.Equal(t, 9, reminder.Time.Hour)
}

// TestDeleteInfo 测试DeleteInfo结构
func TestDeleteInfo(t *testing.T) {
	deleteInfo := DeleteInfo{
		Keywords: []string{"健身", "晚上"},
		Criteria: "删除今晚的健身提醒",
	}

	assert.Len(t, deleteInfo.Keywords, 2)
	assert.Contains(t, deleteInfo.Keywords, "健身")
	assert.Equal(t, "删除今晚的健身提醒", deleteInfo.Criteria)
}

// TestEditInfo 测试EditInfo结构
func TestEditInfo(t *testing.T) {
	newTitle := "新标题"
	editInfo := EditInfo{
		Keywords: []string{"健身"},
		NewTitle: newTitle,
	}

	assert.Len(t, editInfo.Keywords, 1)
	assert.Equal(t, "新标题", editInfo.NewTitle)
}

// TestPauseInfo 测试PauseInfo结构
func TestPauseInfo(t *testing.T) {
	pauseInfo := PauseInfo{
		Keywords: []string{"健身"},
		Duration: "P1W",
		Reason:   "出差",
	}

	assert.Len(t, pauseInfo.Keywords, 1)
	assert.Equal(t, "P1W", pauseInfo.Duration)
	assert.Equal(t, "出差", pauseInfo.Reason)
}

// TestResumeInfo 测试ResumeInfo结构
func TestResumeInfo(t *testing.T) {
	resumeInfo := ResumeInfo{
		Keywords: []string{"健身", "跑步"},
	}

	assert.Len(t, resumeInfo.Keywords, 2)
	assert.Contains(t, resumeInfo.Keywords, "健身")
	assert.Contains(t, resumeInfo.Keywords, "跑步")
}

// TestChatResponse 测试ChatResponse结构
func TestChatResponse(t *testing.T) {
	response := ChatResponse{
		Response:    "你好！",
		ParsedBy:    "openai-gpt-4",
		ProcessTime: 100 * time.Millisecond,
		Timestamp:   time.Now(),
	}

	assert.Equal(t, "你好！", response.Response)
	assert.Equal(t, "openai-gpt-4", response.ParsedBy)
	assert.Equal(t, 100*time.Millisecond, response.ProcessTime)
	assert.False(t, response.Timestamp.IsZero())
}

// TestWeatherInfo 测试WeatherInfo结构
func TestWeatherInfo(t *testing.T) {
	weatherInfo := WeatherInfo{
		Location: "北京",
		Date:     "今天",
	}

	assert.Equal(t, "北京", weatherInfo.Location)
	assert.Equal(t, "今天", weatherInfo.Date)
}

// TestParseIntent_Weather 测试天气意图
func TestParseIntent_Weather(t *testing.T) {
	weatherIntent := IntentWeather
	assert.True(t, weatherIntent.IsValid(), "IntentWeather应该是有效的")
	assert.Equal(t, "weather", weatherIntent.String(), "IntentWeather的字符串表示应该是'weather'")
}

// TestParseResult_WeatherValidation 测试天气解析结果验证
func TestParseResult_WeatherValidation(t *testing.T) {
	// 测试有效的天气解析结果
	validResult := &ParseResult{
		Intent:     IntentWeather,
		Confidence: 0.95,
		Weather: &WeatherInfo{
			Location: "上海",
			Date:     "明天",
		},
		Timestamp: time.Now(),
	}

	validation := validResult.Validate()
	assert.True(t, validation.IsValid, "有效的天气解析结果应该通过验证")
	assert.Empty(t, validation.Errors, "有效的天气解析结果不应该有错误")

	// 测试缺少Weather信息的解析结果
	invalidResult1 := &ParseResult{
		Intent:     IntentWeather,
		Confidence: 0.95,
		Timestamp:  time.Now(),
	}

	validation1 := invalidResult1.Validate()
	assert.False(t, validation1.IsValid, "缺少Weather信息的解析结果应该验证失败")
	assert.Contains(t, validation1.Errors, "weather info is required for weather intent", "应该提示缺少天气信息")

	// 测试缺少Location信息的解析结果
	invalidResult2 := &ParseResult{
		Intent:     IntentWeather,
		Confidence: 0.95,
		Weather: &WeatherInfo{
			Location: "", // 空位置
			Date:     "今天",
		},
		Timestamp: time.Now(),
	}

	validation2 := invalidResult2.Validate()
	assert.False(t, validation2.IsValid, "缺少Location信息的解析结果应该验证失败")
	assert.Contains(t, validation2.Errors, "weather location is required", "应该提示缺少位置信息")
}
