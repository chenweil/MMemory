package service

import (
	"context"
	"testing"

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
			wantExpr: "0 30 19 * * *",
			wantErr:  false,
		},
		{
			name: "每周一三五提醒",
			reminder: &models.Reminder{
				SchedulePattern: "weekly:1,3,5",
				TargetTime:      "08:00:00",
			},
			wantExpr: "0 0 8 * * 1,3,5",
			wantErr:  false,
		},
		{
			name: "一次性提醒",
			reminder: &models.Reminder{
				SchedulePattern: "once:2025-12-25",
				TargetTime:      "10:30:00",
			},
			wantExpr: "0 30 10 25 12 *",
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
		})
	}
}

func TestSchedulerService_WeeklyPattern(t *testing.T) {
	scheduler := &schedulerService{}

	tests := []struct {
		name        string
		pattern     string
		wantDays    []string
		wantErr     bool
	}{
		{
			name:        "工作日",
			pattern:     "weekly:1,2,3,4,5",
			wantDays:    []string{"1", "2", "3", "4", "5"},
			wantErr:     false,
		},
		{
			name:        "周末",
			pattern:     "weekly:6,7",
			wantDays:    []string{"6", "7"},
			wantErr:     false,
		},
		{
			name:        "无效格式",
			pattern:     "daily",
			wantDays:    nil,
			wantErr:     true,
		},
		{
			name:        "无效星期数",
			pattern:     "weekly:8,9",
			wantDays:    nil,
			wantErr:     true,
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