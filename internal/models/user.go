package models

import (
	"time"
)

// User 用户模型
type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TelegramID   int64     `gorm:"uniqueIndex;not null" json:"telegram_id"`
	Username     string    `gorm:"size:255" json:"username"`
	FirstName    string    `gorm:"size:255" json:"first_name"`
	LastName     string    `gorm:"size:255" json:"last_name"`
	Timezone     string    `gorm:"size:50;default:'Asia/Shanghai'" json:"timezone"`
	LanguageCode string    `gorm:"size:10;default:'zh-CN'" json:"language_code"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联关系
	Reminders     []Reminder     `gorm:"foreignKey:UserID" json:"reminders,omitempty"`
	Conversations []Conversation `gorm:"foreignKey:UserID" json:"conversations,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}