package models

import (
	"encoding/json"
	"time"
)

// ConditionType 条件类型
type ConditionType string

const (
	ConditionTypeWeather     ConditionType = "weather"     // 天气条件
	ConditionTypeTime        ConditionType = "time"        // 时间条件
	ConditionTypeDayOfWeek   ConditionType = "day_of_week" // 星期条件
	ConditionTypeTemperature ConditionType = "temperature" // 温度条件
	ConditionTypeLocation    ConditionType = "location"    // 位置条件
	ConditionTypeCustom      ConditionType = "custom"      // 自定义条件
)

// ConditionOperator 条件操作符
type ConditionOperator string

const (
	ConditionOpEquals    ConditionOperator = "eq"    // 等于
	ConditionOpNotEquals ConditionOperator = "ne"    // 不等于
	ConditionOpGreater   ConditionOperator = "gt"    // 大于
	ConditionOpLess      ConditionOperator = "lt"    // 小于
	ConditionOpContains  ConditionOperator = "contains" // 包含
	ConditionOpIn        ConditionOperator = "in"    // 在列表中
)

// ReminderCondition 提醒条件模型
type ReminderCondition struct {
	ID           uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	ReminderID   uint            `gorm:"not null;index" json:"reminder_id"`
	Type         ConditionType   `gorm:"size:20;not null" json:"type"`
	Operator     ConditionOperator `gorm:"size:20;not null" json:"operator"`
	Value        string          `gorm:"type:text" json:"value"`       // JSON格式的值
	ValueType    string          `gorm:"size:20" json:"value_type"`    // string, number, boolean, array
	Field        string          `gorm:"size:50" json:"field"`         // 条件字段，如 weather, temp, humidity
	Description  string          `gorm:"size:200" json:"description"`
	Priority     int             `gorm:"default:0" json:"priority"`    // 条件优先级
	IsNegated    bool            `gorm:"default:false" json:"is_negated"` // 是否取反
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`

	// 关联关系
	Reminder Reminder `gorm:"foreignKey:ReminderID" json:"reminder,omitempty"`
}

// TableName 指定表名
func (ReminderCondition) TableName() string {
	return "reminder_conditions"
}

// GetValue 获取解析后的值
func (rc *ReminderCondition) GetValue() interface{} {
	switch rc.ValueType {
	case "number":
		var num float64
		if err := json.Unmarshal([]byte(rc.Value), &num); err == nil {
			return num
		}
	case "boolean":
		if rc.Value == "true" {
			return true
		}
		return false
	case "array":
		var arr []string
		if err := json.Unmarshal([]byte(rc.Value), &arr); err == nil {
			return arr
		}
	}
	return rc.Value
}

// SetValue 设置值（自动推断类型）
func (rc *ReminderCondition) SetValue(v interface{}) error {
	switch val := v.(type) {
	case float64:
		rc.ValueType = "number"
		rc.Value = jsonString(val)
	case int:
		rc.ValueType = "number"
		rc.Value = jsonString(float64(val))
	case bool:
		rc.ValueType = "boolean"
		rc.Value = jsonString(val)
	case []string:
		rc.ValueType = "array"
		rc.Value = jsonString(val)
	case string:
		rc.ValueType = "string"
		rc.Value = val
	default:
		rc.ValueType = "string"
		rc.Value = jsonString(val)
	}
	return nil
}

func jsonString(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// ConditionalReminder 条件提醒结构（用于内存中处理）
type ConditionalReminder struct {
	BaseReminder Reminder
	Conditions   []ReminderCondition
	Evaluator    string
}

// ConditionEvaluationResult 条件评估结果
type ConditionEvaluationResult struct {
	ConditionID    uint      `json:"condition_id"`
	Type           string    `json:"type"`
	Met            bool      `json:"met"`
	Value          interface{} `json:"value"`
	Expected       interface{} `json:"expected"`
	Operator       string    `json:"operator"`
	Description    string    `json:"description"`
	CheckedAt      time.Time `json:"checked_at"`
	Details        string    `json:"details,omitempty"`
}

// ConditionEvaluationStatus 条件评估状态
type ConditionEvaluationStatus struct {
	ReminderID       uint                      `json:"reminder_id"`
	AllConditionsMet bool                      `json:"all_conditions_met"`
	Results          []ConditionEvaluationResult `json:"results"`
	NextCheckTime    *time.Time                `json:"next_check_time,omitempty"`
	LastCheckedAt    time.Time                 `json:"last_checked_at"`
	FailedConditions []string                  `json:"failed_conditions,omitempty"`
}

// WeatherConditionData 天气条件数据
type WeatherConditionData struct {
	Location     string  `json:"location"`
	WeatherCode  string  `json:"weather_code"`
	Temperature  float64 `json:"temperature"`
	Humidity     float64 `json:"humidity"`
	WindSpeed    float64 `json:"wind_speed"`
	Condition    string  `json:"condition"`
	Precip       float64 `json:"precip"`
	CloudCover   float64 `json:"cloud_cover"`
	Visibility   float64 `json:"visibility"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// TimeConditionData 时间条件数据
type TimeConditionData struct {
	CurrentTime time.Time `json:"current_time"`
	Hour        int       `json:"hour"`
	Minute      int       `json:"minute"`
	DayOfWeek   int       `json:"day_of_week"`
	DayOfMonth  int       `json:"day_of_month"`
	Month       int       `json:"month"`
	Date        string    `json:"date"` // YYYY-MM-DD
	Time        string    `json:"time"` // HH:MM:SS
}

// LocationConditionData 位置条件数据
type LocationConditionData struct {
	LocationID   string    `json:"location_id"`
	Name         string    `json:"name"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Timezone     string    `json:"timezone"`
	LastUpdated  time.Time `json:"last_updated"`
}
