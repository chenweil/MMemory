package models

import (
	"time"
)

// AIParseResult AI解析结果的数据库模型
type AIParseResult struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index" json:"user_id"`
	Message   string    `gorm:"type:text" json:"message"`
	
	// 解析结果
	Intent      string  `json:"intent"`
	Confidence  float32 `json:"confidence"`
	ParsedBy    string  `json:"parsed_by"`
	ProcessTime int64   `json:"process_time"` // 微秒
	
	// 提醒信息 (JSON存储)
	ReminderData string `gorm:"type:text" json:"reminder_data,omitempty"`
	
	// 对话信息 (JSON存储)
	ChatData string `gorm:"type:text" json:"chat_data,omitempty"`
	
	// 原始AI响应
	RawResponse string `gorm:"type:text" json:"raw_response,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MessageRole 消息角色常量
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
)

// MessageType 消息类型常量
const (
	MessageTypeText     = "text"
	MessageTypeReminder = "reminder"
	MessageTypeChat     = "chat"
	MessageTypeSummary  = "summary"
	MessageTypeQuery    = "query"
)