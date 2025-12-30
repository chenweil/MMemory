package sqlite

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/metrics"
)

type reminderRepository struct {
	db            *gorm.DB
	queryOptimizer *QueryOptimizer
}

func NewReminderRepository(db *gorm.DB, queryOptimizer *QueryOptimizer) interfaces.ReminderRepository {
	return &reminderRepository{
		db:            db,
		queryOptimizer: queryOptimizer,
	}
}

func (r *reminderRepository) Create(ctx context.Context, reminder *models.Reminder) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Create(reminder).Error
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
	}

	if r.queryOptimizer != nil {
		r.queryOptimizer.RecordQuery("reminders", "create", "INSERT INTO reminders", duration, 1, false)
		if duration >= r.queryOptimizer.GetSlowThreshold() {
			metrics.RecordSlowQuery("reminders", "create", r.queryOptimizer.GetSlowThreshold().String())
		}
	}

	metrics.RecordQueryDuration("reminders", "create", duration.Seconds())
	metrics.RecordQueryTotal("reminders", "create", status)

	return err
}

func (r *reminderRepository) GetByID(ctx context.Context, id uint) (*models.Reminder, error) {
	start := time.Now()
	var reminder models.Reminder
	err := r.db.WithContext(ctx).Preload("User").First(&reminder, id).Error
	duration := time.Since(start)

	status := "success"
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		status = "error"
	}

	if r.queryOptimizer != nil {
		r.queryOptimizer.RecordQuery("reminders", "get_by_id", "SELECT * FROM reminders WHERE id = ?", duration, 1, false)
	}
	metrics.RecordQueryDuration("reminders", "get_by_id", duration.Seconds())
	metrics.RecordQueryTotal("reminders", "get_by_id", status)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reminder, nil
}

func (r *reminderRepository) GetByUserID(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	start := time.Now()
	var reminders []*models.Reminder
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&reminders).Error
	duration := time.Since(start)

	if r.queryOptimizer != nil {
		r.queryOptimizer.RecordQuery("reminders", "get_by_user", "SELECT * FROM reminders WHERE user_id = ?", duration, int64(len(reminders)), false)
	}
	metrics.RecordQueryDuration("reminders", "get_by_user", duration.Seconds())
	metrics.RecordQueryTotal("reminders", "get_by_user", "success")

	return reminders, err
}

func (r *reminderRepository) GetActiveReminders(ctx context.Context) ([]*models.Reminder, error) {
	start := time.Now()
	var reminders []*models.Reminder
	err := r.db.WithContext(ctx).Preload("User").Where("is_active = ?", true).Find(&reminders).Error
	duration := time.Since(start)

	if r.queryOptimizer != nil {
		r.queryOptimizer.RecordQuery("reminders", "get_active", "SELECT * FROM reminders WHERE is_active = true", duration, int64(len(reminders)), false)
	}
	metrics.RecordQueryDuration("reminders", "get_active", duration.Seconds())
	metrics.RecordQueryTotal("reminders", "get_active", "success")

	return reminders, err
}

func (r *reminderRepository) Update(ctx context.Context, reminder *models.Reminder) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Save(reminder).Error
	duration := time.Since(start)

	if r.queryOptimizer != nil {
		r.queryOptimizer.RecordQuery("reminders", "update", "UPDATE reminders SET ...", duration, 1, false)
	}
	metrics.RecordQueryDuration("reminders", "update", duration.Seconds())
	metrics.RecordQueryTotal("reminders", "update", "success")

	return err
}

func (r *reminderRepository) Delete(ctx context.Context, id uint) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Delete(&models.Reminder{}, id).Error
	duration := time.Since(start)

	if r.queryOptimizer != nil {
		r.queryOptimizer.RecordQuery("reminders", "delete", "DELETE FROM reminders WHERE id = ?", duration, 1, false)
	}
	metrics.RecordQueryDuration("reminders", "delete", duration.Seconds())
	metrics.RecordQueryTotal("reminders", "delete", "success")

	return err
}

func (r *reminderRepository) CountByStatus(ctx context.Context, status models.ReminderStatStatus) (int64, error) {
	var count int64

	switch status {
	case models.ReminderStatStatusActive:
		start := time.Now()
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).Where("is_active = ?", true).Count(&count).Error
		duration := time.Since(start)
		if r.queryOptimizer != nil {
			r.queryOptimizer.RecordQuery("reminders", "count_active", "SELECT COUNT(*) FROM reminders WHERE is_active = true", duration, 0, false)
		}
		return count, err
	case models.ReminderStatStatusCompleted:
		start := time.Now()
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).Where("is_active = ?", false).Count(&count).Error
		duration := time.Since(start)
		if r.queryOptimizer != nil {
			r.queryOptimizer.RecordQuery("reminders", "count_completed", "SELECT COUNT(*) FROM reminders WHERE is_active = false", duration, 0, false)
		}
		return count, err
	case models.ReminderStatStatusExpired:
		start := time.Now()
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).
			Where("is_active = ? AND schedule_pattern LIKE ?", true, string(models.SchedulePatternOnce)+"%").
			Count(&count).Error
		duration := time.Since(start)
		if r.queryOptimizer != nil {
			r.queryOptimizer.RecordQuery("reminders", "count_expired", "SELECT COUNT(*) FROM reminders WHERE ...", duration, 0, false)
		}
		return count, err
	default:
		return 0, nil
	}
}
