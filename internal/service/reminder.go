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
}

func NewReminderService(reminderRepo interfaces.ReminderRepository) ReminderService {
	return &reminderService{
		reminderRepo: reminderRepo,
		parser:       NewParserService(),
	}
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

	return s.reminderRepo.Create(ctx, reminder)
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
	return s.reminderRepo.Update(ctx, reminder)
}

func (s *reminderService) DeleteReminder(ctx context.Context, id uint) error {
	if id == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}
	return s.reminderRepo.Delete(ctx, id)
}