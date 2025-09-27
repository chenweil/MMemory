package service

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mmemory/internal/models"
	"mmemory/pkg/logger"
)

type notificationService struct {
	bot *tgbotapi.BotAPI
}

func NewNotificationService(bot *tgbotapi.BotAPI) NotificationService {
	return &notificationService{
		bot: bot,
	}
}

func (s *notificationService) SendReminder(ctx context.Context, log *models.ReminderLog) error {
	if log.Reminder.User.TelegramID == 0 {
		return fmt.Errorf("用户Telegram ID为空")
	}
	
	// 构建提醒消息
	message := s.buildReminderMessage(&log.Reminder)
	
	// 创建键盘按钮
	keyboard := s.buildReminderKeyboard(log.ID)
	
	// 发送消息
	msg := tgbotapi.NewMessage(log.Reminder.User.TelegramID, message)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeHTML
	
	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("发送Telegram消息失败: %w", err)
	}
	
	logger.Infof("📤 提醒消息已发送: 用户=%d, 提醒=%s", 
		log.Reminder.User.TelegramID, log.Reminder.Title)
	
	return nil
}

func (s *notificationService) SendFollowUp(ctx context.Context, log *models.ReminderLog) error {
	if log.Reminder.User.TelegramID == 0 {
		return fmt.Errorf("用户Telegram ID为空")
	}
	
	// 构建关怀消息
	message := s.buildFollowUpMessage(&log.Reminder, log.FollowUpCount)
	
	// 创建键盘按钮
	keyboard := s.buildReminderKeyboard(log.ID)
	
	// 发送消息
	msg := tgbotapi.NewMessage(log.Reminder.User.TelegramID, message)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeHTML
	
	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("发送关怀消息失败: %w", err)
	}
	
	logger.Infof("💌 关怀消息已发送: 用户=%d, 次数=%d", 
		log.Reminder.User.TelegramID, log.FollowUpCount+1)
	
	return nil
}

// buildReminderMessage 构建提醒消息
func (s *notificationService) buildReminderMessage(reminder *models.Reminder) string {
	var message string
	
	// 根据提醒类型使用不同的emoji和措辞
	switch reminder.Type {
	case models.ReminderTypeHabit:
		message = fmt.Sprintf("⏰ <b>习惯提醒</b>\n\n"+
			"📝 %s\n\n"+
			"已经到了约定的时间，完成了吗？", reminder.Title)
	case models.ReminderTypeTask:
		message = fmt.Sprintf("📋 <b>任务提醒</b>\n\n"+
			"📝 %s\n\n"+
			"该处理这个任务了，准备好了吗？", reminder.Title)
	default:
		message = fmt.Sprintf("🔔 <b>提醒</b>\n\n"+
			"📝 %s\n\n"+
			"时间到了，请查看！", reminder.Title)
	}
	
	return message
}

// buildFollowUpMessage 构建关怀消息
func (s *notificationService) buildFollowUpMessage(reminder *models.Reminder, followUpCount int) string {
	var message string
	
	switch followUpCount {
	case 0:
		message = fmt.Sprintf("🤔 <b>还没完成吗？</b>\n\n"+
			"📝 %s\n\n"+
			"没关系，有什么困难吗？需要延期还是跳过？", reminder.Title)
	case 1:
		message = fmt.Sprintf("😊 <b>温馨提醒</b>\n\n"+
			"📝 %s\n\n"+
			"这个任务还在等着你呢，要不要处理一下？", reminder.Title)
	default:
		message = fmt.Sprintf("💪 <b>最后提醒</b>\n\n"+
			"📝 %s\n\n"+
			"今天确实不方便的话，可以选择跳过哦～", reminder.Title)
	}
	
	return message
}

// buildReminderKeyboard 构建回复键盘
func (s *notificationService) buildReminderKeyboard(logID uint) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ 完成了", fmt.Sprintf("reminder_complete_%d", logID)),
			tgbotapi.NewInlineKeyboardButtonData("⏰ 延期1小时", fmt.Sprintf("reminder_delay_%d_1", logID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ 延期3小时", fmt.Sprintf("reminder_delay_%d_3", logID)),
			tgbotapi.NewInlineKeyboardButtonData("😴 今天跳过", fmt.Sprintf("reminder_skip_%d", logID)),
		),
	)
}