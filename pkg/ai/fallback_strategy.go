package ai

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

// FallbackStrategy 降级策略配置
type FallbackStrategy struct {
	MaxResponseTime    time.Duration `yaml:"max_response_time"`      // 最大响应时间
	ErrorRateThreshold float64     `yaml:"error_rate_threshold"` // 错误率阈值
	CostWeight         float64     `yaml:"cost_weight"`          // 成本权重
	PerformanceWeight  float64     `yaml:"performance_weight"`  // 性能权重
	ReliabilityWeight  float64     `yaml:"reliability_weight"`   // 可靠性权重
}

// FallbackCondition 降级条件
type FallbackCondition struct {
	ResponseTime   time.Duration `json:"response_time"`
	ErrorRate      float64       `json:"error_rate"`
	CostExceeded  bool           `json:"cost_exceeded"`
	ProviderDown   bool           `json:"provider_down"`
}

// FallbackDecision 降级决策
type FallbackDecision struct {
	ShouldFallback    bool                `json:"should_fallback"`
	Reason           string               `json:"reason"`
	PreferredProvider string              `json:"preferred_provider,omitempty"`
	Conditions       FallbackCondition   `json:"conditions"`
}

// StrategyEngine 策略引擎
type StrategyEngine struct {
	strategy FallbackStrategy
	logger   *logrus.Logger
}

// NewStrategyEngine 创建策略引擎
func NewStrategyEngine(strategy FallbackStrategy, logger *logrus.Logger) *StrategyEngine {
	return &StrategyEngine{
		strategy: strategy,
		logger:   logger,
	}
}

// ShouldFallback 判断是否应该降级
func (s *StrategyEngine) ShouldFallback(
	ctx context.Context,
	providerStats map[string]*ProviderStats,
	costController *CostController,
) *FallbackDecision {
	decision := &FallbackDecision{}

	// 检查各Provider状态
	for name, stats := range providerStats {
		conditions := s.evaluateConditions(stats, name, costController)

		// 如果任何一个条件触发降级
		if s.shouldTriggerFallback(conditions) {
			decision.ShouldFallback = true
			decision.Conditions = conditions
			decision.Reason = s.generateReason(conditions)
			decision.PreferredProvider = s.selectPreferredProvider(providerStats, conditions)

			s.logger.WithFields(logrus.Fields{
				"provider":  name,
				"decision": decision,
			}).Info("Fallback triggered by strategy")

			return decision
		}
	}

	// 所有条件都满足，不需要降级
	decision.ShouldFallback = false
	decision.Reason = "All providers healthy within acceptable limits"

	return decision
}

// evaluateConditions 评估降级条件
func (s *StrategyEngine) evaluateConditions(stats *ProviderStats, providerName string, costController *CostController) FallbackCondition {
	conditions := FallbackCondition{}

	// 响应时间检查
	if stats.AvgResponseTime > s.strategy.MaxResponseTime {
		conditions.ResponseTime = stats.AvgResponseTime
	}

	// 错误率检查
	if stats.TotalRequests > 10 { // 至少要有足够样本
		errorRate := float64(stats.ErrorCount) / float64(stats.TotalRequests)
		if errorRate > s.strategy.ErrorRateThreshold {
			conditions.ErrorRate = errorRate
		}
	}

	// 成本检查
	if costController != nil && costController.IsOverBudget() {
		conditions.CostExceeded = true
	}

	// Provider可用性检查
	if stats.ErrorCount > 0 && float64(stats.ErrorCount)/float64(stats.TotalRequests) > 0.5 {
		conditions.ProviderDown = true
	}

	return conditions
}

// shouldTriggerFallback 判断是否应该触发降级
func (s *StrategyEngine) shouldTriggerFallback(conditions FallbackCondition) bool {
	// 响应时间过长
	if conditions.ResponseTime > 0 {
		return true
	}

	// 错误率过高
	if conditions.ErrorRate > s.strategy.ErrorRateThreshold {
		return true
	}

	// 成本超限
	if conditions.CostExceeded {
		return true
	}

	// Provider不可用
	if conditions.ProviderDown {
		return true
	}

	return false
}

// generateReason 生成降级原因
func (s *StrategyEngine) generateReason(conditions FallbackCondition) string {
	var reasons []string

	if conditions.ResponseTime > 0 {
		reasons = append(reasons, fmt.Sprintf("Response time exceeded: %v", conditions.ResponseTime))
	}

	if conditions.ErrorRate > s.strategy.ErrorRateThreshold {
		reasons = append(reasons, fmt.Sprintf("Error rate exceeded: %.2f%%", conditions.ErrorRate*100))
	}

	if conditions.CostExceeded {
		reasons = append(reasons, "Cost budget exceeded")
	}

	if conditions.ProviderDown {
		reasons = append(reasons, "Provider health issues detected")
	}

	if len(reasons) == 0 {
		return "No specific condition triggered"
	}

	return fmt.Sprintf("Fallback due to: %s", reasons[0])
}

// selectPreferredProvider 选择优选的备选Provider
func (s *StrategyEngine) selectPreferredProvider(providerStats map[string]*ProviderStats, currentConditions FallbackCondition) string {
	var candidates []ProviderCandidate

	for name, stats := range providerStats {
		// 排除当前有问题的Provider
		if !s.isProviderHealthy(stats) {
			continue
		}

		// 计算综合评分
		score := s.calculateProviderScore(stats)
		candidates = append(candidates, ProviderCandidate{
			Name:  name,
			Score: score,
			Stats: stats,
		})
	}

	if len(candidates) == 0 {
		return ""
	}

	// 选择评分最高的Provider
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates[0].Name
}

// isProviderHealthy 检查Provider是否健康
func (s *StrategyEngine) isProviderHealthy(stats *ProviderStats) bool {
	if stats.TotalRequests < 5 {
		return true // 样本太少，默认健康
	}

	errorRate := float64(stats.ErrorCount) / float64(stats.TotalRequests)
	return errorRate < 0.2 // 错误率低于20%
}

// calculateProviderScore 计算Provider评分
func (s *StrategyEngine) calculateProviderScore(stats *ProviderStats) float64 {
	if stats.TotalRequests == 0 {
		return 0
	}

	// 性能评分 (响应时间越短评分越高)
	performanceScore := 100.0
	if stats.AvgResponseTime > 0 {
		performanceScore = math.Max(0, 100.0-stats.AvgResponseTime.Seconds()*10)
	}

	// 可靠性评分 (基于成功率)
	reliabilityScore := float64(stats.SuccessCount) / float64(stats.TotalRequests) * 100

	// 综合评分
	totalScore := performanceScore*s.strategy.PerformanceWeight +
		reliabilityScore*s.strategy.ReliabilityWeight

	s.logger.WithFields(logrus.Fields{
		"performance_score": performanceScore,
		"reliability_score": reliabilityScore,
		"total_score":        totalScore,
	}).Debug("Provider score calculated")

	return totalScore
}

// ProviderScore Provider候选者
type ProviderCandidate struct {
	Name  string
	Score float64
	Stats *ProviderStats
}

// GetStrategyConfig 获取策略配置
func (s *StrategyEngine) GetStrategyConfig() FallbackStrategy {
	return s.strategy
}

// UpdateStrategy 更新策略配置
func (s *StrategyEngine) UpdateStrategy(newStrategy FallbackStrategy) {
	s.strategy = newStrategy
	s.logger.WithFields(logrus.Fields{
		"max_response_time":     newStrategy.MaxResponseTime,
		"error_rate_threshold": newStrategy.ErrorRateThreshold,
		"cost_weight":           newStrategy.CostWeight,
		"performance_weight":    newStrategy.PerformanceWeight,
		"reliability_weight":   newStrategy.ReliabilityWeight,
	}).Info("Fallback strategy updated")
}