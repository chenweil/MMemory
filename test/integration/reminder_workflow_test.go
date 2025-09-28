package integration

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
)

// TestReminderWorkflow 测试完整的提醒工作流程
func TestReminderWorkflow(t *testing.T) {
	// 设置测试数据库
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 初始化仓储层
	userRepo := sqliterepo.NewUserRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)
	reminderLogRepo := sqliterepo.NewReminderLogRepository(db)

	// 初始化服务层
	userService := service.NewUserService(userRepo)
	reminderService := service.NewReminderService(reminderRepo)
	reminderLogService := service.NewReminderLogService(reminderLogRepo, reminderRepo)

	ctx := context.Background()

	// 创建测试用户
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := userService.CreateUser(ctx, user)
	require.NoError(t, err)

	// 测试用例1: 创建提醒并验证调度
	t.Run("创建提醒并验证即时调度", func(t *testing.T) {
		reminder := &models.Reminder{
			UserID:       user.ID,
			Title:        "测试提醒",
			Description:  "这是一个测试提醒",
			TargetTime:   "14:30",
			SchedulePattern: "daily",
			IsActive:     true,
		}

		// 创建提醒（应该自动注册到调度器）
		err := reminderService.CreateReminder(ctx, reminder)
		require.NoError(t, err)
		assert.NotZero(t, reminder.ID)

		// 验证提醒已创建
		savedReminder, err := reminderRepo.GetByID(ctx, reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, "测试提醒", savedReminder.Title)
		assert.True(t, savedReminder.IsActive)
	})

	// 测试用例2: 延期提醒功能
	t.Run("延期提醒功能测试", func(t *testing.T) {
		// 创建原始提醒记录（模拟已发送的提醒）
		originalLog := &models.ReminderLog{
			ReminderID:    1, // 使用上面创建的提醒
			ScheduledTime: time.Now().Add(-time.Hour), // 1小时前
			Status:        models.ReminderStatusSent,
			SentTime:      &[]time.Time{time.Now().Add(-time.Hour)}[0],
		}
		err := reminderLogRepo.Create(ctx, originalLog)
		require.NoError(t, err)

		// 创建延期提醒（延期1小时）
		delayTime := time.Now().Add(time.Hour)
		err = reminderLogService.CreateDelayReminder(ctx, originalLog.ID, delayTime, 1)
		require.NoError(t, err)

		// 验证原始记录已更新为延期状态
		updatedLog, err := reminderLogRepo.GetByID(ctx, originalLog.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ReminderStatusSkipped, updatedLog.Status)
		assert.Contains(t, updatedLog.UserResponse, "延期1小时")

		// 验证新的延期提醒记录已创建
		// 由于CreateDelayReminder只创建新的ReminderLog，不创建Reminder，
		// 我们需要检查是否有新的记录
		// 注意：这里可能需要调整实现，让延期提醒也能被调度
	})

	// 测试用例3: 提醒执行流程
	t.Run("提醒执行流程测试", func(t *testing.T) {
		// 创建提醒记录（模拟待执行的提醒）
		reminderLog := &models.ReminderLog{
			ReminderID:    1,
			ScheduledTime: time.Now(),
			Status:        models.ReminderStatusPending,
		}
		err := reminderLogRepo.Create(ctx, reminderLog)
		require.NoError(t, err)

		// 验证记录已创建且状态正确
		savedLog, err := reminderLogRepo.GetByID(ctx, reminderLog.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ReminderStatusPending, savedLog.Status)
		assert.NotNil(t, savedLog.Reminder) // 应该预加载提醒信息
		assert.NotNil(t, savedLog.Reminder.User) // 应该预加载用户信息
	})

	// 测试用例4: 超时关怀流程
	t.Run("超时关怀流程测试", func(t *testing.T) {
		// 创建已发送但未回复的提醒记录（超过1小时）
		sentTime := time.Now().Add(-2 * time.Hour) // 2小时前发送
		overdueLog := &models.ReminderLog{
			ReminderID:    1,
			ScheduledTime: sentTime,
			Status:        models.ReminderStatusSent,
			SentTime:      &sentTime,
		}
		err := reminderLogRepo.Create(ctx, overdueLog)
		require.NoError(t, err)

		// 获取超时提醒
		overdueReminders, err := reminderLogService.GetOverdueReminders(ctx)
		require.NoError(t, err)
		
		// 应该能找到超时的提醒
		found := false
		for _, log := range overdueReminders {
			if log.ID == overdueLog.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "应该找到超时的提醒")
	})
}

// TestDelayReminderWorkflow 测试延期提醒的完整工作流程
func TestDelayReminderWorkflow(t *testing.T) {
	// 设置测试数据库
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 初始化仓储层
	userRepo := sqliterepo.NewUserRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)
	reminderLogRepo := sqliterepo.NewReminderLogRepository(db)

	// 初始化服务层
	userService := service.NewUserService(userRepo)
	reminderService := service.NewReminderService(reminderRepo)
	reminderLogService := service.NewReminderLogService(reminderLogRepo, reminderRepo)
	
	// 注意：为了完整测试延期功能，我们需要调度器服务
	// 但由于循环依赖，我们需要特殊处理
	
	ctx := context.Background()

	// 创建测试用户
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := userService.CreateUser(ctx, user)
	require.NoError(t, err)

	// 创建测试提醒
	reminder := &models.Reminder{
		UserID:       user.ID,
		Title:        "每日运动提醒",
		Description:  "该去跑步了",
		TargetTime:   "18:00",
		SchedulePattern: "daily",
		IsActive:     true,
	}
	err = reminderService.CreateReminder(ctx, reminder)
	require.NoError(t, err)

	// 模拟完整的延期流程
	t.Run("完整延期流程测试", func(t *testing.T) {
		// 1. 创建原始提醒记录（模拟已发送的提醒）
		originalLog := &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: time.Now().Add(-30 * time.Minute), // 30分钟前
			Status:        models.ReminderStatusSent,
			SentTime:      &[]time.Time{time.Now().Add(-30 * time.Minute)}[0],
		}
		err := reminderLogRepo.Create(ctx, originalLog)
		require.NoError(t, err)

		// 2. 模拟用户点击"延期1小时"
		delayTime := time.Now().Add(time.Hour)
		err = reminderLogService.CreateDelayReminder(ctx, originalLog.ID, delayTime, 1)
		require.NoError(t, err)

		// 3. 验证原始记录状态
		updatedLog, err := reminderLogRepo.GetByID(ctx, originalLog.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ReminderStatusSkipped, updatedLog.Status)
		assert.Contains(t, updatedLog.UserResponse, "延期1小时")

		// 4. 验证新的延期提醒记录
		// 注意：当前的CreateDelayReminder只创建ReminderLog，不创建Reminder
		// 这意味着延期提醒不会被调度器自动执行
		// 这是一个需要修复的问题
		
		// 为了完整测试，我们需要验证延期提醒是否能被正确调度
		// 但由于架构限制，这里只能验证记录创建
		logs, err := reminderLogRepo.GetByReminderID(ctx, reminder.ID, 10, 0)
		require.NoError(t, err)
		
		// 应该至少有2条记录（原始记录和新的延期记录）
		assert.GreaterOrEqual(t, len(logs), 2)
		
		// 找到延期记录
		var delayLog *models.ReminderLog
		for _, log := range logs {
			if log.Status == models.ReminderStatusPending && log.ScheduledTime.Equal(delayTime.Truncate(time.Second)) {
				delayLog = log
				break
			}
		}
		
		assert.NotNil(t, delayLog, "应该找到延期提醒记录")
		assert.Equal(t, models.ReminderStatusPending, delayLog.Status)
	})
}

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	// 创建内存数据库用于测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(
		&models.User{},
		&models.Reminder{},
		&models.ReminderLog{},
	)
	require.NoError(t, err)

	// 返回数据库和清理函数
	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}