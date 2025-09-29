package service

import (
	"context"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
	"mmemory/pkg/metrics"
)

// MonitoringService 监控服务接口
type MonitoringService interface {
	Start(ctx context.Context) error
	Stop() error
	UpdateMetrics(ctx context.Context) error
	RecordReminderOperation(operation string, status bool)
	RecordDatabaseOperation(operation string, duration time.Duration, err error)
	RecordNotificationSend(notificationType string, duration time.Duration, err error)
	RecordBotMessage(messageType string, err error)
	RecordReminderParse(parserType string, duration time.Duration, err error)
}

// monitoringService 监控服务实现
type monitoringService struct {
	userRepo     interfaces.UserRepository
	reminderRepo interfaces.ReminderRepository
	logRepo      interfaces.ReminderLogRepository
	
	startTime    time.Time
	stopChan     chan struct{}
	updateTicker *time.Ticker
}

// NewMonitoringService 创建监控服务
func NewMonitoringService(
	userRepo interfaces.UserRepository,
	reminderRepo interfaces.ReminderRepository,
	logRepo interfaces.ReminderLogRepository,
) MonitoringService {
	return &monitoringService{
		userRepo:     userRepo,
		reminderRepo: reminderRepo,
		logRepo:      logRepo,
		startTime:    time.Now(),
		stopChan:     make(chan struct{}),
	}
}

// Start 启动监控服务
func (s *monitoringService) Start(ctx context.Context) error {
	logger.Info("🔍 监控服务启动")
	
	// 设置初始指标
	s.updateSystemMetrics()
	
	// 立即更新一次指标
	if err := s.UpdateMetrics(ctx); err != nil {
		logger.Errorf("初始指标更新失败: %v", err)
	}
	
	// 启动定时更新
	s.updateTicker = time.NewTicker(30 * time.Second)
	
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("监控服务收到停止信号")
				return
			case <-s.stopChan:
				logger.Info("监控服务停止")
				return
			case <-s.updateTicker.C:
				if err := s.UpdateMetrics(ctx); err != nil {
					logger.Errorf("指标更新失败: %v", err)
				}
			}
		}
	}()
	
	return nil
}

// Stop 停止监控服务
func (s *monitoringService) Stop() error {
	logger.Info("🔍 监控服务停止")
	
	if s.updateTicker != nil {
		s.updateTicker.Stop()
	}
	
	close(s.stopChan)
	return nil
}

// UpdateMetrics 更新指标
func (s *monitoringService) UpdateMetrics(ctx context.Context) error {
	start := time.Now()
	
	// 更新用户相关指标
	if err := s.updateUserMetrics(ctx); err != nil {
		logger.Errorf("更新用户指标失败: %v", err)
	}
	
	// 更新提醒相关指标
	if err := s.updateReminderMetrics(ctx); err != nil {
		logger.Errorf("更新提醒指标失败: %v", err)
	}
	
	// 更新系统指标
	s.updateSystemMetrics()
	
	duration := time.Since(start)
	logger.Debugf("📊 指标更新完成，耗时: %v", duration)
	
	return nil
}

// updateUserMetrics 更新用户指标
func (s *monitoringService) updateUserMetrics(ctx context.Context) error {
	// 获取用户总数
	userCount, err := s.userRepo.Count(ctx)
	if err != nil {
		return err
	}
	
	metrics.SetBotUsers(float64(userCount))
	
	return nil
}

// updateReminderMetrics 更新提醒指标
func (s *monitoringService) updateReminderMetrics(ctx context.Context) error {
	// 获取活跃提醒数
	activeCount, err := s.reminderRepo.CountByStatus(ctx, models.ReminderStatStatusActive)
	if err != nil {
		return err
	}
	
	// 获取已完成提醒数
	completedCount, err := s.reminderRepo.CountByStatus(ctx, models.ReminderStatStatusCompleted)
	if err != nil {
		return err
	}
	
	// 获取已过期提醒数
	expiredCount, err := s.reminderRepo.CountByStatus(ctx, models.ReminderStatStatusExpired)
	if err != nil {
		return err
	}
	
	metrics.SetReminders("active", float64(activeCount))
	metrics.SetReminders("completed", float64(completedCount))
	metrics.SetReminders("expired", float64(expiredCount))
	
	return nil
}

// updateSystemMetrics 更新系统指标
func (s *monitoringService) updateSystemMetrics() {
	// 更新系统运行时间
	uptime := time.Since(s.startTime).Seconds()
	metrics.SetSystemUptime(uptime)
}

// RecordReminderOperation 记录提醒操作
func (s *monitoringService) RecordReminderOperation(operation string, status bool) {
	switch operation {
	case "created":
		if status {
			metrics.RecordReminderCreated()
		}
	case "completed":
		if status {
			metrics.RecordReminderCompleted()
		}
	case "skipped":
		if status {
			metrics.RecordReminderSkipped()
		}
	}
}

// RecordDatabaseOperation 记录数据库操作
func (s *monitoringService) RecordDatabaseOperation(operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	
	metrics.RecordDatabaseQuery(operation, status)
	metrics.RecordDatabaseQueryDuration(operation, duration.Seconds())
}

// RecordNotificationSend 记录通知发送
func (s *monitoringService) RecordNotificationSend(notificationType string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	
	metrics.RecordNotification(notificationType, status)
	metrics.RecordNotificationSend(notificationType, status, duration.Seconds())
}

// RecordBotMessage 记录Bot消息
func (s *monitoringService) RecordBotMessage(messageType string, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	
	metrics.RecordBotMessage(messageType, status)
}

// RecordReminderParse 记录提醒解析
func (s *monitoringService) RecordReminderParse(parserType string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	
	metrics.RecordReminderParse(parserType, status, duration.Seconds())
}