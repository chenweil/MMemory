package ai

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ProviderScore Provider评分
type ProviderScore struct {
	Name         string
	Performance  float64 // 响应时间评分 (越快越低)
	Reliability  float64 // 可靠性评分 (成功率)
	Cost         float64 // 成本评分 (越低越高)
	Overall      float64 // 综合评分
	ResponseTime time.Duration
}

// IntelligentManager 智能Provider管理器
type IntelligentManager struct {
	*ProviderManager
	strategy     FallbackStrategy
	scores      map[string]*ProviderScore
	stats       map[string]*ProviderStats
	monthlyCost map[string]float64 // 月成本统计
	mu          sync.RWMutex
}

// ProviderStats Provider统计
type ProviderStats struct {
	TotalRequests   int64
	SuccessCount    int64
	ErrorCount      int64
	AvgResponseTime time.Duration
	TotalTokens    int64
	LastReset      time.Time
}

// NewIntelligentManager 创建智能管理器
func NewIntelligentManager(
	providers map[string]AIProviderInterface,
	primary string,
	fallback []string,
	strategy FallbackStrategy,
	logger *logrus.Logger,
) *IntelligentManager {
	baseManager := NewProviderManager(providers, primary, fallback, logger)

	return &IntelligentManager{
		ProviderManager: baseManager,
		strategy:       strategy,
		scores:         make(map[string]*ProviderScore),
		stats:          make(map[string]*ProviderStats),
		monthlyCost:    make(map[string]float64),
	}
}

// ParseWithIntelligentSelection 使用智能选择进行解析
func (m *IntelligentManager) ParseWithIntelligentSelection(ctx context.Context, text string) (*ProviderParseResult, error) {
	// 1. 更新Provider评分
	m.updateProviderScores()

	// 2. 选择最优Provider
	selectedProvider := m.selectOptimalProvider(ctx)

	if selectedProvider == "" {
		// 回退到基础降级
		return m.ParseWithFallback(ctx, text)
	}

	// 3. 使用选定的Provider
	provider := m.selectProvider(selectedProvider)
	if provider == nil {
		return nil, fmt.Errorf("selected provider not found: %s", selectedProvider)
	}

	result, err := m.tryProvider(ctx, provider, text)
	if err != nil {
		// 智能选择失败，回退到基础降级
		m.logger.WithError(err).WithField("provider", selectedProvider).
			Warn("Intelligent selection failed, falling back")
		return m.ParseWithFallback(ctx, text)
	}

	// 4. 更新成功统计
	m.recordSuccess(selectedProvider, result)

	return result, nil
}

// updateProviderScores 更新Provider评分
func (m *IntelligentManager) updateProviderScores() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name := range m.providers {
		stats := m.stats[name]
		if stats == nil {
			stats = &ProviderStats{}
			m.stats[name] = stats
		}

		// 计算评分
		score := &ProviderScore{
			Name: name,
		}

		// 响应时间评分 (越快越低，评分越高)
		if stats.AvgResponseTime > 0 {
			// 归一化到0-100分，100分为最好
			score.Performance = math.Max(0, 100-math.Min(stats.AvgResponseTime.Seconds()*100, 100))
			score.ResponseTime = time.Duration(stats.AvgResponseTime)
		}

		// 可靠性评分 (成功率)
		if stats.TotalRequests > 0 {
			successRate := float64(stats.SuccessCount) / float64(stats.TotalRequests)
			score.Reliability = successRate * 100
		}

		// 成本评分 (基于每token成本，越低评分越高)
		if stats.TotalTokens > 0 {
			costPerToken := m.monthlyCost[name] / float64(stats.TotalTokens)
			// 反向评分，成本越低评分越高，限制在0-100
			score.Cost = math.Max(0, 100-math.Min(costPerToken*1000, 100))
		}

		// 综合评分
		score.Overall = score.Performance*m.strategy.PerformanceWeight +
			score.Reliability*m.strategy.ReliabilityWeight +
			score.Cost*m.strategy.CostWeight

		m.scores[name] = score
	}
}

// selectOptimalProvider 选择最优Provider
func (m *IntelligentManager) selectOptimalProvider(ctx context.Context) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var candidates []ProviderScore

	// 筛选可用的Provider（不在熔断状态且响应时间合理）
	for name, score := range m.scores {
		if !m.isProviderAvailable(name) {
			continue
		}

		// 过滤响应时间过长的Provider
		if score.ResponseTime > 10*time.Second {
			continue
		}

		candidates = append(candidates, *score)
	}

	if len(candidates) == 0 {
		m.logger.Warn("No available providers found")
		return ""
	}

	// 按综合评分排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Overall > candidates[j].Overall
	})

	bestProvider := candidates[0].Name

	m.logger.WithFields(logrus.Fields{
		"selected": bestProvider,
		"score":     candidates[0].Overall,
		"candidates": len(candidates),
	}).Info("Provider selected intelligently")

	return bestProvider
}

// isProviderAvailable 检查Provider是否可用
func (m *IntelligentManager) isProviderAvailable(name string) bool {
	// 检查熔断器状态
	if breaker := m.breakers[name]; breaker != nil {
		if !breaker.CanRequest() {
			return false
		}
	}

	// 检查统计信息
	stats := m.stats[name]
	if stats == nil {
		return true
	}

	// 检查成功率
	if stats.TotalRequests > 10 {
		successRate := float64(stats.SuccessCount) / float64(stats.TotalRequests)
		if successRate < 0.5 { // 成功率低于50%
			return false
		}
	}

	return true
}

// recordRequest 记录请求
func (m *IntelligentManager) recordRequest(providerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.stats[providerName]
	if stats == nil {
		stats = &ProviderStats{}
		m.stats[providerName] = stats
	}

	stats.TotalRequests++
}

// recordSuccess 记录成功
func (m *IntelligentManager) recordSuccess(providerName string, result *ProviderParseResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.stats[providerName]
	if stats == nil {
		return
	}

	stats.SuccessCount++

	// 更新Token统计
	if result != nil && result.TokensUsed > 0 {
		stats.TotalTokens += int64(result.TokensUsed)
	}
}

// recordError 记录错误
func (m *IntelligentManager) recordError(providerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.stats[providerName]
	if stats == nil {
		return
	}

	stats.ErrorCount++
}

// updateStats 更新统计信息
func (m *IntelligentManager) updateStats(providerName string, responseTime time.Duration, result *ProviderParseResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.stats[providerName]
	if stats == nil {
		return
	}

	// 更新平均响应时间
	requestCount := float64(stats.TotalRequests)
	if requestCount > 0 {
		oldAvgSeconds := stats.AvgResponseTime.Seconds()
		newAvgSeconds := responseTime.Seconds()
		newAvgDuration := time.Duration((oldAvgSeconds*(requestCount-1)+newAvgSeconds) / requestCount * float64(time.Second))
		stats.AvgResponseTime = newAvgDuration
	}

	// 重置月度统计（如果需要）
	now := time.Now()
	if now.Sub(stats.LastReset) > 30*24*time.Hour {
		m.resetMonthlyStats(providerName)
	}
}

// resetMonthlyStats 重置月度统计
func (m *IntelligentManager) resetMonthlyStats(providerName string) {
	stats := m.stats[providerName]
	if stats != nil {
		stats.TotalRequests = 0
		stats.SuccessCount = 0
		stats.ErrorCount = 0
		stats.TotalTokens = 0
		stats.AvgResponseTime = 0
		stats.LastReset = time.Now()
	}

	m.monthlyCost[providerName] = 0
}

// GetIntelligentStats 获取智能统计
func (m *IntelligentManager) GetIntelligentStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{})

	for name, score := range m.scores {
		result[name] = map[string]interface{}{
			"score":        score,
			"stats":        m.stats[name],
			"monthly_cost": m.monthlyCost[name],
		}
	}

	return result
}

// SetMonthlyCost 设置月成本
func (m *IntelligentManager) SetMonthlyCost(providerName string, cost float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.monthlyCost[providerName] = cost
}

// GetProviders 获取Provider列表
func (m *IntelligentManager) GetProviders() map[string]AIProviderInterface {
	return m.providers
}