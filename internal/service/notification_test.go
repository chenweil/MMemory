package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mmemory/internal/models"
)

// testNotificationService 测试专用的通知服务
type testNotificationService struct {
	sentMessages []string
	sentUsers    []int64
}

func newTestNotificationService() *testNotificationService {
	return &testNotificationService{
		sentMessages: make([]string, 0),
		sentUsers:    make([]int64, 0),
	}
}

func (s *testNotificationService) SendReminder(ctx context.Context, log *models.ReminderLog) error {
	if log.Reminder.User.TelegramID == 0 {
		return fmt.Errorf("用户Telegram ID为空")
	}
	
	// 构建提醒消息
	message := s.buildReminderMessage(&log.Reminder)
	
	// 记录发送的消息
	s.sentMessages = append(s.sentMessages, message)
	s.sentUsers = append(s.sentUsers, log.Reminder.User.TelegramID)
	
	return nil
}

func (s *testNotificationService) SendFollowUp(ctx context.Context, log *models.ReminderLog) error {
	if log.Reminder.User.TelegramID == 0 {
		return fmt.Errorf("用户Telegram ID为空")
	}
	
	// 构建关怀消息
	message := s.buildFollowUpMessage(&log.Reminder, log.FollowUpCount)
	
	// 记录发送的消息
	s.sentMessages = append(s.sentMessages, message)
	s.sentUsers = append(s.sentUsers, log.Reminder.User.TelegramID)
	
	return nil
}

// buildReminderMessage 构建提醒消息
func (s *testNotificationService) buildReminderMessage(reminder *models.Reminder) string {
	var message string
	
	// 根据提醒类型使用不同的emoji和措辞
	switch reminder.Type {
	case models.ReminderTypeHabit:
		message = fmt.Sprintf("⏰ 习惯提醒\n\n"+
			"📝 %s\n\n"+
			"已经到了约定的时间，完成了吗？", reminder.Title)
	case models.ReminderTypeTask:
		message = fmt.Sprintf("📋 任务提醒\n\n"+
			"📝 %s\n\n"+
			"该处理这个任务了，准备好了吗？", reminder.Title)
	default:
		message = fmt.Sprintf("🔔 提醒\n\n"+
			"📝 %s\n\n"+
			"时间到了，请查看！", reminder.Title)
	}
	
	return message
}

// buildFollowUpMessage 构建关怀消息
func (s *testNotificationService) buildFollowUpMessage(reminder *models.Reminder, followUpCount int) string {
	var message string
	
	switch followUpCount {
	case 0:
		message = fmt.Sprintf("🤔 还没完成吗？\n\n"+
			"📝 %s\n\n"+
			"没关系，有什么困难吗？需要延期还是跳过？", reminder.Title)
	case 1:
		message = fmt.Sprintf("😊 温馨提醒\n\n"+
			"📝 %s\n\n"+
			"这个任务还在等着你呢，要不要处理一下？", reminder.Title)
	default:
		message = fmt.Sprintf("💪 最后提醒\n\n"+
			"📝 %s\n\n"+
			"今天确实不方便的话，可以选择跳过哦～", reminder.Title)
	}
	
	return message
}

func TestNotificationService_SendReminder(t *testing.T) {
	service := newTestNotificationService()
	
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
		Title:       "喝水提醒",
		Description: "每天要喝8杯水",
		Type:        models.ReminderTypeHabit,
		User:        *user,
	}
	
	// 创建测试日志
	log := &models.ReminderLog{
		ID:            1,
		ReminderID:    1,
		ScheduledTime: time.Now(),
		Status:        models.ReminderStatusPending,
		Reminder:      *reminder,
	}
	
	ctx := context.Background()
	
	tests := []struct {
		name          string
		log           *models.ReminderLog
		wantErr       bool
		wantMsgCount  int
	}{
		{
			name:         "成功发送习惯提醒",
			log:          log,
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name: "成功发送任务提醒",
			log: &models.ReminderLog{
				ID:            2,
				ReminderID:    2,
				ScheduledTime: time.Now(),
				Status:        models.ReminderStatusPending,
				Reminder: models.Reminder{
					ID:          2,
					UserID:      1,
					Title:       "开会提醒",
					Description: "下午3点开会",
					Type:        models.ReminderTypeTask,
					User:        *user,
				},
			},
			wantErr:      false,
			wantMsgCount: 1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialCount := len(service.sentMessages)
			
			err := service.SendReminder(ctx, tt.log)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("SendReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				finalCount := len(service.sentMessages)
				actualCount := finalCount - initialCount
				
				if actualCount != tt.wantMsgCount {
					t.Errorf("SendReminder() 发送消息数 = %d, want %d", actualCount, tt.wantMsgCount)
				}
				
				// 验证消息发送到正确的用户
				if finalCount > 0 {
					lastUserID := service.sentUsers[finalCount-1]
					if lastUserID != tt.log.Reminder.User.TelegramID {
						t.Errorf("SendReminder() UserID = %d, want %d", 
							lastUserID, tt.log.Reminder.User.TelegramID)
					}
				}
			}
		})
	}
}

func TestNotificationService_SendFollowUp(t *testing.T) {
	service := newTestNotificationService()
	
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
		name         string
		followUpCount int
		wantErr      bool
		wantMsgCount int
	}{
		{
			name:         "第一次关怀消息",
			followUpCount: 0,
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name:         "第二次关怀消息",
			followUpCount: 1,
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name:         "第三次关怀消息",
			followUpCount: 2,
			wantErr:      false,
			wantMsgCount: 1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试日志
			log := &models.ReminderLog{
				ID:            1,
				ReminderID:    1,
				ScheduledTime: time.Now().Add(-2 * time.Hour),
				Status:        models.ReminderStatusSent,
				FollowUpCount: tt.followUpCount,
				Reminder:      *reminder,
			}
			
			initialCount := len(service.sentMessages)
			
			err := service.SendFollowUp(ctx, log)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("SendFollowUp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				finalCount := len(service.sentMessages)
				actualCount := finalCount - initialCount
				
				if actualCount != tt.wantMsgCount {
					t.Errorf("SendFollowUp() 发送消息数 = %d, want %d", actualCount, tt.wantMsgCount)
				}
				
				// 验证消息发送到正确的用户
				if finalCount > 0 {
					lastUserID := service.sentUsers[finalCount-1]
					if lastUserID != user.TelegramID {
						t.Errorf("SendFollowUp() UserID = %d, want %d", 
							lastUserID, user.TelegramID)
					}
				}
			}
		})
	}
}

func TestNotificationService_MessageContent(t *testing.T) {
	service := newTestNotificationService()
	
	// 创建测试用户
	user := &models.User{
		ID:         1,
		TelegramID: 123456789,
		FirstName:  "张三",
	}
	
	ctx := context.Background()
	
	tests := []struct {
		name           string
		reminderType   models.ReminderType
		reminderTitle  string
		wantContains   []string
	}{
		{
			name:          "习惯提醒消息格式",
			reminderType:  models.ReminderTypeHabit,
			reminderTitle: "每日阅读",
			wantContains:  []string{"习惯提醒", "每日阅读", "完成了吗"},
		},
		{
			name:          "任务提醒消息格式",
			reminderType:  models.ReminderTypeTask,
			reminderTitle: "项目会议",
			wantContains:  []string{"任务提醒", "项目会议", "准备好了吗"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试提醒和日志
			reminder := &models.Reminder{
				ID:     1,
				UserID: 1,
				Title:  tt.reminderTitle,
				Type:   tt.reminderType,
				User:   *user,
			}
			
			log := &models.ReminderLog{
				ID:            1,
				ReminderID:    1,
				ScheduledTime: time.Now(),
				Status:        models.ReminderStatusPending,
				Reminder:      *reminder,
			}
			
			initialCount := len(service.sentMessages)
			
			err := service.SendReminder(ctx, log)
			if err != nil {
				t.Fatalf("SendReminder() error = %v", err)
			}
			
			// 检查消息内容
			if len(service.sentMessages) > initialCount {
				msg := service.sentMessages[len(service.sentMessages)-1]
				for _, want := range tt.wantContains {
					if !testContains(msg, want) {
						t.Errorf("消息内容缺少 '%s': %s", want, msg)
					}
				}
			}
		})
	}
}

// 辅助函数：检查字符串是否包含子字符串
func testContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}