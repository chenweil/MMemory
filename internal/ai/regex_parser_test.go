package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/internal/models"
	"mmemory/pkg/ai"
)

// TestRegexParser_DailyReminder 测试每日提醒解析
func TestRegexParser_DailyReminder(t *testing.T) {
	parser := NewRegexParser()
	ctx := context.Background()

	tests := []struct {
		name           string
		message        string
		expectedTitle  string
		expectedHour   int
		expectedMinute int
		expectedType   models.ReminderType
	}{
		{
			name:           "每天提醒-小时",
			message:        "每天早上8点提醒我喝水",
			expectedTitle:  "喝水",
			expectedHour:   8,
			expectedMinute: 0,
			expectedType:   models.ReminderTypeHabit,
		},
		{
			name:           "每天提醒-小时分钟",
			message:        "每天9点30分提醒我吃药",
			expectedTitle:  "吃药",
			expectedHour:   9,
			expectedMinute: 30,
			expectedType:   models.ReminderTypeHabit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(ctx, "user1", tt.message)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, ai.IntentReminder, result.Intent)
			assert.Equal(t, tt.expectedTitle, result.Reminder.Title)
			assert.Equal(t, tt.expectedHour, result.Reminder.Time.Hour)
			assert.Equal(t, tt.expectedMinute, result.Reminder.Time.Minute)
			assert.Equal(t, tt.expectedType, result.Reminder.Type)
			assert.Equal(t, models.SchedulePatternDaily, result.Reminder.SchedulePattern)
			assert.Equal(t, "regex-parser", result.ParsedBy)
			assert.Equal(t, float32(0.75), result.Confidence)
		})
	}
}

// TestRegexParser_WeeklyReminder 测试每周提醒解析
func TestRegexParser_WeeklyReminder(t *testing.T) {
	parser := NewRegexParser()
	ctx := context.Background()

	result, err := parser.Parse(ctx, "user1", "每周一下午3点提醒我开会")

	require.NoError(t, err)
	assert.Equal(t, "开会", result.Reminder.Title)
	assert.Equal(t, 3, result.Reminder.Time.Hour)
	assert.Equal(t, "weekly:1", string(result.Reminder.SchedulePattern))
}

// TestRegexParser_WorkdayReminder 测试工作日提醒解析
func TestRegexParser_WorkdayReminder(t *testing.T) {
	parser := NewRegexParser()
	ctx := context.Background()

	result, err := parser.Parse(ctx, "user1", "工作日早上9点提醒我上班")

	require.NoError(t, err)
	assert.Equal(t, "上班", result.Reminder.Title)
	assert.Equal(t, 9, result.Reminder.Time.Hour)
	assert.Equal(t, "weekly:1,2,3,4,5", string(result.Reminder.SchedulePattern))
}

// TestRegexParser_TomorrowReminder 测试明天提醒解析
func TestRegexParser_TomorrowReminder(t *testing.T) {
	parser := NewRegexParser()
	ctx := context.Background()

	result, err := parser.Parse(ctx, "user1", "明天下午2点提醒我取快递")

	require.NoError(t, err)
	assert.Equal(t, "取快递", result.Reminder.Title)
	assert.Equal(t, 2, result.Reminder.Time.Hour)
	assert.Equal(t, models.ReminderTypeTask, result.Reminder.Type)
	assert.Contains(t, string(result.Reminder.SchedulePattern), "once:")
}

// TestRegexParser_SpecificDateReminder 测试具体日期提醒解析
func TestRegexParser_SpecificDateReminder(t *testing.T) {
	parser := NewRegexParser()
	ctx := context.Background()

	result, err := parser.Parse(ctx, "user1", "2025年10月15日上午10点提醒我体检")

	require.NoError(t, err)
	assert.Equal(t, "体检", result.Reminder.Title)
	assert.Equal(t, 10, result.Reminder.Time.Hour)
	assert.Equal(t, "once:2025-10-15", string(result.Reminder.SchedulePattern))
}

// TestRegexParser_NoMatch 测试无法匹配的消息
func TestRegexParser_NoMatch(t *testing.T) {
	parser := NewRegexParser()
	ctx := context.Background()

	result, err := parser.Parse(ctx, "user1", "这是一个无法解析的消息")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no regex pattern matched")
}

// TestRegexParser_IsHealthy 测试健康检查
func TestRegexParser_IsHealthy(t *testing.T) {
	parser := NewRegexParser()
	assert.True(t, parser.IsHealthy())
}

// TestRegexParser_Priority 测试优先级
func TestRegexParser_Priority(t *testing.T) {
	parser := NewRegexParser()
	assert.Equal(t, 3, parser.GetPriority())
}

// TestRegexParser_Name 测试名称
func TestRegexParser_Name(t *testing.T) {
	parser := NewRegexParser()
	assert.Equal(t, "regex-parser", parser.GetName())
}
