package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	botinterface "mmemory/internal/bot"
	"mmemory/internal/service"
	"mmemory/pkg/logger"
)

type CallbackHandler struct {
	reminderService    service.ReminderService
	reminderLogService service.ReminderLogService
	schedulerService   service.SchedulerService
}

func NewCallbackHandler(
	reminderService service.ReminderService,
	reminderLogService service.ReminderLogService,
	schedulerService service.SchedulerService,
) *CallbackHandler {
	return &CallbackHandler{
		reminderService:    reminderService,
		reminderLogService: reminderLogService,
		schedulerService:   schedulerService,
	}
}

func (h *CallbackHandler) HandleCallback(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery) error {
	// 解析回调数据
	parts := strings.Split(callback.Data, "_")
	if len(parts) < 3 {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的操作")
	}

	action := parts[1]
	resourceIDStr := parts[2]
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 64)
	if err != nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的提醒ID")
	}

	switch action {
	case "complete":
		return h.handleComplete(ctx, bot, callback, uint(resourceID))
	case "delay":
		if len(parts) < 4 {
			return h.sendCallbackResponse(bot, callback.ID, "❌ 缺少延期时间")
		}
		hours, err := strconv.Atoi(parts[3])
		if err != nil {
			return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的延期时间")
		}
		return h.handleDelay(ctx, bot, callback, uint(resourceID), hours)
	case "skip":
		return h.handleSkip(ctx, bot, callback, uint(resourceID))
	case "delete":
		return h.handleReminderDelete(ctx, bot, callback, uint(resourceID))
	case "pause":
		return h.handleReminderPause(ctx, bot, callback, uint(resourceID))
	case "resume":
		return h.handleReminderResume(ctx, bot, callback, uint(resourceID))
	case "edit":
		return h.handleReminderEdit(ctx, bot, callback, uint(resourceID))
	default:
		return h.sendCallbackResponse(bot, callback.ID, "❌ 未知操作")
	}
}

func (h *CallbackHandler) handleComplete(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery, logID uint) error {
	// 获取提醒记录
	log, err := h.reminderLogService.GetByID(ctx, logID)
	if err != nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 获取提醒记录失败")
	}

	if log == nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 提醒记录不存在")
	}

	// 标记为已完成
	if err := h.reminderLogService.MarkAsCompleted(ctx, logID, "用户确认完成"); err != nil {
		logger.Errorf("标记提醒完成失败: %v", err)
		return h.sendCallbackResponse(bot, callback.ID, "❌ 操作失败，请稍后重试")
	}

	// 编辑原消息
	response := fmt.Sprintf("✅ <b>太棒了！</b>\n\n📝 %s\n\n🎉 已记录完成，继续保持！", log.Reminder.Title)
	if err := h.editMessage(bot, callback.Message, response); err != nil {
		logger.Errorf("编辑消息失败: %v", err)
	}

	// 发送回调响应
	return h.sendCallbackResponse(bot, callback.ID, "✅ 已标记为完成")
}

func (h *CallbackHandler) handleDelay(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery, logID uint, hours int) error {
	// 获取提醒记录
	log, err := h.reminderLogService.GetByID(ctx, logID)
	if err != nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 获取提醒记录失败")
	}

	if log == nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 提醒记录不存在")
	}

	// 创建延期提醒
	delayTime := time.Now().Add(time.Duration(hours) * time.Hour)
	if err := h.reminderLogService.CreateDelayReminder(ctx, logID, delayTime, hours); err != nil {
		logger.Errorf("创建延期提醒失败: %v", err)
		return h.sendCallbackResponse(bot, callback.ID, "❌ 延期失败，请稍后重试")
	}

	// 编辑原消息
	response := fmt.Sprintf("⏰ <b>已延期 %d 小时</b>\n\n📝 %s\n\n🕐 将在 %s 再次提醒你",
		hours, log.Reminder.Title, delayTime.Format("15:04"))
	if err := h.editMessage(bot, callback.Message, response); err != nil {
		logger.Errorf("编辑消息失败: %v", err)
	}

	// 发送回调响应
	return h.sendCallbackResponse(bot, callback.ID, fmt.Sprintf("⏰ 已延期%d小时", hours))
}

func (h *CallbackHandler) handleSkip(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery, logID uint) error {
	// 获取提醒记录
	log, err := h.reminderLogService.GetByID(ctx, logID)
	if err != nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 获取提醒记录失败")
	}

	if log == nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 提醒记录不存在")
	}

	// 标记为已跳过
	if err := h.reminderLogService.MarkAsSkipped(ctx, logID, "用户选择跳过"); err != nil {
		logger.Errorf("标记提醒跳过失败: %v", err)
		return h.sendCallbackResponse(bot, callback.ID, "❌ 操作失败，请稍后重试")
	}

	// 编辑原消息
	response := fmt.Sprintf("😴 <b>今天跳过</b>\n\n📝 %s\n\n💤 没关系，明天再来！", log.Reminder.Title)
	if err := h.editMessage(bot, callback.Message, response); err != nil {
		logger.Errorf("编辑消息失败: %v", err)
	}

	// 发送回调响应
	return h.sendCallbackResponse(bot, callback.ID, "😴 已跳过")
}

func (h *CallbackHandler) handleReminderDelete(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery, reminderID uint) error {
	if reminderID == 0 {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的提醒ID")
	}

	if err := h.reminderService.DeleteReminder(ctx, reminderID); err != nil {
		logger.Errorf("删除提醒失败 (ID: %d): %v", reminderID, err)
		return h.sendCallbackResponse(bot, callback.ID, "❌ 删除失败，请稍后重试")
	}

	if callback.Message != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, fmt.Sprintf("✅ 已删除提醒 #%d", reminderID))
		msg.ParseMode = tgbotapi.ModeHTML
		if _, err := bot.Send(msg); err != nil {
			logger.Warnf("发送删除提示失败: %v", err)
		}
	}

	return h.sendCallbackResponse(bot, callback.ID, "✅ 删除成功")
}

func (h *CallbackHandler) handleReminderPause(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery, reminderID uint) error {
	if reminderID == 0 {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的提醒ID")
	}

	duration := 24 * time.Hour
	if err := h.reminderService.PauseReminder(ctx, reminderID, duration, "用户通过按钮暂停"); err != nil {
		logger.Errorf("按钮暂停提醒失败 (ID: %d): %v", reminderID, err)
		return h.sendCallbackResponse(bot, callback.ID, "❌ 暂停失败，请稍后重试")
	}

	reminder, _ := h.reminderService.GetReminderByID(ctx, reminderID)
	until := time.Now().Add(duration).Format("2006-01-02 15:04")
	if reminder != nil && reminder.PausedUntil != nil {
		until = reminder.PausedUntil.Format("2006-01-02 15:04")
	}

	if callback.Message != nil && reminder != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID,
			fmt.Sprintf("⏸️ 已暂停提醒 #%d\n📝 %s\n⏳ 暂停至 %s", reminderID, reminder.Title, until))
		msg.ParseMode = tgbotapi.ModeHTML
		if _, err := bot.Send(msg); err != nil {
			logger.Warnf("发送暂停提示失败: %v", err)
		}
	}

	return h.sendCallbackResponse(bot, callback.ID, "⏸️ 已暂停")
}

func (h *CallbackHandler) handleReminderResume(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery, reminderID uint) error {
	if reminderID == 0 {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的提醒ID")
	}

	if err := h.reminderService.ResumeReminder(ctx, reminderID); err != nil {
		logger.Errorf("按钮恢复提醒失败 (ID: %d): %v", reminderID, err)
		return h.sendCallbackResponse(bot, callback.ID, "❌ 恢复失败，请稍后重试")
	}

	reminder, _ := h.reminderService.GetReminderByID(ctx, reminderID)
	if callback.Message != nil && reminder != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID,
			fmt.Sprintf("▶️ 已恢复提醒 #%d\n📝 %s\n⏰ %s", reminderID, reminder.Title, reminder.TargetTime[:5]))
		msg.ParseMode = tgbotapi.ModeHTML
		if _, err := bot.Send(msg); err != nil {
			logger.Warnf("发送恢复提示失败: %v", err)
		}
	}

	return h.sendCallbackResponse(bot, callback.ID, "▶️ 已恢复")
}

func (h *CallbackHandler) handleReminderEdit(ctx context.Context, bot botinterface.BotAPI, callback *tgbotapi.CallbackQuery, reminderID uint) error {
	if reminderID == 0 {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的提醒ID")
	}

	// 获取提醒详情
	reminder, err := h.reminderService.GetReminderByID(ctx, reminderID)
	if err != nil {
		logger.Errorf("获取提醒失败 (ID: %d): %v", reminderID, err)
		return h.sendCallbackResponse(bot, callback.ID, "❌ 获取提醒失败")
	}
	if reminder == nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 提醒不存在")
	}

	// 构建编辑提示消息
	editText := fmt.Sprintf(`🛠️ <b>编辑提醒 #%d</b>

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
		reminderID,
		reminder.Title,
		reminder.TargetTime[:5],
		reminder.SchedulePattern,
		reminder.Title,
		reminder.Title,
		reminder.Title,
	)

	if callback.Message != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, editText)
		msg.ParseMode = tgbotapi.ModeHTML
		if _, err := bot.Send(msg); err != nil {
			logger.Warnf("发送编辑提示失败: %v", err)
		}
	}

	return h.sendCallbackResponse(bot, callback.ID, "📝 请通过文字描述你的修改")
}

func (h *CallbackHandler) sendCallbackResponse(bot botinterface.BotAPI, callbackID, text string) error {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := bot.Request(callback)
	return err
}

func (h *CallbackHandler) editMessage(bot botinterface.BotAPI, message *tgbotapi.Message, newText string) error {
	edit := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, newText)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = nil // 移除键盘

	_, err := bot.Send(edit)
	return err
}
