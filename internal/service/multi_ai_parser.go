package service

import (
	"context"
	"fmt"
	"time"

	"mmemory/pkg/ai"
	"mmemory/internal/models"
	"mmemory/pkg/logger"
)

// MultiAIParserService 多AI Provider解析服务
type MultiAIParserService struct {
	providerManager *ai.ProviderManager
	enabled         bool
}

// NewMultiAIParserService 创建多AI解析服务
func NewMultiAIParserService(providers map[string]ai.AIProviderInterface, primary string, fallback []string, enabled bool) *MultiAIParserService {
	if !enabled {
		return &MultiAIParserService{enabled: false}
	}

	manager := ai.NewProviderManager(
		providers,
		primary,
		fallback,
		logger.GetLogger(),
	)

	// 启动健康检查
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		healthResults := manager.HealthCheck(ctx)
		for provider, err := range healthResults {
			if err != nil {
				logger.GetLogger().WithError(err).WithField("provider", provider).Warn("Provider health check failed")
			} else {
				logger.GetLogger().WithField("provider", provider).Info("Provider healthy")
			}
		}
	}()

	return &MultiAIParserService{
		providerManager: manager,
		enabled:         enabled,
	}
}

// ParseReminderMultiProvider 使用多Provider解析提醒
func (s *MultiAIParserService) ParseReminderMultiProvider(ctx context.Context, text string) (*ai.ProviderParseResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("multi-AI parser is disabled")
	}

	return s.providerManager.ParseWithFallback(ctx, text)
}

// ChatMultiProvider 使用多Provider聊天
func (s *MultiAIParserService) ChatMultiProvider(ctx context.Context, text string) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("multi-AI parser is disabled")
	}

	return s.providerManager.ChatWithFallback(ctx, text)
}

// GetMetrics 获取Provider指标
func (s *MultiAIParserService) GetMetrics() *ai.ProviderMetrics {
	if s.providerManager != nil {
		return s.providerManager.GetMetrics()
	}
	return ai.NewProviderMetrics()
}

// IsEnabled 检查是否启用
func (s *MultiAIParserService) IsEnabled() bool {
	return s.enabled
}

// ConvertToLegacyResult 将Provider解析结果转换为legacy ParseResult（兼容现有系统）
func ConvertToLegacyResult(providerResult *ai.ProviderParseResult, userID uint) (*models.Reminder, error) {
	if providerResult == nil {
		return nil, fmt.Errorf("provider result is nil")
	}

	reminder := &models.Reminder{
		UserID:  userID,
		Title:   providerResult.Content,
		IsActive: true,
		Type:    models.ReminderTypeTask,
	}

	// 转换时间
	if !providerResult.Time.IsZero() {
		hour := providerResult.Time.Hour()
		minute := providerResult.Time.Minute()
		reminder.TargetTime = fmt.Sprintf("%02d:%02d", hour, minute)

		// 根据时间差推断模式
		now := time.Now()
		hoursUntil := int(providerResult.Time.Sub(now).Hours())

		if hoursUntil < 24 {
			// 24小时内，判断为一次性
			reminder.SchedulePattern = fmt.Sprintf("once:%s", providerResult.Time.Format("2006-01-02"))
		} else if hoursUntil < 24*7 {
			// 一周内，可能是每日或每周
			if providerResult.Time.Weekday() == now.Weekday() {
				reminder.SchedulePattern = "daily"
			} else {
				weekday := int(providerResult.Time.Weekday())
				if weekday == 0 {
					weekday = 7 // 周日转换为7
				}
				reminder.SchedulePattern = fmt.Sprintf("weekly:%d", weekday)
			}
		} else {
			// 更远的未来，设为每月
			reminder.SchedulePattern = fmt.Sprintf("monthly:%d", providerResult.Time.Day())
		}
	} else {
		// 没有明确时间，设置为明天同一时间
		now := time.Now()
		tomorrow := now.Add(24 * time.Hour)
		reminder.TargetTime = fmt.Sprintf("%02d:%02d", tomorrow.Hour(), tomorrow.Minute())
		reminder.SchedulePattern = fmt.Sprintf("once:%s", tomorrow.Format("2006-01-02"))
	}

	return reminder, nil
}