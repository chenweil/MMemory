package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mmemory/internal/service"
	"mmemory/pkg/logger"
)

type CallbackHandler struct {
	reminderLogService service.ReminderLogService
	schedulerService   service.SchedulerService
}

func NewCallbackHandler(
	reminderLogService service.ReminderLogService,
	schedulerService service.SchedulerService,
) *CallbackHandler {
	return &CallbackHandler{
		reminderLogService: reminderLogService,
		schedulerService:   schedulerService,
	}
}

func (h *CallbackHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) error {
	// 解析回调数据
	parts := strings.Split(callback.Data, "_")
	if len(parts) < 3 {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的操作")
	}
	
	action := parts[1]
	logIDStr := parts[2]
	logID, err := strconv.ParseUint(logIDStr, 10, 32)
	if err != nil {
		return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的提醒ID")
	}
	
	switch action {
	case "complete":
		return h.handleComplete(ctx, bot, callback, uint(logID))
	case "delay":
		if len(parts) < 4 {
			return h.sendCallbackResponse(bot, callback.ID, "❌ 缺少延期时间")
		}
		hours, err := strconv.Atoi(parts[3])
		if err != nil {
			return h.sendCallbackResponse(bot, callback.ID, "❌ 无效的延期时间")
		}
		return h.handleDelay(ctx, bot, callback, uint(logID), hours)
	case "skip":
		return h.handleSkip(ctx, bot, callback, uint(logID))
	default:
		return h.sendCallbackResponse(bot, callback.ID, "❌ 未知操作")
	}
}

func (h *CallbackHandler) handleComplete(ctx context.Context, bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, logID uint) error {
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

func (h *CallbackHandler) handleDelay(ctx context.Context, bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, logID uint, hours int) error {
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

func (h *CallbackHandler) handleSkip(ctx context.Context, bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, logID uint) error {
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

func (h *CallbackHandler) sendCallbackResponse(bot *tgbotapi.BotAPI, callbackID, text string) error {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := bot.Request(callback)
	return err
}

func (h *CallbackHandler) editMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, newText string) error {
	edit := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, newText)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = nil // 移除键盘
	
	_, err := bot.Send(edit)
	return err
}