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

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.User{}, &models.Reminder{}, &models.ReminderLog{})
	require.NoError(t, err)

	return db
}

// createTestUser 创建测试用户
func createTestUser(t *testing.T, db *gorm.DB) *models.User {
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

// createTestReminder 创建测试提醒
func createTestReminder(t *testing.T, db *gorm.DB, userID uint) *models.Reminder {
	reminder := &models.Reminder{
		UserID:          userID,
		Title:           "测试提醒",
		Description:     "测试描述",
		Type:            models.ReminderTypeTask,
		SchedulePattern: "daily",
		TargetTime:      "14:30:00",
		IsActive:        true,
	}
	err := db.Create(reminder).Error
	require.NoError(t, err)
	return reminder
}

// TestReminderLogRepository_Create 测试创建提醒日志
func TestReminderLogRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	t.Run("创建成功", func(t *testing.T) {
		log := &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: time.Now().Add(time.Hour),
			Status:        models.ReminderStatusPending,
		}

		err := repo.Create(ctx, log)
		assert.NoError(t, err)
		assert.NotZero(t, log.ID)
	})

	t.Run("创建多个日志", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			log := &models.ReminderLog{
				ReminderID:    reminder.ID,
				ScheduledTime: time.Now().Add(time.Hour * time.Duration(i+1)),
				Status:        models.ReminderStatusPending,
			}
			err := repo.Create(ctx, log)
			assert.NoError(t, err)
		}
	})
}

// TestReminderLogRepository_GetByID 测试根据ID获取提醒日志
func TestReminderLogRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	// 创建测试日志
	scheduledTime := time.Now().Add(time.Hour)
	log := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: scheduledTime,
		Status:        models.ReminderStatusPending,
		UserResponse:  "测试响应",
	}
	err := repo.Create(ctx, log)
	require.NoError(t, err)

	t.Run("获取存在的日志", func(t *testing.T) {
		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedLog)
		assert.Equal(t, log.ID, retrievedLog.ID)
		assert.Equal(t, reminder.ID, retrievedLog.ReminderID)
		assert.NotNil(t, retrievedLog.Reminder) // 验证预加载
	})

	t.Run("获取不存在的日志", func(t *testing.T) {
		retrievedLog, err := repo.GetByID(ctx, 9999)
		assert.NoError(t, err)
		assert.Nil(t, retrievedLog)
	})

	t.Run("预加载用户信息", func(t *testing.T) {
		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedLog.Reminder.User)
		assert.Equal(t, user.ID, retrievedLog.Reminder.User.ID)
	})
}

// TestReminderLogRepository_GetByReminderID 测试根据提醒ID获取日志列表
func TestReminderLogRepository_GetByReminderID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	// 创建多个日志
	for i := 0; i < 5; i++ {
		log := &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: time.Now().Add(time.Hour * time.Duration(i+1)),
			Status:        models.ReminderStatusPending,
		}
		err := repo.Create(ctx, log)
		require.NoError(t, err)
	}

	t.Run("获取所有日志", func(t *testing.T) {
		logs, err := repo.GetByReminderID(ctx, reminder.ID, 0, 0)
		assert.NoError(t, err)
		assert.Len(t, logs, 5)
	})

	t.Run("测试分页 - limit", func(t *testing.T) {
		logs, err := repo.GetByReminderID(ctx, reminder.ID, 3, 0)
		assert.NoError(t, err)
		assert.Len(t, logs, 3)
	})

	t.Run("测试分页 - offset", func(t *testing.T) {
		logs, err := repo.GetByReminderID(ctx, reminder.ID, 0, 2)
		assert.NoError(t, err)
		assert.Len(t, logs, 3)
	})

	t.Run("测试分页 - limit和offset", func(t *testing.T) {
		logs, err := repo.GetByReminderID(ctx, reminder.ID, 2, 2)
		assert.NoError(t, err)
		assert.Len(t, logs, 2)
	})

	t.Run("获取不存在的提醒的日志", func(t *testing.T) {
		logs, err := repo.GetByReminderID(ctx, 9999, 0, 0)
		assert.NoError(t, err)
		assert.Len(t, logs, 0)
	})
}

// TestReminderLogRepository_GetPendingLogs 测试获取待处理日志
func TestReminderLogRepository_GetPendingLogs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	// 创建不同状态的日志
	pendingLog := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(time.Hour),
		Status:        models.ReminderStatusPending,
	}
	err := repo.Create(ctx, pendingLog)
	require.NoError(t, err)

	sentLog := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(-time.Hour),
		Status:        models.ReminderStatusSent,
	}
	err = repo.Create(ctx, sentLog)
	require.NoError(t, err)

	completedLog := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(-time.Hour),
		Status:        models.ReminderStatusCompleted,
	}
	err = repo.Create(ctx, completedLog)
	require.NoError(t, err)

	t.Run("获取待处理日志", func(t *testing.T) {
		logs, err := repo.GetPendingLogs(ctx)
		assert.NoError(t, err)

		// 应该只包含pending和sent状态的日志
		foundPending := false
		foundSent := false
		for _, log := range logs {
			if log.Status == models.ReminderStatusPending {
				foundPending = true
			}
			if log.Status == models.ReminderStatusSent {
				foundSent = true
			}
		}
		assert.True(t, foundPending, "应该包含pending状态的日志")
		assert.True(t, foundSent, "应该包含sent状态的日志")
	})
}

// TestReminderLogRepository_Update 测试更新提醒日志
func TestReminderLogRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	log := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(time.Hour),
		Status:        models.ReminderStatusPending,
	}
	err := repo.Create(ctx, log)
	require.NoError(t, err)

	t.Run("更新状态为已发送", func(t *testing.T) {
		log.Status = models.ReminderStatusSent
		err := repo.Update(ctx, log)
		assert.NoError(t, err)

		// 验证更新
		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.Equal(t, models.ReminderStatusSent, retrievedLog.Status)
	})

	t.Run("更新响应内容", func(t *testing.T) {
		log.UserResponse = "用户确认完成"
		err := repo.Update(ctx, log)
		assert.NoError(t, err)

		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.Equal(t, "用户确认完成", retrievedLog.UserResponse)
	})
}

// TestReminderLogRepository_Delete 测试删除提醒日志
func TestReminderLogRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	log := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(time.Hour),
		Status:        models.ReminderStatusPending,
	}
	err := repo.Create(ctx, log)
	require.NoError(t, err)

	t.Run("删除存在的日志", func(t *testing.T) {
		err := repo.Delete(ctx, log.ID)
		assert.NoError(t, err)

		// 验证已删除
		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.Nil(t, retrievedLog)
	})

	t.Run("删除不存在的日志", func(t *testing.T) {
		// GORM的Delete方法不返回错误当记录不存在时
		err := repo.Delete(ctx, 9999)
		assert.NoError(t, err) // GORM的Delete不返回错误
	})
}

// TestReminderLogRepository_GetUserLogs 测试获取用户的日志
func TestReminderLogRepository_GetUserLogs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	// 创建日志
	for i := 0; i < 5; i++ {
		log := &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: time.Now().Add(time.Hour * time.Duration(i+1)),
			Status:        models.ReminderStatusPending,
		}
		err := repo.Create(ctx, log)
		require.NoError(t, err)
	}

	t.Run("获取用户所有日志", func(t *testing.T) {
		logs, err := repo.GetUserLogs(ctx, user.ID, time.Time{})
		assert.NoError(t, err)
		assert.Len(t, logs, 5)
	})

	t.Run("按时间筛选日志", func(t *testing.T) {
		since := time.Now().Add(2 * time.Hour)
		logs, err := repo.GetUserLogs(ctx, user.ID, since)
		assert.NoError(t, err)
		// 应该只包含2小时之后的日志
		for _, log := range logs {
			assert.True(t, log.ScheduledTime.After(since))
		}
	})

	t.Run("获取其他用户的日志", func(t *testing.T) {
		logs, err := repo.GetUserLogs(ctx, 9999, time.Time{})
		assert.NoError(t, err)
		assert.Len(t, logs, 0)
	})
}

// TestReminderLogRepository_CreateDelayReminder 测试创建延期提醒
func TestReminderLogRepository_CreateDelayReminder(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	// 创建原始日志
	originalTime := time.Now().Add(time.Hour)
	originalLog := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: originalTime,
		Status:        models.ReminderStatusPending,
	}
	err := repo.Create(ctx, originalLog)
	require.NoError(t, err)

	t.Run("创建延期提醒", func(t *testing.T) {
		delayTime := time.Now().Add(2 * time.Hour)
		err := repo.CreateDelayReminder(ctx, originalLog.ID, delayTime, 1)
		assert.NoError(t, err)

		// 验证延期提醒已创建
		logs, err := repo.GetByReminderID(ctx, reminder.ID, 0, 0)
		assert.NoError(t, err)
		assert.Len(t, logs, 2) // 原始日志 + 延期日志
	})

	t.Run("原始日志不存在", func(t *testing.T) {
		delayTime := time.Now().Add(2 * time.Hour)
		err := repo.CreateDelayReminder(ctx, 9999, delayTime, 1)
		assert.Error(t, err)
	})
}

// TestReminderLogRepository_MarkAsCompleted 测试标记为已完成
func TestReminderLogRepository_MarkAsCompleted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	log := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(time.Hour),
		Status:        models.ReminderStatusPending,
	}
	err := repo.Create(ctx, log)
	require.NoError(t, err)

	t.Run("标记为已完成", func(t *testing.T) {
		err := repo.MarkAsCompleted(ctx, log.ID, "用户完成了提醒")
		assert.NoError(t, err)

		// 验证
		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.Equal(t, models.ReminderStatusCompleted, retrievedLog.Status)
		assert.Equal(t, "用户完成了提醒", retrievedLog.UserResponse)
	})

	t.Run("标记不存在的日志", func(t *testing.T) {
		// GORM的Updates方法不返回错误当没有匹配记录时
		err := repo.MarkAsCompleted(ctx, 9999, "测试")
		assert.NoError(t, err)
	})
}

// TestReminderLogRepository_MarkAsSkipped 测试标记为已跳过
func TestReminderLogRepository_MarkAsSkipped(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	log := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(time.Hour),
		Status:        models.ReminderStatusPending,
	}
	err := repo.Create(ctx, log)
	require.NoError(t, err)

	t.Run("标记为已跳过", func(t *testing.T) {
		err := repo.MarkAsSkipped(ctx, log.ID, "用户跳过了")
		assert.NoError(t, err)

		// 验证
		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.Equal(t, models.ReminderStatusSkipped, retrievedLog.Status)
		assert.Equal(t, "用户跳过了", retrievedLog.UserResponse)
	})
}

// TestReminderLogRepository_GetOverdueReminders 测试获取逾期提醒
func TestReminderLogRepository_GetOverdueReminders(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	// 创建逾期日志（2小时前）
	overdueLog := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(-3 * time.Hour),
		Status:        models.ReminderStatusSent,
	}
	err := repo.Create(ctx, overdueLog)
	require.NoError(t, err)

	// 创建未逾期日志
	futureLog := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(time.Hour),
		Status:        models.ReminderStatusPending,
	}
	err = repo.Create(ctx, futureLog)
	require.NoError(t, err)

	t.Run("获取逾期提醒", func(t *testing.T) {
		logs, err := repo.GetOverdueReminders(ctx)
		assert.NoError(t, err)

		// 应该只包含逾期的日志
		for _, log := range logs {
			assert.True(t, log.ScheduledTime.Before(time.Now()))
			assert.Equal(t, models.ReminderStatusSent, log.Status)
		}
	})
}

// TestReminderLogRepository_UpdateFollowUpCount 测试更新跟进次数
func TestReminderLogRepository_UpdateFollowUpCount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderLogRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	reminder := createTestReminder(t, db, user.ID)

	log := &models.ReminderLog{
		ReminderID:    reminder.ID,
		ScheduledTime: time.Now().Add(time.Hour),
		Status:        models.ReminderStatusPending,
		FollowUpCount: 0,
	}
	err := repo.Create(ctx, log)
	require.NoError(t, err)

	t.Run("更新跟进次数", func(t *testing.T) {
		err := repo.UpdateFollowUpCount(ctx, log.ID)
		assert.NoError(t, err)

		// 验证
		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, retrievedLog.FollowUpCount)
	})

	t.Run("多次更新跟进次数", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			err := repo.UpdateFollowUpCount(ctx, log.ID)
			assert.NoError(t, err)
		}

		retrievedLog, err := repo.GetByID(ctx, log.ID)
		assert.NoError(t, err)
		assert.Equal(t, 4, retrievedLog.FollowUpCount) // 1 + 3
	})
}
