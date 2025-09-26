package interfaces

import (
	"context"
	"mmemory/internal/models"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id uint) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
}

// ReminderRepository 提醒仓储接口
type ReminderRepository interface {
	Create(ctx context.Context, reminder *models.Reminder) error
	GetByID(ctx context.Context, id uint) (*models.Reminder, error)
	GetByUserID(ctx context.Context, userID uint) ([]*models.Reminder, error)
	GetActiveReminders(ctx context.Context) ([]*models.Reminder, error)
	Update(ctx context.Context, reminder *models.Reminder) error
	Delete(ctx context.Context, id uint) error
}

// ReminderLogRepository 提醒记录仓储接口
type ReminderLogRepository interface {
	Create(ctx context.Context, log *models.ReminderLog) error
	GetByID(ctx context.Context, id uint) (*models.ReminderLog, error)
	GetByReminderID(ctx context.Context, reminderID uint, limit, offset int) ([]*models.ReminderLog, error)
	GetPendingLogs(ctx context.Context) ([]*models.ReminderLog, error)
	Update(ctx context.Context, log *models.ReminderLog) error
	Delete(ctx context.Context, id uint) error
}

// ConversationRepository 对话仓储接口
type ConversationRepository interface {
	Create(ctx context.Context, conversation *models.Conversation) error
	GetByUserID(ctx context.Context, userID uint, contextType models.ContextType) (*models.Conversation, error)
	Update(ctx context.Context, conversation *models.Conversation) error
	Delete(ctx context.Context, id uint) error
	DeleteExpired(ctx context.Context) error
}