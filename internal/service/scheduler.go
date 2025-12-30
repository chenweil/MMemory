package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

type schedulerService struct {
	cron                *cron.Cron
	location            *time.Location
	reminderRepo        interfaces.ReminderRepository
	reminderLogRepo     interfaces.ReminderLogRepository
	notificationService NotificationService
	jobs                map[uint]cron.EntryID
	onceTimers          map[uint]*time.Timer
	mu                  sync.RWMutex

	// 动态工作池
	workerPool          chan func()
	activeWorkers       int32
	minWorkers          int
	maxWorkers          int

	// 监控统计
	stats               SchedulerStats
	statsMu             sync.RWMutex
}

type SchedulerStats struct {
	TotalTasks         int64         // 总任务数
	CompletedTasks     int64         // 完成的任务数
	FailedTasks        int64         // 失败的任务数
	ActiveWorkers      int32         // 当前活跃工作线程数
	QueueSize          int           // 当前队列大小
	LastTaskTime       time.Time     // 最后执行任务时间
	AverageTaskLatency time.Duration // 平均任务延迟
}

func NewSchedulerService(
	reminderRepo interfaces.ReminderRepository,
	reminderLogRepo interfaces.ReminderLogRepository,
	notificationService NotificationService,
) SchedulerService {
	// 使用北京时区
	loc, _ := time.LoadLocation("Asia/Shanghai")

	// 默认工作池配置
	maxWorkers := 10
	minWorkers := 2
	workQueueSize := 100

	return &schedulerService{
		cron:                cron.New(cron.WithLocation(loc)),
		location:            loc,
		reminderRepo:        reminderRepo,
		reminderLogRepo:     reminderLogRepo,
		notificationService: notificationService,
		jobs:                make(map[uint]cron.EntryID),
		onceTimers:          make(map[uint]*time.Timer),
		workerPool:          make(chan func(), workQueueSize),
		activeWorkers:       0,
		minWorkers:          minWorkers,
		maxWorkers:          maxWorkers,
		stats: SchedulerStats{
			QueueSize: 0,
		},
	}
}

// NewSchedulerServiceWithConfig 创建带配置的调度服务
func NewSchedulerServiceWithConfig(
	reminderRepo interfaces.ReminderRepository,
	reminderLogRepo interfaces.ReminderLogRepository,
	notificationService NotificationService,
	maxWorkers, minWorkers, workQueueSize int,
) SchedulerService {
	loc, _ := time.LoadLocation("Asia/Shanghai")

	return &schedulerService{
		cron:                cron.New(cron.WithLocation(loc)),
		location:            loc,
		reminderRepo:        reminderRepo,
		reminderLogRepo:     reminderLogRepo,
		notificationService: notificationService,
		jobs:                make(map[uint]cron.EntryID),
		onceTimers:          make(map[uint]*time.Timer),
		workerPool:          make(chan func(), workQueueSize),
		activeWorkers:       0,
		minWorkers:          minWorkers,
		maxWorkers:          maxWorkers,
		stats: SchedulerStats{
			QueueSize: workQueueSize,
		},
	}
}

func (s *schedulerService) Start() error {
	logger.Info("🕰️ 定时调度器启动中...")

	// 启动cron调度器
	s.cron.Start()

	// 从数据库恢复所有有效提醒
	ctx := context.Background()
	reminders, err := s.reminderRepo.GetActiveReminders(ctx)
	if err != nil {
		return fmt.Errorf("获取有效提醒失败: %w", err)
	}

	// 为每个提醒添加调度任务
	for _, reminder := range reminders {
		if err := s.AddReminder(reminder); err != nil {
			logger.Errorf("添加提醒调度失败 (ID: %d): %v", reminder.ID, err)
			continue
		}
	}

	logger.Infof("✅ 定时调度器启动成功，已加载 %d 个提醒", len(reminders))
	return nil
}

func (s *schedulerService) Stop() error {
	logger.Info("🔄 定时调度器停止中...")
	s.cron.Stop()
	s.mu.Lock()
	for id, timer := range s.onceTimers {
		if timer != nil {
			timer.Stop()
		}
		delete(s.onceTimers, id)
	}
	s.jobs = make(map[uint]cron.EntryID)
	s.mu.Unlock()
	logger.Info("✅ 定时调度器已停止")
	return nil
}

func (s *schedulerService) AddReminder(reminder *models.Reminder) error {
	if reminder == nil {
		return fmt.Errorf("提醒信息不能为空")
	}

	if !reminder.IsActive {
		return fmt.Errorf("提醒未激活，无法添加调度")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果存在旧的定时器/任务，先清理
	s.clearReminderLocked(reminder.ID)

	if reminder.IsPaused() {
		logger.Debugf("⏸️ 提醒处于暂停状态，跳过调度: ID=%d", reminder.ID)
		return nil
	}

	if reminder.IsOnce() {
		return s.addOnceReminderLocked(reminder)
	}

	cronExpr, err := s.buildCronExpression(reminder)
	if err != nil {
		return fmt.Errorf("构建cron表达式失败: %w", err)
	}

	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeReminder(reminder.ID)
	})
	if err != nil {
		return fmt.Errorf("添加cron任务失败: %w", err)
	}

	s.jobs[reminder.ID] = entryID

	logger.Debugf("📅 添加提醒调度: ID=%d, Cron=%s", reminder.ID, cronExpr)
	return nil
}

func (s *schedulerService) RemoveReminder(reminderID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := s.clearReminderLocked(reminderID)

	if removed {
		logger.Debugf("🗑️ 移除提醒调度: ID=%d", reminderID)
		return nil
	}

	return fmt.Errorf("提醒调度不存在: %d", reminderID)
}

func (s *schedulerService) RefreshSchedules() error {
	logger.Info("🔄 刷新所有调度任务...")

	// 停止所有现有任务
	s.mu.Lock()
	for id, entry := range s.jobs {
		s.cron.Remove(entry)
		delete(s.jobs, id)
	}
	for id, timer := range s.onceTimers {
		if timer != nil {
			timer.Stop()
		}
		delete(s.onceTimers, id)
	}
	s.mu.Unlock()

	// 重新加载所有有效提醒
	ctx := context.Background()
	reminders, err := s.reminderRepo.GetActiveReminders(ctx)
	if err != nil {
		return fmt.Errorf("获取有效提醒失败: %w", err)
	}

	// 重新添加所有任务
	for _, reminder := range reminders {
		if err := s.AddReminder(reminder); err != nil {
			logger.Errorf("重新添加提醒调度失败 (ID: %d): %v", reminder.ID, err)
			continue
		}
	}

	s.mu.RLock()
	activeJobs := len(s.jobs) + len(s.onceTimers)
	s.mu.RUnlock()

	logger.Infof("✅ 调度任务刷新完成，当前活跃任务: %d", activeJobs)
	return nil
}

// buildCronExpression 根据提醒配置构建cron表达式
func (s *schedulerService) buildCronExpression(reminder *models.Reminder) (string, error) {
	// 解析目标时间
	timeParts := strings.Split(reminder.TargetTime, ":")
	if len(timeParts) < 2 {
		return "", fmt.Errorf("无效的时间格式: %s", reminder.TargetTime)
	}

	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return "", fmt.Errorf("无效的小时: %s", timeParts[0])
	}

	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return "", fmt.Errorf("无效的分钟: %s", timeParts[1])
	}

	// 根据调度模式构建表达式
	switch {
	case reminder.IsDaily():
		// 每天指定时间：分 时 * * *
		return fmt.Sprintf("%02d %d * * *", minute, hour), nil

	case reminder.IsWeekly():
		// 解析星期几
		weekdays, err := s.parseWeeklyPattern(reminder.SchedulePattern)
		if err != nil {
			return "", err
		}
		// 每周指定天：分 时 * * 星期
		return fmt.Sprintf("%02d %d * * %s", minute, hour, strings.Join(weekdays, ",")), nil

	case reminder.IsOnce():
		// 一次性提醒需要特殊处理
		return s.buildOnceExpression(reminder.SchedulePattern, hour, minute)

	default:
		return "", fmt.Errorf("不支持的调度模式: %s", reminder.SchedulePattern)
	}
}

// parseWeeklyPattern 解析每周模式 "weekly:1,3,5"
func (s *schedulerService) parseWeeklyPattern(pattern string) ([]string, error) {
	if !strings.HasPrefix(pattern, "weekly:") {
		return nil, fmt.Errorf("无效的每周模式: %s", pattern)
	}

	weekdaysStr := strings.TrimPrefix(pattern, "weekly:")
	weekdays := strings.Split(weekdaysStr, ",")

	// 验证星期数字有效性
	for _, weekday := range weekdays {
		day, err := strconv.Atoi(strings.TrimSpace(weekday))
		if err != nil || day < 0 || day > 7 {
			return nil, fmt.Errorf("无效的星期数字: %s", weekday)
		}
	}

	return weekdays, nil
}

func (s *schedulerService) addOnceReminderLocked(reminder *models.Reminder) error {
	timeParts := strings.Split(reminder.TargetTime, ":")
	if len(timeParts) < 2 {
		return fmt.Errorf("无效的时间格式: %s", reminder.TargetTime)
	}

	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return fmt.Errorf("无效的小时: %s", timeParts[0])
	}

	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return fmt.Errorf("无效的分钟: %s", timeParts[1])
	}

	targetTime, err := s.parseOnceTargetTime(reminder.SchedulePattern, hour, minute)
	if err != nil {
		return err
	}

	delay := time.Until(targetTime)
	if delay <= 0 {
		return fmt.Errorf("目标时间已过期: %v", targetTime)
	}

	timer := time.AfterFunc(delay, func() {
		s.executeReminder(reminder.ID)
	})

	s.onceTimers[reminder.ID] = timer
	logger.Debugf("⏰ 一次性提醒定时器已创建: ID=%d, 触发时间=%s", reminder.ID, targetTime.Format(time.RFC3339))
	return nil
}

func (s *schedulerService) parseOnceTargetTime(pattern string, hour, minute int) (time.Time, error) {
	if !strings.HasPrefix(pattern, string(models.SchedulePatternOnce)) {
		return time.Time{}, fmt.Errorf("无效的一次性模式: %s", pattern)
	}

	dateStr := strings.TrimPrefix(pattern, string(models.SchedulePatternOnce))
	loc := s.location
	if loc == nil {
		loc = time.Local
	}

	date, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("无效的日期格式: %s", dateStr)
	}

	targetTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)
	currentTime := time.Now().In(loc)
	if !targetTime.After(currentTime) {
		return time.Time{}, fmt.Errorf("目标时间已过期: %v", targetTime)
	}

	return targetTime, nil
}

func (s *schedulerService) clearReminderLocked(reminderID uint) bool {
	removed := false

	if entryID, exists := s.jobs[reminderID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, reminderID)
		removed = true
	}

	if timer, exists := s.onceTimers[reminderID]; exists {
		if timer != nil {
			timer.Stop()
		}
		delete(s.onceTimers, reminderID)
		removed = true
	}

	return removed
}

// buildOnceExpression 构建一次性提醒表达式
func (s *schedulerService) buildOnceExpression(pattern string, hour, minute int) (string, error) {
	targetTime, err := s.parseOnceTargetTime(pattern, hour, minute)
	if err != nil {
		return "", err
	}

	// 一次性任务：分 时 日 月 *
	return fmt.Sprintf("%02d %d %d %d *", minute, hour, targetTime.Day(), int(targetTime.Month())), nil
}

// executeReminder 执行提醒任务
func (s *schedulerService) executeReminder(reminderID uint) {
	startTime := time.Now()
	ctx := context.Background()

	logger.Debugf("⏰ 执行提醒任务: ID=%d", reminderID)

	// 获取提醒详情
	reminder, err := s.reminderRepo.GetByID(ctx, reminderID)
	if err != nil {
		logger.Errorf("获取提醒失败 (ID: %d): %v", reminderID, err)
		return
	}

	if reminder == nil || !reminder.IsActive {
		logger.Warnf("提醒不存在或已禁用 (ID: %d)", reminderID)
		return
	}

	// 创建提醒记录
	reminderLog := &models.ReminderLog{
		ReminderID:    reminderID,
		ScheduledTime: time.Now(),
		Status:        models.ReminderStatusPending,
	}

	if err := s.reminderLogRepo.Create(ctx, reminderLog); err != nil {
		logger.Errorf("创建提醒记录失败 (ID: %d): %v", reminderID, err)
		return
	}

	// 重新加载提醒记录，确保包含提醒与用户信息
	if reminderLog, err = s.reminderLogRepo.GetByID(ctx, reminderLog.ID); err != nil {
		logger.Errorf("加载提醒记录失败 (ID: %d): %v", reminderID, err)
		return
	}
	if reminderLog == nil {
		logger.Errorf("提醒记录不存在 (ID: %d)", reminderID)
		return
	}

	// 发送提醒通知
	if err := s.notificationService.SendReminder(ctx, reminderLog); err != nil {
		logger.Errorf("发送提醒通知失败 (ID: %d): %v", reminderID, err)
		return
	}

	// 更新提醒记录状态
	reminderLog.MarkAsSent()
	if err := s.reminderLogRepo.Update(ctx, reminderLog); err != nil {
		logger.Errorf("更新提醒记录失败 (ID: %d): %v", reminderID, err)
	}

	// 如果是一次性提醒，完成后禁用
	if reminder.IsOnce() {
		reminder.IsActive = false
		if err := s.reminderRepo.Update(ctx, reminder); err != nil {
			logger.Errorf("禁用一次性提醒失败 (ID: %d): %v", reminderID, err)
		} else {
			s.RemoveReminder(reminderID)
			logger.Infof("✅ 一次性提醒已完成并禁用 (ID: %d)", reminderID)
		}
	}

	// 更新统计
	s.updateStats(true, time.Since(startTime))
}

// submitTask 提交任务到工作池
func (s *schedulerService) submitTask(task func()) bool {
	select {
	case s.workerPool <- task:
		s.statsMu.Lock()
		s.stats.QueueSize = len(s.workerPool)
		s.stats.TotalTasks++
		s.statsMu.Unlock()
		s.adjustWorkerPool()
		return true
	default:
		logger.Warn("调度器工作队列已满，任务被丢弃")
		s.statsMu.Lock()
		s.stats.FailedTasks++
		s.statsMu.Unlock()
		return false
	}
}

// adjustWorkerPool 动态调整工作池大小
func (s *schedulerService) adjustWorkerPool() {
	currentQueueSize := len(s.workerPool)
	activeWorkers := atomic.LoadInt32(&s.activeWorkers)

	// 队列拥堵时增加工作线程
	if currentQueueSize > int(activeWorkers) && activeWorkers < int32(s.maxWorkers) {
		go s.startWorker()
	}

	// 队列空闲且有多余线程时减少工作线程
	if currentQueueSize == 0 && activeWorkers > int32(s.minWorkers) {
		// 渐进式减少，不直接减少，让空闲线程自然退出
	}
}

// startWorker 启动一个工作线程
func (s *schedulerService) startWorker() {
	atomic.AddInt32(&s.activeWorkers, 1)
	defer atomic.AddInt32(&s.activeWorkers, -1)

	for task := range s.workerPool {
		start := time.Now()
		task()
		s.updateStats(true, time.Since(start))
	}
}

// updateStats 更新统计信息
func (s *schedulerService) updateStats(success bool, latency time.Duration) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.stats.LastTaskTime = time.Now()
	if success {
		s.stats.CompletedTasks++
	} else {
		s.stats.FailedTasks++
	}

	// 更新平均延迟
	totalTasks := s.stats.CompletedTasks + s.stats.FailedTasks
	if totalTasks > 1 {
		oldAvg := s.stats.AverageTaskLatency
		s.stats.AverageTaskLatency = oldAvg + (latency-oldAvg)/time.Duration(totalTasks)
	} else {
		s.stats.AverageTaskLatency = latency
	}

	s.stats.QueueSize = len(s.workerPool)
	s.stats.ActiveWorkers = atomic.LoadInt32(&s.activeWorkers)
}

// GetStats 获取调度器统计信息
func (s *schedulerService) GetStats() SchedulerStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	stats := s.stats
	stats.QueueSize = len(s.workerPool)
	stats.ActiveWorkers = atomic.LoadInt32(&s.activeWorkers)
	return stats
}
