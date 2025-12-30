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

// setupTestDBForReminder 创建测试数据库
func setupTestDBForReminder(t *testing.T) (*gorm.DB, *QueryOptimizer) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.User{}, &models.Reminder{})
	require.NoError(t, err)

	// 创建查询优化器
	queryOptimizer := NewQueryOptimizer(10 * time.Millisecond)

	return db, queryOptimizer
}

// createTestUserForReminder 创建测试用户
func createTestUserForReminder(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// TestReminderRepository_Create 测试创建提醒
func TestReminderRepository_Create(t *testing.T) {
	db, queryOptimizer := setupTestDBForReminder(t)
	defer queryOptimizer.Stop()
	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	user := createTestUserForReminder(t, db)

	t.Run("创建成功", func(t *testing.T) {
		reminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "测试提醒",
			Description:     "测试描述",
			Type:            models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:      "14:30:00",
			IsActive:        true,
		}

		err := repo.Create(ctx, reminder)
		assert.NoError(t, err)
		assert.NotZero(t, reminder.ID)
	})

	t.Run("创建多个提醒", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			reminder := &models.Reminder{
				UserID:          user.ID,
				Title:           "测试提醒",
				Description:     "测试描述",
				Type:            models.ReminderTypeTask,
				SchedulePattern: "daily",
				TargetTime:      "14:30:00",
				IsActive:        true,
			}
			err := repo.Create(ctx, reminder)
			assert.NoError(t, err)
		}
	})
}

// TestReminderRepository_GetByID 测试根据ID获取提醒
func TestReminderRepository_GetByID(t *testing.T) {
	db, queryOptimizer := setupTestDBForReminder(t)
	defer queryOptimizer.Stop()
	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	user := createTestUserForReminder(t, db)

	// 创建测试提醒
	reminder := &models.Reminder{
		UserID:          user.ID,
		Title:           "测试提醒",
		Description:     "测试描述",
		Type:            models.ReminderTypeTask,
		SchedulePattern: "daily",
		TargetTime:      "14:30:00",
		IsActive:        true,
	}
	err := repo.Create(ctx, reminder)
	require.NoError(t, err)

	t.Run("获取存在的提醒", func(t *testing.T) {
		retrievedReminder, err := repo.GetByID(ctx, reminder.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedReminder)
		assert.Equal(t, reminder.ID, retrievedReminder.ID)
		assert.Equal(t, reminder.Title, retrievedReminder.Title)
		assert.NotNil(t, retrievedReminder.User) // 验证预加载
	})

	t.Run("获取不存在的提醒", func(t *testing.T) {
		retrievedReminder, err := repo.GetByID(ctx, 9999)
		assert.NoError(t, err)
		assert.Nil(t, retrievedReminder)
	})
}

// TestReminderRepository_GetByUserID 测试根据用户ID获取提醒
func TestReminderRepository_GetByUserID(t *testing.T) {
	db, queryOptimizer := setupTestDBForReminder(t)
	defer queryOptimizer.Stop()
	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	user := createTestUserForReminder(t, db)

	// 创建多个提醒
	for i := 0; i < 5; i++ {
		reminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "测试提醒",
			Description:     "测试描述",
			Type:            models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:      "14:30:00",
			IsActive:        true,
		}
		err := repo.Create(ctx, reminder)
		require.NoError(t, err)
	}

	t.Run("获取用户所有提醒", func(t *testing.T) {
		reminders, err := repo.GetByUserID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Len(t, reminders, 5)
	})

	t.Run("获取其他用户的提醒", func(t *testing.T) {
		reminders, err := repo.GetByUserID(ctx, 9999)
		assert.NoError(t, err)
		assert.Len(t, reminders, 0)
	})
}

// TestReminderRepository_GetActiveReminders 测试获取活跃提醒
func TestReminderRepository_GetActiveReminders(t *testing.T) {
	db, queryOptimizer := setupTestDBForReminder(t)
	defer queryOptimizer.Stop()
	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	user := createTestUserForReminder(t, db)

	// 创建活跃提醒
	activeReminder := &models.Reminder{
		UserID:          user.ID,
		Title:           "活跃提醒",
		Description:     "这是活跃的",
		Type:            models.ReminderTypeTask,
		SchedulePattern: "daily",
		TargetTime:      "14:30:00",
		IsActive:        true,
	}
	err := repo.Create(ctx, activeReminder)
	require.NoError(t, err)

	// 创建非活跃提醒
	inactiveReminder := &models.Reminder{
		UserID:          user.ID,
		Title:           "非活跃提醒",
		Description:     "这是非活跃的",
		Type:            models.ReminderTypeTask,
		SchedulePattern: "daily",
		TargetTime:      "15:30:00",
		IsActive:        false,
	}
	err = repo.Create(ctx, inactiveReminder)
	require.NoError(t, err)

	t.Run("获取活跃提醒", func(t *testing.T) {
		activeReminders, err := repo.GetActiveReminders(ctx)
		assert.NoError(t, err)

		// 验证只返回活跃提醒
		for _, reminder := range activeReminders {
			assert.True(t, reminder.IsActive)
		}
	})
}

// TestReminderRepository_Update 测试更新提醒
func TestReminderRepository_Update(t *testing.T) {
	db, queryOptimizer := setupTestDBForReminder(t)
	defer queryOptimizer.Stop()
	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	user := createTestUserForReminder(t, db)

	reminder := &models.Reminder{
		UserID:          user.ID,
		Title:           "原始标题",
		Description:     "原始描述",
		Type:            models.ReminderTypeTask,
		SchedulePattern: "daily",
		TargetTime:      "14:30:00",
		IsActive:        true,
	}
	err := repo.Create(ctx, reminder)
	require.NoError(t, err)

	t.Run("更新提醒", func(t *testing.T) {
		reminder.Title = "更新后的标题"
		reminder.Description = "更新后的描述"
		reminder.IsActive = false

		err := repo.Update(ctx, reminder)
		assert.NoError(t, err)

		// 验证更新
		updatedReminder, err := repo.GetByID(ctx, reminder.ID)
		assert.NoError(t, err)
		assert.Equal(t, "更新后的标题", updatedReminder.Title)
		assert.Equal(t, "更新后的描述", updatedReminder.Description)
		assert.False(t, updatedReminder.IsActive)
	})
}

// TestReminderRepository_Delete 测试删除提醒
func TestReminderRepository_Delete(t *testing.T) {
	db, queryOptimizer := setupTestDBForReminder(t)
	defer queryOptimizer.Stop()
	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	user := createTestUserForReminder(t, db)

	reminder := &models.Reminder{
		UserID:          user.ID,
		Title:           "待删除提醒",
		Description:     "这个提醒将被删除",
		Type:            models.ReminderTypeTask,
		SchedulePattern: "daily",
		TargetTime:      "14:30:00",
		IsActive:        true,
	}
	err := repo.Create(ctx, reminder)
	require.NoError(t, err)

	t.Run("删除存在的提醒", func(t *testing.T) {
		err := repo.Delete(ctx, reminder.ID)
		assert.NoError(t, err)

		// 验证已删除
		deletedReminder, err := repo.GetByID(ctx, reminder.ID)
		assert.NoError(t, err)
		assert.Nil(t, deletedReminder)
	})

	t.Run("删除不存在的提醒", func(t *testing.T) {
		// GORM的Delete方法不返回错误当记录不存在时
		err := repo.Delete(ctx, 9999)
		assert.NoError(t, err)
	})
}

// TestReminderRepository_CountByStatus 测试根据状态统计提醒数量
func TestReminderRepository_CountByStatus(t *testing.T) {
	t.Run("统计活跃提醒", func(t *testing.T) {
		db, queryOptimizer := setupTestDBForReminder(t)
		defer queryOptimizer.Stop()
		repo := NewReminderRepository(db, queryOptimizer)
		ctx := context.Background()
		user := createTestUserForReminder(t, db)

		// 创建3个活跃提醒
		for i := 0; i < 3; i++ {
			reminder := &models.Reminder{
				UserID:          user.ID,
				Title:           "活跃提醒",
				Description:     "这是活跃的",
				Type:            models.ReminderTypeTask,
				SchedulePattern: "daily",
				TargetTime:      "14:30:00",
				IsActive:        true,
			}
			err := repo.Create(ctx, reminder)
			require.NoError(t, err)
		}

		count, err := repo.CountByStatus(ctx, models.ReminderStatStatusActive)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("统计已完成提醒", func(t *testing.T) {
		db, queryOptimizer := setupTestDBForReminder(t)
		defer queryOptimizer.Stop()
		repo := NewReminderRepository(db, queryOptimizer)
		ctx := context.Background()
		user := createTestUserForReminder(t, db)

		// 创建2个提醒，然后设置为非活跃
		for i := 0; i < 2; i++ {
			reminder := &models.Reminder{
				UserID:          user.ID,
				Title:           "非活跃提醒",
				Description:     "这是非活跃的",
				Type:            models.ReminderTypeTask,
				SchedulePattern: "daily",
				TargetTime:      "15:30:00",
				IsActive:        true, // 先创建为活跃
			}
			err := repo.Create(ctx, reminder)
			require.NoError(t, err)

			// 然后更新为非活跃
			reminder.IsActive = false
			err = repo.Update(ctx, reminder)
			require.NoError(t, err)
		}

		// 验证提醒确实被创建了
		var allCount int64
		err := db.Model(&models.Reminder{}).Count(&allCount).Error
		require.NoError(t, err)
		t.Logf("数据库中总共的提醒数: %d", allCount)

		// 验证非活跃提醒数量
		var inactiveCount int64
		err = db.Model(&models.Reminder{}).Where("is_active = ?", false).Count(&inactiveCount).Error
		require.NoError(t, err)
		t.Logf("数据库中非活跃提醒数: %d", inactiveCount)

		count, err := repo.CountByStatus(ctx, models.ReminderStatStatusCompleted)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("统计过期提醒", func(t *testing.T) {
		db, queryOptimizer := setupTestDBForReminder(t)
		defer queryOptimizer.Stop()
		repo := NewReminderRepository(db, queryOptimizer)
		ctx := context.Background()
		user := createTestUserForReminder(t, db)

		// 创建一次性提醒（过期）
		for i := 0; i < 2; i++ {
			reminder := &models.Reminder{
				UserID:          user.ID,
				Title:           "一次性提醒",
				Description:     "这是过期的",
				Type:            models.ReminderTypeTask,
				SchedulePattern: "once:2024-01-01",
				TargetTime:      "14:30:00",
				IsActive:        true,
			}
			err := repo.Create(ctx, reminder)
			require.NoError(t, err)
		}

		count, err := repo.CountByStatus(ctx, models.ReminderStatStatusExpired)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}