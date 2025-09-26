package models

import (
	"time"
)

// ContextType 对话上下文类型
type ContextType string

const (
	ContextTypeCreatingReminder   ContextType = "creating_reminder"   // 创建提醒中
	ContextTypeRespondingReminder ContextType = "responding_reminder" // 回复提醒中
	ContextTypeEditingReminder    ContextType = "editing_reminder"    // 编辑提醒中
)

// Conversation 对话上下文模型
type Conversation struct {
	ID          uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint        `gorm:"not null;index" json:"user_id"`
	ContextType ContextType `gorm:"size:50;not null" json:"context_type"`
	ContextData string      `gorm:"type:text" json:"context_data"` // JSON 格式存储上下文信息
	ExpiresAt   *time.Time  `json:"expires_at"`
	CreatedAt   time.Time   `json:"created_at"`

	// 关联关系
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (Conversation) TableName() string {
	return "conversations"
}

// IsExpired 检查是否已过期
func (c *Conversation) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return c.ExpiresAt.Before(time.Now())
}

// SetExpiry 设置过期时间
func (c *Conversation) SetExpiry(duration time.Duration) {
	expiry := time.Now().Add(duration)
	c.ExpiresAt = &expiry
}