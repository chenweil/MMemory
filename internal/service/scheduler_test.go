package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/robfig/cron/v3"

	"mmemory/internal/models"
)

// Mock NotificationService for testing
type mockNotificationService struct {
	sentReminders []uint
	sentFollowUps []uint
}

func newMockNotificationService() *mockNotificationService {
	return &mockNotificationService{
		sentReminders: make([]uint, 0),
		sentFollowUps: make([]uint, 0),
	}
}

func (m *mockNotificationService) SendReminder(ctx context.Context, log *models.ReminderLog) error {
	m.sentReminders = append(m.sentReminders, log.ID)
	return nil
}

func (m *mockNotificationService) SendFollowUp(ctx context.Context, log *models.ReminderLog) error {
	m.sentFollowUps = append(m.sentFollowUps, log.ID)
	return nil
}

func TestSchedulerService_CronExpression(t *testing.T) {
	mockReminderRepo := newMockReminderRepository()
	mockLogRepo := newMockReminderLogRepository()
	mockNotification := newMockNotificationService()

	scheduler := NewSchedulerService(mockReminderRepo, mockLogRepo, mockNotification).(*schedulerService)

	tests := []struct {
		name     string
		reminder *models.Reminder
		wantExpr string
		wantErr  bool
	}{
		{
			name: "每日提醒",
			reminder: &models.Reminder{
				SchedulePattern: "daily",
				TargetTime:      "19:30:00",
			},
			wantExpr: "30 19 * * *",
			wantErr:  false,
		},
		{
			name: "每周一三五提醒",
			reminder: &models.Reminder{
				SchedulePattern: "weekly:1,3,5",
				TargetTime:      "08:00:00",
			},
			wantExpr: "00 8 * * 1,3,5",
			wantErr:  false,
		},
		{
			name: "一次性提醒",
			reminder: &models.Reminder{
				SchedulePattern: "once:2025-12-25",
				TargetTime:      "10:30:00",
			},
			wantExpr: "30 10 25 12 *",
			wantErr:  false,
		},
		{
			name: "无效时间格式",
			reminder: &models.Reminder{
				SchedulePattern: "daily",
				TargetTime:      "invalid",
			},
			wantExpr: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := scheduler.buildCronExpression(tt.reminder)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildCronExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && expr != tt.wantExpr {
				t.Errorf("buildCronExpression() = %v, want %v", expr, tt.wantExpr)
			}

			if !tt.wantErr {
				if _, parseErr := cron.ParseStandard(expr); parseErr != nil {
					t.Errorf("cron expression %q invalid: %v", expr, parseErr)
				}
			}
		})
	}
}

func TestSchedulerService_BuildOnceExpression_Timezone(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	scheduler := &schedulerService{
		location: loc,
	}

	now := time.Now().In(loc)
	futureDate := now.Add(24 * time.Hour)

	pattern := fmt.Sprintf("%s%s", string(models.SchedulePatternOnce), futureDate.Format("2006-01-02"))
	expr, err := scheduler.buildOnceExpression(pattern, 10, 30)
	if err != nil {
		t.Fatalf("buildOnceExpression() unexpected error: %v", err)
	}

	expected := fmt.Sprintf("%02d %d %d %d *", 30, 10, futureDate.Day(), int(futureDate.Month()))
	if expr != expected {
		t.Errorf("buildOnceExpression() = %s, want %s", expr, expected)
	}

	pastDate := now.Add(-24 * time.Hour)
	pastPattern := fmt.Sprintf("%s%s", string(models.SchedulePatternOnce), pastDate.Format("2006-01-02"))

	if _, err := scheduler.buildOnceExpression(pastPattern, now.Hour(), now.Minute()); err == nil {
		t.Errorf("buildOnceExpression() expected error for past date, got nil")
	}
}

func TestSchedulerService_WeeklyPattern(t *testing.T) {
	scheduler := &schedulerService{}

	tests := []struct {
		name     string
		pattern  string
		wantDays []string
		wantErr  bool
	}{
		{
			name:     "工作日",
			pattern:  "weekly:1,2,3,4,5",
			wantDays: []string{"1", "2", "3", "4", "5"},
			wantErr:  false,
		},
		{
			name:     "周末",
			pattern:  "weekly:6,7",
			wantDays: []string{"6", "7"},
			wantErr:  false,
		},
		{
			name:     "无效格式",
			pattern:  "daily",
			wantDays: nil,
			wantErr:  true,
		},
		{
			name:     "无效星期数",
			pattern:  "weekly:8,9",
			wantDays: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days, err := scheduler.parseWeeklyPattern(tt.pattern)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseWeeklyPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(days) != len(tt.wantDays) {
					t.Errorf("parseWeeklyPattern() days count = %v, want %v", len(days), len(tt.wantDays))
					return
				}

				for i, day := range days {
					if day != tt.wantDays[i] {
						t.Errorf("parseWeeklyPattern() day[%d] = %v, want %v", i, day, tt.wantDays[i])
					}
				}
			}
		})
	}
}

func TestSchedulerService_AddReminder_Paused(t *testing.T) {
	mockReminderRepo := newMockReminderRepository()
	mockLogRepo := newMockReminderLogRepository()
	mockNotification := newMockNotificationService()

	scheduler := NewSchedulerService(mockReminderRepo, mockLogRepo, mockNotification).(*schedulerService)

	pausedUntil := time.Now().Add(2 * time.Hour)
	reminder := &models.Reminder{
		ID:              100,
		UserID:          1,
		Title:           "测试提醒",
		SchedulePattern: "daily",
		TargetTime:      "12:00:00",
		IsActive:        true,
		PausedUntil:     &pausedUntil,
	}

	if err := scheduler.AddReminder(reminder); err != nil {
		t.Fatalf("AddReminder() unexpected error: %v", err)
	}

	scheduler.mu.RLock()
	_, jobExists := scheduler.jobs[reminder.ID]
	_, timerExists := scheduler.onceTimers[reminder.ID]
	scheduler.mu.RUnlock()

	if jobExists || timerExists {
		t.Fatalf("期待暂停提醒不被调度，jobs=%v timers=%v", jobExists, timerExists)
	}
}

func TestSchedulerService_AddReminder_OnceUsesTimer(t *testing.T) {
	mockReminderRepo := newMockReminderRepository()
	mockLogRepo := newMockReminderLogRepository()
	mockNotification := newMockNotificationService()

	scheduler := NewSchedulerService(mockReminderRepo, mockLogRepo, mockNotification).(*schedulerService)

	loc := scheduler.location
	if loc == nil {
		loc = time.Local
	}
	future := time.Now().In(loc).Add(48 * time.Hour)

	reminder := &models.Reminder{
		ID:              200,
		UserID:          1,
		Title:           "一次性提醒",
		SchedulePattern: fmt.Sprintf("%s%s", string(models.SchedulePatternOnce), future.Format("2006-01-02")),
		TargetTime:      future.Format("15:04:05"),
		IsActive:        true,
	}

	if err := scheduler.AddReminder(reminder); err != nil {
		t.Fatalf("AddReminder() unexpected error: %v", err)
	}

	scheduler.mu.RLock()
	timer, exists := scheduler.onceTimers[reminder.ID]
	scheduler.mu.RUnlock()

	if !exists || timer == nil {
		t.Fatalf("期待一次性提醒创建定时器")
	}

	// 清理
	if err := scheduler.RemoveReminder(reminder.ID); err != nil {
		t.Fatalf("RemoveReminder() unexpected error: %v", err)
	}
}

// Mock repositories for scheduler tests
type mockReminderLogRepository struct {
	logs      map[uint]*models.ReminderLog
	idCounter uint
}

func newMockReminderLogRepository() *mockReminderLogRepository {
	return &mockReminderLogRepository{
		logs:      make(map[uint]*models.ReminderLog),
		idCounter: 1,
	}
}

func (m *mockReminderLogRepository) Create(ctx context.Context, log *models.ReminderLog) error {
	log.ID = m.idCounter
	m.logs[m.idCounter] = log
	m.idCounter++
	return nil
}

func (m *mockReminderLogRepository) GetByID(ctx context.Context, id uint) (*models.ReminderLog, error) {
	log := m.logs[id]
	return log, nil
}

func (m *mockReminderLogRepository) GetByReminderID(ctx context.Context, reminderID uint, limit, offset int) ([]*models.ReminderLog, error) {
	var result []*models.ReminderLog
	for _, log := range m.logs {
		if log.ReminderID == reminderID {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *mockReminderLogRepository) GetPendingLogs(ctx context.Context) ([]*models.ReminderLog, error) {
	var result []*models.ReminderLog
	for _, log := range m.logs {
		if log.Status == models.ReminderStatusPending || log.Status == models.ReminderStatusSent {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *mockReminderLogRepository) Update(ctx context.Context, log *models.ReminderLog) error {
	if existing := m.logs[log.ID]; existing != nil {
		m.logs[log.ID] = log
	}
	return nil
}

func (m *mockReminderLogRepository) Delete(ctx context.Context, id uint) error {
	delete(m.logs, id)
	return nil
}
