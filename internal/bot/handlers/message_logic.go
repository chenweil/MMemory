package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/service"
	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
	"mmemory/pkg/version"
)

// MessageHandlerLogic 消息处理业务逻辑层（可独立测试）
type MessageHandlerLogic struct {
	reminderService    service.ReminderService
	reminderLogService service.ReminderLogService
	aiParserService    service.AIParserService
}

// NewMessageHandlerLogic 创建业务逻辑层
func NewMessageHandlerLogic(
	reminderService service.ReminderService,
	reminderLogService service.ReminderLogService,
	aiParserService service.AIParserService,
) *MessageHandlerLogic {
	return &MessageHandlerLogic{
		reminderService:    reminderService,
		reminderLogService: reminderLogService,
		aiParserService:    aiParserService,
	}
}

// ====== 文本构建方法（100%可测试） ======

// BuildWelcomeText 构建欢迎消息
func (l *MessageHandlerLogic) BuildWelcomeText() string {
	return `👋 欢迎使用 MMemory 智能提醒助手！

我可以帮助你：
• 设置日常习惯提醒
• 创建一次性任务提醒
• 跟踪完成进度

🗣️ 你可以直接对我说：
"每天19点提醒我复盘工作"
"明天上午10点提醒我开会"

输入 /help 查看更多帮助信息`
}

// BuildHelpText 构建帮助消息
func (l *MessageHandlerLogic) BuildHelpText() string {
	return `📖 MMemory 使用指南

🔹 设置提醒：
• "每天X点提醒我做某事"
• "每周一三五19点提醒我健身"
• "2024年10月1日提醒我交房租"

🔹 管理提醒：
• /list - 查看我的提醒列表
• 回复提醒时可选择：完成/延期/跳过

🔹 其他命令：
• /start - 重新开始
• /help - 查看帮助
• /stats - 查看统计数据
• /version - 查看版本信息

💡 直接发送文字消息即可创建提醒，我会智能识别你的需求！`
}

// BuildVersionText 构建版本信息消息
func (l *MessageHandlerLogic) BuildVersionText() string {
	versionInfo := version.GetInfo()

	return fmt.Sprintf(`ℹ️ <b>MMemory 版本信息</b>

<b>版本:</b> %s
<b>Git提交:</b> <code>%s</code>
<b>Git分支:</b> <code>%s</code>
<b>构建时间:</b> %s
<b>Go版本:</b> %s
<b>运行平台:</b> %s

🚀 <i>MMemory - 你的智能提醒助手</i>`,
		versionInfo.Version,
		versionInfo.GitCommit,
		versionInfo.GitBranch,
		version.FormatBuildTime(),
		versionInfo.GoVersion,
		versionInfo.Platform,
	)
}

// ListItem 提醒列表项
type ListItem struct {
	ID           uint
	TypeIcon     string
	Title        string
	Schedule     string
	StatusIcon   string
	StatusText   string
	IsPaused     bool
	ButtonEdit   string
	ButtonDelete string
	ButtonAction string
}

// BuildReminderListText 构建提醒列表文本
func (l *MessageHandlerLogic) BuildReminderListText(reminders []*models.Reminder) (string, []ListItem, bool) {
	if len(reminders) == 0 {
		return "📋 你还没有设置任何提醒\n\n💡 试试对我说：\"每天19点提醒我复盘工作\"", nil, false
	}

	listText := "📋 <b>你的提醒列表</b>\n\n"
	var items []ListItem
	activeCount := 0

	for _, reminder := range reminders {
		if !reminder.IsActive {
			// 非活跃但仍处于暂停状态的提醒也展示，便于恢复
			if !reminder.IsPaused() {
				continue
			}
		}

		activeCount++

		// 提醒类型图标
		typeIcon := "🔔"
		if reminder.Type == models.ReminderTypeHabit {
			typeIcon = "🔄"
		} else if reminder.Type == models.ReminderTypeTask {
			typeIcon = "📋"
		}

		// 状态图标
		statusIcon := "✅"
		statusText := "活跃中"
		isPaused := reminder.IsPaused()

		if isPaused {
			statusIcon = "⏸️"
			statusText = "已暂停"
		}

		schedule := FormatSchedule(reminder)

		listText += fmt.Sprintf("<b>#%d</b> %s <i>%s</i>\n", reminder.ID, typeIcon, reminder.Title)
		listText += fmt.Sprintf("    ⏰ %s\n", schedule)
		listText += fmt.Sprintf("    📊 %s %s\n\n", statusIcon, statusText)

		items = append(items, ListItem{
			ID:           reminder.ID,
			TypeIcon:     typeIcon,
			Title:        reminder.Title,
			Schedule:     schedule,
			StatusIcon:   statusIcon,
			StatusText:   statusText,
			IsPaused:     isPaused,
			ButtonEdit:   fmt.Sprintf("reminder_edit_%d", reminder.ID),
			ButtonDelete: fmt.Sprintf("reminder_delete_%d", reminder.ID),
			ButtonAction: fmt.Sprintf("reminder_%s_%d", map[bool]string{true: "resume", false: "pause"}[isPaused], reminder.ID),
		})
	}

	if activeCount == 0 {
		return "📋 你目前没有活跃的提醒\n\n💡 试试对我说：\"每天19点提醒我复盘工作\"", nil, false
	}

	listText += fmt.Sprintf("🔢 共有 <b>%d</b> 个活跃提醒\n", activeCount)
	listText += "\n💡 <i>点击下方按钮快速删除提醒，或回复提示消息进行操作</i>"

	return listText, items, true
}

// BuildStatsText 构建统计信息文本
func (l *MessageHandlerLogic) BuildStatsText(stats *service.UserStatistics) string {
	statsText := "📊 <b>你的使用统计</b>\n\n"

	// 基础统计
	statsText += fmt.Sprintf("📝 <b>提醒总数:</b> %d 个\n", stats.TotalReminders)
	statsText += fmt.Sprintf("✅ <b>活跃提醒:</b> %d 个\n\n", stats.ActiveReminders)

	// 今日统计
	statsText += "📅 <b>今日数据:</b>\n"
	statsText += fmt.Sprintf("  ✅ 完成: %d 个\n", stats.CompletedToday)
	statsText += fmt.Sprintf("  😴 跳过: %d 个\n\n", stats.SkippedToday)

	// 本周统计
	statsText += "📆 <b>本周数据:</b>\n"
	statsText += fmt.Sprintf("  ✅ 完成: %d 个\n\n", stats.CompletedWeek)

	// 本月统计
	statsText += "📈 <b>本月数据:</b>\n"
	statsText += fmt.Sprintf("  ✅ 完成: %d 个\n", stats.CompletedMonth)

	// 完成率
	if stats.CompletionRate > 0 {
		rateEmoji := "📊"
		if stats.CompletionRate >= 80 {
			rateEmoji = "🎉"
		} else if stats.CompletionRate >= 60 {
			rateEmoji = "👍"
		}
		statsText += fmt.Sprintf("  %s 完成率: %d%%\n\n", rateEmoji, stats.CompletionRate)
	} else {
		statsText += "  📊 完成率: 暂无数据\n\n"
	}

	// 鼓励信息
	if stats.CompletedToday > 0 {
		statsText += "🌟 <i>今天做得很棒！继续保持！</i>"
	} else if stats.ActiveReminders > 0 {
		statsText += "💪 <i>今天还有提醒等着你完成哦～</i>"
	} else {
		statsText += "🚀 <i>快去设置一些提醒开始你的习惯养成之旅吧！</i>"
	}

	return statsText
}

// BuildReminderSuccessText 构建提醒创建成功文本
func (l *MessageHandlerLogic) BuildReminderSuccessText(reminder *models.Reminder, parseResult *ai.ParseResult) string {
	successText := fmt.Sprintf("✅ 提醒已设置成功！\n\n📝 %s\n⏰ %s", reminder.Title, FormatSchedule(reminder))

	// 如果置信度不是很高，添加提示
	if parseResult != nil && parseResult.IsLowConfidence() {
		successText += "\n\n💡 如果这不是你想要的，请告诉我更详细的信息。"
	}

	return successText
}

// BuildDeleteSuccessText 构建删除成功文本
func (l *MessageHandlerLogic) BuildDeleteSuccessText(reminder *models.Reminder) string {
	return fmt.Sprintf("✅ 已删除提醒\n\n📝 %s\n⏰ %s", reminder.Title, FormatSchedule(reminder))
}

// BuildEditSuccessText 构建编辑成功文本
func (l *MessageHandlerLogic) BuildEditSuccessText(reminder *models.Reminder) string {
	response := "✅ 已成功修改提醒\n\n"
	response += fmt.Sprintf("📝 %s\n⏰ %s", reminder.Title, FormatSchedule(reminder))
	if reminder.Description != "" {
		response += fmt.Sprintf("\n📄 %s", reminder.Description)
	}
	return response
}

// BuildPauseSuccessText 构建暂停成功文本
func (l *MessageHandlerLogic) BuildPauseSuccessText(reminder *models.Reminder, untilTime string, reason string) string {
	response := fmt.Sprintf("⏸️ 已暂停提醒\n\n📝 %s\n⏳ 暂停至 %s", reminder.Title, untilTime)
	if reason := strings.TrimSpace(reason); reason != "" {
		response += fmt.Sprintf("\n💬 理由：%s", reason)
	}
	response += "\n\n▶️ 想恢复时可以说：\"恢复" + reminder.Title + "\" 或使用 /list 按钮。"
	return response
}

// BuildResumeSuccessText 构建恢复成功文本
func (l *MessageHandlerLogic) BuildResumeSuccessText(reminder *models.Reminder) string {
	return fmt.Sprintf("▶️ 已恢复提醒\n\n📝 %s\n⏰ %s", reminder.Title, FormatSchedule(reminder))
}

// BuildSummaryText 构建总结文本
func (l *MessageHandlerLogic) BuildSummaryText(stats *service.UserStatistics, aiResponse string) string {
	summaryText := "📊 <b>你的使用总结</b>\n\n"
	summaryText += fmt.Sprintf("📝 活跃提醒: %d 个\n", stats.ActiveReminders)
	summaryText += fmt.Sprintf("✅ 本周完成: %d 个\n", stats.CompletedWeek)
	summaryText += fmt.Sprintf("📈 本月完成: %d 个\n\n", stats.CompletedMonth)

	if stats.CompletionRate > 0 {
		summaryText += fmt.Sprintf("🎯 完成率: %d%%\n", stats.CompletionRate)
	}

	// 如果AI有额外的总结回复
	if aiResponse != "" {
		summaryText += "\n💬 " + aiResponse
	}

	return summaryText
}

// BuildQueryText 构建查询文本
func (l *MessageHandlerLogic) BuildQueryText(reminders []*models.Reminder, aiResponse string) string {
	if len(reminders) == 0 {
		return "📋 你还没有设置任何提醒\n\n💡 试试对我说：\"每天19点提醒我复盘工作\""
	}

	// 构建提醒列表
	listText := "📋 <b>你的提醒列表</b>\n\n"

	activeCount := 0
	for _, reminder := range reminders {
		if !reminder.IsActive {
			continue
		}

		activeCount++
		typeIcon := "🔔"
		if reminder.Type == models.ReminderTypeHabit {
			typeIcon = "🔄"
		} else if reminder.Type == models.ReminderTypeTask {
			typeIcon = "📋"
		}

		listText += fmt.Sprintf("<b>%d.</b> %s <i>%s</i>\n", activeCount, typeIcon, reminder.Title)
		listText += fmt.Sprintf("    ⏰ %s\n\n", FormatSchedule(reminder))
	}

	if activeCount == 0 {
		return "📋 你目前没有活跃的提醒"
	}

	listText += fmt.Sprintf("🔢 共有 <b>%d</b> 个活跃提醒", activeCount)

	// 如果AI有额外的回复
	if aiResponse != "" {
		listText += "\n\n💬 " + aiResponse
	}

	return listText
}

// BuildMultipleMatchesText 构建多个匹配结果文本
func (l *MessageHandlerLogic) BuildMultipleMatchesText(matches []reminderMatch, action string) string {
	text := "🔍 找到多个可能的提醒，请更具体一些：\n\n"
	for i, match := range matches {
		text += fmt.Sprintf("%d. #%d %s\n    ⏰ %s\n", i+1, match.reminder.ID, match.reminder.Title, FormatSchedule(match.reminder))
	}

	switch action {
	case "delete":
		text += "\n💡 你可以说：\"删除" + matches[0].reminder.Title + "\" 或使用 /delete <ID>"
	case "edit":
		text += "\n💡 试试：\"修改" + matches[0].reminder.Title + "到晚上7点\" 或使用 /list 按钮操作。"
	case "pause":
		text += "\n💡 试试：\"暂停" + matches[0].reminder.Title + "一周\" 或者使用 /list 按钮操作。"
	case "resume":
		text += "\n💡 试试：\"恢复每天的喝水提醒\"。"
	}

	return text
}

// ====== 业务逻辑处理方法（可独立测试） ======

// ProcessReminderCreation 处理提醒创建业务逻辑
func (l *MessageHandlerLogic) ProcessReminderCreation(ctx context.Context, userID uint, reminderInfo *ai.ReminderInfo) (*models.Reminder, error) {
	if reminderInfo == nil {
		return nil, fmt.Errorf("提醒信息为空")
	}

	// 构造时间字符串 HH:MM:SS
	targetTime := fmt.Sprintf("%02d:%02d:00", reminderInfo.Time.Hour, reminderInfo.Time.Minute)

	// 创建提醒对象
	reminder := &models.Reminder{
		UserID:          userID,
		Title:           reminderInfo.Title,
		Description:     reminderInfo.Description,
		Type:            reminderInfo.Type,
		TargetTime:      targetTime,
		SchedulePattern: string(reminderInfo.SchedulePattern),
		IsActive:        true,
		Timezone:        reminderInfo.Time.Timezone,
	}

	// 保存提醒
	if err := l.reminderService.CreateReminder(ctx, reminder); err != nil {
		logger.Errorf("创建提醒失败: %v", err)
		return nil, err
	}

	return reminder, nil
}

// FindRemindersByKeywords 根据关键词查找提醒
func (l *MessageHandlerLogic) FindRemindersByKeywords(ctx context.Context, userID uint, keywords []string) ([]reminderMatch, error) {
	keywords = filterKeywords(keywords)
	if len(keywords) == 0 {
		return nil, fmt.Errorf("关键词为空")
	}

	reminders, err := l.reminderService.GetUserReminders(ctx, userID)
	if err != nil {
		logger.Errorf("获取用户提醒失败: %v", err)
		return nil, err
	}

	matches := matchReminders(reminders, keywords)
	return matches, nil
}

// CalculatePauseUntilTime 计算暂停截止时间
func (l *MessageHandlerLogic) CalculatePauseUntilTime(duration string) (time.Duration, string) {
	dur := parsePauseDuration(duration)
	if dur <= 0 {
		dur = 7 * 24 * time.Hour
	}

	untilTime := time.Now().Add(dur)
	untilText := untilTime.Format("2006-01-02 15:04")

	return dur, untilText
}

// UpdatePauseUntilTime 根据实际提醒更新暂停时间文本
func (l *MessageHandlerLogic) UpdatePauseUntilTime(reminder *models.Reminder, fallbackDuration time.Duration) string {
	if reminder != nil && reminder.PausedUntil != nil {
		return reminder.PausedUntil.In(time.Now().Location()).Format("2006-01-02 15:04")
	}
	return time.Now().Add(fallbackDuration).Format("2006-01-02 15:04")
}

// FormatSchedule 格式化提醒计划（全局可用）
func FormatSchedule(reminder *models.Reminder) string {
	switch {
	case reminder.IsDaily():
		return fmt.Sprintf("每天 %s", reminder.TargetTime[:5])
	case reminder.IsWeekly():
		// 解析周几
		weekdayMap := map[string]string{
			"1": "周一", "2": "周二", "3": "周三", "4": "周四",
			"5": "周五", "6": "周六", "7": "周日",
		}

		pattern := reminder.SchedulePattern
		if len(pattern) > 7 && pattern[:7] == "weekly:" {
			weekdaysStr := pattern[7:]
			weekdays := []string{}
			for _, day := range strings.Split(weekdaysStr, ",") {
				day = strings.TrimSpace(day)
				if dayName, ok := weekdayMap[day]; ok {
					weekdays = append(weekdays, dayName)
				}
			}
			if len(weekdays) > 0 {
				return fmt.Sprintf("%s %s", strings.Join(weekdays, "、"), reminder.TargetTime[:5])
			}
		}
		return fmt.Sprintf("每周指定时间 %s", reminder.TargetTime[:5])
	case reminder.IsOnce():
		// 解析日期
		pattern := reminder.SchedulePattern
		if strings.HasPrefix(pattern, string(models.SchedulePatternOnce)) {
			dateStr := strings.TrimPrefix(pattern, string(models.SchedulePatternOnce))
			return fmt.Sprintf("%s %s", dateStr, reminder.TargetTime[:5])
		}
		return fmt.Sprintf("一次性提醒 %s", reminder.TargetTime[:5])
	default:
		return reminder.SchedulePattern
	}
}
