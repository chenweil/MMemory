package ai

import (
	"strings"
	"time"

	"mmemory/internal/models"
)

// ParseIntent 解析意图类型
type ParseIntent string

const (
	IntentReminder ParseIntent = "reminder" // 创建提醒
	IntentDelete   ParseIntent = "delete"   // 删除提醒
	IntentEdit     ParseIntent = "edit"     // 编辑提醒
	IntentPause    ParseIntent = "pause"    // 暂停提醒
	IntentResume   ParseIntent = "resume"   // 恢复提醒
	IntentChat     ParseIntent = "chat"     // 普通对话
	IntentSummary  ParseIntent = "summary"  // 总结请求
	IntentQuery    ParseIntent = "query"    // 查询提醒
	IntentUnknown  ParseIntent = "unknown"  // 未知意图
)

// IsValid 检查意图是否有效
func (i ParseIntent) IsValid() bool {
	switch i {
	case IntentReminder,
		IntentDelete,
		IntentEdit,
		IntentPause,
		IntentResume,
		IntentChat,
		IntentSummary,
		IntentQuery,
		IntentUnknown:
		return true
	default:
		return false
	}
}

// String 返回意图的字符串表示
func (i ParseIntent) String() string {
	return string(i)
}

// ParseResult 解析结果统一结构
type ParseResult struct {
	// 解析意图
	Intent     ParseIntent `json:"intent"`
	Confidence float32     `json:"confidence"` // 0.0-1.0

	// 提醒相关（当Intent为reminder时）
	Reminder *ReminderInfo `json:"reminder,omitempty"`

	// 删除相关
	Delete *DeleteInfo `json:"delete,omitempty"`

	// 编辑相关
	Edit *EditInfo `json:"edit,omitempty"`

	// 暂停相关
	Pause *PauseInfo `json:"pause,omitempty"`

	// 恢复相关
	Resume *ResumeInfo `json:"resume,omitempty"`

	// 对话相关（当Intent为chat时）
	ChatResponse *ChatInfo `json:"chat_response,omitempty"`

	// 元信息
	ParsedBy    string        `json:"parsed_by"` // "openai-gpt-4"
	ProcessTime time.Duration `json:"process_time"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ReminderInfo 提醒信息结构
type ReminderInfo struct {
	Title           string                 `json:"title"`
	Type            models.ReminderType    `json:"type"`
	Time            TimeInfo               `json:"time"`
	SchedulePattern models.SchedulePattern `json:"schedule_pattern"`
	Description     string                 `json:"description,omitempty"`
}

// TimeInfo 时间信息结构
type TimeInfo struct {
	Hour            int    `json:"hour"`
	Minute          int    `json:"minute"`
	Timezone        string `json:"timezone"`
	ScheduleDetails string `json:"schedule_details,omitempty"` // "weekly:1,3,5"
	IsRelativeTime  bool   `json:"is_relative_time"`           // 是否为相对时间
	RelativeDesc    string `json:"relative_desc,omitempty"`    // "明天", "下周一"
}

// ChatInfo 对话信息结构
type ChatInfo struct {
	Response       string `json:"response"`
	NeedFollowUp   bool   `json:"need_follow_up"`
	FollowUpPrompt string `json:"follow_up_prompt,omitempty"`
}

// DeleteInfo 删除提醒信息
type DeleteInfo struct {
	Keywords []string `json:"keywords"`
	Criteria string   `json:"criteria"`
	Reason   string   `json:"reason,omitempty"`
}

// EditInfo 编辑提醒信息
type EditInfo struct {
	Keywords   []string  `json:"keywords"`
	NewTime    *TimeInfo `json:"new_time,omitempty"`
	NewPattern string    `json:"new_pattern,omitempty"`
	NewTitle   string    `json:"new_title,omitempty"`
	NewText    string    `json:"new_text,omitempty"`
}

// PauseInfo 暂停提醒信息
type PauseInfo struct {
	Keywords []string `json:"keywords"`
	Duration string   `json:"duration"`
	Reason   string   `json:"reason,omitempty"`
}

// ResumeInfo 恢复提醒信息
type ResumeInfo struct {
	Keywords []string `json:"keywords"`
}

// ChatResponse 对话响应（用于Chat接口）
type ChatResponse struct {
	Response    string        `json:"response"`
	ParsedBy    string        `json:"parsed_by"`
	ProcessTime time.Duration `json:"process_time"`
	Timestamp   time.Time     `json:"timestamp"`
}

// PromptContext Prompt模板上下文
type PromptContext struct {
	Message             string `json:"message"`
	CurrentTime         string `json:"current_time"`
	ConversationHistory string `json:"conversation_history,omitempty"`
	UserID              string `json:"user_id,omitempty"`
}

// AIProvider AI服务提供商
type AIProvider string

const (
	ProviderOpenAI AIProvider = "openai"
	ProviderClaude AIProvider = "claude" // 未来扩展用
)

// ParserType 解析器类型
type ParserType string

const (
	ParserTypePrimaryAI ParserType = "primary_ai" // 主要AI解析器
	ParserTypeBackupAI  ParserType = "backup_ai"  // 兜底AI解析器
	ParserTypeRegex     ParserType = "regex"      // 正则解析器
	ParserTypeFallback  ParserType = "fallback"   // 兜底对话
)

// String 返回解析器类型的字符串表示
func (p ParserType) String() string {
	return string(p)
}

// Priority 返回解析器的优先级（数字越小优先级越高）
func (p ParserType) Priority() int {
	switch p {
	case ParserTypePrimaryAI:
		return 1
	case ParserTypeBackupAI:
		return 2
	case ParserTypeRegex:
		return 3
	case ParserTypeFallback:
		return 4
	default:
		return 999
	}
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`
}

// IsHighConfidence 检查置信度是否足够高
func (pr *ParseResult) IsHighConfidence() bool {
	return pr.Confidence >= 0.8
}

// IsMediumConfidence 检查置信度是否中等
func (pr *ParseResult) IsMediumConfidence() bool {
	return pr.Confidence >= 0.5 && pr.Confidence < 0.8
}

// IsLowConfidence 检查置信度是否偏低
func (pr *ParseResult) IsLowConfidence() bool {
	return pr.Confidence < 0.5
}

// Validate 验证解析结果
func (pr *ParseResult) Validate() ValidationResult {
	var errors []string

	// 检查意图是否有效
	if !pr.Intent.IsValid() {
		errors = append(errors, "invalid intent")
	}

	// 检查置信度范围
	if pr.Confidence < 0.0 || pr.Confidence > 1.0 {
		errors = append(errors, "confidence must be between 0.0 and 1.0")
	}

	// 根据意图验证相关字段
	switch pr.Intent {
	case IntentReminder:
		if pr.Reminder == nil {
			errors = append(errors, "reminder info is required for reminder intent")
		} else {
			errors = append(errors, pr.validateReminderInfo()...)
		}
	case IntentDelete:
		if pr.Delete == nil {
			errors = append(errors, "delete info is required for delete intent")
		} else {
			errors = append(errors, pr.validateDeleteInfo()...)
		}
	case IntentEdit:
		if pr.Edit == nil {
			errors = append(errors, "edit info is required for edit intent")
		} else {
			errors = append(errors, pr.validateEditInfo()...)
		}
	case IntentPause:
		if pr.Pause == nil {
			errors = append(errors, "pause info is required for pause intent")
		} else {
			errors = append(errors, pr.validatePauseInfo()...)
		}
	case IntentResume:
		if pr.Resume == nil {
			errors = append(errors, "resume info is required for resume intent")
		} else {
			errors = append(errors, pr.validateResumeInfo()...)
		}
	case IntentChat:
		if pr.ChatResponse == nil {
			errors = append(errors, "chat response is required for chat intent")
		} else if pr.ChatResponse.Response == "" {
			errors = append(errors, "chat response cannot be empty")
		}
	}

	return ValidationResult{
		IsValid: len(errors) == 0,
		Errors:  errors,
	}
}

// validateReminderInfo 验证提醒信息
func (pr *ParseResult) validateReminderInfo() []string {
	var errors []string
	r := pr.Reminder

	if r.Title == "" {
		errors = append(errors, "reminder title cannot be empty")
	}

	if r.Type != models.ReminderTypeHabit && r.Type != models.ReminderTypeTask {
		errors = append(errors, "invalid reminder type")
	}

	// 验证时间信息
	if r.Time.Hour < 0 || r.Time.Hour > 23 {
		errors = append(errors, "hour must be between 0 and 23")
	}

	if r.Time.Minute < 0 || r.Time.Minute > 59 {
		errors = append(errors, "minute must be between 0 and 59")
	}

	if r.Time.Timezone == "" {
		errors = append(errors, "timezone is required")
	}

	return errors
}

func (pr *ParseResult) validateDeleteInfo() []string {
	var errors []string

	if pr.Delete == nil {
		return []string{"delete info is missing"}
	}

	if len(filterEmpty(pr.Delete.Keywords)) == 0 && strings.TrimSpace(pr.Delete.Criteria) == "" {
		errors = append(errors, "delete keywords or criteria required")
	}

	return errors
}

func (pr *ParseResult) validateEditInfo() []string {
	var errors []string

	if pr.Edit == nil {
		return []string{"edit info is missing"}
	}

	if len(filterEmpty(pr.Edit.Keywords)) == 0 {
		errors = append(errors, "edit keywords required")
	}

	if pr.Edit.NewTime == nil && pr.Edit.NewPattern == "" && pr.Edit.NewTitle == "" && pr.Edit.NewText == "" {
		errors = append(errors, "edit requires at least one field to update")
	}

	return errors
}

func (pr *ParseResult) validatePauseInfo() []string {
	var errors []string

	if pr.Pause == nil {
		return []string{"pause info is missing"}
	}

	if len(filterEmpty(pr.Pause.Keywords)) == 0 {
		errors = append(errors, "pause keywords required")
	}

	if strings.TrimSpace(pr.Pause.Duration) == "" {
		errors = append(errors, "pause duration required")
	}

	return errors
}

func (pr *ParseResult) validateResumeInfo() []string {
	var errors []string

	if pr.Resume == nil {
		return []string{"resume info is missing"}
	}

	if len(filterEmpty(pr.Resume.Keywords)) == 0 {
		errors = append(errors, "resume keywords required")
	}

	return errors
}

func filterEmpty(values []string) []string {
	var filtered []string
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			filtered = append(filtered, strings.TrimSpace(v))
		}
	}
	return filtered
}
