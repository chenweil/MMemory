package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mmemory/internal/models"
	"mmemory/internal/service"
	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
)

type MessageHandler struct {
	reminderService    service.ReminderService
	userService        service.UserService
	reminderLogService service.ReminderLogService

	// AI服务（可选，用于智能解析和对话）
	aiParserService    service.AIParserService
	conversationService service.ConversationService
}

func NewMessageHandler(
	reminderService service.ReminderService,
	userService service.UserService,
	reminderLogService service.ReminderLogService,
	aiParserService service.AIParserService,
	conversationService service.ConversationService,
) *MessageHandler {
	return &MessageHandler{
		reminderService:     reminderService,
		userService:         userService,
		reminderLogService:  reminderLogService,
		aiParserService:     aiParserService,
		conversationService: conversationService,
	}
}

func (h *MessageHandler) HandleMessage(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message) error {
	// 确保用户存在
	user, err := h.ensureUser(ctx, message.From)
	if err != nil {
		logger.Errorf("确保用户存在失败: %v", err)
		return h.sendErrorMessage(bot, message.Chat.ID, "系统错误，请稍后重试")
	}

	// 处理不同类型的消息
	if message.IsCommand() {
		return h.handleCommand(ctx, bot, message, user)
	}

	return h.handleTextMessage(ctx, bot, message, user)
}

func (h *MessageHandler) handleCommand(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User) error {
	switch message.Command() {
	case "start":
		return h.handleStartCommand(bot, message)
	case "help":
		return h.handleHelpCommand(bot, message)
	case "list":
		return h.handleListCommand(ctx, bot, message, user)
	case "stats":
		return h.handleStatsCommand(ctx, bot, message, user)
	default:
		return h.sendMessage(bot, message.Chat.ID, "未知命令，请输入 /help 查看帮助")
	}
}

func (h *MessageHandler) handleStartCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) error {
	welcomeText := `👋 欢迎使用 MMemory 智能提醒助手！

我可以帮助你：
• 设置日常习惯提醒
• 创建一次性任务提醒  
• 跟踪完成进度

🗣️ 你可以直接对我说：
"每天19点提醒我复盘工作"
"明天上午10点提醒我开会"

输入 /help 查看更多帮助信息`

	return h.sendMessage(bot, message.Chat.ID, welcomeText)
}

func (h *MessageHandler) handleHelpCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) error {
	helpText := `📖 MMemory 使用指南

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

💡 直接发送文字消息即可创建提醒，我会智能识别你的需求！`

	return h.sendMessage(bot, message.Chat.ID, helpText)
}

func (h *MessageHandler) handleListCommand(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User) error {
	reminders, err := h.reminderService.GetUserReminders(ctx, user.ID)
	if err != nil {
		logger.Errorf("获取用户提醒列表失败: %v", err)
		return h.sendErrorMessage(bot, message.Chat.ID, "获取提醒列表失败，请稍后重试")
	}

	if len(reminders) == 0 {
		return h.sendMessage(bot, message.Chat.ID, "📋 你还没有设置任何提醒\n\n💡 试试对我说：\"每天19点提醒我复盘工作\"")
	}

	// 构建提醒列表消息
	listText := "📋 <b>你的提醒列表</b>\n\n"
	
	activeCount := 0
	for _, reminder := range reminders {
		if !reminder.IsActive {
			continue
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
		
		listText += fmt.Sprintf("<b>%d.</b> %s <i>%s</i>\n", activeCount, typeIcon, reminder.Title)
		listText += fmt.Sprintf("    ⏰ %s\n", h.formatSchedule(reminder))
		listText += fmt.Sprintf("    📊 %s %s\n\n", statusIcon, statusText)
	}
	
	if activeCount == 0 {
		return h.sendMessage(bot, message.Chat.ID, "📋 你目前没有活跃的提醒\n\n💡 试试对我说：\"每天19点提醒我复盘工作\"")
	}
	
	listText += fmt.Sprintf("🔢 共有 <b>%d</b> 个活跃提醒\n", activeCount)
	listText += "\n💡 <i>回复提醒消息时可以选择完成、延期或跳过</i>"

	return h.sendMessage(bot, message.Chat.ID, listText)
}

func (h *MessageHandler) handleStatsCommand(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User) error {
	stats, err := h.reminderLogService.GetUserStatistics(ctx, user.ID)
	if err != nil {
		logger.Errorf("获取用户统计数据失败: %v", err)
		return h.sendErrorMessage(bot, message.Chat.ID, "获取统计数据失败，请稍后重试")
	}

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

	return h.sendMessage(bot, message.Chat.ID, statsText)
}

func (h *MessageHandler) handleTextMessage(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User) error {
	// 如果启用了AI服务，优先使用AI解析
	if h.aiParserService != nil {
		logger.Infof("使用AI解析器处理用户 %d 的消息", user.ID)
		return h.handleWithAI(ctx, bot, message, user)
	}

	// 降级到传统解析器
	logger.Infof("使用传统解析器处理用户 %d 的消息", user.ID)
	return h.handleWithLegacyParser(ctx, bot, message, user)
}

// handleWithAI 使用AI解析器处理消息
func (h *MessageHandler) handleWithAI(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User) error {
	// 调用AI解析服务
	userIDStr := fmt.Sprintf("%d", user.TelegramID)
	parseResult, err := h.aiParserService.ParseMessage(ctx, userIDStr, message.Text)
	if err != nil {
		logger.Errorf("AI解析失败，降级到传统解析器: %v", err)
		return h.handleWithLegacyParser(ctx, bot, message, user)
	}

	// 验证解析结果
	validation := parseResult.Validate()
	if !validation.IsValid {
		logger.Warnf("AI解析结果验证失败: %v，降级到传统解析器", validation.Errors)
		return h.handleWithLegacyParser(ctx, bot, message, user)
	}

	logger.Infof("AI解析成功 - Intent: %s, Confidence: %.2f, ParsedBy: %s",
		parseResult.Intent, parseResult.Confidence, parseResult.ParsedBy)

	// 根据意图路由到不同的处理器
	switch parseResult.Intent {
	case ai.IntentReminder:
		return h.handleReminderIntent(ctx, bot, message, user, parseResult)
	case ai.IntentChat:
		return h.handleChatIntent(ctx, bot, message, user, parseResult)
	case ai.IntentSummary:
		return h.handleSummaryIntent(ctx, bot, message, user, parseResult)
	case ai.IntentQuery:
		return h.handleQueryIntent(ctx, bot, message, user, parseResult)
	case ai.IntentUnknown:
		return h.sendMessage(bot, message.Chat.ID, "抱歉，我没有完全理解你的意思。\n\n💡 你可以：\n• 设置提醒：\"每天19点提醒我复盘工作\"\n• 查看列表：/list\n• 查看帮助：/help")
	default:
		logger.Warnf("未知的意图类型: %s", parseResult.Intent)
		return h.sendMessage(bot, message.Chat.ID, "抱歉，我暂时无法处理这类请求。请尝试其他方式或查看 /help")
	}
}

// handleWithLegacyParser 使用传统解析器处理消息
func (h *MessageHandler) handleWithLegacyParser(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User) error {
	// 尝试解析提醒创建请求
	reminder, err := h.reminderService.ParseReminderFromText(ctx, message.Text, user.ID)
	if err != nil {
		logger.Errorf("解析提醒失败: %v", err)
		return h.sendMessage(bot, message.Chat.ID, "抱歉，我没有理解你的意思。请尝试这样说：\n\"每天19点提醒我复盘工作\"")
	}

	if reminder == nil {
		return h.sendMessage(bot, message.Chat.ID, "请告诉我你想要设置什么提醒？\n\n例如：\"每天19点提醒我复盘工作\"")
	}

	// 创建提醒
	if err := h.reminderService.CreateReminder(ctx, reminder); err != nil {
		logger.Errorf("创建提醒失败: %v", err)
		return h.sendErrorMessage(bot, message.Chat.ID, "创建提醒失败，请稍后重试")
	}

	successText := fmt.Sprintf("✅ 提醒已设置成功！\n\n📝 %s\n⏰ %s", reminder.Title, h.formatSchedule(reminder))
	return h.sendMessage(bot, message.Chat.ID, successText)
}

// handleReminderIntent 处理提醒创建意图
func (h *MessageHandler) handleReminderIntent(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User, parseResult *ai.ParseResult) error {
	if parseResult.Reminder == nil {
		logger.Error("提醒意图但缺少提醒信息")
		return h.sendErrorMessage(bot, message.Chat.ID, "抱歉，无法提取提醒信息，请重新描述")
	}

	reminderInfo := parseResult.Reminder

	// 构造时间字符串 HH:MM:SS
	targetTime := fmt.Sprintf("%02d:%02d:00", reminderInfo.Time.Hour, reminderInfo.Time.Minute)

	// 创建提醒对象
	reminder := &models.Reminder{
		UserID:          user.ID,
		Title:           reminderInfo.Title,
		Description:     reminderInfo.Description,
		Type:            reminderInfo.Type,
		TargetTime:      targetTime,
		SchedulePattern: string(reminderInfo.SchedulePattern),
		IsActive:        true,
		Timezone:        reminderInfo.Time.Timezone,
	}

	// 保存提醒
	if err := h.reminderService.CreateReminder(ctx, reminder); err != nil {
		logger.Errorf("创建提醒失败: %v", err)
		return h.sendErrorMessage(bot, message.Chat.ID, "创建提醒失败，请稍后重试")
	}

	// 构造成功消息
	successText := fmt.Sprintf("✅ 提醒已设置成功！\n\n📝 %s\n⏰ %s",
		reminder.Title, h.formatSchedule(reminder))

	// 如果置信度不是很高，添加提示
	if parseResult.IsLowConfidence() {
		successText += "\n\n💡 如果这不是你想要的，请告诉我更详细的信息。"
	}

	// 添加解析器信息（调试用）
	if parseResult.ParsedBy != "" {
		logger.Infof("提醒由 %s 解析", parseResult.ParsedBy)
	}

	return h.sendMessage(bot, message.Chat.ID, successText)
}

// handleChatIntent 处理对话意图
func (h *MessageHandler) handleChatIntent(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User, parseResult *ai.ParseResult) error {
	if parseResult.ChatResponse == nil || parseResult.ChatResponse.Response == "" {
		logger.Error("对话意图但缺少回复内容")
		return h.sendMessage(bot, message.Chat.ID, "我在想怎么回答你...但好像有点卡住了 🤔\n\n试试问我其他问题？")
	}

	// 保存对话上下文（如果有ConversationService）
	if h.conversationService != nil {
		// 构造对话上下文数据
		contextData := map[string]interface{}{
			"last_message":  message.Text,
			"last_response": parseResult.ChatResponse.Response,
			"timestamp":     time.Now().Unix(),
		}

		// 尝试获取现有对话
		conversation, err := h.conversationService.GetConversation(ctx, user.ID, models.ContextTypeChat)
		if err != nil {
			logger.Warnf("获取对话上下文失败: %v", err)
		}

		if conversation != nil {
			// 更新现有对话
			if err := h.conversationService.UpdateConversation(ctx, conversation, contextData); err != nil {
				logger.Warnf("更新对话上下文失败: %v", err)
			}
		} else {
			// 创建新对话（30天有效期）
			_, err := h.conversationService.CreateConversation(ctx, user.ID, models.ContextTypeChat, contextData, 30*24*time.Hour)
			if err != nil {
				logger.Warnf("创建对话上下文失败: %v", err)
			}
		}
	}

	// 发送AI的回复
	return h.sendMessage(bot, message.Chat.ID, parseResult.ChatResponse.Response)
}

// handleSummaryIntent 处理总结意图
func (h *MessageHandler) handleSummaryIntent(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User, parseResult *ai.ParseResult) error {
	// 获取用户的提醒统计
	stats, err := h.reminderLogService.GetUserStatistics(ctx, user.ID)
	if err != nil {
		logger.Errorf("获取用户统计失败: %v", err)
		return h.sendErrorMessage(bot, message.Chat.ID, "获取统计数据失败，请稍后重试")
	}

	// 构造总结消息
	summaryText := "📊 <b>你的使用总结</b>\n\n"
	summaryText += fmt.Sprintf("📝 活跃提醒: %d 个\n", stats.ActiveReminders)
	summaryText += fmt.Sprintf("✅ 本周完成: %d 个\n", stats.CompletedWeek)
	summaryText += fmt.Sprintf("📈 本月完成: %d 个\n\n", stats.CompletedMonth)

	if stats.CompletionRate > 0 {
		summaryText += fmt.Sprintf("🎯 完成率: %d%%\n", stats.CompletionRate)
	}

	// 如果AI有额外的总结回复
	if parseResult.ChatResponse != nil && parseResult.ChatResponse.Response != "" {
		summaryText += "\n💬 " + parseResult.ChatResponse.Response
	}

	return h.sendMessage(bot, message.Chat.ID, summaryText)
}

// handleQueryIntent 处理查询意图
func (h *MessageHandler) handleQueryIntent(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *models.User, parseResult *ai.ParseResult) error {
	// 获取用户的提醒列表
	reminders, err := h.reminderService.GetUserReminders(ctx, user.ID)
	if err != nil {
		logger.Errorf("获取提醒列表失败: %v", err)
		return h.sendErrorMessage(bot, message.Chat.ID, "获取提醒列表失败，请稍后重试")
	}

	if len(reminders) == 0 {
		return h.sendMessage(bot, message.Chat.ID, "📋 你还没有设置任何提醒\n\n💡 试试对我说：\"每天19点提醒我复盘工作\"")
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
		listText += fmt.Sprintf("    ⏰ %s\n\n", h.formatSchedule(reminder))
	}

	if activeCount == 0 {
		return h.sendMessage(bot, message.Chat.ID, "📋 你目前没有活跃的提醒")
	}

	listText += fmt.Sprintf("🔢 共有 <b>%d</b> 个活跃提醒", activeCount)

	// 如果AI有额外的回复
	if parseResult.ChatResponse != nil && parseResult.ChatResponse.Response != "" {
		listText += "\n\n💬 " + parseResult.ChatResponse.Response
	}

	return h.sendMessage(bot, message.Chat.ID, listText)
}

func (h *MessageHandler) ensureUser(ctx context.Context, from *tgbotapi.User) (*models.User, error) {
	user, err := h.userService.GetByTelegramID(ctx, from.ID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		// 创建新用户
		user = &models.User{
			TelegramID:   from.ID,
			Username:     from.UserName,
			FirstName:    from.FirstName,
			LastName:     from.LastName,
			LanguageCode: from.LanguageCode,
		}

		if err := h.userService.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (h *MessageHandler) formatSchedule(reminder *models.Reminder) string {
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
		if len(pattern) > 5 && pattern[:5] == "once:" {
			dateStr := pattern[5:]
			return fmt.Sprintf("%s %s", dateStr, reminder.TargetTime[:5])
		}
		return fmt.Sprintf("一次性提醒 %s", reminder.TargetTime[:5])
	default:
		return reminder.SchedulePattern
	}
}

func (h *MessageHandler) sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := bot.Send(msg)
	return err
}

func (h *MessageHandler) sendErrorMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	errorText := "⚠️ " + text
	return h.sendMessage(bot, chatID, errorText)
}