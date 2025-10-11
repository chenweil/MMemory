package models

import (
	"strings"
	"time"
)

// ReminderType 提醒类型
type ReminderType string

const (
	ReminderTypeHabit ReminderType = "habit" // 习惯提醒
	ReminderTypeTask  ReminderType = "task"  // 任务提醒
)

// ReminderStatus 提醒状态（用于统计）
type ReminderStatStatus string

const (
	ReminderStatStatusActive    ReminderStatStatus = "active"    // 活跃
	ReminderStatStatusCompleted ReminderStatStatus = "completed" // 已完成
	ReminderStatStatusExpired   ReminderStatStatus = "expired"   // 已过期
)

// SchedulePattern 调度模式
type SchedulePattern string

const (
	SchedulePatternDaily   SchedulePattern = "daily"   // 每天
	SchedulePatternWeekly  SchedulePattern = "weekly"  // 每周，格式: weekly:1,3,5
	SchedulePatternMonthly SchedulePattern = "monthly" // 每月，格式: monthly:1,15
	SchedulePatternOnce    SchedulePattern = "once:"   // 一次性前缀，格式: once:2024-10-01
)

// Reminder 提醒配置模型
type Reminder struct {
	ID              uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint         `gorm:"not null;index" json:"user_id"`
	Title           string       `gorm:"size:500;not null" json:"title"`
	Description     string       `gorm:"type:text" json:"description"`
	Type            ReminderType `gorm:"size:20;not null" json:"type"`
	SchedulePattern string       `gorm:"size:100;not null" json:"schedule_pattern"`
	TargetTime      string       `gorm:"size:8;not null" json:"target_time"` // HH:MM:SS 格式
	Timezone        string       `gorm:"size:50" json:"timezone"`
	IsActive        bool         `gorm:"default:true" json:"is_active"`
	PausedUntil     *time.Time   `gorm:"index" json:"paused_until,omitempty"`
	PauseReason     string       `gorm:"type:text" json:"pause_reason,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`

	// 关联关系
	User         User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ReminderLogs []ReminderLog `gorm:"foreignKey:ReminderID" json:"reminder_logs,omitempty"`
}

// TableName 指定表名
func (Reminder) TableName() string {
	return "reminders"
}

// IsDaily 检查是否为每日提醒
func (r *Reminder) IsDaily() bool {
	return r.SchedulePattern == string(SchedulePatternDaily)
}

// IsWeekly 检查是否为每周提醒
func (r *Reminder) IsWeekly() bool {
	return len(r.SchedulePattern) > 7 && r.SchedulePattern[:7] == "weekly:"
}

// IsOnce 检查是否为一次性提醒
func (r *Reminder) IsOnce() bool {
	return strings.HasPrefix(r.SchedulePattern, string(SchedulePatternOnce)) &&
		len(r.SchedulePattern) > len(string(SchedulePatternOnce))
}

// IsPaused 检查是否处于暂停状态
func (r *Reminder) IsPaused() bool {
	if r.PausedUntil == nil {
		return false
	}
	return time.Now().Before(*r.PausedUntil)
}
