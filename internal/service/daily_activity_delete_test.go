package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"mmemory/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDeleteActivityRepository 用于测试的 mock repository
type mockDeleteActivityRepository struct {
	activities []*models.DailyActivity
	deletedIDs []uint
}

// 确保实现了接口
var _ interface {
	Create(context.Context, *models.DailyActivity) error
	GetByID(context.Context, uint) (*models.DailyActivity, error)
	Delete(context.Context, uint) error
	GetByType(context.Context, uint, models.ActivityType, int, int) ([]*models.DailyActivity, error)
} = (*mockDeleteActivityRepository)(nil)

func newMockDeleteActivityRepository() *mockDeleteActivityRepository {
	return &mockDeleteActivityRepository{
		activities: make([]*models.DailyActivity, 0),
		deletedIDs: make([]uint, 0),
	}
}

func (m *mockDeleteActivityRepository) Create(ctx context.Context, activity *models.DailyActivity) error {
	activity.ID = uint(len(m.activities) + 1)
	m.activities = append(m.activities, activity)
	return nil
}

func (m *mockDeleteActivityRepository) GetByID(ctx context.Context, id uint) (*models.DailyActivity, error) {
	for _, act := range m.activities {
		if act.ID == id {
			return act, nil
		}
	}
	return nil, nil
}

func (m *mockDeleteActivityRepository) GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*models.DailyActivity, error) {
	var result []*models.DailyActivity
	for _, act := range m.activities {
		if act.UserID == userID {
			result = append(result, act)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockDeleteActivityRepository) GetByType(ctx context.Context, userID uint, activityType models.ActivityType, limit, offset int) ([]*models.DailyActivity, error) {
	var result []*models.DailyActivity
	for _, act := range m.activities {
		if act.UserID == userID && act.ActivityType == activityType {
			result = append(result, act)
		}
	}
	return result, nil
}

func (m *mockDeleteActivityRepository) GetByDateRange(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*models.DailyActivity, error) {
	var result []*models.DailyActivity
	for _, act := range m.activities {
		if act.UserID == userID && act.OccurredAt.After(startTime) && act.OccurredAt.Before(endTime) {
			result = append(result, act)
		}
	}
	return result, nil
}

func (m *mockDeleteActivityRepository) GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error) {
	return m.GetByUserID(ctx, userID, limit, 0)
}

func (m *mockDeleteActivityRepository) Update(ctx context.Context, activity *models.DailyActivity) error {
	for i, act := range m.activities {
		if act.ID == activity.ID {
			m.activities[i] = activity
			return nil
		}
	}
	return nil
}

func (m *mockDeleteActivityRepository) Delete(ctx context.Context, id uint) error {
	m.deletedIDs = append(m.deletedIDs, id)
	for i, act := range m.activities {
		if act.ID == id {
			m.activities = append(m.activities[:i], m.activities[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockDeleteActivityRepository) GetStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (map[string]int64, error) {
	stats := make(map[string]int64)
	for _, act := range m.activities {
		if act.UserID == userID {
			stats[string(act.ActivityType)]++
		}
	}
	return stats, nil
}

func TestDailyActivityService_DeleteActivities(t *testing.T) {
	ctx := context.Background()
	mockRepo := newMockDeleteActivityRepository()
	service := NewDailyActivityService(mockRepo)

	t.Run("删除指定书名的阅读记录", func(t *testing.T) {
		// 创建测试数据
		book1Details := &models.ActivityDetails{BookName: "？", Chapter: "未知"}
		details1, _ := json.Marshal(book1Details)
		activity1 := &models.DailyActivity{
			UserID:       1,
			ActivityType: models.ActivityTypeReadBook,
			Details:      string(details1),
			OccurredAt:   time.Now(),
		}
		mockRepo.Create(ctx, activity1)

		book2Details := &models.ActivityDetails{BookName: "如何阅读一本书", Chapter: "第10章"}
		details2, _ := json.Marshal(book2Details)
		activity2 := &models.DailyActivity{
			UserID:       1,
			ActivityType: models.ActivityTypeReadBook,
			Details:      string(details2),
			OccurredAt:   time.Now(),
		}
		mockRepo.Create(ctx, activity2)

		// 执行删除
		criteria := map[string]interface{}{"book_name": "？"}
		deleted, err := service.DeleteActivities(ctx, 1, models.ActivityTypeReadBook, criteria)

		// 验证结果
		require.NoError(t, err)
		assert.Equal(t, 1, deleted)
		assert.Len(t, mockRepo.deletedIDs, 1)

		// 验证只剩下一条记录
		activities, _ := mockRepo.GetByType(ctx, 1, models.ActivityTypeReadBook, 10, 0)
		assert.Len(t, activities, 1)
		// 验证Details包含正确的书名
		details, _ := activities[0].GetDetails()
		assert.Equal(t, "如何阅读一本书", details.BookName)
	})

	t.Run("删除不存在的记录返回0", func(t *testing.T) {
		mockRepo.deletedIDs = nil // 清空删除记录
		criteria := map[string]interface{}{"book_name": "不存在的书"}
		deleted, err := service.DeleteActivities(ctx, 1, models.ActivityTypeReadBook, criteria)

		require.NoError(t, err)
		assert.Equal(t, 0, deleted)
		assert.Len(t, mockRepo.deletedIDs, 0)
	})

	t.Run("删除所有匹配的记录", func(t *testing.T) {
		// 创建多条相同书名的记录
		for i := 0; i < 3; i++ {
			details := &models.ActivityDetails{BookName: "测试书", Chapter: "第1章"}
			detailsJSON, _ := json.Marshal(details)
			activity := &models.DailyActivity{
				UserID:       2,
				ActivityType: models.ActivityTypeReadBook,
				Details:      string(detailsJSON),
				OccurredAt:   time.Now(),
			}
			mockRepo.Create(ctx, activity)
		}

		criteria := map[string]interface{}{"book_name": "测试书"}
		deleted, err := service.DeleteActivities(ctx, 2, models.ActivityTypeReadBook, criteria)

		require.NoError(t, err)
		assert.Equal(t, 3, deleted)
	})

	t.Run("删除喝水记录 - 无时间范围", func(t *testing.T) {
		// 创建喝水记录
		for i := 0; i < 2; i++ {
			details := &models.ActivityDetails{Amount: "1杯"}
			detailsJSON, _ := json.Marshal(details)
			activity := &models.DailyActivity{
				UserID:       3,
				ActivityType: models.ActivityTypeDrinkWater,
				Details:      string(detailsJSON),
				OccurredAt:   time.Now(),
			}
			mockRepo.Create(ctx, activity)
		}

		criteria := map[string]interface{}{} // 没有时间范围，删除所有
		deleted, err := service.DeleteActivities(ctx, 3, models.ActivityTypeDrinkWater, criteria)

		require.NoError(t, err)
		assert.Equal(t, 2, deleted)
	})

	t.Run("删除喝水记录 - 指定时间范围", func(t *testing.T) {
		// 创建不同时间的喝水记录
		yesterday := time.Now().Add(-24 * time.Hour)

		oldDetails := &models.ActivityDetails{Amount: "1杯"}
		oldDetailsJSON, _ := json.Marshal(oldDetails)
		oldActivity := &models.DailyActivity{
			UserID:       4,
			ActivityType: models.ActivityTypeDrinkWater,
			Details:      string(oldDetailsJSON),
			OccurredAt:   yesterday,
		}
		mockRepo.Create(ctx, oldActivity)

		newDetails := &models.ActivityDetails{Amount: "1杯"}
		newDetailsJSON, _ := json.Marshal(newDetails)
		newActivity := &models.DailyActivity{
			UserID:       4,
			ActivityType: models.ActivityTypeDrinkWater,
			Details:      string(newDetailsJSON),
			OccurredAt:   time.Now(),
		}
		mockRepo.Create(ctx, newActivity)

		// 删除昨天的记录
		criteria := map[string]interface{}{"time_range": "昨天"}
		deleted, err := service.DeleteActivities(ctx, 4, models.ActivityTypeDrinkWater, criteria)

		require.NoError(t, err)
		assert.Equal(t, 1, deleted) // 只删除昨天的
	})
}
