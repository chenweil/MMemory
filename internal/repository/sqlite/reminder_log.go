package sqlite

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *reminderLogRepository) GetUserLogs(ctx context.Context, userID uint, since time.Time) ([]*models.ReminderLog, error) {
	query := r.db.WithContext(ctx).
		Table(models.ReminderLog{}.TableName()).
		Joins("JOIN reminders ON reminders.id = reminder_logs.reminder_id").
		Where("reminders.user_id = ?", userID).
		Order("reminder_logs.created_at DESC")

	if !since.IsZero() {
		query = query.Where("reminder_logs.created_at >= ?", since)
	}

	var logs []*models.ReminderLog
	if err := query.Preload("Reminder").Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// CreateDelayReminder 创建延期提醒
func (r *reminderLogRepository) CreateDelayReminder(ctx context.Context, originalLogID uint, delayTime time.Time, delayHours int) error {
	// 获取原始提醒记录
	var originalLog models.ReminderLog
	if err := r.db.WithContext(ctx).
		Preload("Reminder").
		First(&originalLog, originalLogID).Error; err != nil {
		return err
	}

	// 创建新的提醒记录
	delayLog := &models.ReminderLog{
		ReminderID:    originalLog.ReminderID,
		ScheduledTime: delayTime,
		Status:        models.ReminderStatusPending,
		UserResponse:   fmt.Sprintf("延期%d小时", delayHours),
	}

	if err := r.db.WithContext(ctx).Create(delayLog).Error; err != nil {
		return err
	}

	return nil
}

// MarkAsCompleted 标记为已完成
func (r *reminderLogRepository) MarkAsCompleted(ctx context.Context, logID uint, note string) error {
	return r.db.WithContext(ctx).
		Model(&models.ReminderLog{}).
		Where("id = ?", logID).
		Updates(map[string]interface{}{
			"status":        models.ReminderStatusCompleted,
			"user_response":  note,
			"response_time":  time.Now(),
		}).Error
}

// MarkAsSkipped 标记为已跳过
func (r *reminderLogRepository) MarkAsSkipped(ctx context.Context, logID uint, note string) error {
	return r.db.WithContext(ctx).
		Model(&models.ReminderLog{}).
		Where("id = ?", logID).
		Updates(map[string]interface{}{
			"status":       models.ReminderStatusSkipped,
			"user_response": note,
			"response_time": time.Now(),
		}).Error
}

// UpdateFollowUpCount 更新关怀次数
func (r *reminderLogRepository) UpdateFollowUpCount(ctx context.Context, logID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.ReminderLog{}).
		Where("id = ?", logID).
		UpdateColumn("follow_up_count", gorm.Expr("follow_up_count + 1")).Error
}

// GetOverdueReminders 获取超时的提醒
func (r *reminderLogRepository) GetOverdueReminders(ctx context.Context) ([]*models.ReminderLog, error) {
	var logs []*models.ReminderLog

	// 超时时间定义为超过2小时未处理的提醒
	overdueTime := time.Now().Add(-2 * time.Hour)

	err := r.db.WithContext(ctx).
		Preload("Reminder").
		Preload("Reminder.User").
		Where("status = ? AND scheduled_time < ?", models.ReminderStatusSent, overdueTime).
		Find(&logs).Error

	return logs, err
}
