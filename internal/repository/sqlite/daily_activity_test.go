package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
)

// setupTestDBForDailyActivity 创建测试数据库
func setupTestDBForDailyActivity(t *testing.T) (*gorm.DB, *QueryOptimizer) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.User{}, &models.DailyActivity{})
	require.NoError(t, err)

	// 创建查询优化器
	queryOptimizer := NewQueryOptimizer(10 * time.Millisecond)

	return db, queryOptimizer
}

// createTestUserForDailyActivity 创建测试用户
func createTestUserForDailyActivity(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		TelegramID:   123456789,
		Username:     "activity_test_user",
		FirstName:    "ActivityTest",
		LanguageCode: "zh-CN",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// TestDailyActivityRepository_Create 测试创建活动记录
func TestDailyActivityRepository_Create(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	t.Run("创建喝水活动成功", func(t *testing.T) {
		activity := &models.DailyActivity{
			UserID:       user.ID,
			ActivityType: models.ActivityTypeDrinkWater,
			OccurredAt:   time.Now(),
			Details:      `{"amount": "200ml", "waterType": "温水"}`,
			Source:       models.SourceConversation,
		}

		err := repo.Create(ctx, activity)

		require.NoError(t, err)
		assert.NotZero(t, activity.ID)
		assert.NotZero(t, activity.CreatedAt)
	})

	t.Run("创建运动活动成功", func(t *testing.T) {
		activity := &models.DailyActivity{
			UserID:       user.ID,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:   time.Now(),
			Details:      `{"exercise_type": "跑步", "exercise_duration": "30分钟", "distance": "5公里"}`,
			Source:       models.SourceReminder,
		}

		err := repo.Create(ctx, activity)

		require.NoError(t, err)
		assert.NotZero(t, activity.ID)
	})

	t.Run("创建多条活动记录", func(t *testing.T) {
		// 先清理之前的活动
		db.Exec("DELETE FROM daily_activities WHERE user_id = ?", user.ID)

		for i := 0; i < 3; i++ {
			activity := &models.DailyActivity{
				UserID:       user.ID,
				ActivityType: models.ActivityTypeReadBook,
				OccurredAt:   time.Now(),
				Details:      `{"book_name": "书"}`,
				Source:       models.SourceManual,
			}
			err := repo.Create(ctx, activity)
			require.NoError(t, err)
		}

		activities, err := repo.GetByUserID(ctx, user.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, activities, 3)
	})
}

// TestDailyActivityRepository_GetByID 测试按ID获取活动
func TestDailyActivityRepository_GetByID(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	// 先创建一个活动
	activity := &models.DailyActivity{
		UserID:       user.ID,
		ActivityType: models.ActivityTypeDrinkWater,
		OccurredAt:   time.Now(),
		Details:      `{"amount": "1杯"}`,
		Source:       models.SourceConversation,
	}
	err := repo.Create(ctx, activity)
	require.NoError(t, err)

	t.Run("获取存在的活动", func(t *testing.T) {
		result, err := repo.GetByID(ctx, activity.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, activity.ID, result.ID)
		assert.Equal(t, models.ActivityTypeDrinkWater, result.ActivityType)
	})

	t.Run("获取不存在的活动", func(t *testing.T) {
		result, err := repo.GetByID(ctx, 99999)

		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

// TestDailyActivityRepository_GetByUserID 测试按用户ID获取活动
func TestDailyActivityRepository_GetByUserID(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	// 创建多个活动
	for i := 0; i < 5; i++ {
		activity := &models.DailyActivity{
			UserID:       user.ID,
			ActivityType: models.ActivityTypeDrinkWater,
			OccurredAt:   time.Now(),
			Details:      `{"amount": "1杯"}`,
			Source:       models.SourceConversation,
		}
		err := repo.Create(ctx, activity)
		require.NoError(t, err)
	}

	t.Run("获取用户所有活动", func(t *testing.T) {
		activities, err := repo.GetByUserID(ctx, user.ID, 10, 0)

		require.NoError(t, err)
		assert.Len(t, activities, 5)
	})

	t.Run("限制返回数量", func(t *testing.T) {
		activities, err := repo.GetByUserID(ctx, user.ID, 3, 0)

		require.NoError(t, err)
		assert.Len(t, activities, 3)
	})

	t.Run("测试分页偏移", func(t *testing.T) {
		// 先获取前3个
		page1, _ := repo.GetByUserID(ctx, user.ID, 3, 0)
		// 再获取后2个
		page2, _ := repo.GetByUserID(ctx, user.ID, 10, 3)

		assert.Len(t, page1, 3)
		assert.Len(t, page2, 2)

		// 确保不重复
		page1IDs := make(map[uint]bool)
		for _, a := range page1 {
			page1IDs[a.ID] = true
		}
		for _, a := range page2 {
			assert.False(t, page1IDs[a.ID])
		}
	})
}

// TestDailyActivityRepository_GetByType 测试按类型获取活动
func TestDailyActivityRepository_GetByType(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	// 创建不同类型的活动
	activities := []*models.DailyActivity{
		{UserID: user.ID, ActivityType: models.ActivityTypeDrinkWater, OccurredAt: time.Now(), Source: models.SourceConversation},
		{UserID: user.ID, ActivityType: models.ActivityTypeDrinkWater, OccurredAt: time.Now(), Source: models.SourceConversation},
		{UserID: user.ID, ActivityType: models.ActivityTypeReadBook, OccurredAt: time.Now(), Source: models.SourceConversation},
		{UserID: user.ID, ActivityType: models.ActivityTypeExercise, OccurredAt: time.Now(), Source: models.SourceConversation},
	}
	for _, a := range activities {
		err := repo.Create(ctx, a)
		require.NoError(t, err)
	}

	t.Run("获取喝水活动", func(t *testing.T) {
		result, err := repo.GetByType(ctx, user.ID, models.ActivityTypeDrinkWater, 10, 0)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		for _, a := range result {
			assert.Equal(t, models.ActivityTypeDrinkWater, a.ActivityType)
		}
	})

	t.Run("获取看书活动", func(t *testing.T) {
		result, err := repo.GetByType(ctx, user.ID, models.ActivityTypeReadBook, 10, 0)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, models.ActivityTypeReadBook, result[0].ActivityType)
	})
}

// TestDailyActivityRepository_GetByDateRange 测试按日期范围获取活动
func TestDailyActivityRepository_GetByDateRange(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	now := time.Now()

	// 创建今天的活动
	todayActivity := &models.DailyActivity{
		UserID:       user.ID,
		ActivityType: models.ActivityTypeDrinkWater,
		OccurredAt:   now,
		Source:       models.SourceConversation,
	}
	err := repo.Create(ctx, todayActivity)
	require.NoError(t, err)

	// 创建昨天的活动
	yesterday := now.AddDate(0, 0, -1)
	yesterdayActivity := &models.DailyActivity{
		UserID:       user.ID,
		ActivityType: models.ActivityTypeExercise,
		OccurredAt:   yesterday,
		Source:       models.SourceReminder,
	}
	err = repo.Create(ctx, yesterdayActivity)
	require.NoError(t, err)

	t.Run("获取今天的活动", func(t *testing.T) {
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime := startTime.Add(24 * time.Hour)

		result, err := repo.GetByDateRange(ctx, user.ID, startTime, endTime)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 1)
		for _, a := range result {
			assert.True(t, a.OccurredAt.After(startTime) || a.OccurredAt.Equal(startTime))
			assert.True(t, a.OccurredAt.Before(endTime) || a.OccurredAt.Equal(endTime))
		}
	})

	t.Run("获取昨天到今天的活动", func(t *testing.T) {
		startTime := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		endTime := startTime.Add(48 * time.Hour)

		result, err := repo.GetByDateRange(ctx, user.ID, startTime, endTime)

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

// TestDailyActivityRepository_GetRecentActivities 测试获取最近活动
func TestDailyActivityRepository_GetRecentActivities(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	// 创建多个活动
	for i := 0; i < 5; i++ {
		activity := &models.DailyActivity{
			UserID:       user.ID,
			ActivityType: models.ActivityTypeDrinkWater,
			OccurredAt:   time.Now(),
			Source:       models.SourceConversation,
		}
		err := repo.Create(ctx, activity)
		require.NoError(t, err)
	}

	t.Run("获取最近3条", func(t *testing.T) {
		result, err := repo.GetRecentActivities(ctx, user.ID, 3)

		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("限制为较大值返回所有", func(t *testing.T) {
		// GORM中limit=0会返回0行，使用较大的limit值
		result, err := repo.GetRecentActivities(ctx, user.ID, 100)

		require.NoError(t, err)
		assert.Len(t, result, 5)
	})
}

// TestDailyActivityRepository_Update 测试更新活动
func TestDailyActivityRepository_Update(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	// 创建活动
	activity := &models.DailyActivity{
		UserID:       user.ID,
		ActivityType: models.ActivityTypeDrinkWater,
		OccurredAt:   time.Now(),
		Details:      `{"amount": "1杯"}`,
		Source:       models.SourceConversation,
	}
	err := repo.Create(ctx, activity)
	require.NoError(t, err)

	t.Run("更新活动详情", func(t *testing.T) {
		activity.Details = `{"amount": "2杯", "waterType": "温水"}`

		err := repo.Update(ctx, activity)

		require.NoError(t, err)

		// 验证更新
		result, _ := repo.GetByID(ctx, activity.ID)
		assert.Contains(t, result.Details, "2杯")
	})
}

// TestDailyActivityRepository_Delete 测试删除活动
func TestDailyActivityRepository_Delete(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	// 创建活动
	activity := &models.DailyActivity{
		UserID:       user.ID,
		ActivityType: models.ActivityTypeDrinkWater,
		OccurredAt:   time.Now(),
		Source:       models.SourceConversation,
	}
	err := repo.Create(ctx, activity)
	require.NoError(t, err)

	t.Run("删除活动", func(t *testing.T) {
		err := repo.Delete(ctx, activity.ID)

		require.NoError(t, err)

		// 验证删除
		result, _ := repo.GetByID(ctx, activity.ID)
		assert.Nil(t, result)
	})
}

// TestDailyActivityRepository_GetStatistics 测试获取统计信息
func TestDailyActivityRepository_GetStatistics(t *testing.T) {
	db, queryOptimizer := setupTestDBForDailyActivity(t)
	defer queryOptimizer.Stop()

	repo := NewDailyActivityRepository(db)
	ctx := context.Background()

	user := createTestUserForDailyActivity(t, db)

	now := time.Now()

	// 创建不同类型的活动
	activities := []*models.DailyActivity{
		{UserID: user.ID, ActivityType: models.ActivityTypeDrinkWater, OccurredAt: now, Source: models.SourceConversation},
		{UserID: user.ID, ActivityType: models.ActivityTypeDrinkWater, OccurredAt: now, Source: models.SourceConversation},
		{UserID: user.ID, ActivityType: models.ActivityTypeExercise, OccurredAt: now, Source: models.SourceConversation},
		{UserID: user.ID, ActivityType: models.ActivityTypeReadBook, OccurredAt: now, Source: models.SourceConversation},
	}
	for _, a := range activities {
		err := repo.Create(ctx, a)
		require.NoError(t, err)
	}

	t.Run("获取时间范围内统计", func(t *testing.T) {
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime := startTime.Add(24 * time.Hour)

		stats, err := repo.GetStatistics(ctx, user.ID, startTime, endTime)

		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats["drink_water"], int64(2))
		assert.GreaterOrEqual(t, stats["exercise"], int64(1))
		assert.GreaterOrEqual(t, stats["read_book"], int64(1))
	})
}
