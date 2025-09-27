package service

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"
)

func TestReminderLogService_MarkAsCompleted(t *testing.T) {
	mockLogRepo := newMockReminderLogRepository()
	mockReminderRepo := newMockReminderRepository()
	
	service := NewReminderLogService(mockLogRepo, mockReminderRepo)

	// 创建测试日志
	log := &models.ReminderLog{
		ReminderID:    1,
		ScheduledTime: time.Now(),
		Status:        models.ReminderStatusSent,
	}
	ctx := context.Background()
	err := mockLogRepo.Create(ctx, log)
	if err != nil {
		t.Fatalf("创建测试日志失败: %v", err)
	}

	tests := []struct {
		name     string
		logID    uint
		response string
		wantErr  bool
	}{
		{
			name:     "成功标记完成",
			logID:    log.ID,
			response: "用户确认完成",
			wantErr:  false,
		},
		{
			name:     "记录不存在",
			logID:    999,
			response: "不存在的记录",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.MarkAsCompleted(ctx, tt.logID, tt.response)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("MarkAsCompleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 验证状态已更新
				updatedLog, err := service.GetByID(ctx, tt.logID)
				if err != nil {
					t.Errorf("获取更新后的日志失败: %v", err)
					return
				}
				
				if updatedLog.Status != models.ReminderStatusCompleted {
					t.Errorf("状态未正确更新: got %v, want %v", updatedLog.Status, models.ReminderStatusCompleted)
				}
				
				if updatedLog.UserResponse != tt.response {
					t.Errorf("用户回复未正确保存: got %v, want %v", updatedLog.UserResponse, tt.response)
				}
				
				if updatedLog.ResponseTime == nil {
					t.Errorf("回复时间未设置")
				}
			}
		})
	}
}

func TestReminderLogService_CreateDelayReminder(t *testing.T) {
	mockLogRepo := newMockReminderLogRepository()
	mockReminderRepo := newMockReminderRepository()
	
	service := NewReminderLogService(mockLogRepo, mockReminderRepo)

	// 创建原始提醒记录
	originalLog := &models.ReminderLog{
		ReminderID:    1,
		ScheduledTime: time.Now(),
		Status:        models.ReminderStatusSent,
	}
	ctx := context.Background()
	err := mockLogRepo.Create(ctx, originalLog)
	if err != nil {
		t.Fatalf("创建原始日志失败: %v", err)
	}

	delayTime := time.Now().Add(2 * time.Hour)
	
	tests := []struct {
		name           string
		originalLogID  uint
		delayTime      time.Time
		hours          int
		wantErr        bool
	}{
		{
			name:          "成功创建延期提醒",
			originalLogID: originalLog.ID,
			delayTime:     delayTime,
			hours:         2,
			wantErr:       false,
		},
		{
			name:          "原始记录不存在",
			originalLogID: 999,
			delayTime:     delayTime,
			hours:         2,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CreateDelayReminder(ctx, tt.originalLogID, tt.delayTime, tt.hours)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDelayReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 验证原始记录已更新
				updatedOriginal, err := service.GetByID(ctx, tt.originalLogID)
				if err != nil {
					t.Errorf("获取原始记录失败: %v", err)
					return
				}
				
				if updatedOriginal.Status != models.ReminderStatusSkipped {
					t.Errorf("原始记录状态未正确更新: got %v, want %v", updatedOriginal.Status, models.ReminderStatusSkipped)
				}
				
				// 验证创建了新的延期记录
				allLogs, err := mockLogRepo.GetByReminderID(ctx, originalLog.ReminderID, 0, 0)
				if err != nil {
					t.Errorf("获取提醒记录失败: %v", err)
					return
				}
				
				if len(allLogs) != 2 {
					t.Errorf("延期记录未创建: 期望2个记录，实际%d个", len(allLogs))
				}
			}
		})
	}
}

func TestReminderLogService_GetOverdueReminders(t *testing.T) {
	mockLogRepo := newMockReminderLogRepository()
	mockReminderRepo := newMockReminderRepository()
	
	service := NewReminderLogService(mockLogRepo, mockReminderRepo)
	ctx := context.Background()

	// 创建测试数据
	now := time.Now()
	pastTime := now.Add(-2 * time.Hour) // 2小时前
	recentTime := now.Add(-30 * time.Minute) // 30分钟前

	// 超时的记录
	overdueLog := &models.ReminderLog{
		ReminderID:    1,
		ScheduledTime: pastTime,
		Status:        models.ReminderStatusSent,
		SentTime:      &pastTime,
	}
	
	// 未超时的记录
	recentLog := &models.ReminderLog{
		ReminderID:    2,
		ScheduledTime: recentTime,
		Status:        models.ReminderStatusSent,
		SentTime:      &recentTime,
	}
	
	// 待发送的记录
	pendingLog := &models.ReminderLog{
		ReminderID:    3,
		ScheduledTime: now,
		Status:        models.ReminderStatusPending,
	}

	// 创建测试记录
	mockLogRepo.Create(ctx, overdueLog)
	mockLogRepo.Create(ctx, recentLog)
	mockLogRepo.Create(ctx, pendingLog)

	// 测试获取超时记录
	overdueLogs, err := service.GetOverdueReminders(ctx)
	if err != nil {
		t.Fatalf("GetOverdueReminders() error = %v", err)
	}

	// 应该只返回超时的记录
	if len(overdueLogs) != 1 {
		t.Errorf("GetOverdueReminders() 返回记录数 = %d, want 1", len(overdueLogs))
	}

	if len(overdueLogs) > 0 && overdueLogs[0].ID != overdueLog.ID {
		t.Errorf("GetOverdueReminders() 返回了错误的记录ID = %d, want %d", 
			overdueLogs[0].ID, overdueLog.ID)
	}
}