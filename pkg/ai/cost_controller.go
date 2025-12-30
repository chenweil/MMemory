package ai

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CostAlert 成本告警
type CostAlert struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"` // critical, warning, info
	Message     string    `json:"message"`
	CurrentCost float64   `json:"current_cost"`
	Budget      float64   `json:"budget"`
	UsageRate   float64   `json:"usage_rate"`
	Provider    string    `json:"provider,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

// CostController 成本控制器
type CostController struct {
	providers          map[string]*ProviderCost
	budget             BudgetConfig
	currentMonthlyCost map[string]float64
	dailyCosts         map[string][]float64        // 每日成本历史（最近30天）
	predictionModel    *CostPredictionModel        // 成本预测模型
	optimizationRules  []CostOptimizationRule      // 优化规则
	alertRules         []CostAlertRule             // 告警规则
	activeAlerts       []*CostAlert                // 活跃告警
	alertChan          chan *CostAlert             // 告警通道
	alertThresholds    map[string]float64          // 告警阈值
	alertThreshold     float64
	mu                 sync.RWMutex
	logger             *logrus.Logger
	stopCh             chan struct{}
	wg                 sync.WaitGroup
}

// ProviderCost Provider成本信息
type ProviderCost struct {
	CostPer1KTokens float64 `yaml:"cost_per_1k_tokens"`
	Model            string  `yaml:"model"`
	Provider        string  `yaml:"provider"`
	Enabled         bool    `yaml:"enabled"`        // 是否启用
	Priority        int     `yaml:"priority"`       // 优先级（数值越小优先级越高）
}

// BudgetConfig 预算配置
type BudgetConfig struct {
	MonthlyBudget    float64 `yaml:"monthly_budget"`
	DailyBudget      float64 `yaml:"daily_budget"`       // 每日预算（可选，默认月预算/30）
	UserBudget       float64 `yaml:"user_budget"`        // 每用户每日预算
	AlertThreshold   float64 `yaml:"alert_threshold"`    // 预算阈值百分比，如0.8表示80%
	WarningThreshold float64 `yaml:"warning_threshold"`  // 警告阈值百分比，如0.6表示60%
	ResetDay         int     `yaml:"reset_day"`          // 每月几号重置预算，默认为1号
}

// CostPredictionModel 成本预测模型
type CostPredictionModel struct {
	TrendAnalysis      float64   // 趋势分析（正=上升，负=下降）
	PredictedNextDay   float64   // 预测次日成本
	PredictedNextWeek  float64   // 预测下周成本
	PredictedNextMonth float64   // 预测下月成本
	Confidence         float64   // 置信度 (0-1)
	LastUpdated        time.Time // 最后更新时间
}

// CostOptimizationRule 成本优化规则
type CostOptimizationRule struct {
	Name        string
	Condition   func(*CostController) bool
	Action      func(*CostController) error
	Priority    int // 优先级（数值越小优先级越高）
	Enabled     bool
	Description string
}

// CostAlertRule 成本告警规则
type CostAlertRule struct {
	Name        string
	Type        string  // budget_warning, budget_critical, cost_anomaly, prediction_exceeded
	Threshold   float64 // 阈值百分比
	Severity    string  // critical, warning, info
	Enabled     bool
	Description string
}

// CostReport 成本报告
type CostReport struct {
	CurrentMonth     string                     `json:"current_month"`
	TotalCost        float64                    `json:"total_cost"`
	Providers        map[string]*ProviderReport `json:"providers"`
	Budget           BudgetReport               `json:"budget"`
	Trend            []CostTrend                `json:"trend"`
	Prediction       *CostPredictionResult      `json:"prediction,omitempty"`        // 成本预测
	OptimizationTips []string                   `json:"optimization_tips,omitempty"` // 优化建议
}

// CostPredictionResult 成本预测结果
type CostPredictionResult struct {
	NextDayCost      float64 `json:"next_day_cost"`
	NextWeekCost     float64 `json:"next_week_cost"`
	NextMonthCost    float64 `json:"next_month_cost"`
	DailyAverage     float64 `json:"daily_average"`
	ProjectedMonthly float64 `json:"projected_monthly"`
	Trend            string  `json:"trend"` // increasing, stable, decreasing
	Confidence       float64 `json:"confidence"`
	RiskLevel        string  `json:"risk_level"` // low, medium, high
}

// ProviderReport Provider成本报告
type ProviderReport struct {
	Name         string  `json:"name"`
	Model        string  `json:"model"`
	Usage        int64   `json:"usage"`         // Token使用量
	Cost         float64 `json:"cost"`          // 当月成本
	AvgCost      float64 `json:"avg_cost"`      // 平均每1K token成本
}

// BudgetReport 预算报告
type BudgetReport struct {
	Monthly      float64 `json:"monthly"`
	Used         float64 `json:"used"`
	Remaining    float64 `json:"remaining"`
	UsageRate    float64 `json:"usage_rate"`    // 使用率百分比
	IsExceeded   bool    `json:"is_exceeded"`
}

// CostTrend 成本趋势
type CostTrend struct {
	Date   string  `json:"date"`
	Cost   float64 `json:"cost"`
}

// NewCostController 创建成本控制器
func NewCostController(providers map[string]*ProviderCost, budget BudgetConfig, logger *logrus.Logger) *CostController {
	// 设置默认值
	if budget.AlertThreshold == 0 {
		budget.AlertThreshold = 0.9
	}
	if budget.WarningThreshold == 0 {
		budget.WarningThreshold = 0.6
	}
	if budget.DailyBudget == 0 {
		budget.DailyBudget = budget.MonthlyBudget / 30
	}

	controller := &CostController{
		providers:          providers,
		budget:             budget,
		currentMonthlyCost: make(map[string]float64),
		dailyCosts:         make(map[string][]float64),
		alertThreshold:     budget.AlertThreshold,
		alertChan:          make(chan *CostAlert, 100),
		alertThresholds: map[string]float64{
			"warning":   budget.WarningThreshold,
			"critical":  budget.AlertThreshold,
			"anomaly":   2.0, // 成本异常阈值（2倍）
		},
		logger: logger,
		stopCh: make(chan struct{}),
	}

	// 初始化成本预测模型
	controller.predictionModel = &CostPredictionModel{
		LastUpdated: time.Now(),
	}

	// 初始化优化规则
	controller.initOptimizationRules()

	// 初始化告警规则
	controller.initAlertRules()

	// 启动每日成本记录器
	controller.startDailyRecorder()

	// 启动告警处理器
	controller.startAlertProcessor()

	return controller
}

// initOptimizationRules 初始化优化规则
func (c *CostController) initOptimizationRules() {
	c.optimizationRules = []CostOptimizationRule{
		{
			Name:        "SwitchToCheaperProvider",
			Priority:    1,
			Enabled:     true,
			Description: "当成本超限时自动切换到更便宜的Provider",
			Condition: func(cc *CostController) bool {
				return cc.IsOverBudget() || cc.IsNearBudgetLimit(0.85)
			},
			Action: func(cc *CostController) error {
				var cheapest *ProviderCost
				var cheapestName string
				for name, p := range cc.providers {
					if p.Enabled && (cheapest == nil || p.CostPer1KTokens < cheapest.CostPer1KTokens) {
						cheapest = p
						cheapestName = name
					}
				}
				if cheapest != nil {
					cc.logger.WithField("provider", cheapestName).Info("Switching to cheaper provider for cost optimization")
				}
				return nil
			},
		},
		{
			Name:        "ReduceRequestFrequency",
			Priority:    2,
			Enabled:     true,
			Description: "当成本接近预算时减少请求频率",
			Condition: func(cc *CostController) bool {
				return cc.IsNearBudgetLimit(0.9)
			},
			Action: func(cc *CostController) error {
				cc.logger.Warn("Cost near budget limit - consider reducing request frequency")
				return nil
			},
		},
		{
			Name:        "EnableCacheForRepeatedRequests",
			Priority:    3,
			Enabled:     true,
			Description: "对重复请求启用缓存以降低成本",
			Condition: func(cc *CostController) bool {
				return true
			},
			Action: func(cc *CostController) error {
				cc.logger.Info("Recommendation: Enable response caching for repeated similar requests")
				return nil
			},
		},
	}
}

// initAlertRules 初始化告警规则
func (c *CostController) initAlertRules() {
	c.alertRules = []CostAlertRule{
		{
			Name:        "BudgetCritical",
			Type:        "budget_critical",
			Threshold:   0.95,
			Severity:    "critical",
			Enabled:     true,
			Description: "预算使用率超过95%",
		},
		{
			Name:        "BudgetWarning",
			Type:        "budget_warning",
			Threshold:   0.75,
			Severity:    "warning",
			Enabled:     true,
			Description: "预算使用率超过75%",
		},
		{
			Name:        "CostAnomaly",
			Type:        "cost_anomaly",
			Threshold:   2.0,
			Severity:    "warning",
			Enabled:     true,
			Description: "检测到异常成本增长",
		},
		{
			Name:        "PredictionExceeded",
			Type:        "prediction_exceeded",
			Threshold:   1.0,
			Severity:    "warning",
			Enabled:     true,
			Description: "预测成本将超出预算",
		},
	}
}

// startAlertProcessor 启动告警处理器
func (c *CostController) startAlertProcessor() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for alert := range c.alertChan {
			c.processAlert(alert)
		}
	}()
}

// processAlert 处理告警
func (c *CostController) processAlert(alert *CostAlert) {
	switch alert.Severity {
	case "critical":
		c.logger.WithFields(logrus.Fields{
			"type":        alert.Type,
			"current_cost": alert.CurrentCost,
			"budget":      alert.Budget,
			"usage_rate":  alert.UsageRate,
		}).Error("成本告警: " + alert.Message)
	case "warning":
		c.logger.WithFields(logrus.Fields{
			"type":        alert.Type,
			"current_cost": alert.CurrentCost,
			"budget":      alert.Budget,
		}).Warn("成本告警: " + alert.Message)
	default:
		c.logger.WithFields(logrus.Fields{
			"type": alert.Type,
		}).Info("成本告警: " + alert.Message)
	}
}

// CheckAlerts 检查并触发告警
func (c *CostController) CheckAlerts() []*CostAlert {
	// 先获取需要的数据（避免死锁）
	var totalCost float64
	var usageRate float64
	var prediction *CostPredictionResult

	c.mu.Lock()
	totalCost = c.calculateTotalMonthlyCost()
	usageRate = totalCost / c.budget.MonthlyBudget
	c.mu.Unlock()

	// 在锁外调用 PredictCosts
	prediction = c.PredictCosts()

	var triggered []*CostAlert

	for _, rule := range c.alertRules {
		var shouldAlert bool
		var currentCost float64

		switch rule.Type {
		case "budget_critical", "budget_warning":
			shouldAlert = usageRate >= rule.Threshold
			currentCost = totalCost
		case "cost_anomaly":
			if prediction.Trend == "increasing" {
				shouldAlert = true
			}
		case "prediction_exceeded":
			if prediction.ProjectedMonthly > c.budget.MonthlyBudget {
				shouldAlert = true
				currentCost = prediction.ProjectedMonthly
			}
		}

		if shouldAlert {
			alert := &CostAlert{
				Type:        rule.Type,
				Severity:    rule.Severity,
				Message:     rule.Description,
				CurrentCost: currentCost,
				Budget:      c.budget.MonthlyBudget,
				UsageRate:   usageRate,
				Timestamp:   time.Now(),
			}

			// 获取锁后添加到活跃告警
			c.mu.Lock()
			triggered = append(triggered, alert)
			c.activeAlerts = append(c.activeAlerts, alert)
			c.mu.Unlock()

			// 发送到告警通道
			select {
			case c.alertChan <- alert:
			default:
				// 通道满，丢弃
			}
		}
	}

	return triggered
}

// GetActiveAlerts 获取活跃告警
func (c *CostController) GetActiveAlerts() []*CostAlert {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var active []*CostAlert
	for _, alert := range c.activeAlerts {
		if !alert.Resolved {
			active = append(active, alert)
		}
	}
	return active
}

// ResolveAlert 解决告警
func (c *CostController) ResolveAlert(alertType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, alert := range c.activeAlerts {
		if alert.Type == alertType && !alert.Resolved {
			alert.Resolved = true
		}
	}
}

// AlertChan 返回告警通道
func (c *CostController) AlertChan() <-chan *CostAlert {
	return c.alertChan
}

// startDailyRecorder 启动每日成本记录器
func (c *CostController) startDailyRecorder() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.recordDailyCost()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// recordDailyCost 记录每日成本
func (c *CostController) recordDailyCost() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	totalCost := c.calculateTotalMonthlyCost()

	// 计算今日成本（通过总成本减去昨天之前的成本）
	for name, cost := range c.currentMonthlyCost {
		if c.dailyCosts[name] == nil {
			c.dailyCosts[name] = make([]float64, 0, 30)
		}

		// 添加今日成本估算（简化处理：假设成本均匀分布）
		dailyCost := cost / float64(now.Day())
		c.dailyCosts[name] = append(c.dailyCosts[name], dailyCost)

		// 保持最近30天数据
		if len(c.dailyCosts[name]) > 30 {
			c.dailyCosts[name] = c.dailyCosts[name][len(c.dailyCosts[name])-30:]
		}
	}

	c.logger.WithFields(logrus.Fields{
		"date":   dateStr,
		"total":  totalCost,
	}).Debug("Daily cost recorded")
}

// IsNearBudgetLimit 检查是否接近预算限制
func (c *CostController) IsNearBudgetLimit(threshold float64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalCost := c.calculateTotalMonthlyCost()
	usageRate := totalCost / c.budget.MonthlyBudget
	return usageRate >= threshold
}

// RecordUsage 记录使用量
func (c *CostController) RecordUsage(providerName string, tokensUsed int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	provider, exists := c.providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	// 计算成本
	cost := float64(tokensUsed) * provider.CostPer1KTokens / 1000.0
	c.currentMonthlyCost[providerName] += cost

	c.logger.WithFields(logrus.Fields{
		"provider":  providerName,
		"tokens":    tokensUsed,
		"cost":      cost,
		"monthly":   c.currentMonthlyCost[providerName],
	}).Debug("Cost usage recorded")

	// 检查预算
	c.checkBudget()

	return nil
}

// checkBudget 检查预算是否超限
func (c *CostController) checkBudget() {
	totalCost := c.calculateTotalMonthlyCost()
	usageRate := totalCost / c.budget.MonthlyBudget

	if usageRate >= c.alertThreshold {
		c.logger.WithFields(logrus.Fields{
			"total_cost":  totalCost,
			"budget":      c.budget.MonthlyBudget,
			"usage_rate":  usageRate,
		}).Warn("Cost budget threshold exceeded")

		// 可以在这里触发告警
		c.triggerBudgetAlert(totalCost, usageRate)
	}

	if totalCost > c.budget.MonthlyBudget {
		c.logger.WithFields(logrus.Fields{
			"total_cost": totalCost,
			"budget":      c.budget.MonthlyBudget,
			"exceeded":   totalCost - c.budget.MonthlyBudget,
		}).Error("Cost budget exceeded")
	}
}

// triggerBudgetAlert 触发预算告警
func (c *CostController) triggerBudgetAlert(totalCost float64, usageRate float64) {
	// 这里可以集成到外部告警系统
	c.logger.Info("Budget alert triggered - consider cost optimization")
}

// calculateTotalMonthlyCost 计算当月总成本
func (c *CostController) calculateTotalMonthlyCost() float64 {
	total := 0.0
	for _, cost := range c.currentMonthlyCost {
		total += cost
	}
	return total
}

// GetMonthlyReport 获取月度成本报告
func (c *CostController) GetMonthlyReport() *CostReport {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalCost := c.calculateTotalMonthlyCost()
	usageRate := totalCost / c.budget.MonthlyBudget

	report := &CostReport{
		CurrentMonth: time.Now().Format("2006-01"),
		TotalCost:   totalCost,
		Providers:   make(map[string]*ProviderReport),
		Budget: BudgetReport{
			Monthly:   c.budget.MonthlyBudget,
			Used:      totalCost,
			Remaining:  c.budget.MonthlyBudget - totalCost,
			UsageRate:  usageRate,
			IsExceeded: totalCost > c.budget.MonthlyBudget,
		},
	}

	// 生成各Provider报告
	for name, cost := range c.providers {
		providerCost := c.currentMonthlyCost[name]
		if providerCost == 0 {
			continue
		}

		// 模拟使用量（实际应该从usage统计获取）
		tokensUsed := int64(providerCost * 1000 / cost.CostPer1KTokens)

		report.Providers[name] = &ProviderReport{
			Name:    name,
			Model:   cost.Model,
			Usage:   tokensUsed,
			Cost:    providerCost,
			AvgCost: cost.CostPer1KTokens,
		}
	}

	return report
}

// GetProviderCost 获取Provider成本信息
func (c *CostController) GetProviderCost(providerName string) (*ProviderCost, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cost, exists := c.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	return cost, nil
}

// ResetMonthlyStats 重置月度统计（通常在月初调用）
func (c *CostController) ResetMonthlyStats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.currentMonthlyCost = make(map[string]float64)

	c.logger.Info("Monthly cost statistics reset")
}

// IsOverBudget 检查是否超预算
func (c *CostController) IsOverBudget() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.calculateTotalMonthlyCost() > c.budget.MonthlyBudget
}

// GetRecommendations 获取成本优化建议
func (c *CostController) GetRecommendations() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var recommendations []string
	totalCost := c.calculateTotalMonthlyCost()

	// 基于使用率给出建议
	usageRate := totalCost / c.budget.MonthlyBudget
	if usageRate > 0.9 {
		recommendations = append(recommendations, "成本使用率过高，建议增加预算或优化使用")
	}

	if usageRate > 0.8 {
		recommendations = append(recommendations, "接近预算上限，建议监控使用情况")
	}

	// 基于Provider成本给出建议
	for name, cost := range c.providers {
		providerCost := c.currentMonthlyCost[name]
		if providerCost > 0 {
			// 计算实际每1K token成本
			actualCostPer1K := providerCost * 1000 / (providerCost * 1000 / cost.CostPer1KTokens)
			if actualCostPer1K > cost.CostPer1KTokens*1.5 {
				recommendations = append(recommendations,
					fmt.Sprintf("Provider %s使用成本较高，建议优化Prompt或使用更经济的模型", name))
			}
		}
	}

	return recommendations
}

// SetMonthlyCost 设置Provider月成本（用于测试或手动调整）
func (c *CostController) SetMonthlyCost(providerName string, cost float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.providers[providerName]; exists {
		c.currentMonthlyCost[providerName] = cost
		c.logger.WithField("provider", providerName).WithField("cost", cost).Info("Monthly cost set manually")
	}
}

// ========== 成本预测功能 ==========

// PredictCosts 预测未来成本
func (c *CostController) PredictCosts() *CostPredictionResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	currentDay := float64(now.Day())
	totalCost := c.calculateTotalMonthlyCost()

	// 计算日均成本
	dailyAverage := totalCost / currentDay

	// 计算趋势（基于最近7天的变化）
	trend := c.calculateTrend()

	// 预测未来成本
	nextDayCost := dailyAverage * (1 + trend*0.1)
	nextWeekCost := dailyAverage * 7 * (1 + trend*0.2)
	nextMonthCost := dailyAverage * 30 * (1 + trend*0.3)

	// 计算置信度（基于数据量）
	confidence := math.Min(0.9, 0.5+float64(len(c.dailyCosts))*0.02)

	// 计算预测月成本
	daysInMonth := float64(now.AddDate(0, 1, 0).Day())
	projectedMonthly := dailyAverage * daysInMonth

	// 确定趋势方向
	trendStr := "stable"
	if trend > 0.1 {
		trendStr = "increasing"
	} else if trend < -0.1 {
		trendStr = "decreasing"
	}

	// 确定风险等级
	riskLevel := "low"
	usageRate := totalCost / c.budget.MonthlyBudget
	if usageRate > 0.9 || (trendStr == "increasing" && usageRate > 0.7) {
		riskLevel = "high"
	} else if usageRate > 0.7 || trendStr == "increasing" {
		riskLevel = "medium"
	}

	return &CostPredictionResult{
		NextDayCost:      nextDayCost,
		NextWeekCost:     nextWeekCost,
		NextMonthCost:    nextMonthCost,
		DailyAverage:     dailyAverage,
		ProjectedMonthly: projectedMonthly,
		Trend:            trendStr,
		Confidence:       confidence,
		RiskLevel:        riskLevel,
	}
}

// calculateTrend 计算成本趋势
func (c *CostController) calculateTrend() float64 {
	if len(c.dailyCosts) == 0 {
		return 0
	}

	// 计算所有Provider的总日均成本
	var totalDailyCosts []float64
	for _, costs := range c.dailyCosts {
		if len(costs) > 0 {
			// 取最近7天的数据
			recentCosts := costs
			if len(recentCosts) > 7 {
				recentCosts = recentCosts[len(recentCosts)-7:]
			}
			var sum float64
			for _, d := range recentCosts {
				sum += d
			}
			totalDailyCosts = append(totalDailyCosts, sum/float64(len(recentCosts)))
		}
	}

	if len(totalDailyCosts) < 2 {
		return 0
	}

	// 简单线性回归计算趋势斜率
	n := float64(len(totalDailyCosts))
	var sumX, sumY, sumXY, sumX2 float64
	for i, y := range totalDailyCosts {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	if n*sumX2-sumX*sumX == 0 {
		return 0
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	avgY := sumY / n

	// 归一化趋势值
	if avgY > 0 {
		return slope / avgY
	}
	return 0
}

// GetPrediction 返回成本预测模型
func (c *CostController) GetPrediction() *CostPredictionModel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	prediction := c.PredictCosts()
	return &CostPredictionModel{
		TrendAnalysis:     c.calculateTrend(),
		PredictedNextDay:  prediction.NextDayCost,
		PredictedNextWeek: prediction.NextWeekCost,
		PredictedNextMonth: prediction.NextMonthCost,
		Confidence:        prediction.Confidence,
		LastUpdated:       time.Now(),
	}
}

// ========== 优化规则引擎 ==========

// RunOptimizationRules 运行所有优化规则
func (c *CostController) RunOptimizationRules() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var actions []string
	for _, rule := range c.optimizationRules {
		if !rule.Enabled {
			continue
		}

		if rule.Condition(c) {
			if err := rule.Action(c); err != nil {
				c.logger.WithFields(logrus.Fields{
					"rule":  rule.Name,
					"error": err,
				}).Warn("Optimization rule failed")
			} else {
				actions = append(actions, fmt.Sprintf("执行规则: %s - %s", rule.Name, rule.Description))
			}
		}
	}
	return actions
}

// GetOptimizationRules 获取所有优化规则
func (c *CostController) GetOptimizationRules() []CostOptimizationRule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.optimizationRules
}

// EnableOptimizationRule 启用/禁用优化规则
func (c *CostController) EnableOptimizationRule(ruleName string, enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, rule := range c.optimizationRules {
		if rule.Name == ruleName {
			c.optimizationRules[i].Enabled = enabled
			c.logger.WithField("rule", ruleName).WithField("enabled", enabled).Info("Optimization rule updated")
			return
		}
	}
}

// ========== 增强报告功能 ==========

// GetEnhancedReport 获取增强的成本报告（包含预测和优化建议）
func (c *CostController) GetEnhancedReport() *CostReport {
	c.mu.RLock()
	defer c.mu.RUnlock()

	report := c.GetMonthlyReport()

	// 添加预测
	prediction := c.PredictCosts()
	report.Prediction = &CostPredictionResult{
		NextDayCost:      prediction.NextDayCost,
		NextWeekCost:     prediction.NextWeekCost,
		NextMonthCost:    prediction.NextMonthCost,
		DailyAverage:     prediction.DailyAverage,
		ProjectedMonthly: prediction.ProjectedMonthly,
		Trend:            prediction.Trend,
		Confidence:       prediction.Confidence,
		RiskLevel:        prediction.RiskLevel,
	}

	// 添加优化建议
	report.OptimizationTips = c.getOptimizationTips()

	return report
}

// getOptimizationTips 生成优化建议
func (c *CostController) getOptimizationTips() []string {
	var tips []string

	totalCost := c.calculateTotalMonthlyCost()
	usageRate := totalCost / c.budget.MonthlyBudget
	prediction := c.PredictCosts()

	// 基于使用率的建议
	if usageRate > 0.9 {
		tips = append(tips, "紧急：成本使用率超过90%，建议立即采取措施减少使用或增加预算")
	} else if usageRate > 0.75 {
		tips = append(tips, "警告：成本使用率超过75%，建议开始优化使用策略")
	}

	// 基于趋势的建议
	if prediction.Trend == "increasing" {
		tips = append(tips, "趋势：成本呈上升趋势，建议分析原因并采取预防措施")
	}

	// 基于风险等级的建议
	if prediction.RiskLevel == "high" {
		tips = append(tips, "风险：预测显示高风险，建议启用成本上限或切换到更经济的模型")
	}

	// 基于Provider的建议
	var maxProvider string
	var maxCost float64
	for name, cost := range c.currentMonthlyCost {
		if cost > maxCost {
			maxCost = cost
			maxProvider = name
		}
	}

	if maxProvider != "" && maxCost > 0 {
		providerRatio := maxCost / totalCost
		if providerRatio > 0.5 {
			tips = append(tips, fmt.Sprintf("优化：%s 占据 %.0f%% 的成本，考虑使用更经济的Provider或优化请求", maxProvider, providerRatio*100))
		}
	}

	// 基于预算的建议
	if prediction.ProjectedMonthly > c.budget.MonthlyBudget {
		excess := prediction.ProjectedMonthly - c.budget.MonthlyBudget
		tips = append(tips, fmt.Sprintf("预测：预计月成本将超出预算 $%.2f，建议调整预算或优化策略", excess))
	}

	if len(tips) == 0 {
		tips = append(tips, "状态良好：当前成本在可控范围内")
	}

	return tips
}

// Stop 停止成本控制器（清理资源）
func (c *CostController) Stop() {
	close(c.stopCh)
	close(c.alertChan)
	c.wg.Wait()
	c.logger.Info("Cost controller stopped")
}