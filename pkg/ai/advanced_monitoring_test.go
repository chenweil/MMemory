package ai

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewAdvancedMonitor(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.providerStats)
	assert.NotNil(t, monitor.alertChannel)
	assert.NotNil(t, monitor.logger)
}

func TestAdvancedMonitor_UpdateProviderStats(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	// Test updating stats for a new provider
	monitor.UpdateProviderStats("provider1", 100*time.Millisecond, true, 100)

	stats, exists := monitor.providerStats["provider1"]
	assert.True(t, exists)
	assert.NotNil(t, stats)
	assert.NotNil(t, stats.ProviderStats)
	assert.Equal(t, int64(1), stats.ProviderStats.TotalRequests)
	assert.Equal(t, int64(1), stats.ProviderStats.SuccessCount)
	assert.Equal(t, int64(0), stats.ProviderStats.ErrorCount)
	assert.Equal(t, int64(100), stats.ProviderStats.TotalTokens)

	// Test updating stats for an existing provider
	monitor.UpdateProviderStats("provider1", 200*time.Millisecond, false, 150)

	stats, exists = monitor.providerStats["provider1"]
	assert.True(t, exists)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(2), stats.ProviderStats.TotalRequests)
	assert.Equal(t, int64(1), stats.ProviderStats.SuccessCount)
	assert.Equal(t, int64(1), stats.ProviderStats.ErrorCount)
	assert.Equal(t, int64(250), stats.ProviderStats.TotalTokens)
	// Trend data replaces the first element on subsequent updates
	assert.Len(t, stats.TrendData, 1)
}

func TestAdvancedMonitor_CheckAlerts(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	// No alert rules configured, should not panic
	monitor.CheckAlerts()

	// Add an alert rule
	rule := AlertRule{
		Name:        "high_error_rate",
		Type:        AlertErrorRate,
		Threshold:   0.1,
		Duration:    time.Minute,
		Provider:    "provider1",
		Description: "Error rate too high",
	}

	monitor.UpdateAlertRules([]AlertRule{rule})

	// Add some stats with low error rate
	for i := 0; i < 10; i++ {
		monitor.UpdateProviderStats("provider1", 100*time.Millisecond, true, 100)
	}

	// Should not panic
	monitor.CheckAlerts()
}

func TestAdvancedMonitor_GetProviderTrends(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	// Add some stats
	for i := 0; i < 10; i++ {
		monitor.UpdateProviderStats("provider1", 100*time.Millisecond, true, 100)
	}

	trends := monitor.GetProviderTrends()
	assert.NotNil(t, trends)
}

func TestAdvancedMonitor_RecordDecisionLatency(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	// Record decision latency
	monitor.RecordDecisionLatency("intelligent_c3", 50*time.Millisecond)

	// Should not panic
	assert.True(t, true)
}

func TestAdvancedMonitor_GetDecisionLatency(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	// Record decision latency
	monitor.RecordDecisionLatency("intelligent_c3", 50*time.Millisecond)

	// Get decision latency stats
	latency := monitor.GetDecisionLatency("intelligent_c3")
	assert.NotNil(t, latency)
}

func TestAdvancedMonitor_GetAlertRules(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	// Add alert rules
	rules := []AlertRule{
		{
			Name:        "rule1",
			Type:        AlertErrorRate,
			Threshold:   0.1,
			Duration:    time.Minute,
			Description: "Test rule 1",
		},
		{
			Name:        "rule2",
			Type:        AlertResponseTime,
			Threshold:   500.0,
			Duration:    time.Minute,
			Description: "Test rule 2",
		},
	}

	monitor.UpdateAlertRules(rules)

	// Get alert rules
	retrievedRules := monitor.GetAlertRules()
	assert.Len(t, retrievedRules, 2)
	assert.Equal(t, "rule1", retrievedRules[0].Name)
	assert.Equal(t, "rule2", retrievedRules[1].Name)
}

func TestAdvancedMonitor_UpdateAlertRules(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitor := NewAdvancedMonitor(logger)

	// Add initial alert rules
	rules := []AlertRule{
		{
			Name:        "rule1",
			Type:        AlertErrorRate,
			Threshold:   0.1,
			Duration:    time.Minute,
			Description: "Test rule 1",
		},
	}

	monitor.UpdateAlertRules(rules)
	assert.Len(t, monitor.alertRules, 1)

	// Update alert rules
	newRules := []AlertRule{
		{
			Name:        "rule1",
			Type:        AlertErrorRate,
			Threshold:   0.2, // Updated threshold
			Duration:    time.Minute,
			Description: "Test rule 1 updated",
		},
		{
			Name:        "rule2",
			Type:        AlertResponseTime,
			Threshold:   500.0,
			Duration:    time.Minute,
			Description: "Test rule 2",
		},
	}

	monitor.UpdateAlertRules(newRules)
	assert.Len(t, monitor.alertRules, 2)
	assert.Equal(t, 0.2, monitor.alertRules[0].Threshold)
}