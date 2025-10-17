package models

import "time"

// ConversationMessage 表示上下文中的单条消息
type ConversationMessage struct {
	Role      string                           `json:"role"`
	Content   string                           `json:"content"`
	Intent    string                           `json:"intent,omitempty"`
	Entities  map[string]ConversationEntityRef `json:"entities,omitempty"`
	Timestamp time.Time                        `json:"timestamp"`
}

// ConversationEntityRef 表示消息中引用的实体
type ConversationEntityRef struct {
	Name       string      `json:"name"`
	Value      interface{} `json:"value"`
	Confidence float64     `json:"confidence,omitempty"`
	Source     string      `json:"source,omitempty"`
}

// ConversationEntity 表示聚合后的实体
type ConversationEntity struct {
	Name       string      `json:"name"`
	Value      interface{} `json:"value"`
	Confidence float64     `json:"confidence,omitempty"`
	Source     string      `json:"source,omitempty"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// ConversationContextState 是对外暴露的上下文状态
type ConversationContextState struct {
	UserID       uint                          `json:"user_id"`
	SessionID    string                        `json:"session_id"`
	State        string                        `json:"state"`
	Intent       string                        `json:"intent"`
	Channel      string                        `json:"channel,omitempty"`
	Locale       string                        `json:"locale,omitempty"`
	Messages     []ConversationMessage         `json:"messages,omitempty"`
	Entities     map[string]ConversationEntity `json:"entities,omitempty"`
	CreatedAt    time.Time                     `json:"created_at"`
	LastActivity time.Time                     `json:"last_activity"`
	TTLSeconds   int64                         `json:"ttl_seconds"`
}

// ConversationContext 数据库存储模型
type ConversationContext struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint       `gorm:"not null;uniqueIndex:idx_context_user" json:"user_id"`
	SessionID    string     `gorm:"size:64;not null" json:"session_id"`
	State        string     `gorm:"size:64;not null" json:"state"`
	Intent       string     `gorm:"size:64;not null" json:"intent"`
	Channel      string     `gorm:"size:32" json:"channel"`
	Locale       string     `gorm:"size:10" json:"locale"`
	MessagesJSON string     `gorm:"type:text;not null" json:"messages"`
	EntitiesJSON string     `gorm:"type:text" json:"entities"`
	TTLSeconds   int64      `gorm:"not null;default:0" json:"ttl_seconds"`
	LastActivity time.Time  `gorm:"not null" json:"last_activity"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

// TableName 指定表名
func (ConversationContext) TableName() string {
	return "conversation_contexts"
}
