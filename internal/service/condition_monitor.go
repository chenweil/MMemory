package service

import (
	"context"
	"sync"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/logger"

	"github.com/sirupsen/logrus"
)

// ConditionMonitor 条件监控器
type ConditionMonitor struct {
	conditionSvc    *ConditionService
	reminderSvc     ReminderService
	notificationSvc NotificationService
	checkInterval   time.Duration
	activeMonitors  map[uint]*ConditionalMonitor
	mu              sync.RWMutex
	stopCh          chan struct{}
	wg              sync.WaitGroup
	log             *logrus.Logger
}

// ConditionalMonitor 单个条件监控
type ConditionalMonitor struct {
	ReminderID   uint
	Conditions   []models.ReminderCondition
	LastCheck    time.Time
	LastResult   *models.ConditionEvaluationStatus
	ResultCh     chan *models.ConditionEvaluationStatus
}

// NewConditionMonitor 创建条件监控器
func NewConditionMonitor(
	conditionSvc *ConditionService,
	reminderSvc ReminderService,
	notificationSvc NotificationService,
) *ConditionMonitor {
	return &ConditionMonitor{
		conditionSvc:    conditionSvc,
		reminderSvc:     reminderSvc,
		notificationSvc: notificationSvc,
		checkInterval:   5 * time.Minute,
		activeMonitors:  make(map[uint]*ConditionalMonitor),
		stopCh:          make(chan struct{}),
		log:             logger.GetLogger(),
	}
}

// Start 启动监控器
func (cm *ConditionMonitor) Start(ctx context.Context) {
	cm.wg.Add(1)
	go cm.run(ctx)
}

// Stop 停止监控器
func (cm *ConditionMonitor) Stop() {
	close(cm.stopCh)
	cm.wg.Wait()
}

// run 运行监控循环
func (cm *ConditionMonitor) run(ctx context.Context) {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.checkAllConditions(ctx)
		}
	}
}

// checkAllConditions 检查所有活跃监控
func (cm *ConditionMonitor) checkAllConditions(ctx context.Context) {
	cm.mu.RLock()
	monitors := make([]*ConditionalMonitor, 0, len(cm.activeMonitors))
	for _, m := range cm.activeMonitors {
		monitors = append(monitors, m)
	}
	cm.mu.RUnlock()

	for _, monitor := range monitors {
		result, err := cm.conditionSvc.EvaluateConditions(ctx, monitor.ReminderID)
		if err != nil {
			cm.log.Errorf("检查提醒 %d 条件失败: %v", monitor.ReminderID, err)
			continue
		}

		monitor.LastCheck = time.Now()
		monitor.LastResult = result

		// 发送结果到通道
		select {
		case monitor.ResultCh <- result:
		default:
		}

		// 如果所有条件满足，触发提醒
		if result.AllConditionsMet {
			cm.triggerReminder(ctx, monitor.ReminderID)
		}
	}
}

// triggerReminder 触发提醒
func (cm *ConditionMonitor) triggerReminder(ctx context.Context, reminderID uint) {
	reminder, err := cm.reminderSvc.GetReminderByID(ctx, reminderID)
	if err != nil {
		cm.log.Errorf("获取提醒失败: %v", err)
		return
	}

	if cm.notificationSvc != nil {
		// 创建临时的 ReminderLog 来触发通知
		log := &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: time.Now(),
			Status:        models.ReminderStatusPending,
		}
		_ = cm.notificationSvc.SendReminder(ctx, log)
	}
}

// StartMonitoring 开始监控提醒
func (cm *ConditionMonitor) StartMonitoring(reminderID uint, conditions []models.ReminderCondition) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.activeMonitors[reminderID]; exists {
		return
	}

	monitor := &ConditionalMonitor{
		ReminderID: reminderID,
		Conditions: conditions,
		ResultCh:   make(chan *models.ConditionEvaluationStatus, 10),
	}

	cm.activeMonitors[reminderID] = monitor
}

// StopMonitoring 停止监控提醒
func (cm *ConditionMonitor) StopMonitoring(reminderID uint) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if monitor, exists := cm.activeMonitors[reminderID]; exists {
		close(monitor.ResultCh)
		delete(cm.activeMonitors, reminderID)
	}
}

// GetMonitorStatus 获取监控状态
func (cm *ConditionMonitor) GetMonitorStatus(reminderID uint) (*models.ConditionEvaluationStatus, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if monitor, exists := cm.activeMonitors[reminderID]; exists {
		return monitor.LastResult, true
	}
	return nil, false
}

// GetAllActiveMonitors 获取所有活跃监控
func (cm *ConditionMonitor) GetAllActiveMonitors() []uint {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ids := make([]uint, 0, len(cm.activeMonitors))
	for id := range cm.activeMonitors {
		ids = append(ids, id)
	}
	return ids
}

// SetCheckInterval 设置检查间隔
func (cm *ConditionMonitor) SetCheckInterval(interval time.Duration) {
	cm.checkInterval = interval
}

// ConditionReminderService 条件提醒服务
type ConditionReminderService struct {
	conditionSvc     *ConditionService
	conditionMonitor *ConditionMonitor
	reminderSvc      ReminderService
	conditionRepo    interface {
		GetByReminderID(ctx context.Context, reminderID uint) ([]models.ReminderCondition, error)
		Create(ctx context.Context, condition *models.ReminderCondition) error
		DeleteByReminderID(ctx context.Context, reminderID uint) error
	}
	log *logrus.Logger
}

// NewConditionReminderService 创建条件提醒服务
func NewConditionReminderService(
	conditionSvc *ConditionService,
	conditionMonitor *ConditionMonitor,
	reminderSvc ReminderService,
) *ConditionReminderService {
	return &ConditionReminderService{
		conditionSvc:     conditionSvc,
		conditionMonitor: conditionMonitor,
		reminderSvc:      reminderSvc,
		log:              logger.GetLogger(),
	}
}

// AddConditionToReminder 为提醒添加条件
func (crs *ConditionReminderService) AddConditionToReminder(ctx context.Context, reminderID uint, condition *models.ReminderCondition) error {
	reminder, err := crs.reminderSvc.GetReminderByID(ctx, reminderID)
	if err != nil {
		return err
	}
	if reminder == nil {
		return ErrReminderNotFound
	}

	condition.ReminderID = reminderID

	if err := crs.conditionRepo.Create(ctx, condition); err != nil {
		return err
	}

	conditions, _ := crs.conditionRepo.GetByReminderID(ctx, reminderID)
	crs.conditionMonitor.StartMonitoring(reminderID, conditions)

	return nil
}

// RemoveConditionsFromReminder 移除提醒的所有条件
func (crs *ConditionReminderService) RemoveConditionsFromReminder(ctx context.Context, reminderID uint) error {
	if err := crs.conditionRepo.DeleteByReminderID(ctx, reminderID); err != nil {
		return err
	}

	crs.conditionMonitor.StopMonitoring(reminderID)
	return nil
}

// GetReminderConditions 获取提醒的所有条件
func (crs *ConditionReminderService) GetReminderConditions(ctx context.Context, reminderID uint) ([]models.ReminderCondition, error) {
	return crs.conditionRepo.GetByReminderID(ctx, reminderID)
}

// CheckReminderConditions 检查提醒的条件状态
func (crs *ConditionReminderService) CheckReminderConditions(ctx context.Context, reminderID uint) (*models.ConditionEvaluationStatus, error) {
	return crs.conditionSvc.EvaluateConditions(ctx, reminderID)
}

// GetSupportedConditionTypes 获取支持的条件类型
func (crs *ConditionReminderService) GetSupportedConditionTypes() []models.ConditionType {
	return crs.conditionSvc.GetSupportedTypes()
}
