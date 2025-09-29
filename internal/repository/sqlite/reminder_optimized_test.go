package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
)

// TestOptimizedReminderRepository 测试优化的提醒仓储
func TestOptimizedReminderRepository(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.User{}, &models.Reminder{}, &models.ReminderLog{})
	require.NoError(t, err)

	// 创建优化的仓储
	repo := NewOptimizedReminderRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err = db.Create(user).Error
	require.NoError(t, err)

	t.Run("创建提醒 - 基础功能", func(t *testing.T) {
		reminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "测试提醒",
			Description:     "这是一个测试提醒",
			Type:            models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:      "14:30:00",
			IsActive:        true,
		}

		err := repo.Create(ctx, reminder)
		require.NoError(t, err)
		assert.NotZero(t, reminder.ID)
		assert.Equal(t, "Asia/Shanghai", reminder.Timezone) // 验证默认值
		assert.Equal(t, models.ReminderTypeTask, reminder.Type)
	})

	t.Run("创建提醒 - 验证必填字段", func(t *testing.T) {
		reminder := &models.Reminder{
			// 缺少必填字段
			Description: "缺少必填字段",
		}

		err := repo.Create(ctx, reminder)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "用户ID不能为空")
	})

	t.Run("根据ID获取提醒 - 包含关联数据", func(t *testing.T) {
		// 先创建提醒
		reminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "获取测试提醒",
			Description:     "测试获取功能",
			Type:            models.ReminderTypeHabit,
			SchedulePattern: "weekly:1,3,5",
			TargetTime:      "09:00:00",
			IsActive:        true,
		}
		err := repo.Create(ctx, reminder)
		require.NoError(t, err)

		// 创建相关的提醒记录
		log := &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: time.Now(),
			Status:        models.ReminderStatusPending,
		}
		err = db.Create(log).Error
		require.NoError(t, err)

		// 获取提醒（应该包含关联数据）
		retrievedReminder, err := repo.GetByID(ctx, reminder.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedReminder)
		assert.Equal(t, reminder.Title, retrievedReminder.Title)
		assert.NotNil(t, retrievedReminder.User) // 应该预加载用户信息
		assert.NotNil(t, retrievedReminder.ReminderLogs) // 应该预加载提醒记录
	})

	t.Run("根据用户ID获取提醒", func(t *testing.T) {
		// 创建多个提醒
		for i := 0; i < 3; i++ {
			reminder := &models.Reminder{
				UserID:          user.ID,
				Title:           fmt.Sprintf("用户提醒 %d", i),
				Description:     fmt.Sprintf("描述 %d", i),
				Type:            models.ReminderTypeTask,
				SchedulePattern: "daily",
				TargetTime:      "10:00:00",
				IsActive:        true,
			}
			err := repo.Create(ctx, reminder)
			require.NoError(t, err)
		}

		// 获取用户的提醒
		reminders, err := repo.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(reminders), 3)

		// 验证所有提醒都属于该用户且是活跃的
		for _, reminder := range reminders {
			assert.Equal(t, user.ID, reminder.UserID)
			assert.True(t, reminder.IsActive)
		}
	})

	t.Run("获取活跃提醒", func(t *testing.T) {
		// 创建活跃和非活跃提醒
		activeReminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "活跃提醒",
			Description:     "这是活跃的",
			Type:            models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:      "11:00:00",
			IsActive:        true,
		}
		err := repo.Create(ctx, activeReminder)
		require.NoError(t, err)

		inactiveReminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "非活跃提醒",
			Description:     "这是非活跃的",
			Type:            models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:      "12:00:00",
			IsActive:        false,
		}
		err = repo.Create(ctx, inactiveReminder)
		require.NoError(t, err)

		// 获取活跃提醒
		activeReminders, err := repo.GetActiveReminders(ctx)
		require.NoError(t, err)

		// 验证只返回活跃提醒
		foundActive := false
		for _, reminder := range activeReminders {
			assert.True(t, reminder.IsActive)
			if reminder.ID == activeReminder.ID {
				foundActive = true
			}
		}
		assert.True(t, foundActive, "应该找到活跃的提醒")
	})

	t.Run("更新提醒", func(t *testing.T) {
		reminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "原始标题",
			Description:     "原始描述",
			Type:            models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:      "13:00:00",
			IsActive:        true,
		}
		err := repo.Create(ctx, reminder)
		require.NoError(t, err)

		// 更新提醒
		reminder.Title = "更新后的标题"
		reminder.Description = "更新后的描述"
		reminder.IsActive = false

		err = repo.Update(ctx, reminder)
		require.NoError(t, err)

		// 验证更新
		updatedReminder, err := repo.GetByID(ctx, reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, "更新后的标题", updatedReminder.Title)
		assert.Equal(t, "更新后的描述", updatedReminder.Description)
		assert.False(t, updatedReminder.IsActive)
	})

	t.Run("删除提醒 - 级联删除", func(t *testing.T) {
		reminder := &models.Reminder{
			UserID:          user.ID,
			Title:           "待删除提醒",
			Description:     "这个提醒将被删除",
			Type:            models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:      "14:00:00",
			IsActive:        true,
		}
		err := repo.Create(ctx, reminder)
		require.NoError(t, err)

		// 创建相关的提醒记录
		log := &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: time.Now(),
			Status:        models.ReminderStatusPending,
		}
		err = db.Create(log).Error
		require.NoError(t, err)

		// 删除提醒
		err = repo.Delete(ctx, reminder.ID)
		require.NoError(t, err)

		// 验证提醒已被删除
		deletedReminder, err := repo.GetByID(ctx, reminder.ID)
		assert.NoError(t, err) // 查询本身不应该出错
		assert.Nil(t, deletedReminder) // 但应该返回nil
	})

	t.Run("验证时间格式", func(t *testing.T) {
		validTimes := []string{"00:00:00", "12:30:45", "23:59:59"}
		invalidTimes := []string{"24:00:00", "12:60:00", "12:30:60", "123:00:00", "12:300:00"}

		for _, validTime := range validTimes {
			reminder := &models.Reminder{
				UserID:          user.ID,
				Title:           "有效时间测试",
				Description:     fmt.Sprintf("时间: %s", validTime),
				Type:            models.ReminderTypeTask,
				SchedulePattern: "daily",
				TargetTime:      validTime,
				IsActive:        true,
			}
			err := repo.Create(ctx, reminder)
			assert.NoError(t, err, "时间 %s 应该是有效的", validTime)
		}

		for _, invalidTime := range invalidTimes {
			reminder := &models.Reminder{
				UserID:          user.ID,
				Title:           "无效时间测试",
				Description:     fmt.Sprintf("时间: %s", invalidTime),
				Type:            models.ReminderTypeTask,
				SchedulePattern: "daily",
				TargetTime:      invalidTime,
				IsActive:        true,
			}
			err := repo.Create(ctx, reminder)
			assert.Error(t, err, "时间 %s 应该是无效的", invalidTime)
		}
	})
}