package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/internal/models"
)

// Mock DailyActivityRepository for testing
type mockDailyActivityRepository struct {
	activities map[uint]*models.DailyActivity
	idCounter  uint
	mu         sync.Mutex
	stats      map[string]int64 // 用于统计查询
}

func newMockDailyActivityRepository() *mockDailyActivityRepository {
	return &mockDailyActivityRepository{
		activities: make(map[uint]*models.DailyActivity),
		idCounter:  1,
		stats:      make(map[string]int64),
	}
}

func (m *mockDailyActivityRepository) Create(ctx context.Context, activity *models.DailyActivity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	activity.ID = m.idCounter
	m.activities[m.idCounter] = activity
	m.idCounter++

	// 更新统计
	m.stats[string(activity.ActivityType)]++

	return nil
}

func (m *mockDailyActivityRepository) GetByID(ctx context.Context, id uint) (*models.DailyActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.activities[id], nil
}

func (m *mockDailyActivityRepository) GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*models.DailyActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.DailyActivity
	for _, activity := range m.activities {
		if activity.UserID == userID {
			result = append(result, activity)
		}
	}

	// 应用分页
	if offset > 0 && len(result) > offset {
		result = result[offset:]
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (m *mockDailyActivityRepository) GetByType(ctx context.Context, userID uint, activityType models.ActivityType, limit, offset int) ([]*models.DailyActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.DailyActivity
	for _, activity := range m.activities {
		if activity.UserID == userID && activity.ActivityType == activityType {
			result = append(result, activity)
		}
	}

	// 应用分页
	if offset > 0 && len(result) > offset {
		result = result[offset:]
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (m *mockDailyActivityRepository) GetByDateRange(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*models.DailyActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.DailyActivity
	for _, activity := range m.activities {
		if activity.UserID == userID &&
			(activity.OccurredAt.Equal(startTime) || activity.OccurredAt.After(startTime)) &&
			(activity.OccurredAt.Before(endTime) || activity.OccurredAt.Equal(endTime)) {
			result = append(result, activity)
		}
	}

	return result, nil
}

func (m *mockDailyActivityRepository) GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.DailyActivity
	for _, activity := range m.activities {
		if activity.UserID == userID {
			result = append(result, activity)
		}
	}

	// 按时间倒序排序并限制数量
	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (m *mockDailyActivityRepository) Update(ctx context.Context, activity *models.DailyActivity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing := m.activities[activity.ID]; existing != nil {
		m.activities[activity.ID] = activity
	}
	return nil
}

func (m *mockDailyActivityRepository) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.activities, id)
	return nil
}

func (m *mockDailyActivityRepository) GetStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (map[string]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 简化统计实现
	result := make(map[string]int64)
	for _, activity := range m.activities {
		if activity.UserID == userID &&
			(activity.OccurredAt.Equal(startTime) || activity.OccurredAt.After(startTime)) &&
			(activity.OccurredAt.Before(endTime) || activity.OccurredAt.Equal(endTime)) {
			result[string(activity.ActivityType)]++
		}
	}

	return result, nil
}

// TestDailyActivityService_RecordActivity 测试记录活动功能
func TestDailyActivityService_RecordActivity(t *testing.T) {
	repo := newMockDailyActivityRepository()
	service := NewDailyActivityService(repo)

	ctx := context.Background()
	userID := uint(1)

	t.Run("记录喝水活动", func(t *testing.T) {
		details := map[string]interface{}{
			"amount":    "200ml",
			"waterType": "温水",
		}

		activity, err := service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, details, models.SourceConversation)

		require.NoError(t, err)
		assert.NotNil(t, activity)
		assert.Equal(t, userID, activity.UserID)
		assert.Equal(t, models.ActivityTypeDrinkWater, activity.ActivityType)
		assert.Equal(t, models.SourceConversation, activity.Source)
	})

	t.Run("记录运动活动", func(t *testing.T) {
		details := map[string]interface{}{
			"exerciseType":     "跑步",
			"exerciseDuration": "30分钟",
			"distance":         "5公里",
		}

		activity, err := service.RecordActivity(ctx, userID, models.ActivityTypeExercise, details, models.SourceReminder)

		require.NoError(t, err)
		assert.NotNil(t, activity)
		assert.Equal(t, models.ActivityTypeExercise, activity.ActivityType)
	})

	t.Run("记录读书活动", func(t *testing.T) {
		details := map[string]interface{}{
			"book_name": "如何阅读一本书",
			"chapter":   "第十一章",
		}

		activity, err := service.RecordActivity(ctx, userID, models.ActivityTypeReadBook, details, models.SourceManual)

		require.NoError(t, err)
		assert.NotNil(t, activity)
		assert.Equal(t, models.ActivityTypeReadBook, activity.ActivityType)
	})
}

// TestDailyActivityService_GetRecentActivities 测试获取最近活动
func TestDailyActivityService_GetRecentActivities(t *testing.T) {
	repo := newMockDailyActivityRepository()
	service := NewDailyActivityService(repo)

	ctx := context.Background()
	userID := uint(1)

	// 先创建一些活动
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, map[string]interface{}{"amount": "1杯"}, models.SourceConversation)
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeReadBook, map[string]interface{}{"book_name": "书1"}, models.SourceConversation)
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeExercise, map[string]interface{}{"type": "跑步"}, models.SourceConversation)

	t.Run("获取最近3条活动", func(t *testing.T) {
		activities, err := service.GetRecentActivities(ctx, userID, 3)

		require.NoError(t, err)
		assert.Len(t, activities, 3)
	})

	t.Run("限制获取数量", func(t *testing.T) {
		activities, err := service.GetRecentActivities(ctx, userID, 2)

		require.NoError(t, err)
		assert.Len(t, activities, 2)
	})
}

// TestDailyActivityService_GetActivitiesByType 测试按类型获取活动
func TestDailyActivityService_GetActivitiesByType(t *testing.T) {
	repo := newMockDailyActivityRepository()
	service := NewDailyActivityService(repo)

	ctx := context.Background()
	userID := uint(1)

	// 创建不同类型的活动
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, map[string]interface{}{"amount": "1杯"}, models.SourceConversation)
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, map[string]interface{}{"amount": "2杯"}, models.SourceConversation)
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeReadBook, map[string]interface{}{"book_name": "书1"}, models.SourceConversation)

	t.Run("获取喝水活动", func(t *testing.T) {
		activities, err := service.GetActivitiesByType(ctx, userID, models.ActivityTypeDrinkWater, 10)

		require.NoError(t, err)
		assert.Len(t, activities, 2)
		for _, act := range activities {
			assert.Equal(t, models.ActivityTypeDrinkWater, act.ActivityType)
		}
	})

	t.Run("获取看书活动", func(t *testing.T) {
		activities, err := service.GetActivitiesByType(ctx, userID, models.ActivityTypeReadBook, 10)

		require.NoError(t, err)
		assert.Len(t, activities, 1)
		assert.Equal(t, models.ActivityTypeReadBook, activities[0].ActivityType)
	})

	t.Run("限制返回数量", func(t *testing.T) {
		activities, err := service.GetActivitiesByType(ctx, userID, models.ActivityTypeDrinkWater, 1)

		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})
}

// TestDailyActivityService_GetActivityStatistics 测试获取统计信息
func TestDailyActivityService_GetActivityStatistics(t *testing.T) {
	repo := newMockDailyActivityRepository()
	service := NewDailyActivityService(repo)

	ctx := context.Background()
	userID := uint(1)

	// 创建多个活动
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, map[string]interface{}{"amount": "1杯"}, models.SourceConversation)
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, map[string]interface{}{"amount": "2杯"}, models.SourceConversation)
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeExercise, map[string]interface{}{"type": "跑步"}, models.SourceConversation)

	t.Run("统计最近7天", func(t *testing.T) {
		stats, err := service.GetActivityStatistics(ctx, userID, "最近7天")

		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats["drink_water"], int64(2))
		assert.GreaterOrEqual(t, stats["exercise"], int64(1))
	})

	t.Run("统计最近10天", func(t *testing.T) {
		stats, err := service.GetActivityStatistics(ctx, userID, "最近10天")

		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats["drink_water"], int64(2))
	})

	t.Run("统计今天", func(t *testing.T) {
		stats, err := service.GetActivityStatistics(ctx, userID, "今天")

		require.NoError(t, err)
		assert.NotNil(t, stats)
	})
}

// TestDailyActivityService_QueryActivities 测试查询活动并返回自然语言
func TestDailyActivityService_QueryActivities(t *testing.T) {
	repo := newMockDailyActivityRepository()
	service := NewDailyActivityService(repo)

	ctx := context.Background()
	userID := uint(1)

	// 创建活动
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeReadBook, map[string]interface{}{
		"book_name": "如何阅读一本书",
		"chapter":   "第十一章",
	}, models.SourceConversation)

	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeReadBook, map[string]interface{}{
		"book_name": "深入理解计算机系统",
		"chapter":   "第5章",
	}, models.SourceConversation)

	t.Run("按类型查询读书记录", func(t *testing.T) {
		response, err := service.QueryActivities(ctx, userID, "by_type", "read_book", "最近7天")

		require.NoError(t, err)
		assert.Contains(t, response, "如何阅读一本书")
		assert.Contains(t, response, "深入理解计算机系统")
		assert.Contains(t, response, "第十一章")
	})

	t.Run("按类型查询喝水记录", func(t *testing.T) {
		_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, map[string]interface{}{"amount": "1杯"}, models.SourceConversation)

		response, err := service.QueryActivities(ctx, userID, "by_type", "drink_water", "最近7天")

		require.NoError(t, err)
		assert.Contains(t, response, "喝水")
	})

	t.Run("按时间查询最近10天", func(t *testing.T) {
		response, err := service.QueryActivities(ctx, userID, "by_time", "", "最近10天")

		require.NoError(t, err)
		assert.Contains(t, response, "最近10天")
		assert.Contains(t, response, "看书")
	})

	t.Run("按类型查询无记录", func(t *testing.T) {
		response, err := service.QueryActivities(ctx, userID, "by_type", "sleep", "最近7天")

		require.NoError(t, err)
		assert.Contains(t, response, "没有找到")
	})

	t.Run("查询统计数据", func(t *testing.T) {
		response, err := service.QueryActivities(ctx, userID, "statistics", "", "最近7天")

		require.NoError(t, err)
		assert.Contains(t, response, "活动统计")
	})

	t.Run("未知查询类型", func(t *testing.T) {
		response, err := service.QueryActivities(ctx, userID, "unknown_type", "read_book", "最近7天")

		require.NoError(t, err)
		assert.Contains(t, response, "不理解")
	})
}

// TestDailyActivityService_GetActivitiesByDateRange 测试按日期范围获取活动
func TestDailyActivityService_GetActivitiesByDateRange(t *testing.T) {
	repo := newMockDailyActivityRepository()
	service := NewDailyActivityService(repo)

	ctx := context.Background()
	userID := uint(1)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// 创建活动（模拟不同时间）
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeDrinkWater, map[string]interface{}{"amount": "1杯"}, models.SourceConversation)
	_, _ = service.RecordActivity(ctx, userID, models.ActivityTypeExercise, map[string]interface{}{"type": "跑步"}, models.SourceConversation)

	t.Run("获取今天的活动", func(t *testing.T) {
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime := startTime.Add(24 * time.Hour)

		activities, err := service.GetActivitiesByDateRange(ctx, userID, startTime, endTime)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 1)
	})

	t.Run("获取昨天到今天的活动", func(t *testing.T) {
		startTime := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		endTime := startTime.Add(48 * time.Hour)

		activities, err := service.GetActivitiesByDateRange(ctx, userID, startTime, endTime)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 2)
	})
}

// TestActivityDetails_GetDetails 测试活动详情解析
func TestActivityDetails_GetDetails(t *testing.T) {
	t.Run("解析喝水详情", func(t *testing.T) {
		activity := &models.DailyActivity{
			Details: `{"amount": "200ml", "water_type": "温水"}`,
		}

		details, err := activity.GetDetails()

		require.NoError(t, err)
		assert.Equal(t, "200ml", details.Amount)
		assert.Equal(t, "温水", details.WaterType)
	})

	t.Run("解析运动详情", func(t *testing.T) {
		activity := &models.DailyActivity{
			Details: `{"exercise_type": "跑步", "exercise_duration": "30分钟", "distance": "5公里"}`,
		}

		details, err := activity.GetDetails()

		require.NoError(t, err)
		assert.Equal(t, "跑步", details.ExerciseType)
		assert.Equal(t, "30分钟", details.ExerciseDuration)
		assert.Equal(t, "5公里", details.Distance)
	})

	t.Run("解析空详情", func(t *testing.T) {
		activity := &models.DailyActivity{
			Details: "",
		}

		details, err := activity.GetDetails()

		require.NoError(t, err)
		assert.NotNil(t, details)
	})

	t.Run("解析无效JSON", func(t *testing.T) {
		activity := &models.DailyActivity{
			Details: "invalid json",
		}

		_, err := activity.GetDetails()
		assert.Error(t, err)
	})
}

// TestActivityDetails_SetDetails 测试设置活动详情
func TestActivityDetails_SetDetails(t *testing.T) {
	t.Run("设置喝水详情", func(t *testing.T) {
		activity := &models.DailyActivity{}

		details := &models.ActivityDetails{
			Amount:    "300ml",
			WaterType: "凉水",
		}

		err := activity.SetDetails(details)

		require.NoError(t, err)
		assert.Contains(t, activity.Details, "300ml")
		assert.Contains(t, activity.Details, "凉水")
	})
}

// TestGetActivityTypeDisplayName 测试活动类型显示名称
func TestGetActivityTypeDisplayName(t *testing.T) {
	tests := []struct {
		activityType models.ActivityType
		expected     string
	}{
		{models.ActivityTypeDrinkWater, "喝水"},
		{models.ActivityTypeTakeMedicine, "吃药"},
		{models.ActivityTypeReadBook, "看书"},
		{models.ActivityTypeExercise, "运动"},
		{models.ActivityTypeSleep, "睡眠"},
		{models.ActivityTypeEat, "吃饭"},
		{models.ActivityTypeCustom, "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.activityType), func(t *testing.T) {
			result := getActivityTypeDisplayName(tt.activityType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
