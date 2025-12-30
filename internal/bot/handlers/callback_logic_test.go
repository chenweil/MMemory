package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mmemory/internal/models"
)

// TestNewCallbackHandlerLogic 测试创建回调逻辑层
func TestNewCallbackHandlerLogic(t *testing.T) {
	mockReminder := new(MockReminderService)
	mockLog := new(MockReminderLogService)

	logic := NewCallbackHandlerLogic(mockReminder, mockLog)

	assert.NotNil(t, logic)
	assert.NotNil(t, logic.reminderService)
	assert.NotNil(t, logic.reminderLogService)
}

// TestParseCallbackData 测试解析回调数据
func TestParseCallbackData(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	tests := []struct {
		name        string
		data        string
		expectValid bool
		expectAction string
		expectID    uint
		expectHours int
		expectError string
	}{
		{
			name:        "valid complete",
			data:        "reminder_complete_123",
			expectValid: true,
			expectAction: "complete",
			expectID:    123,
		},
		{
			name:        "valid delay with hours",
			data:        "reminder_delay_456_2",
			expectValid: true,
			expectAction: "delay",
			expectID:    456,
			expectHours: 2,
		},
		{
			name:        "valid skip",
			data:        "reminder_skip_789",
			expectValid: true,
			expectAction: "skip",
			expectID:    789,
		},
		{
			name:        "invalid - too few parts",
			data:        "reminder_complete",
			expectValid: false,
			expectError: "无效的操作",
		},
		{
			name:        "invalid - bad ID",
			data:        "reminder_complete_abc",
			expectValid: false,
			expectError: "无效的提醒ID",
		},
		{
			name:        "invalid delay - bad hours",
			data:        "reminder_delay_123_abc",
			expectValid: false,
			expectError: "无效的延期时间",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := logic.ParseCallbackData(tt.data)

			assert.Equal(t, tt.expectValid, action.IsValid)

			if tt.expectValid {
				assert.Equal(t, tt.expectAction, action.Action)
				assert.Equal(t, tt.expectID, action.ResourceID)
				if tt.expectAction == "delay" && tt.expectHours > 0 {
					assert.Equal(t, tt.expectHours, action.Hours)
				}
			} else {
				assert.Contains(t, action.ErrorMsg, tt.expectError)
			}
		})
	}
}

// TestBuildCompleteText 测试构建完成文本
func TestBuildCompleteText(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	reminder := &models.Reminder{
		ID:    1,
		Title: "喝水",
	}

	text := logic.BuildCompleteText(reminder)

	assert.Contains(t, text, "✅")
	assert.Contains(t, text, "太棒了")
	assert.Contains(t, text, "喝水")
	assert.Contains(t, text, "已记录完成")
}

// TestBuildDelayText 测试构建延期文本
func TestBuildDelayText(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	reminder := &models.Reminder{
		ID:    1,
		Title: "喝水",
	}

	delayTime := time.Now().Add(2 * time.Hour)
	text := logic.BuildDelayText(reminder, 2, delayTime)

	assert.Contains(t, text, "⏰")
	assert.Contains(t, text, "已延期 2 小时")
	assert.Contains(t, text, "喝水")
	assert.Contains(t, text, "再次提醒你")
}

// TestBuildSkipText 测试构建跳过文本
func TestBuildSkipText(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	reminder := &models.Reminder{
		ID:    1,
		Title: "健身",
	}

	text := logic.BuildSkipText(reminder)

	assert.Contains(t, text, "😴")
	assert.Contains(t, text, "今天跳过")
	assert.Contains(t, text, "健身")
	assert.Contains(t, text, "明天再来")
}

// TestBuildDeleteText 测试构建删除文本
func TestBuildDeleteText(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	text := logic.BuildDeleteText(123)

	assert.Contains(t, text, "✅")
	assert.Contains(t, text, "已删除提醒")
	assert.Contains(t, text, "#123")
}

// TestBuildPauseText 测试构建暂停文本
func TestBuildPauseText(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	reminder := &models.Reminder{
		ID:    1,
		Title: "喝水",
	}

	text := logic.BuildPauseText(reminder, "2024-10-20 15:00")

	assert.Contains(t, text, "⏸️")
	assert.Contains(t, text, "已暂停提醒")
	assert.Contains(t, text, "#1")
	assert.Contains(t, text, "喝水")
	assert.Contains(t, text, "2024-10-20 15:00")
}

// TestBuildResumeText 测试构建恢复文本
func TestBuildResumeText(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	reminder := &models.Reminder{
		ID:         1,
		Title:      "健身",
		TargetTime: "19:00:00",
	}

	text := logic.BuildResumeText(reminder)

	assert.Contains(t, text, "▶️")
	assert.Contains(t, text, "已恢复提醒")
	assert.Contains(t, text, "#1")
	assert.Contains(t, text, "健身")
	assert.Contains(t, text, "19:00")
}

// TestBuildEditText 测试构建编辑文本
func TestBuildEditText(t *testing.T) {
	logic := NewCallbackHandlerLogic(nil, nil)

	reminder := &models.Reminder{
		ID:              1,
		Title:           "喝水",
		TargetTime:      "08:00:00",
		SchedulePattern: string(models.SchedulePatternDaily),
	}

	text := logic.BuildEditText(reminder)

	assert.Contains(t, text, "🛠️")
	assert.Contains(t, text, "编辑提醒")
	assert.Contains(t, text, "#1")
	assert.Contains(t, text, "喝水")
	assert.Contains(t, text, "08:00")
	assert.Contains(t, text, "如何编辑")
}

// TestProcessComplete 测试处理完成操作
func TestProcessComplete(t *testing.T) {
	mockLog := new(MockReminderLogService)
	logic := NewCallbackHandlerLogic(nil, mockLog)

	tests := []struct {
		name        string
		logID       uint
		setupMock   func()
		expectError bool
	}{
		{
			name:  "success",
			logID: 1,
			setupMock: func() {
				reminderLog := &models.ReminderLog{
					ID: 1,
					Reminder: models.Reminder{
						ID:    1,
						Title: "喝水",
					},
					Status: models.ReminderStatusPending,
				}
				mockLog.On("GetByID", mock.Anything, uint(1)).Return(reminderLog, nil).Once()
				mockLog.On("MarkAsCompleted", mock.Anything, uint(1), mock.Anything).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name:  "log not found",
			logID: 999,
			setupMock: func() {
				mockLog.On("GetByID", mock.Anything, uint(999)).Return(nil, nil).Once()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			log, err := logic.ProcessComplete(context.Background(), tt.logID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, log)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, log)
			}
		})
	}
}

// TestProcessDelay 测试处理延期操作
func TestProcessDelay(t *testing.T) {
	mockLog := new(MockReminderLogService)
	logic := NewCallbackHandlerLogic(nil, mockLog)

	tests := []struct {
		name        string
		logID       uint
		hours       int
		setupMock   func()
		expectError bool
	}{
		{
			name:  "success",
			logID: 1,
			hours: 2,
			setupMock: func() {
				reminderLog := &models.ReminderLog{
					ID: 1,
					Reminder: models.Reminder{
						ID:    1,
						Title: "喝水",
					},
					Status: models.ReminderStatusPending,
				}
				mockLog.On("GetByID", mock.Anything, uint(1)).Return(reminderLog, nil).Once()
				mockLog.On("CreateDelayReminder", mock.Anything, uint(1), mock.AnythingOfType("time.Time"), 2).
					Return(nil).Once()
			},
			expectError: false,
		},
		{
			name:  "log not found",
			logID: 999,
			hours: 2,
			setupMock: func() {
				mockLog.On("GetByID", mock.Anything, uint(999)).Return(nil, nil).Once()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			log, delayTime, err := logic.ProcessDelay(context.Background(), tt.logID, tt.hours)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, log)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, log)
				assert.NotZero(t, delayTime)
			}
		})
	}
}

// TestProcessSkip 测试处理跳过操作
func TestProcessSkip(t *testing.T) {
	mockLog := new(MockReminderLogService)
	logic := NewCallbackHandlerLogic(nil, mockLog)

	tests := []struct {
		name        string
		logID       uint
		setupMock   func()
		expectError bool
	}{
		{
			name:  "success",
			logID: 1,
			setupMock: func() {
				reminderLog := &models.ReminderLog{
					ID: 1,
					Reminder: models.Reminder{
						ID:    1,
						Title: "健身",
					},
					Status: models.ReminderStatusPending,
				}
				mockLog.On("GetByID", mock.Anything, uint(1)).Return(reminderLog, nil).Once()
				mockLog.On("MarkAsSkipped", mock.Anything, uint(1), mock.Anything).Return(nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			log, err := logic.ProcessSkip(context.Background(), tt.logID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, log)
			}
		})
	}
}

// TestProcessDelete 测试处理删除操作
func TestProcessDelete(t *testing.T) {
	mockReminder := new(MockReminderService)
	logic := NewCallbackHandlerLogic(mockReminder, nil)

	tests := []struct {
		name        string
		reminderID  uint
		setupMock   func()
		expectError bool
	}{
		{
			name:       "success",
			reminderID: 1,
			setupMock: func() {
				mockReminder.On("DeleteReminder", mock.Anything, uint(1)).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name:        "invalid ID",
			reminderID:  0,
			setupMock:   func() {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := logic.ProcessDelete(context.Background(), tt.reminderID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProcessPause 测试处理暂停操作
func TestProcessPause(t *testing.T) {
	mockReminder := new(MockReminderService)
	logic := NewCallbackHandlerLogic(mockReminder, nil)

	tests := []struct {
		name        string
		reminderID  uint
		setupMock   func()
		expectError bool
	}{
		{
			name:       "success",
			reminderID: 1,
			setupMock: func() {
				pausedTime := time.Now().Add(24 * time.Hour)
				reminder := &models.Reminder{
					ID:          1,
					Title:       "喝水",
					PausedUntil: &pausedTime,
				}
				mockReminder.On("PauseReminder", mock.Anything, uint(1), mock.AnythingOfType("time.Duration"), mock.Anything).
					Return(nil).Once()
				mockReminder.On("GetReminderByID", mock.Anything, uint(1)).Return(reminder, nil).Once()
			},
			expectError: false,
		},
		{
			name:        "invalid ID",
			reminderID:  0,
			setupMock:   func() {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			reminder, until, err := logic.ProcessPause(context.Background(), tt.reminderID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reminder)
				assert.Empty(t, until)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, until)
			}
		})
	}
}

// TestProcessResume 测试处理恢复操作
func TestProcessResume(t *testing.T) {
	mockReminder := new(MockReminderService)
	logic := NewCallbackHandlerLogic(mockReminder, nil)

	tests := []struct {
		name        string
		reminderID  uint
		setupMock   func()
		expectError bool
	}{
		{
			name:       "success",
			reminderID: 1,
			setupMock: func() {
				reminder := &models.Reminder{
					ID:          1,
					Title:       "健身",
					TargetTime:  "19:00:00",
					PausedUntil: nil,
				}
				mockReminder.On("ResumeReminder", mock.Anything, uint(1)).Return(nil).Once()
				mockReminder.On("GetReminderByID", mock.Anything, uint(1)).Return(reminder, nil).Once()
			},
			expectError: false,
		},
		{
			name:        "invalid ID",
			reminderID:  0,
			setupMock:   func() {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			reminder, err := logic.ProcessResume(context.Background(), tt.reminderID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reminder)
			} else {
				assert.NoError(t, err)
				// reminder可能为nil（如果GetReminderByID失败）
			}
		})
	}
}

// TestProcessEdit 测试处理编辑操作
func TestProcessEdit(t *testing.T) {
	mockReminder := new(MockReminderService)
	logic := NewCallbackHandlerLogic(mockReminder, nil)

	tests := []struct {
		name        string
		reminderID  uint
		setupMock   func()
		expectError bool
	}{
		{
			name:       "success",
			reminderID: 1,
			setupMock: func() {
				reminder := &models.Reminder{
					ID:              1,
					Title:           "喝水",
					TargetTime:      "08:00:00",
					SchedulePattern: string(models.SchedulePatternDaily),
				}
				mockReminder.On("GetReminderByID", mock.Anything, uint(1)).Return(reminder, nil).Once()
			},
			expectError: false,
		},
		{
			name:        "invalid ID",
			reminderID:  0,
			setupMock:   func() {},
			expectError: true,
		},
		{
			name:       "reminder not found",
			reminderID: 999,
			setupMock: func() {
				mockReminder.On("GetReminderByID", mock.Anything, uint(999)).Return(nil, nil).Once()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			reminder, err := logic.ProcessEdit(context.Background(), tt.reminderID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reminder)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
			}
		})
	}
}
