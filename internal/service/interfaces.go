package service

import (
	"context"
	"mmemory/internal/models"
)

// UserService 用户服务接口
type UserService interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id uint) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
}

// ReminderService 提醒服务接口
type ReminderService interface {
	CreateReminder(ctx context.Context, reminder *models.Reminder) error
	ParseReminderFromText(ctx context.Context, text string, userID uint) (*models.Reminder, error)
	GetUserReminders(ctx context.Context, userID uint) ([]*models.Reminder, error)
	UpdateReminder(ctx context.Context, reminder *models.Reminder) error
	DeleteReminder(ctx context.Context, id uint) error
}

// SchedulerService 调度服务接口
type SchedulerService interface {
	Start() error
	Stop() error
	AddReminder(reminder *models.Reminder) error
	RemoveReminder(reminderID uint) error
	RefreshSchedules() error
}

// NotificationService 通知服务接口
type NotificationService interface {
	SendReminder(ctx context.Context, log *models.ReminderLog) error
	SendFollowUp(ctx context.Context, log *models.ReminderLog) error
}