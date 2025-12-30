package ai

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestNewStrategyEngine 测试创建策略引擎
func TestNewStrategyEngine(t *testing.T) {
	t.Run("创建成功", func(t *testing.T) {
		strategy := FallbackStrategy{
			MaxResponseTime:    5 * time.Second,
			ErrorRateThreshold: 0.1,
			CostWeight:         0.3,
			PerformanceWeight:  0.4,
			ReliabilityWeight:  0.3,
		}

		logger := logrus.New()
		engine := NewStrategyEngine(strategy, logger)

		assert.NotNil(t, engine)
		assert.Equal(t, strategy.MaxResponseTime, engine.strategy.MaxResponseTime)
		assert.Equal(t, strategy.ErrorRateThreshold, engine.strategy.ErrorRateThreshold)
	})

	t.Run("使用默认值", func(t *testing.T) {
		strategy := FallbackStrategy{}
		logger := logrus.New()
		engine := NewStrategyEngine(strategy, logger)

		assert.NotNil(t, engine)
		assert.Equal(t, 0*time.Second, engine.strategy.MaxResponseTime)
		assert.Equal(t, 0.0, engine.strategy.ErrorRateThreshold)
	})
}

// TestFallbackStrategy_ShouldFallback 测试降级判断
func TestFallbackStrategy_ShouldFallback(t *testing.T) {
	strategy := FallbackStrategy{
		MaxResponseTime:    5 * time.Second,
		ErrorRateThreshold: 0.1,
		CostWeight:         0.3,
		PerformanceWeight:  0.4,
		ReliabilityWeight:  0.3,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // 减少日志输出
	engine := NewStrategyEngine(strategy, logger)

	t.Run("不需要降级 - 所有Provider健康", func(t *testing.T) {
		ctx := context.Background()
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    95,
				ErrorCount:      5,
				AvgResponseTime: 2 * time.Second,
			},
		}

		decision := engine.ShouldFallback(ctx, providerStats, nil)

		assert.False(t, decision.ShouldFallback)
		assert.Equal(t, "All providers healthy within acceptable limits", decision.Reason)
	})

	t.Run("需要降级 - 响应时间过长", func(t *testing.T) {
		ctx := context.Background()
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    95,
				ErrorCount:      5,
				AvgResponseTime: 10 * time.Second, // 超过5秒阈值
			},
		}

		decision := engine.ShouldFallback(ctx, providerStats, nil)

		assert.True(t, decision.ShouldFallback)
		assert.Contains(t, decision.Reason, "Response time exceeded")
	})

	t.Run("需要降级 - 错误率过高", func(t *testing.T) {
		ctx := context.Background()
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    80,
				ErrorCount:      20, // 20%错误率，超过10%阈值
				AvgResponseTime: 2 * time.Second,
			},
		}

		decision := engine.ShouldFallback(ctx, providerStats, nil)

		assert.True(t, decision.ShouldFallback)
		assert.Contains(t, decision.Reason, "Error rate exceeded")
	})

	t.Run("需要降级 - 成本超限", func(t *testing.T) {
		ctx := context.Background()
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    95,
				ErrorCount:      5,
				AvgResponseTime: 2 * time.Second,
			},
		}

		// 创建一个预算很小的成本控制器
		providers := map[string]*ProviderCost{
			"openai": {
				CostPer1KTokens: 0.002,
				Model:           "gpt-3.5-turbo",
				Provider:        "openai",
			},
		}
		budget := BudgetConfig{
			MonthlyBudget:   1.0, // 很小的预算
			AlertThreshold: 0.8,
		}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		costController := NewCostController(providers, budget, logger)
		// 设置超过预算的成本
		costController.SetMonthlyCost("openai", 2.0) // 超过1.0的预算

		decision := engine.ShouldFallback(ctx, providerStats, costController)

		assert.True(t, decision.ShouldFallback)
		assert.Contains(t, decision.Reason, "Cost budget exceeded")
	})

	t.Run("需要降级 - Provider不可用", func(t *testing.T) {
		ctx := context.Background()
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    40,
				ErrorCount:      60, // 60%错误率，超过50%
				AvgResponseTime: 2 * time.Second,
			},
		}

		decision := engine.ShouldFallback(ctx, providerStats, nil)

		assert.True(t, decision.ShouldFallback)
		// Provider不可用时，会触发降级，但原因可能是错误率
		assert.NotEmpty(t, decision.Reason)
	})

	t.Run("样本不足 - 不触发降级", func(t *testing.T) {
		ctx := context.Background()
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   5, // 样本不足
				SuccessCount:    4,
				ErrorCount:      1, // 错误率20%，不会触发降级
				AvgResponseTime: 2 * time.Second,
			},
		}

		decision := engine.ShouldFallback(ctx, providerStats, nil)

		// 样本不足时，不会触发降级
		assert.False(t, decision.ShouldFallback)
	})
}

// TestFallbackStrategy_evaluateConditions 测试条件评估
func TestFallbackStrategy_evaluateConditions(t *testing.T) {
	strategy := FallbackStrategy{
		MaxResponseTime:    5 * time.Second,
		ErrorRateThreshold: 0.1,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	engine := NewStrategyEngine(strategy, logger)

	t.Run("所有条件正常", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   100,
			SuccessCount:    95,
			ErrorCount:      5,
			AvgResponseTime: 2 * time.Second,
		}

		conditions := engine.evaluateConditions(stats, "openai", nil)

		assert.Zero(t, conditions.ResponseTime)
		assert.Zero(t, conditions.ErrorRate)
		assert.False(t, conditions.CostExceeded)
		assert.False(t, conditions.ProviderDown)
	})

	t.Run("响应时间过长", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   100,
			SuccessCount:    95,
			ErrorCount:      5,
			AvgResponseTime: 10 * time.Second,
		}

		conditions := engine.evaluateConditions(stats, "openai", nil)

		assert.Equal(t, 10*time.Second, conditions.ResponseTime)
	})

	t.Run("错误率过高", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   100,
			SuccessCount:    80,
			ErrorCount:      20,
			AvgResponseTime: 2 * time.Second,
		}

		conditions := engine.evaluateConditions(stats, "openai", nil)

		assert.Equal(t, 0.2, conditions.ErrorRate)
	})
}

// TestFallbackStrategy_shouldTriggerFallback 测试降级触发
func TestFallbackStrategy_shouldTriggerFallback(t *testing.T) {
	strategy := FallbackStrategy{
		MaxResponseTime:    5 * time.Second,
		ErrorRateThreshold: 0.1,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	engine := NewStrategyEngine(strategy, logger)

	t.Run("响应时间触发", func(t *testing.T) {
		conditions := FallbackCondition{
			ResponseTime: 10 * time.Second,
		}

		shouldFallback := engine.shouldTriggerFallback(conditions)
		assert.True(t, shouldFallback)
	})

	t.Run("错误率触发", func(t *testing.T) {
		conditions := FallbackCondition{
			ErrorRate: 0.2,
		}

		shouldFallback := engine.shouldTriggerFallback(conditions)
		assert.True(t, shouldFallback)
	})

	t.Run("成本触发", func(t *testing.T) {
		conditions := FallbackCondition{
			CostExceeded: true,
		}

		shouldFallback := engine.shouldTriggerFallback(conditions)
		assert.True(t, shouldFallback)
	})

	t.Run("Provider不可用触发", func(t *testing.T) {
		conditions := FallbackCondition{
			ProviderDown: true,
		}

		shouldFallback := engine.shouldTriggerFallback(conditions)
		assert.True(t, shouldFallback)
	})

	t.Run("不触发降级", func(t *testing.T) {
		conditions := FallbackCondition{}

		shouldFallback := engine.shouldTriggerFallback(conditions)
		assert.False(t, shouldFallback)
	})
}

// TestFallbackStrategy_generateReason 测试生成原因
func TestFallbackStrategy_generateReason(t *testing.T) {
	strategy := FallbackStrategy{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	engine := NewStrategyEngine(strategy, logger)

	t.Run("响应时间原因", func(t *testing.T) {
		conditions := FallbackCondition{
			ResponseTime: 10 * time.Second,
		}

		reason := engine.generateReason(conditions)
		assert.Contains(t, reason, "Response time exceeded")
	})

	t.Run("错误率原因", func(t *testing.T) {
		conditions := FallbackCondition{
			ErrorRate: 0.2,
		}

		reason := engine.generateReason(conditions)
		assert.Contains(t, reason, "Error rate exceeded")
	})

	t.Run("成本原因", func(t *testing.T) {
		conditions := FallbackCondition{
			CostExceeded: true,
		}

		reason := engine.generateReason(conditions)
		assert.Contains(t, reason, "Cost budget exceeded")
	})

	t.Run("多个原因", func(t *testing.T) {
		conditions := FallbackCondition{
			ResponseTime: 10 * time.Second,
			ErrorRate:    0.2,
		}

		reason := engine.generateReason(conditions)
		assert.NotEmpty(t, reason)
	})
}

// TestFallbackStrategy_selectPreferredProvider 测试选择优选Provider
func TestFallbackStrategy_selectPreferredProvider(t *testing.T) {
	strategy := FallbackStrategy{
		MaxResponseTime:    5 * time.Second,
		ErrorRateThreshold: 0.1,
		PerformanceWeight:  0.5,
		ReliabilityWeight:  0.5,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	engine := NewStrategyEngine(strategy, logger)

	t.Run("选择评分最高的Provider", func(t *testing.T) {
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    90,
				ErrorCount:      10,
				AvgResponseTime: 3 * time.Second,
			},
			"claude": {
				TotalRequests:   100,
				SuccessCount:    95,
				ErrorCount:      5,
				AvgResponseTime: 2 * time.Second, // 更快，更可靠
			},
		}

		currentConditions := FallbackCondition{}

		selected := engine.selectPreferredProvider(providerStats, currentConditions)
		assert.Equal(t, "claude", selected)
	})

	t.Run("只有一个可用Provider", func(t *testing.T) {
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    95,
				ErrorCount:      5,
				AvgResponseTime: 2 * time.Second,
			},
		}

		currentConditions := FallbackCondition{}

		selected := engine.selectPreferredProvider(providerStats, currentConditions)
		assert.Equal(t, "openai", selected)
	})

	t.Run("没有可用Provider", func(t *testing.T) {
		providerStats := map[string]*ProviderStats{
			"openai": {
				TotalRequests:   100,
				SuccessCount:    30,
				ErrorCount:      70, // 错误率过高
				AvgResponseTime: 2 * time.Second,
			},
		}

		currentConditions := FallbackCondition{}

		selected := engine.selectPreferredProvider(providerStats, currentConditions)
		assert.Empty(t, selected)
	})
}

// TestFallbackStrategy_isProviderHealthy 测试Provider健康检查
func TestFallbackStrategy_isProviderHealthy(t *testing.T) {
	strategy := FallbackStrategy{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	engine := NewStrategyEngine(strategy, logger)

	t.Run("Provider健康", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   100,
			SuccessCount:    95,
			ErrorCount:      5,
			AvgResponseTime: 2 * time.Second,
		}

		healthy := engine.isProviderHealthy(stats)
		assert.True(t, healthy)
	})

	t.Run("Provider不健康 - 错误率过高", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   100,
			SuccessCount:    70,
			ErrorCount:      30, // 30%错误率
			AvgResponseTime: 2 * time.Second,
		}

		healthy := engine.isProviderHealthy(stats)
		assert.False(t, healthy)
	})

	t.Run("样本不足 - 默认健康", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   3,
			SuccessCount:    1,
			ErrorCount:      2,
			AvgResponseTime: 2 * time.Second,
		}

		healthy := engine.isProviderHealthy(stats)
		assert.True(t, healthy)
	})
}

// TestFallbackStrategy_calculateProviderScore 测试计算Provider评分
func TestFallbackStrategy_calculateProviderScore(t *testing.T) {
	strategy := FallbackStrategy{
		PerformanceWeight: 0.5,
		ReliabilityWeight: 0.5,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	engine := NewStrategyEngine(strategy, logger)

	t.Run("计算评分", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   100,
			SuccessCount:    95,
			ErrorCount:      5,
			AvgResponseTime: 2 * time.Second,
		}

		score := engine.calculateProviderScore(stats)
		assert.Greater(t, score, 0.0)
		assert.LessOrEqual(t, score, 100.0)
	})

	t.Run("零请求 - 返回0", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   0,
			SuccessCount:    0,
			ErrorCount:      0,
			AvgResponseTime: 0,
		}

		score := engine.calculateProviderScore(stats)
		assert.Equal(t, 0.0, score)
	})

	t.Run("快速响应 - 高性能评分", func(t *testing.T) {
		stats := &ProviderStats{
			TotalRequests:   100,
			SuccessCount:    95,
			ErrorCount:      5,
			AvgResponseTime: 1 * time.Second, // 快速
		}

		score := engine.calculateProviderScore(stats)
		assert.Greater(t, score, 0.0)
	})
}

// TestFallbackStrategy_GetStrategyConfig 测试获取策略配置
func TestFallbackStrategy_GetStrategyConfig(t *testing.T) {
	strategy := FallbackStrategy{
		MaxResponseTime:    5 * time.Second,
		ErrorRateThreshold: 0.1,
		CostWeight:         0.3,
		PerformanceWeight:  0.4,
		ReliabilityWeight:  0.3,
	}

	logger := logrus.New()
	engine := NewStrategyEngine(strategy, logger)

	config := engine.GetStrategyConfig()
	assert.Equal(t, strategy.MaxResponseTime, config.MaxResponseTime)
	assert.Equal(t, strategy.ErrorRateThreshold, config.ErrorRateThreshold)
	assert.Equal(t, strategy.CostWeight, config.CostWeight)
}

// TestFallbackStrategy_UpdateStrategy 测试更新策略
func TestFallbackStrategy_UpdateStrategy(t *testing.T) {
	strategy := FallbackStrategy{
		MaxResponseTime:    5 * time.Second,
		ErrorRateThreshold: 0.1,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	engine := NewStrategyEngine(strategy, logger)

	newStrategy := FallbackStrategy{
		MaxResponseTime:    10 * time.Second,
		ErrorRateThreshold: 0.2,
		CostWeight:         0.4,
		PerformanceWeight:  0.3,
		ReliabilityWeight:  0.3,
	}

	engine.UpdateStrategy(newStrategy)

	config := engine.GetStrategyConfig()
	assert.Equal(t, 10*time.Second, config.MaxResponseTime)
	assert.Equal(t, 0.2, config.ErrorRateThreshold)
	assert.Equal(t, 0.4, config.CostWeight)
}

// TestFallbackCondition 测试FallbackCondition结构
func TestFallbackCondition(t *testing.T) {
	t.Run("创建完整条件", func(t *testing.T) {
		condition := FallbackCondition{
			ResponseTime:   10 * time.Second,
			ErrorRate:      0.2,
			CostExceeded:  true,
			ProviderDown:   false,
		}

		assert.Equal(t, 10*time.Second, condition.ResponseTime)
		assert.Equal(t, 0.2, condition.ErrorRate)
		assert.True(t, condition.CostExceeded)
		assert.False(t, condition.ProviderDown)
	})

	t.Run("创建空条件", func(t *testing.T) {
		condition := FallbackCondition{}

		assert.Zero(t, condition.ResponseTime)
		assert.Zero(t, condition.ErrorRate)
		assert.False(t, condition.CostExceeded)
		assert.False(t, condition.ProviderDown)
	})
}

// TestFallbackDecision 测试FallbackDecision结构
func TestFallbackDecision(t *testing.T) {
	t.Run("创建降级决策", func(t *testing.T) {
		decision := &FallbackDecision{
			ShouldFallback:    true,
			Reason:           "Response time exceeded: 10s",
			PreferredProvider: "claude",
			Conditions: FallbackCondition{
				ResponseTime: 10 * time.Second,
			},
		}

		assert.True(t, decision.ShouldFallback)
		assert.Equal(t, "Response time exceeded: 10s", decision.Reason)
		assert.Equal(t, "claude", decision.PreferredProvider)
		assert.Equal(t, 10*time.Second, decision.Conditions.ResponseTime)
	})
}