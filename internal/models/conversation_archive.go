package models

import (
	"encoding/json"
	"time"
)

// ArchiveType 存档类型
type ArchiveType string

const (
	ArchiveTypeFull    ArchiveType = "full"
	ArchiveTypeSummary ArchiveType = "summary"
)

// String 返回类型的字符串表示
// Patch 1: 在模型定义中添加此方法，避免在Task 5重复添加
func (t ArchiveType) String() string {
	return string(t)
}

// ConversationArchive 对话存档
type ConversationArchive struct {
	ID         uint        `gorm:"primarykey"`
	UserID     uint        `gorm:"not null;index:idx_archives_user_id"`
	ArchiveType ArchiveType `gorm:"not null;index:idx_archives_type;size:20"`

	// 内容
	Content string `gorm:"type:text"` // 完整内容
	Summary string `gorm:"type:text"` // AI摘要

	// 关键信息 (JSON)
	KeyEntities string `gorm:"type:text"` // JSON: 关键实体

	// 元信息
	ImportanceScore float64 `gorm:"default:0.5"`
	MessageCount    int     `gorm:"default:1"`
	DateRangeStart  *time.Time `gorm:"index"`
	DateRangeEnd    *time.Time `gorm:"index"`

	// 时间戳
	CreatedAt time.Time  `gorm:"autoCreateTime;index:idx_archives_created"`
	ExpiresAt *time.Time `gorm:"index"`

	// 关联
	User *User `gorm:"foreignKey:UserID"`
}

// KeyEntities 关键实体结构
type KeyEntities struct {
	BookName string   `json:"book_name,omitempty"`
	Topic    string   `json:"topic,omitempty"`
	Insights []string `json:"insights,omitempty"`
	People   []string `json:"people,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
}

// GetKeyEntities 获取关键实体
func (a *ConversationArchive) GetKeyEntities() (*KeyEntities, error) {
	if a.KeyEntities == "" {
		return &KeyEntities{}, nil
	}

	var entities KeyEntities
	err := json.Unmarshal([]byte(a.KeyEntities), &entities)
	if err != nil {
		return nil, err
	}

	return &entities, nil
}

// SetKeyEntities 设置关键实体
func (a *ConversationArchive) SetKeyEntities(entities *KeyEntities) error {
	data, err := json.Marshal(entities)
	if err != nil {
		return err
	}

	a.KeyEntities = string(data)
	return nil
}

// IsExpired 检查是否过期
func (a *ConversationArchive) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// SetExpiry 设置过期时间
func (a *ConversationArchive) SetExpiry(duration time.Duration) {
	if duration == 0 {
		// 0表示永不过期
		a.ExpiresAt = nil
		return
	}

	expiry := time.Now().Add(duration)
	a.ExpiresAt = &expiry
}

// TableName 指定表名
func (ConversationArchive) TableName() string {
	return "conversation_archives"
}
