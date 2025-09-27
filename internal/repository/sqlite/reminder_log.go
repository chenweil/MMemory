package sqlite

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type reminderLogRepository struct {
	db *gorm.DB
}

func NewReminderLogRepository(db *gorm.DB) interfaces.ReminderLogRepository {
	return &reminderLogRepository{db: db}
}

func (r *reminderLogRepository) Create(ctx context.Context, log *models.ReminderLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *reminderLogRepository) GetByID(ctx context.Context, id uint) (*models.ReminderLog, error) {
	var log models.ReminderLog
	err := r.db.WithContext(ctx).
		Preload("Reminder").
		Preload("Reminder.User").
		First(&log, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

func (r *reminderLogRepository) GetByReminderID(ctx context.Context, reminderID uint, limit, offset int) ([]*models.ReminderLog, error) {
	var logs []*models.ReminderLog
	query := r.db.WithContext(ctx).Where("reminder_id = ?", reminderID).Order("scheduled_time DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	err := query.Find(&logs).Error
	return logs, err
}

func (r *reminderLogRepository) GetPendingLogs(ctx context.Context) ([]*models.ReminderLog, error) {
	var logs []*models.ReminderLog
	err := r.db.WithContext(ctx).
		Preload("Reminder").
		Preload("Reminder.User").
		Where("status IN ?", []models.ReminderStatus{models.ReminderStatusPending, models.ReminderStatusSent}).
		Find(&logs).Error
	return logs, err
}

func (r *reminderLogRepository) Update(ctx context.Context, log *models.ReminderLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *reminderLogRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ReminderLog{}, id).Error
}
