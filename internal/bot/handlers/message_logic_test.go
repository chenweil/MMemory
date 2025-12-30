package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mmemory/internal/models"
	"mmemory/internal/service"
	"mmemory/pkg/ai"
)

// TestNewMessageHandlerLogic 测试创建逻辑层
func TestNewMessageHandlerLogic(t *testing.T) {
	mockReminder := new(MockReminderService)
	mockLog := new(MockReminderLogService)
	mockAI := new(MockAIParserService)

	logic := NewMessageHandlerLogic(mockReminder, mockLog, mockAI)

	assert.NotNil(t, logic)
	assert.NotNil(t, logic.reminderService)
	assert.NotNil(t, logic.reminderLogService)
}

// TestBuildWelcomeText 测试构建欢迎消息
func TestBuildWelcomeText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	text := logic.BuildWelcomeText()

	assert.Contains(t, text, "欢迎使用")
	assert.Contains(t, text, "MMemory")
	assert.Contains(t, text, "智能提醒助手")
	assert.Contains(t, text, "/help")
}

// TestBuildHelpText 测试构建帮助消息
func TestBuildHelpText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	text := logic.BuildHelpText()

	assert.Contains(t, text, "使用指南")
	assert.Contains(t, text, "/list")
	assert.Contains(t, text, "/start")
	assert.Contains(t, text, "/stats")
	assert.Contains(t, text, "/version")
}

// TestBuildVersionText 测试构建版本信息
func TestBuildVersionText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	text := logic.BuildVersionText()

	assert.Contains(t, text, "版本信息")
	assert.Contains(t, text, "MMemory")
	assert.Contains(t, text, "版本:")
	assert.Contains(t, text, "Git提交:")
}

// TestBuildReminderListText 测试构建提醒列表
func TestBuildReminderListText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	tests := []struct {
		name          string
		reminders     []*models.Reminder
		expectEmpty   bool
		expectCount   int
		expectItems   int
		expectContain string
	}{
		{
			name:          "empty list",
			reminders:     []*models.Reminder{},
			expectEmpty:   true,
			expectCount:   0,
			expectItems:   0,
			expectContain: "还没有设置任何提醒",
		},
		{
			name: "one active reminder",
			reminders: []*models.Reminder{
				{
					ID:              1,
					Title:           "喝水",
					Type:            models.ReminderTypeHabit,
					TargetTime:      "08:00:00",
					SchedulePattern: string(models.SchedulePatternDaily),
					IsActive:        true,
				},
			},
			expectEmpty:   false,
			expectCount:   1,
			expectItems:   1,
			expectContain: "喝水",
		},
		{
			name: "multiple reminders with inactive",
			reminders: []*models.Reminder{
				{
					ID:              1,
					Title:           "喝水",
					Type:            models.ReminderTypeHabit,
					TargetTime:      "08:00:00",
					SchedulePattern: string(models.SchedulePatternDaily),
					IsActive:        true,
				},
				{
					ID:              2,
					Title:           "健身",
					Type:            models.ReminderTypeTask,
					TargetTime:      "19:00:00",
					SchedulePattern: "weekly:1,3,5",
					IsActive:        false, // inactive
				},
				{
					ID:              3,
					Title:           "阅读",
					Type:            models.ReminderTypeHabit,
					TargetTime:      "21:00:00",
					SchedulePattern: string(models.SchedulePatternDaily),
					IsActive:        true,
				},
			},
			expectEmpty:   false,
			expectCount:   2,
			expectItems:   2,
			expectContain: "喝水",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, items, hasActive := logic.BuildReminderListText(tt.reminders)

			assert.Contains(t, text, tt.expectContain)
			assert.Equal(t, len(items), tt.expectItems)
			assert.Equal(t, hasActive, !tt.expectEmpty)

			if !tt.expectEmpty {
				assert.Contains(t, text, "提醒列表")
			}
		})
	}
}

// TestBuildStatsText 测试构建统计信息
func TestBuildStatsText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	tests := []struct {
		name   string
		stats  *service.UserStatistics
		expect []string
	}{
		{
			name: "with completion rate",
			stats: &service.UserStatistics{
				TotalReminders:  10,
				ActiveReminders: 8,
				CompletedToday:  3,
				SkippedToday:    1,
				CompletedWeek:   15,
				CompletedMonth:  50,
				CompletionRate:  85,
			},
			expect: []string{"使用统计", "<b>活跃提醒:</b> 8", "今日数据", "完成率: 85%", "今天做得很棒"},
		},
		{
			name: "no completion rate",
			stats: &service.UserStatistics{
				TotalReminders:  5,
				ActiveReminders: 5,
				CompletedToday:  0,
				CompletionRate:  0,
			},
			expect: []string{"使用统计", "<b>活跃提醒:</b> 5", "暂无数据", "今天还有提醒"},
		},
		{
			name: "no active reminders",
			stats: &service.UserStatistics{
				TotalReminders:  0,
				ActiveReminders: 0,
				CompletionRate:  0,
			},
			expect: []string{"使用统计", "<b>活跃提醒:</b> 0", "快去设置一些提醒"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := logic.BuildStatsText(tt.stats)

			for _, exp := range tt.expect {
				assert.Contains(t, text, exp)
			}
		})
	}
}

// TestBuildReminderSuccessText 测试构建提醒成功消息
func TestBuildReminderSuccessText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	reminder := &models.Reminder{
		ID:              1,
		Title:           "喝水",
		Type:            models.ReminderTypeHabit,
		TargetTime:      "08:00:00",
		SchedulePattern: string(models.SchedulePatternDaily),
		IsActive:        true,
	}

	tests := []struct {
		name        string
		parseResult *ai.ParseResult
		expectHint  bool
	}{
		{
			name: "high confidence",
			parseResult: &ai.ParseResult{
				Confidence: 0.95,
			},
			expectHint: false,
		},
		{
			name: "low confidence",
			parseResult: &ai.ParseResult{
				Confidence: 0.4, // 低于0.5才算低置信度
			},
			expectHint: true,
		},
		{
			name:        "no parse result",
			parseResult: nil,
			expectHint:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := logic.BuildReminderSuccessText(reminder, tt.parseResult)

			assert.Contains(t, text, "✅")
			assert.Contains(t, text, "提醒已设置成功")
			assert.Contains(t, text, "喝水")

			if tt.expectHint {
				assert.Contains(t, text, "更详细的信息")
			} else {
				assert.NotContains(t, text, "更详细的信息")
			}
		})
	}
}

// TestBuildDeleteSuccessText 测试构建删除成功消息
func TestBuildDeleteSuccessText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	reminder := &models.Reminder{
		ID:              1,
		Title:           "喝水",
		TargetTime:      "08:00:00",
		SchedulePattern: string(models.SchedulePatternDaily),
	}

	text := logic.BuildDeleteSuccessText(reminder)

	assert.Contains(t, text, "✅")
	assert.Contains(t, text, "已删除提醒")
	assert.Contains(t, text, "喝水")
}

// TestBuildEditSuccessText 测试构建编辑成功消息
func TestBuildEditSuccessText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	tests := []struct {
		name       string
		reminder   *models.Reminder
		expectDesc bool
	}{
		{
			name: "with description",
			reminder: &models.Reminder{
				ID:              1,
				Title:           "喝水",
				Description:     "每天8杯水",
				TargetTime:      "08:00:00",
				SchedulePattern: string(models.SchedulePatternDaily),
			},
			expectDesc: true,
		},
		{
			name: "without description",
			reminder: &models.Reminder{
				ID:              1,
				Title:           "喝水",
				TargetTime:      "08:00:00",
				SchedulePattern: string(models.SchedulePatternDaily),
			},
			expectDesc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := logic.BuildEditSuccessText(tt.reminder)

			assert.Contains(t, text, "✅")
			assert.Contains(t, text, "已成功修改提醒")
			assert.Contains(t, text, "喝水")

			if tt.expectDesc {
				assert.Contains(t, text, "每天8杯水")
			}
		})
	}
}

// TestBuildPauseSuccessText 测试构建暂停成功消息
func TestBuildPauseSuccessText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	reminder := &models.Reminder{
		ID:    1,
		Title: "喝水",
	}

	tests := []struct {
		name         string
		untilTime    string
		reason       string
		expectReason bool
	}{
		{
			name:         "with reason",
			untilTime:    "2024-10-20 15:00",
			reason:       "出差",
			expectReason: true,
		},
		{
			name:         "without reason",
			untilTime:    "2024-10-20 15:00",
			reason:       "",
			expectReason: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := logic.BuildPauseSuccessText(reminder, tt.untilTime, tt.reason)

			assert.Contains(t, text, "⏸️")
			assert.Contains(t, text, "已暂停提醒")
			assert.Contains(t, text, "喝水")
			assert.Contains(t, text, tt.untilTime)

			if tt.expectReason {
				assert.Contains(t, text, "理由")
				assert.Contains(t, text, tt.reason)
			}
		})
	}
}

// TestBuildResumeSuccessText 测试构建恢复成功消息
func TestBuildResumeSuccessText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	reminder := &models.Reminder{
		ID:              1,
		Title:           "喝水",
		TargetTime:      "08:00:00",
		SchedulePattern: string(models.SchedulePatternDaily),
	}

	text := logic.BuildResumeSuccessText(reminder)

	assert.Contains(t, text, "▶️")
	assert.Contains(t, text, "已恢复提醒")
	assert.Contains(t, text, "喝水")
}

// TestBuildSummaryText 测试构建总结消息
func TestBuildSummaryText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	stats := &service.UserStatistics{
		ActiveReminders: 5,
		CompletedWeek:   10,
		CompletedMonth:  35,
		CompletionRate:  75,
	}

	tests := []struct {
		name       string
		aiResponse string
		expectAI   bool
	}{
		{
			name:       "with AI response",
			aiResponse: "你本周表现很好！",
			expectAI:   true,
		},
		{
			name:       "without AI response",
			aiResponse: "",
			expectAI:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := logic.BuildSummaryText(stats, tt.aiResponse)

			assert.Contains(t, text, "使用总结")
			assert.Contains(t, text, "活跃提醒: 5")
			assert.Contains(t, text, "完成率: 75%")

			if tt.expectAI {
				assert.Contains(t, text, tt.aiResponse)
			}
		})
	}
}

// TestBuildQueryText 测试构建查询消息
func TestBuildQueryText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	tests := []struct {
		name       string
		reminders  []*models.Reminder
		aiResponse string
		expectText string
	}{
		{
			name:       "no reminders",
			reminders:  []*models.Reminder{},
			aiResponse: "",
			expectText: "还没有设置任何提醒",
		},
		{
			name: "with reminders",
			reminders: []*models.Reminder{
				{
					ID:              1,
					Title:           "喝水",
					Type:            models.ReminderTypeHabit,
					TargetTime:      "08:00:00",
					SchedulePattern: string(models.SchedulePatternDaily),
					IsActive:        true,
				},
			},
			aiResponse: "你有1个提醒",
			expectText: "提醒列表",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := logic.BuildQueryText(tt.reminders, tt.aiResponse)

			assert.Contains(t, text, tt.expectText)

			if tt.aiResponse != "" && len(tt.reminders) > 0 {
				assert.Contains(t, text, tt.aiResponse)
			}
		})
	}
}

// TestBuildMultipleMatchesText 测试构建多个匹配结果消息
func TestBuildMultipleMatchesText(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	matches := []reminderMatch{
		{
			reminder: &models.Reminder{
				ID:              1,
				Title:           "喝水提醒",
				TargetTime:      "08:00:00",
				SchedulePattern: string(models.SchedulePatternDaily),
			},
			score: 2,
		},
		{
			reminder: &models.Reminder{
				ID:              2,
				Title:           "喝水",
				TargetTime:      "12:00:00",
				SchedulePattern: string(models.SchedulePatternDaily),
			},
			score: 1,
		},
	}

	tests := []struct {
		name       string
		action     string
		expectHint string
	}{
		{
			name:       "delete action",
			action:     "delete",
			expectHint: "删除",
		},
		{
			name:       "edit action",
			action:     "edit",
			expectHint: "修改",
		},
		{
			name:       "pause action",
			action:     "pause",
			expectHint: "暂停",
		},
		{
			name:       "resume action",
			action:     "resume",
			expectHint: "恢复",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := logic.BuildMultipleMatchesText(matches, tt.action)

			assert.Contains(t, text, "找到多个")
			assert.Contains(t, text, "喝水提醒")
			assert.Contains(t, text, tt.expectHint)
		})
	}
}

// TestProcessReminderCreation 测试处理提醒创建
func TestProcessReminderCreation(t *testing.T) {
	mockReminder := new(MockReminderService)
	logic := NewMessageHandlerLogic(mockReminder, nil, nil)

	tests := []struct {
		name         string
		reminderInfo *ai.ReminderInfo
		setupMock    func()
		expectError  bool
	}{
		{
			name:         "nil reminder info",
			reminderInfo: nil,
			setupMock:    func() {},
			expectError:  true,
		},
		{
			name: "valid reminder",
			reminderInfo: &ai.ReminderInfo{
				Title:       "喝水",
				Description: "每天8杯水",
				Type:        models.ReminderTypeHabit,
				Time: ai.TimeInfo{
					Hour:     8,
					Minute:   0,
					Timezone: "Asia/Shanghai",
				},
				SchedulePattern: models.SchedulePatternDaily,
			},
			setupMock: func() {
				mockReminder.On("CreateReminder", mock.Anything, mock.AnythingOfType("*models.Reminder")).
					Return(nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			reminder, err := logic.ProcessReminderCreation(context.Background(), 1, tt.reminderInfo)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reminder)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
				assert.Equal(t, "喝水", reminder.Title)
				assert.Equal(t, "08:00:00", reminder.TargetTime)
			}
		})
	}
}

// TestFindRemindersByKeywords 测试根据关键词查找提醒
func TestFindRemindersByKeywords(t *testing.T) {
	mockReminder := new(MockReminderService)
	logic := NewMessageHandlerLogic(mockReminder, nil, nil)

	reminders := []*models.Reminder{
		{
			ID:       1,
			Title:    "喝水提醒",
			IsActive: true,
		},
		{
			ID:       2,
			Title:    "健身打卡",
			IsActive: true,
		},
	}

	tests := []struct {
		name          string
		keywords      []string
		setupMock     func()
		expectError   bool
		expectMatches int
	}{
		{
			name:        "empty keywords",
			keywords:    []string{},
			setupMock:   func() {},
			expectError: true,
		},
		{
			name:     "keywords with spaces",
			keywords: []string{"  ", "喝水", "  "},
			setupMock: func() {
				mockReminder.On("GetUserReminders", mock.Anything, uint(1)).
					Return(reminders, nil).Once()
			},
			expectError:   false,
			expectMatches: 1,
		},
		{
			name:     "multiple matches",
			keywords: []string{"提醒", "打卡"},
			setupMock: func() {
				mockReminder.On("GetUserReminders", mock.Anything, uint(1)).
					Return(reminders, nil).Once()
			},
			expectError:   false,
			expectMatches: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			matches, err := logic.FindRemindersByKeywords(context.Background(), 1, tt.keywords)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, matches, tt.expectMatches)
			}
		})
	}
}

// TestCalculatePauseUntilTime 测试计算暂停时间
func TestCalculatePauseUntilTime(t *testing.T) {
	logic := NewMessageHandlerLogic(nil, nil, nil)

	tests := []struct {
		name            string
		duration        string
		expectDuration  time.Duration
		expectTimeValid bool
	}{
		{
			name:            "1 week",
			duration:        "1week",
			expectDuration:  7 * 24 * time.Hour,
			expectTimeValid: true,
		},
		{
			name:            "3 days",
			duration:        "3天",
			expectDuration:  3 * 24 * time.Hour,
			expectTimeValid: true,
		},
		{
			name:            "empty duration",
			duration:        "",
			expectDuration:  7 * 24 * time.Hour,
			expectTimeValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dur, timeText := logic.CalculatePauseUntilTime(tt.duration)

			assert.Equal(t, tt.expectDuration, dur)
			assert.NotEmpty(t, timeText)
			if tt.expectTimeValid {
				assert.Len(t, timeText, 16) // "2024-10-20 15:00" format
			}
		})
	}
}

// TestFormatSchedule 测试格式化时间表
func TestFormatSchedule(t *testing.T) {
	tests := []struct {
		name       string
		reminder   *models.Reminder
		expectText string
	}{
		{
			name: "daily reminder",
			reminder: &models.Reminder{
				TargetTime:      "08:00:00",
				SchedulePattern: string(models.SchedulePatternDaily),
			},
			expectText: "每天 08:00",
		},
		{
			name: "weekly reminder",
			reminder: &models.Reminder{
				TargetTime:      "19:00:00",
				SchedulePattern: "weekly:1,3,5",
			},
			expectText: "周一、周三、周五 19:00",
		},
		{
			name: "once reminder",
			reminder: &models.Reminder{
				TargetTime:      "10:00:00",
				SchedulePattern: "once:2024-12-25",
			},
			expectText: "2024-12-25 10:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSchedule(tt.reminder)
			assert.Equal(t, tt.expectText, result)
		})
	}
}
