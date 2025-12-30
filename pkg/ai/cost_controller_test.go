package ai

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCostController 测试创建成本控制器
func TestNewCostController(t *testing.T) {
	t.Run("创建成功", func(t *testing.T) {
		providers := map[string]*ProviderCost{
			"openai": {
				CostPer1KTokens: 0.002,
				Model:           "gpt-3.5-turbo",
				Provider:        "openai",
			},
			"claude": {
				CostPer1KTokens: 0.001,
				Model:           "claude-3-haiku",
				Provider:        "anthropic",
			},
		}

		budget := BudgetConfig{
			MonthlyBudget:   100.0,
			AlertThreshold: 0.8,
			ResetDay:       1,
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		controller := NewCostController(providers, budget, logger)

		assert.NotNil(t, controller)
		assert.Equal(t, 2, len(controller.providers))
		assert.Equal(t, 100.0, controller.budget.MonthlyBudget)
		assert.Equal(t, 0.8, controller.alertThreshold)
	})

	t.Run("空Provider列表", func(t *testing.T) {
		providers := map[string]*ProviderCost{}
		budget := BudgetConfig{
			MonthlyBudget:   100.0,
			AlertThreshold: 0.8,
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		controller := NewCostController(providers, budget, logger)

		assert.NotNil(t, controller)
		assert.Empty(t, controller.providers)
	})
}

// TestCostController_RecordUsage 测试记录使用量
func TestCostController_RecordUsage(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   100.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("记录成功", func(t *testing.T) {
		err := controller.RecordUsage("openai", 1000)
		assert.NoError(t, err)
	})

	t.Run("记录多次", func(t *testing.T) {
		err := controller.RecordUsage("openai", 1000)
		require.NoError(t, err)

		err = controller.RecordUsage("openai", 2000)
		assert.NoError(t, err)
	})

	t.Run("Provider不存在", func(t *testing.T) {
		err := controller.RecordUsage("unknown", 1000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("计算成本正确", func(t *testing.T) {
		controller.ResetMonthlyStats()
		err := controller.RecordUsage("openai", 1000)
		require.NoError(t, err)

		// 1000 tokens * 0.002 per 1K tokens = 0.002
		report := controller.GetMonthlyReport()
		assert.InDelta(t, 0.002, report.TotalCost, 0.001)
	})
}

// TestCostController_checkBudget 测试预算检查
func TestCostController_checkBudget(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   10.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("未超过阈值", func(t *testing.T) {
		controller.ResetMonthlyStats()
		err := controller.RecordUsage("openai", 1000)
		require.NoError(t, err)

		// 成本很低，不会触发告警
		// 这个测试主要是确保不会panic
	})

	t.Run("超过阈值", func(t *testing.T) {
		controller.ResetMonthlyStats()
		// 设置超过80%阈值的成本
		controller.SetMonthlyCost("openai", 9.0) // 90%

		// 重新记录使用量会触发预算检查
		err := controller.RecordUsage("openai", 100)
		assert.NoError(t, err)
	})

	t.Run("超过预算", func(t *testing.T) {
		controller.ResetMonthlyStats()
		// 设置超过预算的成本
		controller.SetMonthlyCost("openai", 11.0) // 110%

		err := controller.RecordUsage("openai", 100)
		assert.NoError(t, err)
	})
}

// TestCostController_GetMonthlyReport 测试获取月度报告
func TestCostController_GetMonthlyReport(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
		"claude": {
			CostPer1KTokens: 0.001,
			Model:           "claude-3-haiku",
			Provider:        "anthropic",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   100.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("空报告", func(t *testing.T) {
		controller.ResetMonthlyStats()
		report := controller.GetMonthlyReport()

		assert.NotNil(t, report)
		assert.Equal(t, time.Now().Format("2006-01"), report.CurrentMonth)
		assert.Equal(t, 0.0, report.TotalCost)
		assert.Empty(t, report.Providers)
		assert.Equal(t, 100.0, report.Budget.Monthly)
		assert.Equal(t, 100.0, report.Budget.Remaining)
	})

	t.Run("有使用记录的报告", func(t *testing.T) {
		controller.ResetMonthlyStats()
		err := controller.RecordUsage("openai", 1000)
		require.NoError(t, err)

		report := controller.GetMonthlyReport()

		assert.NotNil(t, report)
		assert.Greater(t, report.TotalCost, 0.0)
		assert.NotEmpty(t, report.Providers)
		assert.Contains(t, report.Providers, "openai")
		assert.Equal(t, "gpt-3.5-turbo", report.Providers["openai"].Model)
	})

	t.Run("多个Provider的报告", func(t *testing.T) {
		controller.ResetMonthlyStats()
		err := controller.RecordUsage("openai", 1000)
		require.NoError(t, err)

		err = controller.RecordUsage("claude", 2000)
		require.NoError(t, err)

		report := controller.GetMonthlyReport()

		assert.Equal(t, 2, len(report.Providers))
		assert.Contains(t, report.Providers, "openai")
		assert.Contains(t, report.Providers, "claude")
	})

	t.Run("预算使用率计算", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 50.0) // 50%

		report := controller.GetMonthlyReport()

		assert.Equal(t, 0.5, report.Budget.UsageRate)
		assert.Equal(t, 50.0, report.Budget.Remaining)
		assert.False(t, report.Budget.IsExceeded)
	})

	t.Run("预算超限", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 110.0) // 110%

		report := controller.GetMonthlyReport()

		assert.Equal(t, 1.1, report.Budget.UsageRate)
		assert.Equal(t, -10.0, report.Budget.Remaining)
		assert.True(t, report.Budget.IsExceeded)
	})
}

// TestCostController_GetProviderCost 测试获取Provider成本
func TestCostController_GetProviderCost(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   100.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("获取存在的Provider", func(t *testing.T) {
		cost, err := controller.GetProviderCost("openai")
		assert.NoError(t, err)
		assert.NotNil(t, cost)
		assert.Equal(t, "openai", cost.Provider)
		assert.Equal(t, "gpt-3.5-turbo", cost.Model)
		assert.Equal(t, 0.002, cost.CostPer1KTokens)
	})

	t.Run("获取不存在的Provider", func(t *testing.T) {
		cost, err := controller.GetProviderCost("unknown")
		assert.Error(t, err)
		assert.Nil(t, cost)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestCostController_ResetMonthlyStats 测试重置月度统计
func TestCostController_ResetMonthlyStats(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   100.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("重置成功", func(t *testing.T) {
		err := controller.RecordUsage("openai", 1000)
		require.NoError(t, err)

		report := controller.GetMonthlyReport()
		assert.Greater(t, report.TotalCost, 0.0)

		controller.ResetMonthlyStats()

		report = controller.GetMonthlyReport()
		assert.Equal(t, 0.0, report.TotalCost)
		assert.Empty(t, report.Providers)
	})
}

// TestCostController_IsOverBudget 测试检查是否超预算
func TestCostController_IsOverBudget(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   100.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("未超预算", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 50.0)

		isOver := controller.IsOverBudget()
		assert.False(t, isOver)
	})

	t.Run("刚好达到预算", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 100.0)

		isOver := controller.IsOverBudget()
		assert.False(t, isOver)
	})

	t.Run("超过预算", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 101.0)

		isOver := controller.IsOverBudget()
		assert.True(t, isOver)
	})
}

// TestCostController_GetRecommendations 测试获取优化建议
func TestCostController_GetRecommendations(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   100.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("低使用率 - 无建议", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 10.0) // 10%

		recommendations := controller.GetRecommendations()
		assert.Empty(t, recommendations)
	})

	t.Run("高使用率 - 有建议", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 95.0) // 95%

		recommendations := controller.GetRecommendations()
		assert.NotEmpty(t, recommendations)
		assert.Contains(t, recommendations[0], "成本使用率过高")
	})

	t.Run("接近预算 - 有建议", func(t *testing.T) {
		controller.ResetMonthlyStats()
		controller.SetMonthlyCost("openai", 85.0) // 85%

		recommendations := controller.GetRecommendations()
		assert.NotEmpty(t, recommendations)
		assert.Contains(t, recommendations[0], "接近预算上限")
	})
}

// TestCostController_SetMonthlyCost 测试设置月成本
func TestCostController_SetMonthlyCost(t *testing.T) {
	providers := map[string]*ProviderCost{
		"openai": {
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		},
	}

	budget := BudgetConfig{
		MonthlyBudget:   100.0,
		AlertThreshold: 0.8,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	controller := NewCostController(providers, budget, logger)

	t.Run("设置成功", func(t *testing.T) {
		controller.SetMonthlyCost("openai", 50.0)

		report := controller.GetMonthlyReport()
		assert.Equal(t, 50.0, report.TotalCost)
	})

	t.Run("设置不存在的Provider", func(t *testing.T) {
		// 不应该panic，只是不设置
		controller.SetMonthlyCost("unknown", 50.0)

		_ = controller.GetMonthlyReport()
		// 应该还是之前的值
	})
}

// TestProviderCost 测试ProviderCost结构
func TestProviderCost(t *testing.T) {
	t.Run("创建完整成本信息", func(t *testing.T) {
		cost := &ProviderCost{
			CostPer1KTokens: 0.002,
			Model:           "gpt-3.5-turbo",
			Provider:        "openai",
		}

		assert.Equal(t, 0.002, cost.CostPer1KTokens)
		assert.Equal(t, "gpt-3.5-turbo", cost.Model)
		assert.Equal(t, "openai", cost.Provider)
	})
}

// TestBudgetConfig 测试BudgetConfig结构
func TestBudgetConfig(t *testing.T) {
	t.Run("创建完整预算配置", func(t *testing.T) {
		config := BudgetConfig{
			MonthlyBudget:   100.0,
			AlertThreshold: 0.8,
			ResetDay:       1,
		}

		assert.Equal(t, 100.0, config.MonthlyBudget)
		assert.Equal(t, 0.8, config.AlertThreshold)
		assert.Equal(t, 1, config.ResetDay)
	})
}

// TestCostReport 测试CostReport结构
func TestCostReport(t *testing.T) {
	t.Run("创建完整成本报告", func(t *testing.T) {
		report := &CostReport{
			CurrentMonth: "2025-01",
			TotalCost:    50.0,
			Providers: map[string]*ProviderReport{
				"openai": {
					Name:    "openai",
					Model:   "gpt-3.5-turbo",
					Usage:   1000,
					Cost:    2.0,
					AvgCost: 0.002,
				},
			},
			Budget: BudgetReport{
				Monthly:   100.0,
				Used:      50.0,
				Remaining: 50.0,
				UsageRate: 0.5,
				IsExceeded: false,
			},
		}

		assert.Equal(t, "2025-01", report.CurrentMonth)
		assert.Equal(t, 50.0, report.TotalCost)
		assert.Equal(t, 1, len(report.Providers))
		assert.Equal(t, 0.5, report.Budget.UsageRate)
	})
}

// TestProviderReport 测试ProviderReport结构
func TestProviderReport(t *testing.T) {
	t.Run("创建Provider报告", func(t *testing.T) {
		report := &ProviderReport{
			Name:    "openai",
			Model:   "gpt-3.5-turbo",
			Usage:   1000,
			Cost:    2.0,
			AvgCost: 0.002,
		}

		assert.Equal(t, "openai", report.Name)
		assert.Equal(t, "gpt-3.5-turbo", report.Model)
		assert.Equal(t, int64(1000), report.Usage)
		assert.Equal(t, 2.0, report.Cost)
	})
}

// TestBudgetReport 测试BudgetReport结构
func TestBudgetReport(t *testing.T) {
	t.Run("创建预算报告", func(t *testing.T) {
		report := BudgetReport{
			Monthly:   100.0,
			Used:      50.0,
			Remaining: 50.0,
			UsageRate: 0.5,
			IsExceeded: false,
		}

		assert.Equal(t, 100.0, report.Monthly)
		assert.Equal(t, 50.0, report.Used)
		assert.Equal(t, 50.0, report.Remaining)
		assert.Equal(t, 0.5, report.UsageRate)
		assert.False(t, report.IsExceeded)
	})
}

// TestCostTrend 测试CostTrend结构
func TestCostTrend(t *testing.T) {
	t.Run("创建成本趋势", func(t *testing.T) {
		trend := CostTrend{
			Date: "2025-01-01",
			Cost: 10.0,
		}

		assert.Equal(t, "2025-01-01", trend.Date)
		assert.Equal(t, 10.0, trend.Cost)
	})
}