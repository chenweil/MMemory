package interfaces

import (
	"context"
	"time"

	"mmemory/internal/models"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id uint) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
	Count(ctx context.Context) (int64, error)
}

// ReminderRepository 提醒仓储接口
type ReminderRepository interface {
	Create(ctx context.Context, reminder *models.Reminder) error
	GetByID(ctx context.Context, id uint) (*models.Reminder, error)
	GetByUserID(ctx context.Context, userID uint) ([]*models.Reminder, error)
	GetActiveReminders(ctx context.Context) ([]*models.Reminder, error)
	Update(ctx context.Context, reminder *models.Reminder) error
	Delete(ctx context.Context, id uint) error
	CountByStatus(ctx context.Context, status models.ReminderStatStatus) (int64, error)
}

// ReminderLogRepository 提醒记录仓储接口
type ReminderLogRepository interface {
	Create(ctx context.Context, log *models.ReminderLog) error
	GetByID(ctx context.Context, id uint) (*models.ReminderLog, error)
	GetByReminderID(ctx context.Context, reminderID uint, limit, offset int) ([]*models.ReminderLog, error)
	GetPendingLogs(ctx context.Context) ([]*models.ReminderLog, error)
	Update(ctx context.Context, log *models.ReminderLog) error
	Delete(ctx context.Context, id uint) error
	GetUserLogs(ctx context.Context, userID uint, since time.Time) ([]*models.ReminderLog, error)
	CreateDelayReminder(ctx context.Context, originalLogID uint, delayTime time.Time, delayHours int) error
	MarkAsCompleted(ctx context.Context, logID uint, note string) error
	MarkAsSkipped(ctx context.Context, logID uint, note string) error
	UpdateFollowUpCount(ctx context.Context, logID uint) error
	GetOverdueReminders(ctx context.Context) ([]*models.ReminderLog, error)
}

// ConversationRepository 对话仓储接口
type ConversationRepository interface {
	Create(ctx context.Context, conversation *models.Conversation) error
	GetByUserID(ctx context.Context, userID uint, contextType models.ContextType) (*models.Conversation, error)
	Update(ctx context.Context, conversation *models.Conversation) error
	Delete(ctx context.Context, id uint) error
	DeleteExpired(ctx context.Context) error
}

// ConversationContextRepository 对话上下文仓储接口
type ConversationContextRepository interface {
	GetByUserID(ctx context.Context, userID uint) (*models.ConversationContext, error)
	CreateOrUpdate(ctx context.Context, ctxModel *models.ConversationContext) error
	DeleteByUserID(ctx context.Context, userID uint) error
	CleanupExpired(ctx context.Context, now time.Time) error
}

// DailyActivityRepository 日常活动仓储接口
type DailyActivityRepository interface {
	Create(ctx context.Context, activity *models.DailyActivity) error
	GetByID(ctx context.Context, id uint) (*models.DailyActivity, error)
	GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*models.DailyActivity, error)
	GetByType(ctx context.Context, userID uint, activityType models.ActivityType, limit, offset int) ([]*models.DailyActivity, error)
	GetByDateRange(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*models.DailyActivity, error)
	GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error)
	Update(ctx context.Context, activity *models.DailyActivity) error
	Delete(ctx context.Context, id uint) error
	GetStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (map[string]int64, error)
}

// ConversationArchiveRepository 存档仓库接口
type ConversationArchiveRepository interface {
	// Create 创建存档
	Create(ctx context.Context, archive *models.ConversationArchive) error

	// GetByUserID 获取用户的所有存档
	GetByUserID(ctx context.Context, userID uint, limit int, offset int) ([]*models.ConversationArchive, error)

	// GetByID 根据ID获取存档
	GetByID(ctx context.Context, id uint) (*models.ConversationArchive, error)

	// Delete 删除存档
	Delete(ctx context.Context, id uint) error

	// DeleteExpired 删除过期存档
	DeleteExpired(ctx context.Context) (int64, error)

	// CountByUserID 统计用户存档数量
	CountByUserID(ctx context.Context, userID uint) (int64, error)
}
