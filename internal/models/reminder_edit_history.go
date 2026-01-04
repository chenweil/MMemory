package models

import "time"

// ReminderEditHistory 提醒编辑历史模型
type ReminderEditHistory struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ReminderID      uint      `gorm:"not null;index" json:"reminder_id"`
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	FieldName       string    `gorm:"size:50;not null" json:"field_name"`       // time, pattern, title, description
	OldValue        string    `gorm:"type:text" json:"old_value"`               // 修改前的值
	NewValue        string    `gorm:"type:text" json:"new_value"`               // 修改后的值
	EditType        string    `gorm:"size:20;not null" json:"edit_type"`        // manual, ai, button
	EditReason      string    `gorm:"type:text" json:"edit_reason"`             // 编辑原因（可选）
	CreatedAt       time.Time `json:"created_at"`

	// 关联关系
	Reminder Reminder `gorm:"foreignKey:ReminderID" json:"reminder,omitempty"`
}

// TableName 指定表名
func (ReminderEditHistory) TableName() string {
	return "reminder_edit_histories"
}