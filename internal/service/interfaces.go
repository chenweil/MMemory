package service

import (
	"context"
	"mmemory/internal/models"
	"time"
)

// UserStatistics 用户统计数据
type UserStatistics struct {
	TotalReminders  int `json:"total_reminders"`  // 总提醒数
	ActiveReminders int `json:"active_reminders"` // 活跃提醒数
	CompletedToday  int `json:"completed_today"`  // 今日完成数
	CompletedWeek   int `json:"completed_week"`   // 本周完成数
	CompletedMonth  int `json:"completed_month"`  // 本月完成数
	SkippedToday    int `json:"skipped_today"`    // 今日跳过数
	CompletionRate  int `json:"completion_rate"`  // 完成率 (百分比)
	LongestStreak   int `json:"longest_streak"`   // 最长连续完成天数
	CurrentStreak   int `json:"current_streak"`   // 当前连续完成天数
}

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
	GetReminderByID(ctx context.Context, id uint) (*models.Reminder, error)
	UpdateReminder(ctx context.Context, reminder *models.Reminder) error
	DeleteReminder(ctx context.Context, id uint) error
	PauseReminder(ctx context.Context, id uint, duration time.Duration, reason string) error
	ResumeReminder(ctx context.Context, id uint) error
}

// ReminderLogService 提醒记录服务接口
type ReminderLogService interface {
	GetByID(ctx context.Context, id uint) (*models.ReminderLog, error)
	MarkAsCompleted(ctx context.Context, id uint, response string) error
	MarkAsSkipped(ctx context.Context, id uint, response string) error
	CreateDelayReminder(ctx context.Context, originalLogID uint, delayTime time.Time, hours int) error
	GetOverdueReminders(ctx context.Context) ([]*models.ReminderLog, error)
	UpdateFollowUpCount(ctx context.Context, id uint) error
	GetUserStatistics(ctx context.Context, userID uint) (*UserStatistics, error)
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

// ConversationService 对话服务接口
type ConversationService interface {
	// CreateConversation 创建对话上下文
	CreateConversation(ctx context.Context, userID uint, contextType models.ContextType, contextData interface{}, ttl time.Duration) (*models.Conversation, error)

	// GetConversation 获取用户对话上下文
	GetConversation(ctx context.Context, userID uint, contextType models.ContextType) (*models.Conversation, error)

	// UpdateConversation 更新对话上下文
	UpdateConversation(ctx context.Context, conversation *models.Conversation, contextData interface{}) error

	// ClearConversation 清除对话上下文
	ClearConversation(ctx context.Context, userID uint, contextType models.ContextType) error

	// IsConversationActive 检查对话是否活跃
	IsConversationActive(ctx context.Context, userID uint, contextType models.ContextType) (bool, error)

	// CleanupExpiredConversations 清理过期对话
	CleanupExpiredConversations(ctx context.Context) error

	// GetContextData 获取上下文数据
	GetContextData(ctx context.Context, userID uint, contextType models.ContextType, target interface{}) error
}
