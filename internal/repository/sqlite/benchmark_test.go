package sqlite

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupBenchmarkDB 设置基准测试数据库
func setupBenchmarkDB(b *testing.B) (*gorm.DB, *QueryOptimizer, func()) {
	b.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatalf("创建数据库失败: %v", err)
	}

	// 自动迁移
	db.AutoMigrate(&models.User{}, &models.Reminder{})

	// 创建查询优化器
	queryOptimizer := NewQueryOptimizer(10 * time.Millisecond)

	cleanup := func() {
		queryOptimizer.Stop()
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	return db, queryOptimizer, cleanup
}

// newTestReminderForBench 为基准测试创建提醒对象（不保存到数据库）
func newTestReminderForBench(userID uint, index int) *models.Reminder {
	return &models.Reminder{
		UserID:          userID,
		Title:           "测试提醒",
		Description:     "Test reminder " + string(rune('A'+index)),
		SchedulePattern: string(models.SchedulePatternDaily),
		TargetTime:      "08:00:00",
		IsActive:        true,
		Type:            models.ReminderTypeHabit,
	}
}

// BenchmarkReminderRepository_Create 基准测试：创建提醒
func BenchmarkReminderRepository_Create(b *testing.B) {
	db, queryOptimizer, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reminder := newTestReminderForBench(1, i%26)
		_ = repo.Create(ctx, reminder)
	}
}

// BenchmarkReminderRepository_GetByID 基准测试：按ID获取提醒
func BenchmarkReminderRepository_GetByID(b *testing.B) {
	db, queryOptimizer, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	// 先创建一些提醒
	for i := 0; i < 100; i++ {
		reminder := newTestReminderForBench(1, i)
		_ = repo.Create(ctx, reminder)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.GetByID(ctx, uint(i%100+1))
	}
}

// BenchmarkReminderRepository_GetByUserID 基准测试：按用户ID获取提醒
func BenchmarkReminderRepository_GetByUserID(b *testing.B) {
	db, queryOptimizer, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	// 创建测试数据
	userID := uint(1)
	for i := 0; i < 100; i++ {
		reminder := newTestReminderForBench(userID, i)
		_ = repo.Create(ctx, reminder)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.GetByUserID(ctx, userID)
	}
}

// BenchmarkReminderRepository_GetActiveReminders 基准测试：获取活跃提醒
func BenchmarkReminderRepository_GetActiveReminders(b *testing.B) {
	db, queryOptimizer, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	// 创建测试数据
	for i := 0; i < 100; i++ {
		reminder := newTestReminderForBench(1, i)
		_ = repo.Create(ctx, reminder)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.GetActiveReminders(ctx)
	}
}

// BenchmarkReminderRepository_Update 基准测试：更新提醒
func BenchmarkReminderRepository_Update(b *testing.B) {
	db, queryOptimizer, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	// 先创建提醒
	reminder := newTestReminderForBench(1, 0)
	_ = repo.Create(ctx, reminder)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reminder.Title = "更新后的标题"
		_ = repo.Update(ctx, reminder)
	}
}

// BenchmarkReminderRepository_Delete 基准测试：删除提醒
func BenchmarkReminderRepository_Delete(b *testing.B) {
	db, queryOptimizer, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 每次测试创建新的提醒ID
		newReminder := newTestReminderForBench(1, i)
		_ = repo.Create(ctx, newReminder)
		_ = repo.Delete(ctx, newReminder.ID)
	}
}

// BenchmarkReminderRepository_CountByStatus 基准测试：按状态统计
func BenchmarkReminderRepository_CountByStatus(b *testing.B) {
	db, queryOptimizer, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	repo := NewReminderRepository(db, queryOptimizer)
	ctx := context.Background()

	// 创建测试数据
	for i := 0; i < 100; i++ {
		reminder := newTestReminderForBench(1, i)
		if i%2 == 0 {
			reminder.IsActive = true
		} else {
			reminder.IsActive = false
		}
		_ = repo.Create(ctx, reminder)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.CountByStatus(ctx, models.ReminderStatStatusActive)
	}
}

// BenchmarkQueryOptimizer_RecordQuery 基准测试：记录查询指标
func BenchmarkQueryOptimizer_RecordQuery(b *testing.B) {
	optimizer := NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		optimizer.RecordQuery("reminders", "test", "SELECT * FROM reminders", time.Millisecond, 10, false)
	}
}

// BenchmarkQueryOptimizer_GetSlowQueries 基准测试：获取慢查询
func BenchmarkQueryOptimizer_GetSlowQueries(b *testing.B) {
	optimizer := NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	// 记录一些慢查询
	for i := 0; i < 100; i++ {
		optimizer.RecordQuery("reminders", "test", "SELECT * FROM reminders", 50*time.Millisecond, 10, false)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = optimizer.GetSlowQueries(50)
	}
}

// BenchmarkQueryOptimizer_GetMetrics 基准测试：获取指标
func BenchmarkQueryOptimizer_GetMetrics(b *testing.B) {
	optimizer := NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	// 记录一些查询
	for i := 0; i < 1000; i++ {
		duration := time.Duration(i%100) * time.Millisecond
		optimizer.RecordQuery("reminders", "test", "SELECT * FROM reminders", duration, 10, false)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = optimizer.GetMetrics()
	}
}

// BenchmarkQueryOptimizer_Concurrency 基准测试：并发查询记录
func BenchmarkQueryOptimizer_Concurrency(b *testing.B) {
	optimizer := NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			optimizer.RecordQuery("reminders", "test", "SELECT * FROM reminders", time.Millisecond, 10, false)
		}
	})
}
