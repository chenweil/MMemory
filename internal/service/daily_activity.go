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
	DeleteActivities(ctx context.Context, userID uint, activityType models.ActivityType, criteria map[string]interface{}) (int, error)
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
	startTime, endTime, _ := resolveActivityTimeRange(timeRange, time.Now())

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
		startTime, endTime, normalizedRange := resolveActivityTimeRange(timeRange, time.Now())
		activities, err := s.GetActivitiesByDateRange(ctx, userID, startTime, endTime)
		if err != nil {
			return "", err
		}
		return s.formatActivitiesByTime(activities, normalizedRange), nil

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
			details, err := act.GetDetails()
			logger.Infof("查询活动 ID=%d: BookName=%q, Chapter=%q, Error=%v",
				act.ID, details.BookName, details.Chapter, err)
			if details.BookName != "" {
				chapter := details.Chapter
				if chapter == "" {
					chapter = "未知章节"
				}
				// 只保留最新的记录（map 中不存在的才添加）
				// 这样确保旧记录不会覆盖新记录
				if _, exists := books[details.BookName]; !exists {
					books[details.BookName] = chapter
				}
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

// DeleteActivities 根据条件删除活动记录
func (s *dailyActivityServiceImpl) DeleteActivities(ctx context.Context, userID uint, activityType models.ActivityType, criteria map[string]interface{}) (int, error) {
	// 获取所有匹配的活动（获取足够多的记录以便筛选）
	activities, err := s.activityRepo.GetByType(ctx, userID, activityType, 100, 0)
	if err != nil {
		logger.Errorf("查询活动记录失败: %v", err)
		return 0, fmt.Errorf("查询活动记录失败: %w", err)
	}

	var toDelete []uint

	for _, activity := range activities {
		details, err := activity.GetDetails()
		if err != nil {
			logger.Warnf("解析活动详情失败 (ID=%d): %v", activity.ID, err)
			continue
		}

		// 根据活动类型和条件筛选
		switch activityType {
		case models.ActivityTypeReadBook:
			// 根据书名删除
			if bookName, ok := criteria["book_name"].(string); ok {
				if details.BookName == bookName {
					toDelete = append(toDelete, activity.ID)
				}
			}

		case models.ActivityTypeDrinkWater:
			// 删除所有喝水记录或按时间范围
			if timeRange, ok := criteria["time_range"].(string); ok {
				// 根据时间范围筛选
				startTime, endTime, _ := resolveActivityTimeRange(timeRange, time.Now())
				if activity.OccurredAt.After(startTime) && activity.OccurredAt.Before(endTime) {
					toDelete = append(toDelete, activity.ID)
				}
			} else {
				// 没有指定时间范围，删除所有
				toDelete = append(toDelete, activity.ID)
			}

		case models.ActivityTypeExercise:
			// 删除运动记录 - 可以根据运动类型筛选
			if exerciseType, ok := criteria["exercise_type"].(string); ok {
				if details.ExerciseType == exerciseType {
					toDelete = append(toDelete, activity.ID)
				}
			} else {
				// 删除所有运动记录
				toDelete = append(toDelete, activity.ID)
			}

		case models.ActivityTypeTakeMedicine:
			// 根据药名删除
			if medicineName, ok := criteria["medicine_name"].(string); ok {
				if details.MedicineName == medicineName {
					toDelete = append(toDelete, activity.ID)
				}
			}

		default:
			// 其他类型，如果没有具体条件，删除所有
			if len(criteria) == 0 {
				toDelete = append(toDelete, activity.ID)
			}
		}
	}

	// 批量删除
	deleted := 0
	for _, id := range toDelete {
		if err := s.activityRepo.Delete(ctx, id); err != nil {
			logger.Errorf("删除活动记录失败 (ID=%d): %v", id, err)
		} else {
			deleted++
		}
	}

	logger.Infof("用户 %d 删除了 %d 条 %s 记录", userID, deleted, activityType)
	return deleted, nil
}
