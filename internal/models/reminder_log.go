package models

import (
	"time"
)

// ReminderStatus 提醒状态
type ReminderStatus string

const (
	ReminderStatusPending   ReminderStatus = "pending"   // 待发送
	ReminderStatusSent      ReminderStatus = "sent"      // 已发送
	ReminderStatusCompleted ReminderStatus = "completed" // 已完成
	ReminderStatusSkipped   ReminderStatus = "skipped"   // 已跳过
	ReminderStatusOverdue   ReminderStatus = "overdue"   // 已超时
	ReminderStatusCancelled ReminderStatus = "cancelled" // 已取消
)

// ReminderLog 提醒记录模型
type ReminderLog struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ReminderID     uint           `gorm:"not null;index" json:"reminder_id"`
	ScheduledTime  time.Time      `gorm:"not null" json:"scheduled_time"`
	SentTime       *time.Time     `json:"sent_time"`
	Status         ReminderStatus `gorm:"size:20;default:'pending'" json:"status"`
	UserResponse   string         `gorm:"type:text" json:"user_response"`
	ResponseTime   *time.Time     `json:"response_time"`
	FollowUpCount  int            `gorm:"default:0" json:"follow_up_count"`
	CreatedAt      time.Time      `json:"created_at"`

	// 关联关系
	Reminder Reminder `gorm:"foreignKey:ReminderID" json:"reminder,omitempty"`
}

// TableName 指定表名
func (ReminderLog) TableName() string {
	return "reminder_logs"
}

// IsCompleted 检查是否已完成
func (rl *ReminderLog) IsCompleted() bool {
	return rl.Status == ReminderStatusCompleted
}

// IsOverdue 检查是否已超时
func (rl *ReminderLog) IsOverdue() bool {
	return rl.Status == ReminderStatusOverdue
}

// MarkAsSent 标记为已发送
func (rl *ReminderLog) MarkAsSent() {
	now := time.Now()
	rl.Status = ReminderStatusSent
	rl.SentTime = &now
}

// MarkAsCompleted 标记为已完成
func (rl *ReminderLog) MarkAsCompleted(response string) {
	now := time.Now()
	rl.Status = ReminderStatusCompleted
	rl.UserResponse = response
	rl.ResponseTime = &now
}

// MarkAsSkipped 标记为已跳过
func (rl *ReminderLog) MarkAsSkipped(response string) {
	now := time.Now()
	rl.Status = ReminderStatusSkipped
	rl.UserResponse = response
	rl.ResponseTime = &now
}