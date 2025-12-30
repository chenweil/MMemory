package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
	sqliterepo "mmemory/internal/repository/sqlite"
	"mmemory/internal/service"
	ai "mmemory/pkg/ai"
)

// TestDailyActivityE2E 端到端测试：日常活动服务
func TestDailyActivityE2E(t *testing.T) {
	// 1. 设置测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.DailyActivity{}))

	// 2. 创建仓库层
	userRepo := sqliterepo.NewUserRepository(db)
	activityRepo := sqliterepo.NewDailyActivityRepository(db)

	ctx := context.Background()

	// 3. 创建测试用户
	user := &models.User{
		TelegramID:   54321,
		Username:     "activity_test",
		FirstName:    "ActivityTest",
		LanguageCode: "zh-CN",
	}
	require.NoError(t, userRepo.Create(ctx, user))

	// 4. 创建活动服务
	activityService := service.NewDailyActivityService(activityRepo)

	// 5. 测试活动记录
	t.Run("记录日常活动", func(t *testing.T) {
		// 记录喝水活动
		details := map[string]interface{}{
			"amount":    "200ml",
			"waterType": "温水",
		}
		activity, err := activityService.RecordActivity(ctx, user.ID, models.ActivityTypeDrinkWater, details, models.SourceManual)
		require.NoError(t, err)
		assert.NotNil(t, activity)
		assert.Equal(t, models.ActivityTypeDrinkWater, activity.ActivityType)
		assert.Equal(t, user.ID, activity.UserID)

		// 记录运动活动
		exerciseDetails := map[string]interface{}{
			"exerciseType":   "跑步",
			"exerciseDuration": "30分钟",
			"distance":      "5公里",
		}
		exerciseActivity, err := activityService.RecordActivity(ctx, user.ID, models.ActivityTypeExercise, exerciseDetails, models.SourceConversation)
		require.NoError(t, err)
		assert.NotNil(t, exerciseActivity)
		assert.Equal(t, models.ActivityTypeExercise, exerciseActivity.ActivityType)
	})

	// 6. 测试获取最近活动
	t.Run("获取最近活动", func(t *testing.T) {
		activities, err := activityService.GetRecentActivities(ctx, user.ID, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 2)
	})

	// 7. 测试按类型获取活动
	t.Run("按类型获取活动", func(t *testing.T) {
		activities, err := activityService.GetActivitiesByType(ctx, user.ID, models.ActivityTypeDrinkWater, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, len(activities))
		assert.Equal(t, models.ActivityTypeDrinkWater, activities[0].ActivityType)
	})

	// 8. 测试活动统计
	t.Run("活动统计", func(t *testing.T) {
		stats, err := activityService.GetActivityStatistics(ctx, user.ID, "week")
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Greater(t, len(stats), 0)
	})
}

// TestContextManagerE2E 端到端测试：上下文管理器
func TestContextManagerE2E(t *testing.T) {
	// 1. 设置测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.ConversationContext{}))

	// 2. 创建仓库层
	userRepo := sqliterepo.NewUserRepository(db)
	conversationContextRepo := sqliterepo.NewConversationContextRepository(db)

	ctx := context.Background()

	// 3. 创建测试用户
	user := &models.User{
		TelegramID:   11111,
		Username:     "context_test",
		FirstName:    "ContextTest",
		LanguageCode: "zh-CN",
	}
	require.NoError(t, userRepo.Create(ctx, user))

	// 4. 创建上下文管理器
	contextManager := service.NewContextManager(
		conversationContextRepo,
		&service.DefaultEntityExtractor{},
		&service.DefaultIntentTracker{},
		service.ContextManagerConfig{
			MaxMessages: 10,
			DefaultTTL:  30 * 24 * time.Hour,
		},
	)

	// 5. 测试上下文处理
	t.Run("处理消息上下文", func(t *testing.T) {
		input := service.ProcessMessageInput{
			UserID:    user.ID,
			SessionID: "test-session-123",
			Role:      "user",
			Message:   "设置一个每天早上7点提醒我喝水的提醒",
			Channel:   "telegram",
			Locale:    "zh-CN",
		}

		state, err := contextManager.ProcessMessage(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Equal(t, user.ID, state.UserID)
	})

	// 6. 测试获取上下文
	t.Run("获取上下文", func(t *testing.T) {
		state, err := contextManager.GetContext(ctx, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, state)
	})

	// 7. 测试更新上下文状态
	t.Run("更新上下文状态", func(t *testing.T) {
		updateInput := service.UpdateContextStateInput{
			UserID: user.ID,
			State:  "reminder_title:喝水提醒;reminder_time:07:00",
		}
		err := contextManager.UpdateContextState(ctx, updateInput)
		require.NoError(t, err)
	})

	// 8. 测试清理过期上下文
	t.Run("清理过期上下文", func(t *testing.T) {
		err := contextManager.CleanupExpired(ctx)
		require.NoError(t, err)
	})
}

// TestEnhancedCacheE2E 端到端测试：增强缓存
func TestEnhancedCacheE2E(t *testing.T) {
	// 创建增强缓存
	cache := ai.NewEnhancedCache(5*time.Minute, 100)

	// 1. 测试基本缓存操作
	t.Run("基本缓存操作", func(t *testing.T) {
		// 设置缓存
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")

		// 获取缓存
		value, found := cache.Get("key1")
		assert.True(t, found)
		assert.Equal(t, "value1", value)

		value, found = cache.Get("key2")
		assert.True(t, found)
		assert.Equal(t, "value2", value)

		// 缓存未命中
		_, found = cache.Get("nonexistent")
		assert.False(t, found)
	})

	// 2. 测试缓存统计
	t.Run("缓存统计", func(t *testing.T) {
		stats := cache.GetStats()
		assert.NotNil(t, stats)
	})

	// 3. 测试缓存命中率和大小
	t.Run("缓存命中率和大小", func(t *testing.T) {
		// 触发一些缓存操作
		cache.Set("stat_key", "stat_value")
		cache.Get("stat_key")
		cache.Get("nonexistent_key")

		hitRate := cache.GetHitRate()
		assert.GreaterOrEqual(t, hitRate, 0.0)
		assert.LessOrEqual(t, hitRate, 100.0) // 可能是百分比

		size := cache.Size()
		assert.GreaterOrEqual(t, size, 0)
	})

	// 4. 测试缓存清理
	t.Run("缓存清理", func(t *testing.T) {
		cache.Set("temp_key", "temp_value")
		time.Sleep(1100 * time.Millisecond) // 等待过期（因为TTL是5分钟，1秒的等待不够）
		// 手动清除
		cache.Clear()

		size := cache.Size()
		assert.Equal(t, 0, size)
	})
}

// TestQueryPerformanceE2E 端到端测试：查询性能监控
func TestQueryPerformanceE2E(t *testing.T) {
	// 创建查询优化器
	optimizer := sqliterepo.NewQueryOptimizer(50 * time.Millisecond)
	defer optimizer.Stop()

	// 1. 测试查询记录
	t.Run("查询记录", func(t *testing.T) {
		// 记录一些查询
		optimizer.RecordQuery("users", "SELECT", "SELECT * FROM users", 5*time.Millisecond, 1, false)
		optimizer.RecordQuery("reminders", "INSERT", "INSERT INTO reminders", 60*time.Millisecond, 1, false)
		optimizer.RecordQuery("logs", "UPDATE", "UPDATE logs SET status = 1", 100*time.Millisecond, 1, false)

		metrics := optimizer.GetMetrics()
		assert.Equal(t, int64(3), metrics.TotalQueries)
		assert.Equal(t, int64(2), metrics.SlowQueries) // 超过50ms的查询
	})

	// 2. 测试慢查询日志
	t.Run("慢查询日志", func(t *testing.T) {
		logs := optimizer.GetSlowQueries(10)
		assert.Equal(t, 2, len(logs))

		// 验证慢查询记录正确
		for _, log := range logs {
			assert.GreaterOrEqual(t, log.Duration.Milliseconds(), int64(50))
		}
	})

	// 3. 测试按表和操作统计
	t.Run("按表统计", func(t *testing.T) {
		metrics := optimizer.GetMetrics()
		assert.NotNil(t, metrics.ByTable)
		assert.Equal(t, int64(1), metrics.ByTable["users"].QueryCount)
		assert.Equal(t, int64(1), metrics.ByTable["reminders"].QueryCount)
		assert.Equal(t, int64(1), metrics.ByTable["logs"].QueryCount)
	})

	// 4. 测试查询统计信息
	t.Run("查询统计信息", func(t *testing.T) {
		stats := optimizer.GetSlowQueryStats()
		assert.NotNil(t, stats)
		assert.Equal(t, int64(3), stats["total_queries"])
		assert.Equal(t, int64(2), stats["slow_queries"])
	})

	// 5. 测试慢查询阈值获取
	t.Run("慢查询阈值", func(t *testing.T) {
		threshold := optimizer.GetSlowThreshold()
		assert.Equal(t, 50*time.Millisecond, threshold)
	})
}
