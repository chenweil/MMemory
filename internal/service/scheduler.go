package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

type schedulerService struct {
	cron                *cron.Cron
	reminderRepo        interfaces.ReminderRepository
	reminderLogRepo     interfaces.ReminderLogRepository
	notificationService NotificationService
	jobs                map[uint]cron.EntryID // reminder_id -> cron_entry_id
}

func NewSchedulerService(
	reminderRepo interfaces.ReminderRepository,
	reminderLogRepo interfaces.ReminderLogRepository,
	notificationService NotificationService,
) SchedulerService {
	// 使用北京时区
	loc, _ := time.LoadLocation("Asia/Shanghai")
	
	return &schedulerService{
		cron:                cron.New(cron.WithLocation(loc)),
		reminderRepo:        reminderRepo,
		reminderLogRepo:     reminderLogRepo,
		notificationService: notificationService,
		jobs:                make(map[uint]cron.EntryID),
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
	s.jobs = make(map[uint]cron.EntryID)
	logger.Info("✅ 定时调度器已停止")
	return nil
}

func (s *schedulerService) AddReminder(reminder *models.Reminder) error {
	if !reminder.IsActive {
		return fmt.Errorf("提醒未激活，无法添加调度")
	}
	
	cronExpr, err := s.buildCronExpression(reminder)
	if err != nil {
		return fmt.Errorf("构建cron表达式失败: %w", err)
	}
	
	// 添加cron任务
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeReminder(reminder.ID)
	})
	if err != nil {
		return fmt.Errorf("添加cron任务失败: %w", err)
	}
	
	// 记录任务ID
	s.jobs[reminder.ID] = entryID
	
	logger.Debugf("📅 添加提醒调度: ID=%d, Cron=%s", reminder.ID, cronExpr)
	return nil
}

func (s *schedulerService) RemoveReminder(reminderID uint) error {
	entryID, exists := s.jobs[reminderID]
	if !exists {
		return fmt.Errorf("提醒调度不存在: %d", reminderID)
	}
	
	s.cron.Remove(entryID)
	delete(s.jobs, reminderID)
	
	logger.Debugf("🗑️ 移除提醒调度: ID=%d", reminderID)
	return nil
}

func (s *schedulerService) RefreshSchedules() error {
	logger.Info("🔄 刷新所有调度任务...")
	
	// 停止所有现有任务
	for reminderID := range s.jobs {
		s.RemoveReminder(reminderID)
	}
	
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
	
	logger.Infof("✅ 调度任务刷新完成，当前活跃任务: %d", len(s.jobs))
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
		// 每天指定时间：秒 分 时 * * *
		return fmt.Sprintf("0 %d %d * * *", minute, hour), nil
		
	case reminder.IsWeekly():
		// 解析星期几
		weekdays, err := s.parseWeeklyPattern(reminder.SchedulePattern)
		if err != nil {
			return "", err
		}
		// 每周指定天：秒 分 时 * * 星期
		return fmt.Sprintf("0 %d %d * * %s", minute, hour, strings.Join(weekdays, ",")), nil
		
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

// buildOnceExpression 构建一次性提醒表达式
func (s *schedulerService) buildOnceExpression(pattern string, hour, minute int) (string, error) {
	if !strings.HasPrefix(pattern, "once:") {
		return "", fmt.Errorf("无效的一次性模式: %s", pattern)
	}
	
	dateStr := strings.TrimPrefix(pattern, "once:")
	
	// 解析日期
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("无效的日期格式: %s", dateStr)
	}
	
	// 检查日期是否在未来
	targetTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())
	if targetTime.Before(time.Now()) {
		return "", fmt.Errorf("目标时间已过期: %v", targetTime)
	}
	
	// 一次性任务：秒 分 时 日 月 *
	return fmt.Sprintf("0 %d %d %d %d *", minute, hour, date.Day(), int(date.Month())), nil
}

// executeReminder 执行提醒任务
func (s *schedulerService) executeReminder(reminderID uint) {
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
}
