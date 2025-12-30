package sqlite

import (
	"context"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"

	"gorm.io/gorm"
)

type dailyActivityRepository struct {
	db *gorm.DB
}

// NewDailyActivityRepository 创建日常活动仓储
func NewDailyActivityRepository(db *gorm.DB) interfaces.DailyActivityRepository {
	return &dailyActivityRepository{db: db}
}

func (r *dailyActivityRepository) Create(ctx context.Context, activity *models.DailyActivity) error {
	return r.db.WithContext(ctx).Create(activity).Error
}

func (r *dailyActivityRepository) GetByID(ctx context.Context, id uint) (*models.DailyActivity, error) {
	var activity models.DailyActivity
	err := r.db.WithContext(ctx).Preload("User").First(&activity, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &activity, nil
}

func (r *dailyActivityRepository) GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*models.DailyActivity, error) {
	var activities []*models.DailyActivity
	query := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("occurred_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&activities).Error
	return activities, err
}

func (r *dailyActivityRepository) GetByType(ctx context.Context, userID uint, activityType models.ActivityType, limit, offset int) ([]*models.DailyActivity, error) {
	var activities []*models.DailyActivity
	query := r.db.WithContext(ctx).Where("user_id = ? AND activity_type = ?", userID, activityType).Order("occurred_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&activities).Error
	return activities, err
}

func (r *dailyActivityRepository) GetByDateRange(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*models.DailyActivity, error) {
	var activities []*models.DailyActivity
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND occurred_at >= ? AND occurred_at <= ?", userID, startTime, endTime).
		Order("occurred_at DESC").
		Find(&activities).Error
	return activities, err
}

func (r *dailyActivityRepository) GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error) {
	var activities []*models.DailyActivity
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&activities).Error
	return activities, err
}

func (r *dailyActivityRepository) Update(ctx context.Context, activity *models.DailyActivity) error {
	return r.db.WithContext(ctx).Save(activity).Error
}

func (r *dailyActivityRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.DailyActivity{}, id).Error
}

func (r *dailyActivityRepository) GetStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (map[string]int64, error) {
	type Stat struct {
		Type  string
		Count int64
	}

	var stats []Stat
	err := r.db.WithContext(ctx).
		Model(&models.DailyActivity{}).
		Select("activity_type as type, count(*) as count").
		Where("user_id = ? AND occurred_at >= ? AND occurred_at <= ?", userID, startTime, endTime).
		Group("activity_type").
		Find(&stats).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, stat := range stats {
		result[stat.Type] = stat.Count
	}

	return result, nil
}