package service

import (
	"context"
	"sync"
	"time"

	"mmemory/pkg/ai"
	"mmemory/pkg/metrics"

	"github.com/sirupsen/logrus"
)

// CostMonitor 成本监控服务
type CostMonitor struct {
	costController *ai.CostController
	logger         *logrus.Logger
	stopCh         chan struct{}
	wg             sync.WaitGroup
	interval       time.Duration
}

// NewCostMonitor 创建成本监控服务
func NewCostMonitor(costController *ai.CostController, interval time.Duration) *CostMonitor {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &CostMonitor{
		costController: costController,
		logger:         logrus.New(),
		stopCh:         make(chan struct{}),
		interval:       interval,
	}
}

// Start 启动监控
func (cm *CostMonitor) Start() {
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		ticker := time.NewTicker(cm.interval)
		defer ticker.Stop()

		// 立即执行一次
		cm.updateMetrics()

		for {
			select {
			case <-ticker.C:
				cm.updateMetrics()
			case <-cm.stopCh:
				return
			}
		}
	}()
	cm.logger.Info("Cost monitor started")
}

// Stop 停止监控
func (cm *CostMonitor) Stop() {
	close(cm.stopCh)
	cm.wg.Wait()
	cm.logger.Info("Cost monitor stopped")
}

// updateMetrics 更新 Prometheus 指标
func (cm *CostMonitor) updateMetrics() {
	if cm.costController == nil {
		return
	}

	// 获取成本报告
	report := cm.costController.GetEnhancedReport()
	if report == nil {
		return
	}

	// 更新预算利用率指标
	budgetUtilization := report.TotalCost / report.Budget.Monthly
	if report.Budget.Monthly > 0 {
		metrics.SetAIBudgetUtilization("daily", budgetUtilization*30) // 估算日预算利用率
		metrics.SetAIBudgetUtilization("monthly", budgetUtilization)
	}

	// 更新成本预测指标
	if report.Prediction != nil {
		metrics.SetAICostPrediction("next_day", "all", report.Prediction.NextDayCost)
		metrics.SetAICostPrediction("next_week", "all", report.Prediction.NextWeekCost)
		metrics.SetAICostPrediction("next_month", "all", report.Prediction.NextMonthCost)

		// 更新趋势指标
		trendValue := 0.0
		switch report.Prediction.Trend {
		case "increasing":
			trendValue = 1.0
		case "decreasing":
			trendValue = -1.0
		}
		metrics.SetAICostTrend(trendValue)

		// 更新风险等级指标
		riskValue := 0.0
		switch report.Prediction.RiskLevel {
		case "low":
			riskValue = 0.2
		case "medium":
			riskValue = 0.5
		case "high":
			riskValue = 1.0
		}
		metrics.SetAICostRiskLevel(riskValue)
	}

	cm.logger.WithFields(logrus.Fields{
		"total_cost":   report.TotalCost,
		"budget":       report.Budget.Monthly,
		"prediction":   report.Prediction,
	}).Debug("Cost metrics updated")
}

// GetCostReport 获取成本报告
func (cm *CostMonitor) GetCostReport(ctx context.Context) *ai.CostReport {
	if cm.costController == nil {
		return nil
	}
	return cm.costController.GetEnhancedReport()
}

// GetOptimizationTips 获取优化建议
func (cm *CostMonitor) GetOptimizationTips(ctx context.Context) []string {
	if cm.costController == nil {
		return nil
	}
	return cm.costController.GetRecommendations()
}

// RunOptimizationRules 运行优化规则
func (cm *CostMonitor) RunOptimizationRules(ctx context.Context) []string {
	if cm.costController == nil {
		return nil
	}
	return cm.costController.RunOptimizationRules()
}

// CheckAlerts 检查告警
func (cm *CostMonitor) CheckAlerts(ctx context.Context) []*ai.CostAlert {
	if cm.costController == nil {
		return nil
	}
	return cm.costController.CheckAlerts()
}
