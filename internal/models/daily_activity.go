package models

import (
	"encoding/json"
	"time"
)

// ActivityType 活动类型
type ActivityType string

const (
	ActivityTypeDrinkWater  ActivityType = "drink_water"  // 喝水
	ActivityTypeTakeMedicine ActivityType = "take_medicine" // 吃药
	ActivityTypeReadBook    ActivityType = "read_book"    // 看书
	ActivityTypeExercise    ActivityType = "exercise"     // 运动
	ActivityTypeSleep       ActivityType = "sleep"        // 睡眠
	ActivityTypeEat         ActivityType = "eat"          // 吃饭
	ActivityTypeCustom      ActivityType = "custom"       // 自定义
)

// ActivitySource 记录来源
type ActivitySource string

const (
	SourceConversation ActivitySource = "conversation" // 对话提取
	SourceManual       ActivitySource = "manual"       // 手动记录
	SourceReminder     ActivitySource = "reminder"     // 提醒关联
)

// DailyActivity 日常活动记录
type DailyActivity struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint           `gorm:"not null;index" json:"user_id"`
	ActivityType ActivityType   `gorm:"size:50;not null;index" json:"activity_type"`
	OccurredAt   time.Time      `gorm:"not null;index" json:"occurred_at"`
	Details      string         `gorm:"type:text" json:"details"` // JSON 格式
	Source       ActivitySource `gorm:"size:20;not null;default:'conversation'" json:"source"`
	Metadata     string         `gorm:"type:text" json:"metadata"` // JSON 格式
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// 关联关系
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (DailyActivity) TableName() string {
	return "daily_activities"
}

// ActivityDetails 活动详情结构
type ActivityDetails struct {
	// 喝水
	Amount string `json:"amount,omitempty"` // "200ml", "1杯"
	WaterType string `json:"water_type,omitempty"`   // "温水", "凉水"

	// 吃药
	MedicineName string `json:"medicine_name,omitempty"` // "阿莫西林"
	Dosage       string `json:"dosage,omitempty"`       // "1粒"
	Frequency    string `json:"frequency,omitempty"`    // "每日3次"

	// 看书
	BookName string `json:"book_name,omitempty"` // "如何阅读一本书"
	Chapter  string `json:"chapter,omitempty"`  // "第十一章"
	Page     string `json:"page,omitempty"`     // "150"

	// 运动
	ExerciseType string `json:"exercise_type,omitempty"`     // "跑步", "健身"
	ExerciseDuration string `json:"exercise_duration,omitempty"` // "30分钟"
	Distance string `json:"distance,omitempty"` // "5公里"

	// 睡眠
	SleepDuration string `json:"sleep_duration,omitempty"` // "8小时"
	Quality  string `json:"quality,omitempty"`  // "良好", "一般"

	// 吃饭
	MealType     string `json:"meal_type,omitempty"`     // "早餐", "午餐", "晚餐"
	Content  string `json:"content,omitempty"`  // "米饭", "面条"
	Calories string `json:"calories,omitempty"` // "500卡"

	// 自定义字段
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// GetDetails 解析详情 JSON
func (da *DailyActivity) GetDetails() (*ActivityDetails, error) {
	if da.Details == "" {
		return &ActivityDetails{}, nil
	}
	var details ActivityDetails
	err := json.Unmarshal([]byte(da.Details), &details)
	return &details, err
}

// SetDetails 设置详情 JSON
func (da *DailyActivity) SetDetails(details *ActivityDetails) error {
	data, err := json.Marshal(details)
	if err != nil {
		return err
	}
	da.Details = string(data)
	return nil
}