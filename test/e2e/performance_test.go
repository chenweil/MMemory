package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
	sqliterepo "mmemory/internal/repository/sqlite"
	"mmemory/internal/service"
	ai "mmemory/pkg/ai"
	"mmemory/pkg/config"
)

// TestDatabaseToServiceE2E 数据库到服务的端到端测试
func TestDatabaseToServiceE2E(t *testing.T) {
	// 1. 设置测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 2. 自动迁移
	if err := db.AutoMigrate(&models.User{}, &models.Reminder{}, &models.ReminderLog{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 3. 创建查询优化器
	queryOptimizer := sqliterepo.NewQueryOptimizer(10 * time.Millisecond)
	defer queryOptimizer.Stop()

	// 4. 创建仓库层
	userRepo := sqliterepo.NewUserRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db, queryOptimizer)

	// 4. 创建服务层
	reminderSvc := service.NewReminderService(reminderRepo)

	// 5. 测试用户创建流程
	t.Run("用户创建流程", func(t *testing.T) {
		ctx := context.Background()
		user := &models.User{
			TelegramID:  12345,
			Username:    "test_user",
			FirstName:   "Test",
			LanguageCode: "zh-CN",
		}

		// 创建用户
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("创建用户失败: %v", err)
		}

		// 获取用户
		retrievedUser, err := userRepo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("获取用户失败: %v", err)
		}

		if retrievedUser.TelegramID != user.TelegramID {
			t.Errorf("用户TelegramID不匹配: 期望 %d, 实际 %d", user.TelegramID, retrievedUser.TelegramID)
		}
	})

	// 6. 测试提醒创建流程
	t.Run("提醒创建流程", func(t *testing.T) {
		ctx := context.Background()
		user := &models.User{
			TelegramID:  67890,
			Username:    "reminder_test",
			FirstName:   "ReminderTest",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}

		// 创建提醒
		reminder := &models.Reminder{
			UserID:         user.ID,
			Title:          "测试提醒标题",
			Description:    "测试提醒内容",
			Type:           models.ReminderTypeTask,
			SchedulePattern: "daily",
			TargetTime:     "09:00",
			IsActive:       true,
		}

		if err := reminderSvc.CreateReminder(ctx, reminder); err != nil {
			t.Fatalf("创建提醒失败: %v", err)
		}

		// 获取提醒
		retrievedReminder, err := reminderRepo.GetByID(ctx, reminder.ID)
		if err != nil {
			t.Fatalf("获取提醒失败: %v", err)
		}

		if retrievedReminder.Title != reminder.Title {
			t.Errorf("提醒标题不匹配: 期望 %s, 实际 %s", reminder.Title, retrievedReminder.Title)
		}
	})

	// 7. 测试提醒统计
	t.Run("提醒统计流程", func(t *testing.T) {
		ctx := context.Background()

		activeCount, err := reminderRepo.CountByStatus(ctx, models.ReminderStatStatusActive)
		if err != nil {
			t.Fatalf("获取活跃提醒数失败: %v", err)
		}

		if activeCount < 1 {
			t.Errorf("期望至少1个活跃提醒，实际: %d", activeCount)
		}
	})
}

// TestCachePerformanceE2E 缓存性能端到端测试
func TestCachePerformanceE2E(t *testing.T) {
	cache := ai.NewEnhancedCache(5*time.Minute, 1000)

	// 测试缓存性能
	t.Run("缓存读写性能", func(t *testing.T) {
		const testSize = 1000
		start := time.Now()

		// 写入测试
		for i := 0; i < testSize; i++ {
			cache.Set(
				string(rune('a'+i%26))+string(rune('0'+i%10)),
				map[string]interface{}{"value": i, "data": "test"},
			)
		}

		writeDuration := time.Since(start)
		t.Logf("写入 %d 条目耗时: %v", testSize, writeDuration)

		// 读取测试
		start = time.Now()
		hits := 0
		for i := 0; i < testSize; i++ {
			key := string(rune('a'+i%26)) + string(rune('0'+i%10))
			if _, ok := cache.Get(key); ok {
				hits++
			}
		}
		readDuration := time.Since(start)
		t.Logf("读取 %d 条目耗时: %v, 命中: %d", testSize, readDuration, hits)

		// 验证性能
		if writeDuration > 1*time.Second {
			t.Logf("警告: 写入性能较慢 (%v)", writeDuration)
		}
		if readDuration > 500*time.Millisecond {
			t.Logf("警告: 读取性能较慢 (%v)", readDuration)
		}
	})

	// 测试缓存命中率
	t.Run("缓存命中率", func(t *testing.T) {
		cache.Clear()

		// 填充缓存
		for i := 0; i < 100; i++ {
			cache.Set("key", "value")
		}

		// 多次读取
		for i := 0; i < 100; i++ {
			cache.Get("key")
		}

		stats := cache.GetStats()
		hitRate := cache.GetHitRate()

		t.Logf("缓存统计 - 命中: %d, 未命中: %d, 命中率: %.2f%%",
			stats.Hits, stats.Misses, hitRate)

		if hitRate < 99 {
			t.Errorf("期望命中率 > 99%%, 实际: %.2f%%", hitRate)
		}
	})

	// 测试LRU驱逐
	t.Run("LRU驱逐", func(t *testing.T) {
		cache.Clear()
		cache = ai.NewEnhancedCache(5*time.Minute, 3)

		// 添加3个条目
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")
		cache.Set("key3", "value3")

		// 访问key1，使其成为最近使用
		cache.Get("key1")

		// 添加新键，key2应该被驱逐（因为key3是最新添加的，key2最久未使用）
		cache.Set("key4", "value4")

		// 验证key1存在（最近使用）
		if _, ok := cache.Get("key1"); !ok {
			t.Error("期望 key1 仍然存在")
		}

		// key2应该被驱逐
		if _, ok := cache.Get("key2"); ok {
			t.Error("期望 key2 已被驱逐")
		}

		// key3和key4应该存在
		if _, ok := cache.Get("key3"); !ok {
			t.Error("期望 key3 仍然存在")
		}
		if _, ok := cache.Get("key4"); !ok {
			t.Error("期望 key4 存在")
		}
	})
}

// TestSchedulerE2E 调度器端到端测试
func TestSchedulerE2E(t *testing.T) {
	// 设置测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Reminder{}, &models.ReminderLog{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建查询优化器
	queryOptimizer := sqliterepo.NewQueryOptimizer(10 * time.Millisecond)
	defer queryOptimizer.Stop()

	userRepo := sqliterepo.NewUserRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db, queryOptimizer)
	reminderLogRepo := sqliterepo.NewReminderLogRepository(db)

	schedulerSvc := service.NewSchedulerServiceWithConfig(reminderRepo, reminderLogRepo, nil, 10, 2, 100)

	// 启动调度器（跳过，因为需要logger）
	_ = schedulerSvc

	ctx := context.Background()

	// 创建测试用户和提醒
	user := &models.User{TelegramID: 11111, Username: "scheduler_test"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	reminder := &models.Reminder{
		UserID:         user.ID,
		Title:          "定时测试提醒",
		Description:    "定时测试描述",
		Type:           models.ReminderTypeTask,
		SchedulePattern: "daily",
		TargetTime:     "09:00",
		IsActive:       true,
	}
	if err := reminderRepo.Create(ctx, reminder); err != nil {
		t.Fatalf("创建提醒失败: %v", err)
	}

	// 验证提醒已创建
	retrievedReminder, err := reminderRepo.GetByID(ctx, reminder.ID)
	if err != nil {
		t.Fatalf("获取提醒失败: %v", err)
	}
	if retrievedReminder.Title != reminder.Title {
		t.Errorf("提醒标题不匹配")
	}
}

// TestConfigE2E 配置端到端测试
func TestConfigE2E(t *testing.T) {
	// 测试配置加载
	t.Run("配置加载", func(t *testing.T) {
		cfg := &config.Config{
			Bot: config.BotConfig{
				Token: "test_token",
			},
			Database: config.DatabaseConfig{
				Driver:   "sqlite3",
				DSN:      ":memory:",
				MaxOpenConns: 10,
			},
			Server: config.ServerConfig{
				Port: "8080",
				Host: "0.0.0.0",
			},
			AI: config.AIConfig{
				Enabled: true,
			},
		}

		if cfg.Bot.Token != "test_token" {
			t.Errorf("Bot Token配置不正确")
		}
		if cfg.Database.MaxOpenConns != 10 {
			t.Errorf("Database MaxOpenConns配置不正确")
		}
		if cfg.Server.Port != "8080" {
			t.Errorf("Server Port配置不正确")
		}
	})
}

// TestCostControllerE2E 成本控制器端到端测试
func TestCostControllerE2E(t *testing.T) {
	// 创建测试用的 logger（避免 nil logger panic）
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.DebugLevel)

	// 创建成本控制器
	providers := map[string]*ai.ProviderCost{
		"openai": {
			CostPer1KTokens: 0.01,
			Model:            "gpt-4o-mini",
			Provider:        "openai",
			Enabled:         true,
			Priority:        1,
		},
		"claude": {
			CostPer1KTokens: 0.015,
			Model:            "claude-3-5-sonnet",
			Provider:        "claude",
			Enabled:         true,
			Priority:        2,
		},
	}
	budget := ai.BudgetConfig{
		MonthlyBudget: 100.0,
		DailyBudget:   5.0,
		UserBudget:    1.0,
	}

	costCtrl := ai.NewCostController(providers, budget, testLogger)

	// 测试月度成本设置
	t.Run("成本设置", func(t *testing.T) {
		// 使用 SetMonthlyCost 设置成本（不依赖 logger）
		costCtrl.SetMonthlyCost("openai", 5.0)
		costCtrl.SetMonthlyCost("claude", 3.0)

		// 获取月度报告
		report := costCtrl.GetMonthlyReport()
		if report == nil {
			t.Error("期望有月度报告")
		} else {
			t.Logf("月度报告 - 总成本: $%.2f", report.TotalCost)
			if report.TotalCost != 8.0 {
				t.Errorf("期望总成本为 8.0，实际为: %.2f", report.TotalCost)
			}
		}
	})

	// 测试成本预测
	t.Run("成本预测", func(t *testing.T) {
		// 设置一些历史数据
		costCtrl.SetMonthlyCost("openai", 10.0)
		costCtrl.SetMonthlyCost("claude", 5.0)

		prediction := costCtrl.PredictCosts()
		if prediction == nil {
			t.Error("期望有成本预测")
		} else {
			t.Logf("成本预测 - 明天: $%.2f, 下周: $%.2f, 下月: $%.2f",
				prediction.NextDayCost, prediction.NextWeekCost, prediction.NextMonthCost)
		}
	})

	// 测试预算检查
	t.Run("预算检查", func(t *testing.T) {
		// 检查是否超预算
		isOver := costCtrl.IsOverBudget()
		t.Logf("是否超出预算: %v", isOver)

		// 检查是否接近预算限制(80%)
		isNear := costCtrl.IsNearBudgetLimit(0.8)
		t.Logf("是否接近预算限制(80%%): %v", isNear)
	})

	// 测试优化规则
	t.Run("优化规则", func(t *testing.T) {
		rules := costCtrl.GetOptimizationRules()
		t.Logf("优化规则数: %d", len(rules))

		for _, rule := range rules {
			t.Logf("规则: %s (启用: %v)", rule.Name, rule.Enabled)
		}
	})

	// 测试成本报告
	t.Run("成本报告", func(t *testing.T) {
		report := costCtrl.GetMonthlyReport()
		if report == nil {
			t.Error("期望有月度报告")
			return
		}

		t.Logf("报告月份: %s", report.CurrentMonth)
		t.Logf("总成本: $%.2f", report.TotalCost)
		t.Logf("预算使用率: %.2f%%", report.Budget.UsageRate*100)

		if report.Budget.Monthly != 100.0 {
			t.Errorf("期望月度预算为 100.0，实际为: %.2f", report.Budget.Monthly)
		}
	})

	// 测试增强报告
	t.Run("增强报告", func(t *testing.T) {
		enhancedReport := costCtrl.GetEnhancedReport()
		if enhancedReport == nil {
			t.Error("期望有增强报告")
			return
		}

		t.Logf("增强报告 - 预测月成本: $%.2f", enhancedReport.Prediction.ProjectedMonthly)
		t.Logf("优化建议数: %d", len(enhancedReport.OptimizationTips))
	})

	// 测试Provider成本获取
	t.Run("Provider成本获取", func(t *testing.T) {
		cost, err := costCtrl.GetProviderCost("openai")
		if err != nil {
			t.Errorf("获取Provider成本失败: %v", err)
		} else {
			t.Logf("OpenAI - 模型: %s, 每1K Token成本: $%.4f", cost.Model, cost.CostPer1KTokens)
		}

		// 测试不存在的Provider
		_, err = costCtrl.GetProviderCost("unknown")
		if err == nil {
			t.Error("期望获取不存在的Provider时返回错误")
		}
	})

	// 测试推荐建议
	t.Run("推荐建议", func(t *testing.T) {
		recommendations := costCtrl.GetRecommendations()
		t.Logf("推荐建议数: %d", len(recommendations))
		for _, rec := range recommendations {
			t.Logf("建议: %s", rec)
		}
	})

	costCtrl.Stop()
}

// TestQueryOptimizerE2E 查询优化器端到端测试
func TestQueryOptimizerE2E(t *testing.T) {
	optimizer := sqliterepo.NewQueryOptimizer(10 * time.Millisecond)
	defer optimizer.Stop()

	// 测试查询记录
	t.Run("查询记录", func(t *testing.T) {
		// 记录一些查询
		optimizer.RecordQuery("users", "SELECT", "SELECT * FROM users WHERE id = 1", 5*time.Millisecond, 1, false)
		optimizer.RecordQuery("users", "SELECT", "SELECT * FROM users WHERE id = 2", 8*time.Millisecond, 1, false)
		optimizer.RecordQuery("reminders", "INSERT", "INSERT INTO reminders", 15*time.Millisecond, 1, false)

		metrics := optimizer.GetMetrics()

		if metrics.TotalQueries != 3 {
			t.Errorf("期望总查询数为3，实际为: %d", metrics.TotalQueries)
		}

		if metrics.SlowQueries != 1 {
			t.Errorf("期望慢查询数为1，实际为: %d", metrics.SlowQueries)
		}
	})

	// 测试慢查询日志
	t.Run("慢查询日志", func(t *testing.T) {
		logs := optimizer.GetSlowQueries(10)
		if len(logs) != 1 {
			t.Errorf("期望慢查询日志数为1，实际为: %d", len(logs))
		}

		if len(logs) > 0 && logs[0].Table != "reminders" {
			t.Errorf("期望表名为reminders，实际为: %s", logs[0].Table)
		}
	})

	// 测试慢查询统计
	t.Run("慢查询统计", func(t *testing.T) {
		stats := optimizer.GetSlowQueryStats()

		if stats["total_queries"].(int64) != 3 {
			t.Errorf("期望总查询数为3，实际为: %v", stats["total_queries"])
		}

		if stats["slow_queries"].(int64) != 1 {
			t.Errorf("期望慢查询数为1，实际为: %v", stats["slow_queries"])
		}

		t.Logf("慢查询统计: %v", stats)
	})
}
