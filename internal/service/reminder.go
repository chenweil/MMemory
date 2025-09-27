package service

import (
	"context"
	"fmt"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type reminderService struct {
	reminderRepo interfaces.ReminderRepository
	parser       *parserService
	scheduler    SchedulerService
}

func NewReminderService(reminderRepo interfaces.ReminderRepository) ReminderService {
	return &reminderService{
		reminderRepo: reminderRepo,
		parser:       NewParserService(),
	}
}

// SetScheduler 设置调度器 (用于避免循环依赖)
func (s *reminderService) SetScheduler(scheduler SchedulerService) {
	s.scheduler = scheduler
}

func (s *reminderService) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	if reminder.UserID == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if reminder.Title == "" {
		return fmt.Errorf("提醒标题不能为空")
	}
	if reminder.TargetTime == "" {
		return fmt.Errorf("提醒时间不能为空")
	}

	// 保存到数据库
	if err := s.reminderRepo.Create(ctx, reminder); err != nil {
		return err
	}

	// 添加到调度器
	if s.scheduler != nil && reminder.IsActive {
		if err := s.scheduler.AddReminder(reminder); err != nil {
			// 调度失败不影响数据库保存，只记录错误
			fmt.Printf("添加调度失败: %v", err)
		}
	}

	return nil
}

func (s *reminderService) ParseReminderFromText(ctx context.Context, text string, userID uint) (*models.Reminder, error) {
	return s.parser.ParseReminderFromText(ctx, text, userID)
}

func (s *reminderService) GetUserReminders(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	return s.reminderRepo.GetByUserID(ctx, userID)
}

func (s *reminderService) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	if reminder.ID == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	// 更新数据库
	if err := s.reminderRepo.Update(ctx, reminder); err != nil {
		return err
	}

	// 更新调度器
	if s.scheduler != nil {
		// 先移除旧的调度
		s.scheduler.RemoveReminder(reminder.ID)
		
		// 如果仍然活跃，添加新的调度
		if reminder.IsActive {
			if err := s.scheduler.AddReminder(reminder); err != nil {
				fmt.Printf("更新调度失败: %v", err)
			}
		}
	}

	return nil
}

func (s *reminderService) DeleteReminder(ctx context.Context, id uint) error {
	if id == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	// 从调度器移除
	if s.scheduler != nil {
		s.scheduler.RemoveReminder(id)
	}

	// 从数据库删除
	return s.reminderRepo.Delete(ctx, id)
}