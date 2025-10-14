package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mmemory/internal/models"
)

// Mock Bot API for testing
type mockBotAPI struct {
	sentMessages []tgbotapi.Chattable
	shouldError  bool
}

func (m *mockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if m.shouldError {
		return tgbotapi.Message{}, fmt.Errorf("mock send error")
	}
	m.sentMessages = append(m.sentMessages, c)
	return tgbotapi.Message{MessageID: 1}, nil
}

func (m *mockBotAPI) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (m *mockBotAPI) GetLastSentMessage() tgbotapi.Chattable {
	if len(m.sentMessages) == 0 {
		return nil
	}
	return m.sentMessages[len(m.sentMessages)-1]
}

func TestNotificationService_SendReminder(t *testing.T) {
	// 创建测试用户
	user := &models.User{
		ID:         1,
		TelegramID: 123456789,
		FirstName:  "测试用户",
	}

	ctx := context.Background()

	tests := []struct {
		name        string
		reminder    *models.Reminder
		wantErr     bool
		wantContains []string
	}{
		{
			name: "成功发送习惯提醒",
			reminder: &models.Reminder{
				ID:          1,
				UserID:      1,
				Title:       "喝水提醒",
				Description: "每天要喝8杯水",
				Type:        models.ReminderTypeHabit,
				User:        *user,
			},
			wantErr:     false,
			wantContains: []string{"习惯提醒", "喝水提醒"},
		},
		{
			name: "成功发送任务提醒",
			reminder: &models.Reminder{
				ID:          2,
				UserID:      1,
				Title:       "开会提醒",
				Description: "下午3点开会",
				Type:        models.ReminderTypeTask,
				User:        *user,
			},
			wantErr:     false,
			wantContains: []string{"任务提醒", "开会提醒"},
		},
		{
			name: "用户TelegramID为空时失败",
			reminder: &models.Reminder{
				ID:          3,
				UserID:      1,
				Title:       "测试",
				Type:        models.ReminderTypeHabit,
				User:        models.User{TelegramID: 0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBot := &mockBotAPI{}
			service := NewNotificationService(mockBot)

			log := &models.ReminderLog{
				ID:            1,
				ReminderID:    tt.reminder.ID,
				ScheduledTime: time.Now(),
				Status:        models.ReminderStatusPending,
				Reminder:      *tt.reminder,
			}

			err := service.SendReminder(ctx, log)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 验证消息已发送
				if len(mockBot.sentMessages) == 0 {
					t.Error("SendReminder() 未发送任何消息")
					return
				}

				// 获取最后发送的消息
				lastMsg := mockBot.GetLastSentMessage()
				if msgConfig, ok := lastMsg.(tgbotapi.MessageConfig); ok {
					// 验证消息内容
					for _, want := range tt.wantContains {
						if !strings.Contains(msgConfig.Text, want) {
							t.Errorf("消息内容缺少 '%s': %s", want, msgConfig.Text)
						}
					}

					// 验证消息发送到正确的用户
					if msgConfig.ChatID != tt.reminder.User.TelegramID {
						t.Errorf("SendReminder() ChatID = %d, want %d",
							msgConfig.ChatID, tt.reminder.User.TelegramID)
					}

					// 验证有键盘按钮
					if msgConfig.ReplyMarkup == nil {
						t.Error("SendReminder() 消息缺少回复键盘")
					}
				}
			}
		})
	}
}

func TestNotificationService_SendFollowUp(t *testing.T) {
	// 创建测试用户
	user := &models.User{
		ID:         1,
		TelegramID: 123456789,
		FirstName:  "测试用户",
	}

	// 创建测试提醒
	reminder := &models.Reminder{
		ID:          1,
		UserID:      1,
		Title:       "运动提醒",
		Description: "每天运动30分钟",
		Type:        models.ReminderTypeHabit,
		User:        *user,
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		followUpCount int
		wantErr       bool
		wantContains  []string
	}{
		{
			name:          "第一次关怀消息",
			followUpCount: 0,
			wantErr:       false,
			wantContains:  []string{"还没完成吗", "运动提醒"},
		},
		{
			name:          "第二次关怀消息",
			followUpCount: 1,
			wantErr:       false,
			wantContains:  []string{"温馨提醒", "运动提醒"},
		},
		{
			name:          "第三次关怀消息",
			followUpCount: 2,
			wantErr:       false,
			wantContains:  []string{"最后提醒", "运动提醒"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBot := &mockBotAPI{}
			service := NewNotificationService(mockBot)

			log := &models.ReminderLog{
				ID:            1,
				ReminderID:    1,
				ScheduledTime: time.Now().Add(-2 * time.Hour),
				Status:        models.ReminderStatusSent,
				FollowUpCount: tt.followUpCount,
				Reminder:      *reminder,
			}

			err := service.SendFollowUp(ctx, log)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendFollowUp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 验证消息已发送
				if len(mockBot.sentMessages) == 0 {
					t.Error("SendFollowUp() 未发送任何消息")
					return
				}

				// 获取最后发送的消息
				lastMsg := mockBot.GetLastSentMessage()
				if msgConfig, ok := lastMsg.(tgbotapi.MessageConfig); ok {
					// 验证消息内容
					for _, want := range tt.wantContains {
						if !strings.Contains(msgConfig.Text, want) {
							t.Errorf("消息内容缺少 '%s': %s", want, msgConfig.Text)
						}
					}
				}
			}
		})
	}
}

func TestNotificationService_SendError(t *testing.T) {
	user := &models.User{
		ID:         1,
		TelegramID: 123456789,
		FirstName:  "测试用户",
	}

	reminder := &models.Reminder{
		ID:     1,
		UserID: 1,
		Title:  "测试提醒",
		Type:   models.ReminderTypeHabit,
		User:   *user,
	}

	log := &models.ReminderLog{
		ID:            1,
		ReminderID:    1,
		ScheduledTime: time.Now(),
		Status:        models.ReminderStatusPending,
		Reminder:      *reminder,
	}

	ctx := context.Background()

	t.Run("Bot发送失败时返回错误", func(t *testing.T) {
		mockBot := &mockBotAPI{shouldError: true}
		service := NewNotificationService(mockBot)

		err := service.SendReminder(ctx, log)
		if err == nil {
			t.Error("SendReminder() 应该返回错误当Bot发送失败")
		}
	})
}

func TestNotificationService_BuildReminderKeyboard(t *testing.T) {
	mockBot := &mockBotAPI{}
	service := NewNotificationService(mockBot).(*notificationService)

	logID := uint(123)
	keyboard := service.buildReminderKeyboard(logID)

	// 验证键盘有两行
	if len(keyboard.InlineKeyboard) != 2 {
		t.Errorf("buildReminderKeyboard() 行数 = %d, want 2", len(keyboard.InlineKeyboard))
	}

	// 验证第一行有2个按钮
	if len(keyboard.InlineKeyboard[0]) != 2 {
		t.Errorf("buildReminderKeyboard() 第一行按钮数 = %d, want 2", len(keyboard.InlineKeyboard[0]))
	}

	// 验证第二行有2个按钮
	if len(keyboard.InlineKeyboard[1]) != 2 {
		t.Errorf("buildReminderKeyboard() 第二行按钮数 = %d, want 2", len(keyboard.InlineKeyboard[1]))
	}

	// 验证按钮数据包含正确的logID
	completeData := *keyboard.InlineKeyboard[0][0].CallbackData
	expectedComplete := fmt.Sprintf("reminder_complete_%d", logID)
	if completeData != expectedComplete {
		t.Errorf("完成按钮数据 = %s, want %s", completeData, expectedComplete)
	}
}
