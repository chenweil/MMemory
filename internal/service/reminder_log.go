package service

import (
	"context"
	"fmt"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type reminderLogService struct {
	reminderLogRepo interfaces.ReminderLogRepository
	reminderRepo    interfaces.ReminderRepository
}

func NewReminderLogService(
	reminderLogRepo interfaces.ReminderLogRepository,
	reminderRepo interfaces.ReminderRepository,
) ReminderLogService {
	return &reminderLogService{
		reminderLogRepo: reminderLogRepo,
		reminderRepo:    reminderRepo,
	}
}

func (s *reminderLogService) GetByID(ctx context.Context, id uint) (*models.ReminderLog, error) {
	return s.reminderLogRepo.GetByID(ctx, id)
}

func (s *reminderLogService) MarkAsCompleted(ctx context.Context, id uint, response string) error {
	log, err := s.reminderLogRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取提醒记录失败: %w", err)
	}
	
	if log == nil {
		return fmt.Errorf("提醒记录不存在")
	}
	
	log.MarkAsCompleted(response)
	return s.reminderLogRepo.Update(ctx, log)
}

func (s *reminderLogService) MarkAsSkipped(ctx context.Context, id uint, response string) error {
	log, err := s.reminderLogRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取提醒记录失败: %w", err)
	}
	
	if log == nil {
		return fmt.Errorf("提醒记录不存在")
	}
	
	log.MarkAsSkipped(response)
	return s.reminderLogRepo.Update(ctx, log)
}

func (s *reminderLogService) CreateDelayReminder(ctx context.Context, originalLogID uint, delayTime time.Time, hours int) error {
	// 获取原始提醒记录
	originalLog, err := s.reminderLogRepo.GetByID(ctx, originalLogID)
	if err != nil {
		return fmt.Errorf("获取原始提醒记录失败: %w", err)
	}
	
	if originalLog == nil {
		return fmt.Errorf("原始提醒记录不存在")
	}
	
	// 标记原记录为已延期
	originalLog.Status = models.ReminderStatusSkipped
	originalLog.UserResponse = fmt.Sprintf("延期%d小时", hours)
	now := time.Now()
	originalLog.ResponseTime = &now
	
	if err := s.reminderLogRepo.Update(ctx, originalLog); err != nil {
		return fmt.Errorf("更新原始记录失败: %w", err)
	}
	
	// 创建新的延期提醒记录
	delayLog := &models.ReminderLog{
		ReminderID:    originalLog.ReminderID,
		ScheduledTime: delayTime,
		Status:        models.ReminderStatusPending,
	}
	
	return s.reminderLogRepo.Create(ctx, delayLog)
}

func (s *reminderLogService) GetOverdueReminders(ctx context.Context) ([]*models.ReminderLog, error) {
	// 获取所有已发送但未回复的提醒
	allLogs, err := s.reminderLogRepo.GetPendingLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取待处理提醒失败: %w", err)
	}
	
	var overdueLogs []*models.ReminderLog
	now := time.Now()
	
	for _, log := range allLogs {
		// 检查是否已发送且超时（发送后1小时未回复）
		if log.Status == models.ReminderStatusSent && 
		   log.SentTime != nil && 
		   now.Sub(*log.SentTime) > time.Hour {
			overdueLogs = append(overdueLogs, log)
		}
	}
	
	return overdueLogs, nil
}

// UpdateFollowUpCount 更新关怀次数
func (s *reminderLogService) UpdateFollowUpCount(ctx context.Context, id uint) error {
	log, err := s.reminderLogRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取提醒记录失败: %w", err)
	}
	
	if log == nil {
		return fmt.Errorf("提醒记录不存在")
	}
	
	log.FollowUpCount++
	return s.reminderLogRepo.Update(ctx, log)
}

// GetUserStatistics 获取用户统计数据
func (s *reminderLogService) GetUserStatistics(ctx context.Context, userID uint) (*UserStatistics, error) {
	// 获取用户的所有提醒
	reminders, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户提醒失败: %w", err)
	}
	
	stats := &UserStatistics{
		TotalReminders: len(reminders),
	}
	
	// 统计活跃提醒数
	for _, reminder := range reminders {
		if reminder.IsActive {
			stats.ActiveReminders++
		}
	}
	
	// 获取时间范围
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(now.Weekday())+1) // 本周一
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	
	// 统计各时期的完成情况
	for _, reminder := range reminders {
		logs, err := s.reminderLogRepo.GetByReminderID(ctx, reminder.ID, 0, 0)
		if err != nil {
			continue
		}
		
		for _, log := range logs {
			if log.ResponseTime == nil {
				continue
			}
			
			responseTime := *log.ResponseTime
			
			// 今日统计
			if responseTime.After(todayStart) {
				if log.Status == models.ReminderStatusCompleted {
					stats.CompletedToday++
				} else if log.Status == models.ReminderStatusSkipped {
					stats.SkippedToday++
				}
			}
			
			// 本周统计
			if responseTime.After(weekStart) {
				if log.Status == models.ReminderStatusCompleted {
					stats.CompletedWeek++
				}
			}
			
			// 本月统计
			if responseTime.After(monthStart) {
				if log.Status == models.ReminderStatusCompleted {
					stats.CompletedMonth++
				}
			}
		}
	}
	
	// 计算完成率 (本月数据)
	totalThisMonth := stats.CompletedMonth + countSkippedThisMonth(reminders, s.reminderLogRepo, ctx, monthStart)
	if totalThisMonth > 0 {
		stats.CompletionRate = (stats.CompletedMonth * 100) / totalThisMonth
	}
	
	// TODO: 计算连续天数 (需要更复杂的逻辑)
	stats.CurrentStreak = 0
	stats.LongestStreak = 0
	
	return stats, nil
}

// 辅助函数：统计本月跳过的数量
func countSkippedThisMonth(reminders []*models.Reminder, repo interfaces.ReminderLogRepository, ctx context.Context, monthStart time.Time) int {
	count := 0
	for _, reminder := range reminders {
		logs, err := repo.GetByReminderID(ctx, reminder.ID, 0, 0)
		if err != nil {
			continue
		}
		
		for _, log := range logs {
			if log.ResponseTime != nil && 
			   log.ResponseTime.After(monthStart) && 
			   log.Status == models.ReminderStatusSkipped {
				count++
			}
		}
	}
	return count
}