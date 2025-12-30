package service

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mmemory/internal/models"
	"mmemory/pkg/logger"
)

// BotAPI 接口（用于测试）
type BotAPI interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

type notificationService struct {
	bot BotAPI
}

func NewNotificationService(bot BotAPI) NotificationService {
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
	// 检查是否需要追问活动详情
	if s.shouldAskForActivityDetails(reminder, followUpCount) {
		return s.buildActivityFollowUp(reminder)
	}

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

// shouldAskForActivityDetails 判断是否需要追问活动详情
func (s *notificationService) shouldAskForActivityDetails(reminder *models.Reminder, followUpCount int) bool {
	// 只在第一次关怀时追问
	if followUpCount != 0 {
		return false
	}

	// 检查提醒标题是否与活动类型相关
	activityKeywords := map[string]string{
		"喝水": "drink_water",
		"吃药": "take_medicine",
		"看书": "read_book",
		"运动": "exercise",
		"跑步": "exercise",
		"健身": "exercise",
	}

	for keyword := range activityKeywords {
		if strings.Contains(reminder.Title, keyword) {
			return true
		}
	}

	return false
}

// buildActivityFollowUp 构建活动详情追问
func (s *notificationService) buildActivityFollowUp(reminder *models.Reminder) string {
	switch {
	case strings.Contains(reminder.Title, "喝水"):
		return "💧 喝水完成！喝了多少？温水还是凉水？"
	case strings.Contains(reminder.Title, "吃药"):
		return "💊 吃药完成！吃了什么药？剂量是多少？"
	case strings.Contains(reminder.Title, "看书"):
		return "📚 看书完成！读了哪一章？有什么感想？"
	case strings.Contains(reminder.Title, "运动") || strings.Contains(reminder.Title, "跑步") || strings.Contains(reminder.Title, "健身"):
		return "🏃 运动完成！运动了多久？什么类型的运动？"
	default:
		return fmt.Sprintf("✅ %s 完成！有什么想记录的吗？", reminder.Title)
	}
}

// buildReminderKeyboard 构建回复键盘
func (s *notificationService) buildReminderKeyboard(logID uint) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ 完成了", fmt.Sprintf("reminder_complete_%d", logID)),
			tgbotapi.NewInlineKeyboardButtonData("📝 编辑", fmt.Sprintf("reminder_edit_%d", logID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ 延期1小时", fmt.Sprintf("reminder_delay_%d_1", logID)),
			tgbotapi.NewInlineKeyboardButtonData("⏰ 延期3小时", fmt.Sprintf("reminder_delay_%d_3", logID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("😴 今天跳过", fmt.Sprintf("reminder_skip_%d", logID)),
		),
	)
}