package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/service"
	"mmemory/pkg/logger"
)

// CallbackHandlerLogic 回调处理业务逻辑层（可独立测试）
type CallbackHandlerLogic struct {
	reminderService    service.ReminderService
	reminderLogService service.ReminderLogService
}

// NewCallbackHandlerLogic 创建回调业务逻辑层
func NewCallbackHandlerLogic(
	reminderService service.ReminderService,
	reminderLogService service.ReminderLogService,
) *CallbackHandlerLogic {
	return &CallbackHandlerLogic{
		reminderService:    reminderService,
		reminderLogService: reminderLogService,
	}
}

// CallbackAction 回调动作
type CallbackAction struct {
	Action     string
	ResourceID uint
	Hours      int    // 用于delay动作
	IsValid    bool
	ErrorMsg   string
}

// ParseCallbackData 解析回调数据
func (l *CallbackHandlerLogic) ParseCallbackData(data string) *CallbackAction {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return &CallbackAction{
			IsValid:  false,
			ErrorMsg: "❌ 无效的操作",
		}
	}

	action := parts[1]
	resourceIDStr := parts[2]
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 64)
	if err != nil {
		return &CallbackAction{
			IsValid:  false,
			ErrorMsg: "❌ 无效的提醒ID",
		}
	}

	result := &CallbackAction{
		Action:     action,
		ResourceID: uint(resourceID),
		IsValid:    true,
	}

	// 处理delay动作的小时数
	if action == "delay" && len(parts) >= 4 {
		hours, err := strconv.Atoi(parts[3])
		if err != nil {
			return &CallbackAction{
				IsValid:  false,
				ErrorMsg: "❌ 无效的延期时间",
			}
		}
		result.Hours = hours
	}

	return result
}

// BuildCompleteText 构建完成提醒文本
func (l *CallbackHandlerLogic) BuildCompleteText(reminder *models.Reminder) string {
	return fmt.Sprintf("✅ <b>太棒了！</b>\n\n📝 %s\n\n🎉 已记录完成，继续保持！", reminder.Title)
}

// BuildDelayText 构建延期提醒文本
func (l *CallbackHandlerLogic) BuildDelayText(reminder *models.Reminder, hours int, delayTime time.Time) string {
	return fmt.Sprintf("⏰ <b>已延期 %d 小时</b>\n\n📝 %s\n\n🕐 将在 %s 再次提醒你",
		hours, reminder.Title, delayTime.Format("15:04"))
}

// BuildSkipText 构建跳过提醒文本
func (l *CallbackHandlerLogic) BuildSkipText(reminder *models.Reminder) string {
	return fmt.Sprintf("😴 <b>今天跳过</b>\n\n📝 %s\n\n💤 没关系，明天再来！", reminder.Title)
}

// BuildDeleteText 构建删除提醒文本
func (l *CallbackHandlerLogic) BuildDeleteText(reminderID uint) string {
	return fmt.Sprintf("✅ 已删除提醒 #%d", reminderID)
}

// BuildPauseText 构建暂停提醒文本
func (l *CallbackHandlerLogic) BuildPauseText(reminder *models.Reminder, until string) string {
	return fmt.Sprintf("⏸️ 已暂停提醒 #%d\n📝 %s\n⏳ 暂停至 %s", reminder.ID, reminder.Title, until)
}

// BuildResumeText 构建恢复提醒文本
func (l *CallbackHandlerLogic) BuildResumeText(reminder *models.Reminder) string {
	return fmt.Sprintf("▶️ 已恢复提醒 #%d\n📝 %s\n⏰ %s", reminder.ID, reminder.Title, reminder.TargetTime[:5])
}

// BuildEditText 构建编辑提示文本
func (l *CallbackHandlerLogic) BuildEditText(reminder *models.Reminder) string {
	return fmt.Sprintf(`🛠️ <b>编辑提醒 #%d</b>

<b>当前信息：</b>
📝 标题：%s
⏰ 时间：%s
🔄 模式：%s

<b>如何编辑：</b>
你可以直接对我说：
• "修改<b>%s</b>到晚上7点"
• "把<b>%s</b>改为每周一三五"
• "把<b>%s</b>的标题改为学习英语"

💡 AI会智能理解你的编辑意图`,
		reminder.ID,
		reminder.Title,
		reminder.TargetTime[:5],
		reminder.SchedulePattern,
		reminder.Title,
		reminder.Title,
		reminder.Title,
	)
}

// ProcessComplete 处理完成操作业务逻辑
func (l *CallbackHandlerLogic) ProcessComplete(ctx context.Context, logID uint) (*models.ReminderLog, error) {
	// 获取提醒记录
	log, err := l.reminderLogService.GetByID(ctx, logID)
	if err != nil {
		return nil, fmt.Errorf("获取提醒记录失败: %w", err)
	}

	if log == nil {
		return nil, fmt.Errorf("提醒记录不存在")
	}

	// 标记为已完成
	if err := l.reminderLogService.MarkAsCompleted(ctx, logID, "用户确认完成"); err != nil {
		logger.Errorf("标记提醒完成失败: %v", err)
		return nil, fmt.Errorf("操作失败: %w", err)
	}

	return log, nil
}

// ProcessDelay 处理延期操作业务逻辑
func (l *CallbackHandlerLogic) ProcessDelay(ctx context.Context, logID uint, hours int) (*models.ReminderLog, time.Time, error) {
	// 获取提醒记录
	log, err := l.reminderLogService.GetByID(ctx, logID)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("获取提醒记录失败: %w", err)
	}

	if log == nil {
		return nil, time.Time{}, fmt.Errorf("提醒记录不存在")
	}

	// 创建延期提醒
	delayTime := time.Now().Add(time.Duration(hours) * time.Hour)
	if err := l.reminderLogService.CreateDelayReminder(ctx, logID, delayTime, hours); err != nil {
		logger.Errorf("创建延期提醒失败: %v", err)
		return nil, time.Time{}, fmt.Errorf("延期失败: %w", err)
	}

	return log, delayTime, nil
}

// ProcessSkip 处理跳过操作业务逻辑
func (l *CallbackHandlerLogic) ProcessSkip(ctx context.Context, logID uint) (*models.ReminderLog, error) {
	// 获取提醒记录
	log, err := l.reminderLogService.GetByID(ctx, logID)
	if err != nil {
		return nil, fmt.Errorf("获取提醒记录失败: %w", err)
	}

	if log == nil {
		return nil, fmt.Errorf("提醒记录不存在")
	}

	// 标记为已跳过
	if err := l.reminderLogService.MarkAsSkipped(ctx, logID, "用户选择跳过"); err != nil {
		logger.Errorf("标记提醒跳过失败: %v", err)
		return nil, fmt.Errorf("操作失败: %w", err)
	}

	return log, nil
}

// ProcessDelete 处理删除操作业务逻辑
func (l *CallbackHandlerLogic) ProcessDelete(ctx context.Context, reminderID uint) error {
	if reminderID == 0 {
		return fmt.Errorf("无效的提醒ID")
	}

	if err := l.reminderService.DeleteReminder(ctx, reminderID); err != nil {
		logger.Errorf("删除提醒失败 (ID: %d): %v", reminderID, err)
		return fmt.Errorf("删除失败: %w", err)
	}

	return nil
}

// ProcessPause 处理暂停操作业务逻辑
func (l *CallbackHandlerLogic) ProcessPause(ctx context.Context, reminderID uint) (*models.Reminder, string, error) {
	if reminderID == 0 {
		return nil, "", fmt.Errorf("无效的提醒ID")
	}

	duration := 24 * time.Hour
	if err := l.reminderService.PauseReminder(ctx, reminderID, duration, "用户通过按钮暂停"); err != nil {
		logger.Errorf("按钮暂停提醒失败 (ID: %d): %v", reminderID, err)
		return nil, "", fmt.Errorf("暂停失败: %w", err)
	}

	reminder, _ := l.reminderService.GetReminderByID(ctx, reminderID)
	until := time.Now().Add(duration).Format("2006-01-02 15:04")
	if reminder != nil && reminder.PausedUntil != nil {
		until = reminder.PausedUntil.Format("2006-01-02 15:04")
	}

	return reminder, until, nil
}

// ProcessResume 处理恢复操作业务逻辑
func (l *CallbackHandlerLogic) ProcessResume(ctx context.Context, reminderID uint) (*models.Reminder, error) {
	if reminderID == 0 {
		return nil, fmt.Errorf("无效的提醒ID")
	}

	if err := l.reminderService.ResumeReminder(ctx, reminderID); err != nil {
		logger.Errorf("按钮恢复提醒失败 (ID: %d): %v", reminderID, err)
		return nil, fmt.Errorf("恢复失败: %w", err)
	}

	reminder, _ := l.reminderService.GetReminderByID(ctx, reminderID)
	return reminder, nil
}

// ProcessEdit 处理编辑操作业务逻辑（只获取提醒信息）
func (l *CallbackHandlerLogic) ProcessEdit(ctx context.Context, reminderID uint) (*models.Reminder, error) {
	if reminderID == 0 {
		return nil, fmt.Errorf("无效的提醒ID")
	}

	// 获取提醒详情
	reminder, err := l.reminderService.GetReminderByID(ctx, reminderID)
	if err != nil {
		logger.Errorf("获取提醒失败 (ID: %d): %v", reminderID, err)
		return nil, fmt.Errorf("获取提醒失败: %w", err)
	}
	if reminder == nil {
		return nil, fmt.Errorf("提醒不存在")
	}

	return reminder, nil
}
