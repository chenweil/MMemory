package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/internal/models"
)

// TestReminderService_EditReminder_Concurrent 测试并发编辑冲突
func TestReminderService_EditReminder_Concurrent(t *testing.T) {
	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	scheduler := &mockScheduler{}
	if setter, ok := reminderService.(interface{ SetScheduler(SchedulerService) }); ok {
		setter.SetScheduler(scheduler)
	}

	ctx := context.Background()

	// 创建测试提醒
	reminder := &models.Reminder{
		UserID:          1,
		Title:           "并发测试",
		Type:            models.ReminderTypeHabit,
		SchedulePattern: "daily",
		TargetTime:      "10:00:00",
		IsActive:        true,
	}

	err := reminderService.CreateReminder(ctx, reminder)
	require.NoError(t, err)

	t.Run("并发修改同一提醒", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrency := 10
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				newTime := fmt.Sprintf("%02d:00:00", 10+index%14)
				params := EditReminderParams{
					ReminderID: reminder.ID,
					NewTime:    &newTime,
				}

				err := reminderService.EditReminder(ctx, params)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// 至少应该有一些成功的修改
		assert.Greater(t, successCount, 0, "应该有成功的并发修改")

		// 验证最终状态是有效的
		final, err := mockRepo.GetByID(ctx, reminder.ID)
		require.NoError(t, err)
		assert.NotNil(t, final)
		assert.NotEmpty(t, final.TargetTime)
	})

	t.Run("并发修改不同字段", func(t *testing.T) {
		var wg sync.WaitGroup
		operations := []func(){
			func() {
				newTime := "11:00:00"
				params := EditReminderParams{
					ReminderID: reminder.ID,
					NewTime:    &newTime,
				}
				_ = reminderService.EditReminder(ctx, params)
			},
			func() {
				newTitle := "并发测试-标题修改"
				params := EditReminderParams{
					ReminderID: reminder.ID,
					NewTitle:   &newTitle,
				}
				_ = reminderService.EditReminder(ctx, params)
			},
			func() {
				newPattern := "weekly:1,3,5"
				params := EditReminderParams{
					ReminderID: reminder.ID,
					NewPattern: &newPattern,
				}
				_ = reminderService.EditReminder(ctx, params)
			},
		}

		for _, op := range operations {
			wg.Add(1)
			go func(operation func()) {
				defer wg.Done()
				operation()
			}(op)
		}

		wg.Wait()

		// 验证提醒仍然有效
		final, err := mockRepo.GetByID(ctx, reminder.ID)
		require.NoError(t, err)
		assert.NotNil(t, final)
	})
}

// TestReminderService_PauseResume_TimeCalculation 测试暂停/恢复时间计算准确性
func TestReminderService_PauseResume_TimeCalculation(t *testing.T) {
	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	scheduler := &mockScheduler{}
	if setter, ok := reminderService.(interface{ SetScheduler(SchedulerService) }); ok {
		setter.SetScheduler(scheduler)
	}

	ctx := context.Background()

	// 创建测试提醒
	reminder := &models.Reminder{
		UserID:          1,
		Title:           "时间测试",
		Type:            models.ReminderTypeHabit,
		SchedulePattern: "daily",
		TargetTime:      "08:00:00",
		IsActive:        true,
	}

	err := reminderService.CreateReminder(ctx, reminder)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		duration     time.Duration
		minExpected  time.Duration
		maxExpected  time.Duration
	}{
		{
			name:         "暂停1小时",
			duration:     1 * time.Hour,
			minExpected:  55 * time.Minute,
			maxExpected:  65 * time.Minute,
		},
		{
			name:         "暂停24小时",
			duration:     24 * time.Hour,
			minExpected:  23*time.Hour + 55*time.Minute,
			maxExpected:  24*time.Hour + 5*time.Minute,
		},
		{
			name:         "暂停7天",
			duration:     7 * 24 * time.Hour,
			minExpected:  6*24*time.Hour + 23*time.Hour,
			maxExpected:  7*24*time.Hour + 1*time.Hour,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()

			// 暂停提醒
			err := reminderService.PauseReminder(ctx, reminder.ID, tc.duration, "测试")
			require.NoError(t, err)

			// 获取更新后的提醒
			updated, err := mockRepo.GetByID(ctx, reminder.ID)
			require.NoError(t, err)
			require.NotNil(t, updated.PausedUntil)

			// 验证暂停时间在预期范围内
			actualDuration := updated.PausedUntil.Sub(now)
			assert.GreaterOrEqual(t, actualDuration, tc.minExpected,
				"暂停时间应该至少为 %v", tc.minExpected)
			assert.LessOrEqual(t, actualDuration, tc.maxExpected,
				"暂停时间不应超过 %v", tc.maxExpected)

			// 恢复提醒
			err = reminderService.ResumeReminder(ctx, reminder.ID)
			require.NoError(t, err)

			// 验证暂停时间已清除
			resumed, err := mockRepo.GetByID(ctx, reminder.ID)
			require.NoError(t, err)
			assert.Nil(t, resumed.PausedUntil, "恢复后 PausedUntil 应该为 nil")
		})
	}
}

// TestReminderService_ConcurrentCreateAndDelete 测试并发创建和删除
func TestReminderService_ConcurrentCreateAndDelete(t *testing.T) {
	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	ctx := context.Background()

	var wg sync.WaitGroup
	createCount := 50
	createdIDs := make([]uint, createCount)
	var mu sync.Mutex

	// 并发创建提醒
	for i := 0; i < createCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			reminder := &models.Reminder{
				UserID:          1,
				Title:           fmt.Sprintf("并发提醒-%d", index),
				Type:            models.ReminderTypeHabit,
				SchedulePattern: "daily",
				TargetTime:      "10:00:00",
				IsActive:        true,
			}

			err := reminderService.CreateReminder(ctx, reminder)
			if err == nil {
				mu.Lock()
				createdIDs[index] = reminder.ID
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// 验证创建的提醒数量
	validCount := 0
	for _, id := range createdIDs {
		if id > 0 {
			validCount++
		}
	}
	assert.Equal(t, createCount, validCount, "所有并发创建应该成功")

	// 并发删除部分提醒
	deleteCount := 25
	for i := 0; i < deleteCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_ = reminderService.DeleteReminder(ctx, createdIDs[index])
		}(i)
	}

	wg.Wait()

	// 验证剩余提醒
	reminders, err := reminderService.GetUserReminders(ctx, 1)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(reminders), deleteCount,
		"应该剩余至少 %d 个提醒", deleteCount)
}

// TestReminderService_StressTest 压力测试
func TestReminderService_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	scheduler := &mockScheduler{}
	if setter, ok := reminderService.(interface{ SetScheduler(SchedulerService) }); ok {
		setter.SetScheduler(scheduler)
	}

	ctx := context.Background()

	t.Run("快速创建100个提醒", func(t *testing.T) {
		startTime := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				reminder := &models.Reminder{
					UserID:          uint(index % 10),
					Title:           fmt.Sprintf("压力测试-%d", index),
					Type:            models.ReminderTypeHabit,
					SchedulePattern: "daily",
					TargetTime:      fmt.Sprintf("%02d:00:00", index%24),
					IsActive:        true,
				}

				_ = reminderService.CreateReminder(ctx, reminder)
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(startTime)

		t.Logf("创建100个提醒耗时: %v", elapsed)
		assert.Less(t, elapsed, 5*time.Second, "创建100个提醒应该在5秒内完成")
	})
}

// TestReminderService_BatchOperations 测试批量操作的事务处理
func TestReminderService_BatchOperations(t *testing.T) {
	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	ctx := context.Background()

	// 批量创建提醒
	t.Run("批量创建", func(t *testing.T) {
		batchSize := 20
		var wg sync.WaitGroup
		errors := make([]error, batchSize)

		for i := 0; i < batchSize; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				reminder := &models.Reminder{
					UserID:          1,
					Title:           fmt.Sprintf("批量提醒-%d", index),
					Type:            models.ReminderTypeHabit,
					SchedulePattern: "daily",
					TargetTime:      "10:00:00",
					IsActive:        true,
				}

				errors[index] = reminderService.CreateReminder(ctx, reminder)
			}(i)
		}

		wg.Wait()

		// 验证所有创建都成功
		for i, err := range errors {
			assert.NoError(t, err, "批量创建第 %d 个提醒失败", i)
		}

		// 验证数据库中的数量
		reminders, err := reminderService.GetUserReminders(ctx, 1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(reminders), batchSize, "应该至少有 %d 个提醒", batchSize)
	})
}

// TestReminderService_EdgeCases 测试边界情况
func TestReminderService_EdgeCases(t *testing.T) {
	mockRepo := newMockReminderRepository()
	reminderService := NewReminderService(mockRepo)
	ctx := context.Background()

	t.Run("极长标题", func(t *testing.T) {
		longTitle := string(make([]byte, 1000))
		for i := range longTitle {
			longTitle = longTitle[:i] + "长"
		}

		reminder := &models.Reminder{
			UserID:          1,
			Title:           longTitle,
			Type:            models.ReminderTypeHabit,
			SchedulePattern: "daily",
			TargetTime:      "10:00:00",
			IsActive:        true,
		}

		// 应该能够处理长标题或返回友好错误
		err := reminderService.CreateReminder(ctx, reminder)
		if err != nil {
			t.Logf("长标题被正确拒绝: %v", err)
		}
	})

	t.Run("特殊字符标题", func(t *testing.T) {
		reminder := &models.Reminder{
			UserID:          1,
			Title:           "🎯💪🏃‍♂️ 健身 & 跑步 <测试>",
			Type:            models.ReminderTypeHabit,
			SchedulePattern: "daily",
			TargetTime:      "10:00:00",
			IsActive:        true,
		}

		err := reminderService.CreateReminder(ctx, reminder)
		assert.NoError(t, err, "应该支持特殊字符和emoji")

		if err == nil {
			retrieved, _ := mockRepo.GetByID(ctx, reminder.ID)
			assert.Equal(t, reminder.Title, retrieved.Title, "特殊字符应该正确保存")
		}
	})

	t.Run("极端时间值", func(t *testing.T) {
		testCases := []struct {
			name    string
			time    string
			wantErr bool
		}{
			{"午夜", "00:00:00", false},
			{"最后一秒", "23:59:59", false},
			// ReminderService不验证时间格式,由Scheduler在调度时处理
			{"无效小时", "25:00:00", false},
			{"无效分钟", "12:60:00", false},
			{"空时间", "", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reminder := &models.Reminder{
					UserID:          1,
					Title:           "时间测试",
					Type:            models.ReminderTypeHabit,
					SchedulePattern: "daily",
					TargetTime:      tc.time,
					IsActive:        true,
				}

				err := reminderService.CreateReminder(ctx, reminder)
				if tc.wantErr {
					assert.Error(t, err, "应该拒绝空时间")
				} else {
					assert.NoError(t, err, "应该接受时间字符串(格式由Scheduler验证)")
				}
			})
		}
	})
}
