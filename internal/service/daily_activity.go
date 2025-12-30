package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

// DailyActivityService 日常活动服务接口
type DailyActivityService interface {
	RecordActivity(ctx context.Context, userID uint, activityType models.ActivityType, details map[string]interface{}, source models.ActivitySource) (*models.DailyActivity, error)
	GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error)
	GetActivitiesByType(ctx context.Context, userID uint, activityType models.ActivityType, limit int) ([]*models.DailyActivity, error)
	GetActivitiesByDateRange(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*models.DailyActivity, error)
	GetActivityStatistics(ctx context.Context, userID uint, timeRange string) (map[string]int64, error)
	QueryActivities(ctx context.Context, userID uint, queryType, activityType, timeRange string) (string, error)
}

type dailyActivityServiceImpl struct {
	activityRepo interfaces.DailyActivityRepository
}

// NewDailyActivityService 创建日常活动服务
func NewDailyActivityService(activityRepo interfaces.DailyActivityRepository) DailyActivityService {
	return &dailyActivityServiceImpl{
		activityRepo: activityRepo,
	}
}

// RecordActivity 记录活动
func (s *dailyActivityServiceImpl) RecordActivity(ctx context.Context, userID uint, activityType models.ActivityType, details map[string]interface{}, source models.ActivitySource) (*models.DailyActivity, error) {
	// 序列化详情
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		logger.Errorf("序列化活动详情失败: %v", err)
		return nil, fmt.Errorf("序列化详情失败: %w", err)
	}

	// 创建活动记录
	activity := &models.DailyActivity{
		UserID:       userID,
		ActivityType: activityType,
		OccurredAt:   time.Now(),
		Details:      string(detailsJSON),
		Source:       source,
	}

	// 保存到数据库
	if err := s.activityRepo.Create(ctx, activity); err != nil {
		logger.Errorf("保存活动记录失败: %v", err)
		return nil, fmt.Errorf("保存失败: %w", err)
	}

	logger.Infof("用户 %d 记录活动: %s", userID, activityType)
	return activity, nil
}

// GetRecentActivities 获取最近活动
func (s *dailyActivityServiceImpl) GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error) {
	return s.activityRepo.GetRecentActivities(ctx, userID, limit)
}

// GetActivitiesByType 按类型获取活动
func (s *dailyActivityServiceImpl) GetActivitiesByType(ctx context.Context, userID uint, activityType models.ActivityType, limit int) ([]*models.DailyActivity, error) {
	return s.activityRepo.GetByType(ctx, userID, activityType, limit, 0)
}

// GetActivitiesByDateRange 按日期范围获取活动
func (s *dailyActivityServiceImpl) GetActivitiesByDateRange(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*models.DailyActivity, error) {
	return s.activityRepo.GetByDateRange(ctx, userID, startTime, endTime)
}

// GetActivityStatistics 获取活动统计
func (s *dailyActivityServiceImpl) GetActivityStatistics(ctx context.Context, userID uint, timeRange string) (map[string]int64, error) {
	now := time.Now()
	var startTime, endTime time.Time

	// 解析时间范围
	switch timeRange {
	case "今天":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime = startTime.Add(24 * time.Hour)
	case "昨天":
		yesterday := now.AddDate(0, 0, -1)
		startTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		endTime = startTime.Add(24 * time.Hour)
	case "这周":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startTime = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 0, 7)
	case "最近7天":
		startTime = now.AddDate(0, 0, -7)
		endTime = now
	default:
		startTime = now.AddDate(0, 0, -7)
		endTime = now
	}

	return s.activityRepo.GetStatistics(ctx, userID, startTime, endTime)
}

// QueryActivities 查询活动并返回自然语言回复
func (s *dailyActivityServiceImpl) QueryActivities(ctx context.Context, userID uint, queryType, activityType, timeRange string) (string, error) {
	switch queryType {
	case "by_type":
		activities, err := s.GetActivitiesByType(ctx, userID, models.ActivityType(activityType), 10)
		if err != nil {
			return "", err
		}
		return s.formatActivitiesByType(activities, activityType), nil

	case "by_time":
		now := time.Now()
		var startTime, endTime time.Time
		if timeRange == "昨天" {
			yesterday := now.AddDate(0, 0, -1)
			startTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
			endTime = startTime.Add(24 * time.Hour)
		} else if timeRange == "今天" {
			startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endTime = startTime.Add(24 * time.Hour)
		}
		activities, err := s.GetActivitiesByDateRange(ctx, userID, startTime, endTime)
		if err != nil {
			return "", err
		}
		return s.formatActivitiesByTime(activities, timeRange), nil

	case "statistics":
		stats, err := s.GetActivityStatistics(ctx, userID, timeRange)
		if err != nil {
			return "", err
		}
		return s.formatStatistics(stats, timeRange), nil

	default:
		return "抱歉，我不理解这个查询", nil
	}
}

func (s *dailyActivityServiceImpl) formatActivitiesByType(activities []*models.DailyActivity, activityType string) string {
	if len(activities) == 0 {
		return "没有找到相关记录"
	}

	switch activityType {
	case "read_book":
		books := make(map[string]string)
		for _, act := range activities {
			details, _ := act.GetDetails()
			if details.BookName != "" {
				chapter := details.Chapter
				if chapter == "" {
					chapter = "未知章节"
				}
				books[details.BookName] = chapter
			}
		}
		result := "根据记录，你最近看过：\n"
		for book, chapter := range books {
			result += fmt.Sprintf("- 《%s》（读到%s）\n", book, chapter)
		}
		return result

	case "drink_water":
		total := len(activities)
		return fmt.Sprintf("最近记录了 %d 次喝水", total)

	case "exercise":
		total := len(activities)
		return fmt.Sprintf("最近运动了 %d 次", total)

	case "take_medicine":
		total := len(activities)
		return fmt.Sprintf("最近记录了 %d 次吃药", total)

	default:
		return fmt.Sprintf("最近记录了 %d 次相关活动", len(activities))
	}
}

func (s *dailyActivityServiceImpl) formatActivitiesByTime(activities []*models.DailyActivity, timeRange string) string {
	if len(activities) == 0 {
		return fmt.Sprintf("%s没有活动记录", timeRange)
	}

	result := fmt.Sprintf("%s的活动记录：\n", timeRange)
	for _, act := range activities {
		timeStr := act.OccurredAt.Format("15:04")
		displayName := getActivityTypeDisplayName(act.ActivityType)
		result += fmt.Sprintf("- %s %s\n", timeStr, displayName)
	}
	return result
}

func (s *dailyActivityServiceImpl) formatStatistics(stats map[string]int64, timeRange string) string {
	if len(stats) == 0 {
		return fmt.Sprintf("%s没有活动统计", timeRange)
	}

	result := fmt.Sprintf("%s的活动统计：\n", timeRange)
	for activityType, count := range stats {
		displayName := getActivityTypeDisplayName(models.ActivityType(activityType))
		result += fmt.Sprintf("- %s: %d 次\n", displayName, count)
	}
	return result
}

// getActivityTypeDisplayName 获取活动类型的显示名称
func getActivityTypeDisplayName(activityType models.ActivityType) string {
	switch activityType {
	case models.ActivityTypeDrinkWater:
		return "喝水"
	case models.ActivityTypeTakeMedicine:
		return "吃药"
	case models.ActivityTypeReadBook:
		return "看书"
	case models.ActivityTypeExercise:
		return "运动"
	case models.ActivityTypeSleep:
		return "睡眠"
	case models.ActivityTypeEat:
		return "吃饭"
	default:
		return string(activityType)
	}
}