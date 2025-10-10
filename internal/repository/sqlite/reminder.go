package sqlite

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type reminderRepository struct {
	db *gorm.DB
}

func NewReminderRepository(db *gorm.DB) interfaces.ReminderRepository {
	return &reminderRepository{db: db}
}

func (r *reminderRepository) Create(ctx context.Context, reminder *models.Reminder) error {
	return r.db.WithContext(ctx).Create(reminder).Error
}

func (r *reminderRepository) GetByID(ctx context.Context, id uint) (*models.Reminder, error) {
	var reminder models.Reminder
	err := r.db.WithContext(ctx).Preload("User").First(&reminder, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reminder, nil
}

func (r *reminderRepository) GetByUserID(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	var reminders []*models.Reminder
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&reminders).Error
	return reminders, err
}

func (r *reminderRepository) GetActiveReminders(ctx context.Context) ([]*models.Reminder, error) {
	var reminders []*models.Reminder
	err := r.db.WithContext(ctx).Preload("User").Where("is_active = ?", true).Find(&reminders).Error
	return reminders, err
}

func (r *reminderRepository) Update(ctx context.Context, reminder *models.Reminder) error {
	return r.db.WithContext(ctx).Save(reminder).Error
}

func (r *reminderRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Reminder{}, id).Error
}

func (r *reminderRepository) CountByStatus(ctx context.Context, status models.ReminderStatStatus) (int64, error) {
	var count int64
	
	switch status {
	case models.ReminderStatStatusActive:
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).Where("is_active = ?", true).Count(&count).Error
		return count, err
	case models.ReminderStatStatusCompleted:
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).Where("is_active = ?", false).Count(&count).Error
		return count, err
	case models.ReminderStatStatusExpired:
		// 这里需要根据业务逻辑定义过期的条件
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).
			Where("is_active = ? AND schedule_pattern = ?", true, string(models.SchedulePatternOnce)).
			Count(&count).Error
		return count, err
	default:
		return 0, nil
	}
}