package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"mmemory/internal/models"
)

// Mock ReminderRepository for testing
type mockReminderRepository struct {
	reminders map[uint]*models.Reminder
	idCounter uint
	mu        sync.Mutex
}

func newMockReminderRepository() *mockReminderRepository {
	return &mockReminderRepository{
		reminders: make(map[uint]*models.Reminder),
		idCounter: 1,
	}
}

func (m *mockReminderRepository) Create(ctx context.Context, reminder *models.Reminder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	reminder.ID = m.idCounter
	m.reminders[m.idCounter] = reminder
	m.idCounter++
	return nil
}

func (m *mockReminderRepository) GetByID(ctx context.Context, id uint) (*models.Reminder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	reminder := m.reminders[id]
	return reminder, nil
}

func (m *mockReminderRepository) GetByUserID(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.Reminder
	for _, reminder := range m.reminders {
		if reminder.UserID == userID {
			result = append(result, reminder)
		}
	}
	return result, nil
}

func (m *mockReminderRepository) GetActiveReminders(ctx context.Context) ([]*models.Reminder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.Reminder
	for _, reminder := range m.reminders {
		if reminder.IsActive {
			result = append(result, reminder)
		}
	}
	return result, nil
}

func (m *mockReminderRepository) Update(ctx context.Context, reminder *models.Reminder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing := m.reminders[reminder.ID]; existing != nil {
		m.reminders[reminder.ID] = reminder
	}
	return nil
}

func (m *mockReminderRepository) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.reminders, id)
	return nil
}

func (m *mockReminderRepository) CountByStatus(ctx context.Context, status models.ReminderStatStatus) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for _, reminder := range m.reminders {
		// 简化实现，实际应该根据状态统计
		if reminder.IsActive && status == models.ReminderStatStatusActive {
			count++
		}
	}
	return count, nil
}

type mockScheduler struct {
	added   []uint
	removed []uint
}

func (m *mockScheduler) Start() error {
	return nil
}

func (m *mockScheduler) Stop() error {
	return nil
}

func (m *mockScheduler) AddReminder(reminder *models.Reminder) error {
	m.added = append(m.added, reminder.ID)
	return nil
}

func (m *mockScheduler) RemoveReminder(reminderID uint) error {
	m.removed = append(m.removed, reminderID)
	return nil
}

func (m *mockScheduler) RefreshSchedules() error {
	return nil
}

func TestReminderService_CreateReminder(t *testing.T) {
	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	ctx := context.Background()

	tests := []struct {
		name     string
		reminder *models.Reminder
		wantErr  bool
	}{
		{
			name: "成功创建提醒",
			reminder: &models.Reminder{
				UserID:          1,
				Title:           "复盘工作",
				Type:            models.ReminderTypeHabit,
				SchedulePattern: "daily",
				TargetTime:      "19:00:00",
				IsActive:        true,
			},
			wantErr: false,
		},
		{
			name: "用户ID为空时失败",
			reminder: &models.Reminder{
				Title:           "复盘工作",
				Type:            models.ReminderTypeHabit,
				SchedulePattern: "daily",
				TargetTime:      "19:00:00",
			},
			wantErr: true,
		},
		{
			name: "标题为空时失败",
			reminder: &models.Reminder{
				UserID:          1,
				Type:            models.ReminderTypeHabit,
				SchedulePattern: "daily",
				TargetTime:      "19:00:00",
			},
			wantErr: true,
		},
		{
			name: "时间为空时失败",
			reminder: &models.Reminder{
				UserID:          1,
				Title:           "复盘工作",
				Type:            models.ReminderTypeHabit,
				SchedulePattern: "daily",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reminderService.CreateReminder(ctx, tt.reminder)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateReminder() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.reminder.ID == 0 {
				t.Errorf("CreateReminder() 成功后应该设置ID")
			}
		})
	}
}

func TestReminderService_GetUserReminders(t *testing.T) {
	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	ctx := context.Background()

	// 创建测试数据
	userID1 := uint(1)
	userID2 := uint(2)

	reminder1 := &models.Reminder{
		UserID:          userID1,
		Title:           "提醒1",
		Type:            models.ReminderTypeHabit,
		SchedulePattern: "daily",
		TargetTime:      "19:00:00",
		IsActive:        true,
	}
	reminder2 := &models.Reminder{
		UserID:          userID1,
		Title:           "提醒2",
		Type:            models.ReminderTypeTask,
		SchedulePattern: "once:2024-10-01",
		TargetTime:      "10:00:00",
		IsActive:        true,
	}
	reminder3 := &models.Reminder{
		UserID:          userID2,
		Title:           "提醒3",
		Type:            models.ReminderTypeHabit,
		SchedulePattern: "daily",
		TargetTime:      "08:00:00",
		IsActive:        true,
	}

	// 创建提醒
	err := reminderService.CreateReminder(ctx, reminder1)
	if err != nil {
		t.Fatalf("创建测试提醒失败: %v", err)
	}
	err = reminderService.CreateReminder(ctx, reminder2)
	if err != nil {
		t.Fatalf("创建测试提醒失败: %v", err)
	}
	err = reminderService.CreateReminder(ctx, reminder3)
	if err != nil {
		t.Fatalf("创建测试提醒失败: %v", err)
	}

	tests := []struct {
		name      string
		userID    uint
		wantCount int
		wantErr   bool
	}{
		{
			name:      "获取用户1的提醒",
			userID:    userID1,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "获取用户2的提醒",
			userID:    userID2,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "获取不存在用户的提醒",
			userID:    999,
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminders, err := reminderService.GetUserReminders(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserReminders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(reminders) != tt.wantCount {
				t.Errorf("GetUserReminders() count = %v, want %v", len(reminders), tt.wantCount)
			}

			for _, reminder := range reminders {
				if reminder.UserID != tt.userID {
					t.Errorf("GetUserReminders() 返回了错误用户的提醒: got userID %v, want %v",
						reminder.UserID, tt.userID)
				}
			}
		})
	}
}

func TestReminderService_PauseReminder(t *testing.T) {
	mockRepo := newMockReminderRepository()
	ctx := context.Background()

	reminderService := NewReminderService(mockRepo)

	scheduler := &mockScheduler{}
	if setter, ok := reminderService.(interface{ SetScheduler(SchedulerService) }); ok {
		setter.SetScheduler(scheduler)
	}

	reminder := &models.Reminder{
		UserID:          1,
		Title:           "健身",
		Type:            models.ReminderTypeHabit,
		SchedulePattern: "daily",
		TargetTime:      "07:30:00",
		IsActive:        true,
	}

	if err := reminderService.CreateReminder(ctx, reminder); err != nil {
		t.Fatalf("创建提醒失败: %v", err)
	}

	duration := 48 * time.Hour
	if err := reminderService.PauseReminder(ctx, reminder.ID, duration, "测试暂停"); err != nil {
		t.Fatalf("PauseReminder 返回错误: %v", err)
	}

	stored, _ := mockRepo.GetByID(ctx, reminder.ID)
	if stored == nil || stored.PausedUntil == nil {
		t.Fatalf("PauseReminder 应设置 PausedUntil")
	}
	if stored.PausedUntil.Before(time.Now().Add(24 * time.Hour)) {
		t.Fatalf("PauseReminder 暂停时间过短: %v", stored.PausedUntil)
	}
	if len(scheduler.removed) == 0 || scheduler.removed[0] != reminder.ID {
		t.Fatalf("PauseReminder 应移除调度，got %v", scheduler.removed)
	}
}

func TestReminderService_ResumeReminder(t *testing.T) {
	mockRepo := newMockReminderRepository()
	ctx := context.Background()

	reminderService := NewReminderService(mockRepo)

	scheduler := &mockScheduler{}
	if setter, ok := reminderService.(interface{ SetScheduler(SchedulerService) }); ok {
		setter.SetScheduler(scheduler)
	}

	reminder := &models.Reminder{
		UserID:          1,
		Title:           "阅读",
		Type:            models.ReminderTypeHabit,
		SchedulePattern: "daily",
		TargetTime:      "21:00:00",
		IsActive:        true,
	}

	if err := reminderService.CreateReminder(ctx, reminder); err != nil {
		t.Fatalf("创建提醒失败: %v", err)
	}

	pauseUntil := time.Now().Add(2 * time.Hour)
	reminder.PausedUntil = &pauseUntil
	reminder.PauseReason = "测试"
	if err := mockRepo.Update(ctx, reminder); err != nil {
		t.Fatalf("更新提醒失败: %v", err)
	}

	if err := reminderService.ResumeReminder(ctx, reminder.ID); err != nil {
		t.Fatalf("ResumeReminder 返回错误: %v", err)
	}

	stored, _ := mockRepo.GetByID(ctx, reminder.ID)
	if stored == nil || stored.PausedUntil != nil {
		t.Fatalf("ResumeReminder 应清空 PausedUntil")
	}
	if len(scheduler.added) == 0 || scheduler.added[len(scheduler.added)-1] != reminder.ID {
		t.Fatalf("ResumeReminder 应重新加入调度，got %v", scheduler.added)
	}
}

func TestReminderService_EditReminder(t *testing.T) {
	mockRepo := newMockReminderRepository()
	ctx := context.Background()

	reminderService := NewReminderService(mockRepo)

	scheduler := &mockScheduler{}
	if setter, ok := reminderService.(interface{ SetScheduler(SchedulerService) }); ok {
		setter.SetScheduler(scheduler)
	}

	// 创建测试提醒
	reminder := &models.Reminder{
		UserID:          1,
		Title:           "健身打卡",
		Description:     "每天坚持健身",
		Type:            models.ReminderTypeHabit,
		SchedulePattern: "daily",
		TargetTime:      "07:00:00",
		IsActive:        true,
	}

	if err := reminderService.CreateReminder(ctx, reminder); err != nil {
		t.Fatalf("创建提醒失败: %v", err)
	}

	t.Run("成功修改时间", func(t *testing.T) {
		newTime := "19:00:00"
		params := EditReminderParams{
			ReminderID: reminder.ID,
			NewTime:    &newTime,
		}

		err := reminderService.EditReminder(ctx, params)
		if err != nil {
			t.Fatalf("EditReminder 返回错误: %v", err)
		}

		// 验证数据库更新
		updated, _ := mockRepo.GetByID(ctx, reminder.ID)
		if updated.TargetTime != newTime {
			t.Errorf("时间未更新: got %v, want %v", updated.TargetTime, newTime)
		}

		// 验证调度器刷新
		if len(scheduler.removed) == 0 {
			t.Error("EditReminder 应该移除旧调度")
		}
		if len(scheduler.added) == 0 {
			t.Error("EditReminder 应该添加新调度")
		}
	})

	t.Run("成功修改模式", func(t *testing.T) {
		scheduler.removed = nil
		scheduler.added = nil

		newPattern := "weekly:1,3,5"
		params := EditReminderParams{
			ReminderID: reminder.ID,
			NewPattern: &newPattern,
		}

		err := reminderService.EditReminder(ctx, params)
		if err != nil {
			t.Fatalf("EditReminder 返回错误: %v", err)
		}

		// 验证数据库更新
		updated, _ := mockRepo.GetByID(ctx, reminder.ID)
		if updated.SchedulePattern != newPattern {
			t.Errorf("模式未更新: got %v, want %v", updated.SchedulePattern, newPattern)
		}
	})

	t.Run("成功修改标题", func(t *testing.T) {
		newTitle := "跑步打卡"
		params := EditReminderParams{
			ReminderID: reminder.ID,
			NewTitle:   &newTitle,
		}

		err := reminderService.EditReminder(ctx, params)
		if err != nil {
			t.Fatalf("EditReminder 返回错误: %v", err)
		}

		// 验证数据库更新
		updated, _ := mockRepo.GetByID(ctx, reminder.ID)
		if updated.Title != newTitle {
			t.Errorf("标题未更新: got %v, want %v", updated.Title, newTitle)
		}
	})

	t.Run("成功修改描述", func(t *testing.T) {
		newDesc := "每周三次跑步"
		params := EditReminderParams{
			ReminderID:     reminder.ID,
			NewDescription: &newDesc,
		}

		err := reminderService.EditReminder(ctx, params)
		if err != nil {
			t.Fatalf("EditReminder 返回错误: %v", err)
		}

		// 验证数据库更新
		updated, _ := mockRepo.GetByID(ctx, reminder.ID)
		if updated.Description != newDesc {
			t.Errorf("描述未更新: got %v, want %v", updated.Description, newDesc)
		}
	})

	t.Run("同时修改多个字段", func(t *testing.T) {
		newTime := "06:30:00"
		newTitle := "晨跑"
		newPattern := "weekly:1,3,5,7"

		params := EditReminderParams{
			ReminderID: reminder.ID,
			NewTime:    &newTime,
			NewTitle:   &newTitle,
			NewPattern: &newPattern,
		}

		err := reminderService.EditReminder(ctx, params)
		if err != nil {
			t.Fatalf("EditReminder 返回错误: %v", err)
		}

		// 验证数据库更新
		updated, _ := mockRepo.GetByID(ctx, reminder.ID)
		if updated.TargetTime != newTime {
			t.Errorf("时间未更新: got %v, want %v", updated.TargetTime, newTime)
		}
		if updated.Title != newTitle {
			t.Errorf("标题未更新: got %v, want %v", updated.Title, newTitle)
		}
		if updated.SchedulePattern != newPattern {
			t.Errorf("模式未更新: got %v, want %v", updated.SchedulePattern, newPattern)
		}
	})

	t.Run("提醒ID为0时失败", func(t *testing.T) {
		newTime := "08:00:00"
		params := EditReminderParams{
			ReminderID: 0,
			NewTime:    &newTime,
		}

		err := reminderService.EditReminder(ctx, params)
		if err == nil {
			t.Error("EditReminder 应该返回错误当 ReminderID 为 0")
		}
	})

	t.Run("提醒不存在时失败", func(t *testing.T) {
		newTime := "08:00:00"
		params := EditReminderParams{
			ReminderID: 9999,
			NewTime:    &newTime,
		}

		err := reminderService.EditReminder(ctx, params)
		if err == nil {
			t.Error("EditReminder 应该返回错误当提醒不存在")
		}
	})

	t.Run("没有提供任何修改参数时失败", func(t *testing.T) {
		params := EditReminderParams{
			ReminderID: reminder.ID,
		}

		err := reminderService.EditReminder(ctx, params)
		if err == nil {
			t.Error("EditReminder 应该返回错误当没有提供任何修改参数")
		}
	})

	t.Run("空字符串参数不应被视为有效修改", func(t *testing.T) {
		emptyTime := ""
		emptyPattern := ""
		emptyTitle := ""

		params := EditReminderParams{
			ReminderID: reminder.ID,
			NewTime:    &emptyTime,
			NewPattern: &emptyPattern,
			NewTitle:   &emptyTitle,
		}

		err := reminderService.EditReminder(ctx, params)
		if err == nil {
			t.Error("EditReminder 应该返回错误当所有参数都是空字符串")
		}
	})
}
