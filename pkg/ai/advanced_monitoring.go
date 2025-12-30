package ai

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// AdvancedMonitor 高级监控和告警系统
type AdvancedMonitor struct {
	providerStats map[string]*ExtendedProviderStats
	alertRules    []AlertRule
	mu           sync.RWMutex
	logger       *logrus.Logger
	alertChannel chan Alert
}

// ExtendedProviderStats 扩展的Provider统计
type ExtendedProviderStats struct {
	*ProviderStats
	TrendData    []float64    // 最近N次响应时间的趋势数据
	ErrorTrend   []float64    // 最近N次错误率趋势
	LastAlert    time.Time     // 最后告警时间
}

// AlertRule 告警规则
type AlertRule struct {
	Name         string          `json:"name"`
	Type         AlertType       `json:"type"`
	Threshold   float64         `json:"threshold"`
	Duration    time.Duration   `json:"duration"`
	Provider    string          `json:"provider,omitempty"`
	Description string          `json:"description"`
}

// AlertType 告警类型
type AlertType string

const (
	AlertResponseTime AlertType = "response_time"
	AlertErrorRate    AlertType = "error_rate"
	AlertCostBudget   AlertType = "cost_budget"
	AlertCircuitBreaker AlertType = "circuit_breaker"
	AlertProviderDown AlertType = "provider_down"
)

var (
	// 高级Prometheus指标
	providerTrendGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_provider_response_trend",
			Help: "Provider response time trend over last N requests",
		},
		[]string{"provider"},
	)

	providerErrorRateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_provider_error_rate_trend",
			Help: "Provider error rate trend over last N requests",
		},
		[]string{"provider"},
	)

	aiCostBudgetGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_cost_budget_usage",
			Help: "AI cost budget usage percentage",
		},
		[]string{"budget"},
	)

	aiDecisionLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_decision_latency_seconds",
			Help:    "Time taken to make provider selection decisions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"decision_type"},
	)

	aiFallbackReasons = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_fallback_reasons_total",
			Help: "Total count of fallback reasons",
		},
		[]string{"reason"},
	)
)

	// NewAdvancedMonitor 创建高级监控
func NewAdvancedMonitor(logger *logrus.Logger) *AdvancedMonitor {
	monitor := &AdvancedMonitor{
		providerStats: make(map[string]*ExtendedProviderStats),
		alertRules:    []AlertRule{
			{
				Name:        "Response Time Alert",
				Type:        AlertResponseTime,
				Threshold:  5.0, // 5秒
				Duration:    time.Minute * 2, // 持续2分钟
				Description: "Provider response time too high",
			},
			{
				Name:        "Error Rate Alert",
				Type:        AlertErrorRate,
				Threshold: 0.15, // 15%
				Duration:    time.Minute * 5, // 持续5分钟
				Description: "Provider error rate too high",
			},
			{
				Name:        "Cost Budget Alert",
				Type:        AlertCostBudget,
				Threshold: 0.90, // 90%预算
				Duration:    time.Hour * 24, // 持续24小时
				Description: "Cost budget nearly exceeded",
			},
		},
		alertChannel: make(chan Alert, 100),
		logger:       logger,
	}

	// 启动告警处理器
	go monitor.alertHandler()

	return monitor
}

// UpdateProviderStats 更新Provider统计（扩展版）
func (m *AdvancedMonitor) UpdateProviderStats(providerName string, responseTime time.Duration, success bool, tokensUsed int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.providerStats[providerName]
	if stats == nil {
		stats = &ExtendedProviderStats{
			ProviderStats: &ProviderStats{},
			TrendData:    make([]float64, 0, 100),
			ErrorTrend:   make([]float64, 0, 100),
		}
		m.providerStats[providerName] = stats
	}

	// 更新基础统计
	stats.TotalRequests++
	if success {
		stats.SuccessCount++
	} else {
		stats.ErrorCount++
	}

	if tokensUsed > 0 {
		stats.TotalTokens += int64(tokensUsed)
	}

	// 更新趋势数据
	responseSeconds := responseTime.Seconds()
	if len(stats.TrendData) == 0 {
		stats.TrendData = append(stats.TrendData, responseSeconds)
	} else {
		stats.TrendData = append(stats.TrendData[1:], responseSeconds)
	}
	if len(stats.TrendData) > 100 {
		stats.TrendData = stats.TrendData[len(stats.TrendData)-100:]
	}

	// 更新错误率趋势
	if stats.TotalRequests > 10 {
		errorRate := float64(stats.ErrorCount) / float64(stats.TotalRequests)
		if len(stats.ErrorTrend) == 0 {
			stats.ErrorTrend = append(stats.ErrorTrend, errorRate)
		} else {
			stats.ErrorTrend = append(stats.ErrorTrend[1:], errorRate)
		}
		if len(stats.ErrorTrend) > 100 {
			stats.ErrorTrend = stats.ErrorTrend[len(stats.ErrorTrend)-100:]
		}
	}

	// 更新平均响应时间
	if stats.TotalRequests > 1 {
		totalResponseTime := float64(stats.AvgResponseTime)*float64(stats.TotalRequests-1) + responseSeconds*float64(time.Second)
		stats.AvgResponseTime = time.Duration(totalResponseTime / float64(stats.TotalRequests))
	} else {
		stats.AvgResponseTime = time.Duration(responseSeconds * float64(time.Second))
	}

	// 更新Prometheus指标
	providerTrendGauge.WithLabelValues(providerName).Set(calculateTrend(stats.TrendData))
	providerErrorRateGauge.WithLabelValues(providerName).Set(calculateTrend(stats.ErrorTrend))

	m.logger.WithFields(logrus.Fields{
		"provider":      providerName,
		"response_time": responseTime,
		"success":       success,
		"tokens_used":   tokensUsed,
		"error_rate":    float64(stats.ErrorCount) / float64(stats.TotalRequests),
		"trend_score":   calculateTrend(stats.TrendData),
	}).Debug("Provider stats updated")
}

// CheckAlerts 检查告警
func (m *AdvancedMonitor) CheckAlerts() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, rule := range m.alertRules {
		if m.evaluateAlertRule(rule) {
			alert := Alert{
				RuleName:     rule.Name,
				Type:         string(rule.Type),
				Message:      fmt.Sprintf("%s: %s", rule.Name, rule.Description),
				Timestamp:    time.Now(),
				Provider:     rule.Provider,
				Threshold:    rule.Threshold,
				CurrentValue: m.getCurrentValue(rule.Type, rule.Provider),
			}

			select {
			case m.alertChannel <- alert:
			default:
				m.logger.Warn("Alert channel full, dropping alert")
			}
		}
	}
}

// evaluateAlertRule 评估告警规则
func (m *AdvancedMonitor) evaluateAlertRule(rule AlertRule) bool {
	if rule.Provider != "" {
		stats := m.providerStats[rule.Provider]
		if stats == nil {
			return false
		}

		switch rule.Type {
		case AlertResponseTime:
			return stats.AvgResponseTime > time.Duration(rule.Threshold*float64(time.Second))
		case AlertErrorRate:
			if stats.TotalRequests > 10 {
				errorRate := float64(stats.ErrorCount) / float64(stats.TotalRequests)
				return errorRate > rule.Threshold
			}
		}
	} else {
		// 全局告警
		switch rule.Type {
		case AlertCostBudget:
			// 这里需要成本控制器支持
			return false // 需要集成成本控制器
		case AlertProviderDown:
			for _, stats := range m.providerStats {
				if stats.ErrorCount > 5 && stats.TotalRequests > 10 {
					errorRate := float64(stats.ErrorCount) / float64(stats.TotalRequests)
					if errorRate > 0.3 { // 30%错误率
						return true
					}
				}
			}
		}
	}

	return false
}

// getCurrentValue 获取当前指标值
func (m *AdvancedMonitor) getCurrentValue(alertType AlertType, provider string) float64 {
	if provider != "" {
		stats := m.providerStats[provider]
		if stats == nil {
			return 0
		}

		switch alertType {
		case AlertResponseTime:
			return stats.AvgResponseTime.Seconds()
		case AlertErrorRate:
			if stats.TotalRequests > 0 {
				return float64(stats.ErrorCount) / float64(stats.TotalRequests)
			}
		}
	}

	return 0
}

// alertHandler 告警处理器
func (m *AdvancedMonitor) alertHandler() {
	for alert := range m.alertChannel {
		m.handleAlert(alert)
	}
}

// handleAlert 处理告警
func (m *AdvancedMonitor) handleAlert(alert Alert) {
	m.logger.WithFields(logrus.Fields{
		"alert_name":   alert.RuleName,
		"alert_type":   alert.Type,
		"message":      alert.Message,
		"provider":     alert.Provider,
		"threshold":    alert.Threshold,
		"value":        alert.CurrentValue,
	}).Warn("AI Provider Alert")

	// 更新Prometheus告警计数
	reason := fmt.Sprintf("%s_%s", alert.RuleName, alert.Provider)
	aiFallbackReasons.WithLabelValues(reason).Inc()

	// 这里可以集成到外部告警系统
	// 例如：发送邮件、Slack、PagerDuty等
}

// calculateTrend 计算趋势分数
func calculateTrend(data []float64) float64 {
	if len(data) < 10 {
		return 50.0
	}

	// 简单的线性回归来计算趋势
	n := float64(len(data))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i := 0; i < len(data); i++ {
		x := float64(i)
		y := data[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	if n*sumX2-sumX*sumX == 0 {
		return 50.0
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// 将斜率转换为0-100分数，50表示稳定趋势
	// 正斜率表示响应时间增长（坏），负斜率表示改善（好）
	trendScore := 50.0 - slope*10
	if trendScore > 100 {
		trendScore = 100
	} else if trendScore < 0 {
		trendScore = 0
	}

	return trendScore
}

// GetProviderTrends 获取Provider趋势
func (m *AdvancedMonitor) GetProviderTrends() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	trends := make(map[string]interface{})
	for name, stats := range m.providerStats {
		trends[name] = map[string]interface{}{
			"response_trend": calculateTrend(stats.TrendData),
			"error_trend":    calculateTrend(stats.ErrorTrend),
			"recent_avg":     stats.AvgResponseTime.Seconds(),
			"error_rate":     float64(stats.ErrorCount) / float64(stats.TotalRequests),
			"last_alert":     stats.LastAlert,
		}
	}

	return trends
}

// RecordDecisionLatency 记录决策延迟
func (m *AdvancedMonitor) RecordDecisionLatency(decisionType string, latency time.Duration) {
	aiDecisionLatency.WithLabelValues(decisionType).Observe(latency.Seconds())
}

// GetDecisionLatency 获取决策延迟统计
func (m *AdvancedMonitor) GetDecisionLatency(decisionType string) interface{} {
	// 这里可以返回更详细的延迟统计信息
	return map[string]interface{}{
		"type": decisionType,
		"note": "Use Prometheus metrics for detailed statistics",
	}
}

// GetAlertRules 获取告警规则
func (m *AdvancedMonitor) GetAlertRules() []AlertRule {
	return m.alertRules
}

// UpdateAlertRules 更新告警规则
func (m *AdvancedMonitor) UpdateAlertRules(newRules []AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertRules = newRules
	m.logger.Info("Alert rules updated")
}