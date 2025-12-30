package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// C3Integration C3阶段智能降级集成
type C3Integration struct {
	intelligentManager *IntelligentManager
	costController    *CostController
	strategyEngine    *StrategyEngine
	advancedMonitor  *AdvancedMonitor
	logger           *logrus.Logger
	mu               sync.RWMutex
}

// C3Config C3配置
type C3Config struct {
	Strategy   FallbackStrategy        `yaml:"strategy"`
	Budget     BudgetConfig            `yaml:"budget"`
	Providers map[string]*ProviderCost `yaml:"providers"`
	Monitoring MonitoringConfig        `yaml:"monitoring"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	AdvancedEnabled bool `yaml:"advanced_enabled"`
	AlertRules     []AlertRule `yaml:"alert_rules"`
}

// NewC3Integration 创建C3集成
func NewC3Integration(
	providers map[string]AIProviderInterface,
	primary string,
	fallback []string,
	config C3Config,
	logger *logrus.Logger,
) (*C3Integration, error) {

	// 创建基础组件
	intelligentManager := NewIntelligentManager(
		providers,
		primary,
		fallback,
		config.Strategy,
		logger,
	)

	costController := NewCostController(config.Providers, config.Budget, logger)
	strategyEngine := NewStrategyEngine(config.Strategy, logger)
	advancedMonitor := NewAdvancedMonitor(logger)

	// 更新监控告警规则
	if config.Monitoring.AdvancedEnabled {
		advancedMonitor.UpdateAlertRules(config.Monitoring.AlertRules)
	}

	return &C3Integration{
		intelligentManager: intelligentManager,
		costController:    costController,
		strategyEngine:    strategyEngine,
		advancedMonitor:  advancedMonitor,
		logger:           logger,
	}, nil
}

// ParseWithIntelligentC3 使用C3智能解析
func (c *C3Integration) ParseWithIntelligentC3(ctx context.Context, text string) (*ProviderParseResult, error) {
	start := time.Now()

	// 1. 记录决策开始
	defer func() {
		c.advancedMonitor.RecordDecisionLatency("intelligent_c3", time.Since(start))
	}()

	// 2. 使用智能选择进行解析
	result, err := c.intelligentManager.ParseWithIntelligentSelection(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("intelligent parsing failed: %w", err)
	}

	// 3. 记录成本使用
	if result != nil && result.TokensUsed > 0 {
		// 假设ParsedBy为provider名称，实际使用时需要设置
		if costErr := c.costController.RecordUsage("intelligent", result.TokensUsed); costErr != nil {
			c.logger.WithError(costErr).Warn("Failed to record cost usage")
		}
	}

	return result, nil
}

// ChatWithIntelligentC3 使用C3智能聊天
func (c *C3Integration) ChatWithIntelligentC3(ctx context.Context, text string) (string, error) {
	// 简化版，使用第一个可用Provider
	providers := c.intelligentManager.providers
	for name, provider := range providers {
		result, err := provider.Chat(ctx, text)
		if err == nil && result != "" {
			c.logger.WithField("provider", name).Info("Chat successful")
			return result, nil
		}
	}

	return "", fmt.Errorf("all providers failed for chat")
}

// ShouldFallback 智能降级决策
func (c *C3Integration) ShouldFallback(
	ctx context.Context,
	providerStats map[string]*ProviderStats,
) *FallbackDecision {
	return c.strategyEngine.ShouldFallback(ctx, providerStats, c.costController)
}

// UpdateProviderStats 更新Provider统计
func (c *C3Integration) UpdateProviderStats(
	providerName string,
	responseTime time.Duration,
	success bool,
	tokensUsed int,
) {
	c.intelligentManager.recordRequest(providerName)
	if success {
		c.intelligentManager.recordSuccess(providerName, &ProviderParseResult{
			TokensUsed: tokensUsed,
		})
	} else {
		c.intelligentManager.recordError(providerName)
	}

	// 更新监控
	c.advancedMonitor.UpdateProviderStats(providerName, responseTime, success, tokensUsed)
}

// GetC3Stats 获取C3统计信息
func (c *C3Integration) GetC3Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"intelligent_stats": c.intelligentManager.GetIntelligentStats(),
		"cost_report":     c.costController.GetMonthlyReport(),
		"provider_trends": c.advancedMonitor.GetProviderTrends(),
		"strategy_config": c.strategyEngine.GetStrategyConfig(),
	}

	// 添加决策统计
	decisionLatency := c.advancedMonitor.GetDecisionLatency("total")
	stats["decision_latency"] = decisionLatency

	return stats
}

// SetMonthlyCost 设置月成本
func (c *C3Integration) SetMonthlyCost(providerName string, cost float64) {
	c.costController.SetMonthlyCost(providerName, cost)
}

// GetCostRecommendations 获取成本优化建议
func (c *C3Integration) GetCostRecommendations() []string {
	return c.costController.GetRecommendations()
}

// IsOverBudget 检查是否超预算
func (c *C3Integration) IsOverBudget() bool {
	return c.costController.IsOverBudget()
}

// UpdateStrategy 更新策略配置
func (c *C3Integration) UpdateStrategy(newStrategy FallbackStrategy) {
	c.strategyEngine.UpdateStrategy(newStrategy)
	c.intelligentManager.mu.Lock()
	c.intelligentManager.strategy = newStrategy
	c.intelligentManager.mu.Unlock()

	c.logger.Info("C3 fallback strategy updated")
}

// GetProviders 获取Provider列表
func (c *C3Integration) GetProviders() map[string]AIProviderInterface {
	return c.intelligentManager.providers
}

// CheckHealth 检查所有Provider健康状态
func (c *C3Integration) CheckHealth(ctx context.Context) map[string]error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := make(map[string]error)
	for name, provider := range c.intelligentManager.providers {
		results[name] = provider.HealthCheck(ctx)
	}

	return results
}

// TriggerAlerts 手动触发告警检查
func (c *C3Integration) TriggerAlerts() {
	c.advancedMonitor.CheckAlerts()
}

// GetMonitoringStatus 获取监控状态
func (c *C3Integration) GetMonitoringStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"alert_rules": c.advancedMonitor.GetAlertRules(),
		"provider_trends": c.advancedMonitor.GetProviderTrends(),
		"intelligent_stats": c.intelligentManager.GetIntelligentStats(),
	}
}