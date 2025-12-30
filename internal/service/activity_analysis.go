package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

// ActivityAnalysisServiceImpl 智能分析服务实现
type ActivityAnalysisServiceImpl struct {
	activityRepo interfaces.DailyActivityRepository
}

// NewActivityAnalysisService 创建智能分析服务
func NewActivityAnalysisService(activityRepo interfaces.DailyActivityRepository) ActivityAnalysisService {
	return &ActivityAnalysisServiceImpl{
		activityRepo: activityRepo,
	}
}

// DetectPatterns 检测用户活动模式
func (s *ActivityAnalysisServiceImpl) DetectPatterns(ctx context.Context, userID uint, activityType string, days int) ([]ActivityPattern, error) {
	if days <= 0 {
		days = 30
	}

	now := time.Now()
	startTime := now.AddDate(0, 0, -days)

	var activities []*models.DailyActivity
	var err error

	if activityType != "" && activityType != "all" {
		activities, err = s.activityRepo.GetByType(ctx, userID, models.ActivityType(activityType), days*2, 0)
	} else {
		activities, err = s.activityRepo.GetByDateRange(ctx, userID, startTime, now)
	}

	if err != nil {
		logger.Errorf("获取活动数据失败: %v", err)
		return nil, err
	}

	// 按活动类型分组
	activitiesByType := make(map[string][]*models.DailyActivity)
	for _, act := range activities {
		activitiesByType[string(act.ActivityType)] = append(activitiesByType[string(act.ActivityType)], act)
	}

	var patterns []ActivityPattern
	for actType, acts := range activitiesByType {
		pattern := s.analyzePattern(actType, acts, days)
		if pattern.Frequency > 0 {
			patterns = append(patterns, pattern)
		}
	}

	// 按一致性评分排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].ConsistencyScore > patterns[j].ConsistencyScore
	})

	return patterns, nil
}

// analyzePattern 分析单个活动类型的模式
func (s *ActivityAnalysisServiceImpl) analyzePattern(activityType string, activities []*models.DailyActivity, days int) ActivityPattern {
	if len(activities) == 0 {
		return ActivityPattern{}
	}

	pattern := ActivityPattern{
		PatternID:      fmt.Sprintf("pattern-%s-%d", activityType, time.Now().Unix()),
		ActivityType:   activityType,
		FirstRecorded:  activities[len(activities)-1].OccurredAt,
		LastRecorded:   activities[0].OccurredAt,
	}

	// 计算频率
	pattern.Frequency = float64(len(activities)) / float64(days)

	// 计算连续天数
	currentStreak := s.calculateStreak(activities)
	pattern.Streak = currentStreak
	pattern.LongestStreak = s.calculateLongestStreak(activities)

	// 计算时间分布
	pattern.WeeklyDistribution = s.calculateWeeklyDistribution(activities)
	pattern.HourlyDistribution = s.calculateHourlyDistribution(activities)

	// 计算平均执行时间
	pattern.AverageTime = s.calculateAverageTime(activities)

	// 计算时间方差
	pattern.TimeVariance = s.calculateTimeVariance(activities)

	// 计算一致性评分 (0-100)
	pattern.ConsistencyScore = s.calculateConsistencyScore(pattern, len(activities), days)

	// 确定模式类型
	pattern.PatternType = s.determinePatternType(pattern.Frequency, pattern.TimeVariance)

	return pattern
}

// calculateStreak 计算当前连续天数
func (s *ActivityAnalysisServiceImpl) calculateStreak(activities []*models.DailyActivity) int {
	if len(activities) == 0 {
		return 0
	}

	// 按时间倒序排序
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].OccurredAt.After(activities[j].OccurredAt)
	})

	streak := 1
	prevDate := activities[0].OccurredAt.Truncate(24 * time.Hour)

	for i := 1; i < len(activities); i++ {
		currDate := activities[i].OccurredAt.Truncate(24 * time.Hour)
		diff := prevDate.Sub(currDate)

		if diff == 24*time.Hour {
			streak++
			prevDate = currDate
		} else if diff < 24*time.Hour {
			// 同一天的活动，不影响连续性
			continue
		} else {
			break
		}
	}

	return streak
}

// calculateLongestStreak 计算最长连续天数
func (s *ActivityAnalysisServiceImpl) calculateLongestStreak(activities []*models.DailyActivity) int {
	if len(activities) == 0 {
		return 0
	}

	// 获取所有唯一日期
	dateSet := make(map[string]struct{})
	for _, act := range activities {
		dateStr := act.OccurredAt.Format("2006-01-02")
		dateSet[dateStr] = struct{}{}
	}

	dates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}

	// 排序
	sort.Strings(dates)

	if len(dates) == 0 {
		return 0
	}

	// 解析日期
	parsedDates := make([]time.Time, 0, len(dates))
	for _, d := range dates {
		t, _ := time.Parse("2006-01-02", d)
		parsedDates = append(parsedDates, t)
	}

	// 计算最长连续
	longest := 1
	current := 1

	for i := 1; i < len(parsedDates); i++ {
		if parsedDates[i].Sub(parsedDates[i-1]) == 24*time.Hour {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 1
		}
	}

	return longest
}

// calculateWeeklyDistribution 计算每周分布
func (s *ActivityAnalysisServiceImpl) calculateWeeklyDistribution(activities []*models.DailyActivity) map[string]float64 {
	weekCount := make(map[string]int)
	weekDays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	for _, act := range activities {
		weekDay := weekDays[act.OccurredAt.Weekday()]
		weekCount[weekDay]++
	}

	// 计算频率
	totalWeeks := 1
	if len(activities) > 0 {
		first := activities[len(activities)-1].OccurredAt
		last := activities[0].OccurredAt
		totalWeeks = int(last.Sub(first).Hours() / 24 / 7)
		if totalWeeks < 1 {
			totalWeeks = 1
		}
	}

	distribution := make(map[string]float64)
	for day, count := range weekCount {
		distribution[day] = float64(count) / float64(totalWeeks)
	}

	return distribution
}

// calculateHourlyDistribution 计算每小时分布
func (s *ActivityAnalysisServiceImpl) calculateHourlyDistribution(activities []*models.DailyActivity) map[int]float64 {
	hourCount := make(map[int]int)

	for _, act := range activities {
		hour := act.OccurredAt.Hour()
		hourCount[hour]++
	}

	// 计算频率
	totalDays := 1
	if len(activities) > 0 {
		first := activities[len(activities)-1].OccurredAt
		last := activities[0].OccurredAt
		totalDays = int(last.Sub(first).Hours() / 24)
		if totalDays < 1 {
			totalDays = 1
		}
	}

	distribution := make(map[int]float64)
	for hour, count := range hourCount {
		distribution[hour] = float64(count) / float64(totalDays)
	}

	return distribution
}

// calculateAverageTime 计算平均执行时间
func (s *ActivityAnalysisServiceImpl) calculateAverageTime(activities []*models.DailyActivity) string {
	if len(activities) == 0 {
		return "00:00"
	}

	var totalMinutes int
	for _, act := range activities {
		totalMinutes += act.OccurredAt.Hour()*60 + act.OccurredAt.Minute()
	}

	avgMinutes := totalMinutes / len(activities)
	hour := avgMinutes / 60
	minute := avgMinutes % 60

	return fmt.Sprintf("%02d:%02d", hour, minute)
}

// calculateTimeVariance 计算时间方差
func (s *ActivityAnalysisServiceImpl) calculateTimeVariance(activities []*models.DailyActivity) float64 {
	if len(activities) == 0 {
		return 0
	}

	var times []float64
	for _, act := range activities {
		times = append(times, float64(act.OccurredAt.Hour()*60+act.OccurredAt.Minute()))
	}

	mean := 0.0
	for _, t := range times {
		mean += t
	}
	mean /= float64(len(times))

	variance := 0.0
	for _, t := range times {
		variance += (t - mean) * (t - mean)
	}

	return variance / float64(len(times))
}

// calculateConsistencyScore 计算一致性评分
func (s *ActivityAnalysisServiceImpl) calculateConsistencyScore(pattern ActivityPattern, totalRecords, days int) float64 {
	// 基于频率、连续性、时间方差计算一致性评分
	frequencyScore := math.Min(pattern.Frequency*100, 100) // 频率权重 30%
	streakScore := math.Min(float64(pattern.Streak)*10, 100) // 连续性权重 40%
	varianceScore := math.Max(100-pattern.TimeVariance/10, 0) // 时间一致性权重 30%

	consistencyScore := frequencyScore*0.3 + streakScore*0.4 + varianceScore*0.3

	return math.Round(consistencyScore*100) / 100
}

// determinePatternType 确定模式类型
func (s *ActivityAnalysisServiceImpl) determinePatternType(frequency, variance float64) string {
	if frequency >= 0.9 && variance < 60 {
		return "stable_daily"
	} else if frequency >= 0.7 {
		return "regular"
	} else if frequency >= 0.4 {
		return "occasional"
	} else {
		return "sporadic"
	}
}

// AnalyzeTimeDistribution 分析时间分布
func (s *ActivityAnalysisServiceImpl) AnalyzeTimeDistribution(ctx context.Context, userID uint, activityType string, days int) (string, error) {
	patterns, err := s.DetectPatterns(ctx, userID, activityType, days)
	if err != nil {
		return "", err
	}

	if len(patterns) == 0 {
		return "📊 暂无足够的活动数据进行分析\n\n请先记录一些活动数据（至少3天），我再为你分析时间分布规律。", nil
	}

	result := "📊 <b>活动时间分布分析</b>\n\n"

	for _, pattern := range patterns {
		result += fmt.Sprintf("📌 <b>%s</b>\n", getActivityTypeDisplayName(models.ActivityType(pattern.ActivityType)))
		result += fmt.Sprintf("  • 频率: %.1f%%/天\n", pattern.Frequency*100)
		result += fmt.Sprintf("  • 平均时间: %s\n", pattern.AverageTime)
		result += fmt.Sprintf("  • 一致性评分: %.1f分\n", pattern.ConsistencyScore)

		// 最佳时间
		bestHour := s.findBestHour(pattern.HourlyDistribution)
		result += fmt.Sprintf("  • 最佳时段: %d:00-%d:00\n", bestHour, bestHour+1)

		result += "\n"
	}

	return result, nil
}

// findBestHour 找到最佳时段
func (s *ActivityAnalysisServiceImpl) findBestHour(distribution map[int]float64) int {
	bestHour := 9 // 默认早上9点
	bestValue := 0.0

	for hour, value := range distribution {
		if value > bestValue {
			bestValue = value
			bestHour = hour
		}
	}

	return bestHour
}

// DetectAnomalies 检测活动异常
func (s *ActivityAnalysisServiceImpl) DetectAnomalies(ctx context.Context, userID uint, activityType string, days int) ([]ActivityAnomaly, error) {
	if days <= 0 {
		days = 7
	}

	now := time.Now()
	startTime := now.AddDate(0, 0, -days)

	var activities []*models.DailyActivity
	var err error

	if activityType != "" && activityType != "all" {
		activities, err = s.activityRepo.GetByType(ctx, userID, models.ActivityType(activityType), days*2, 0)
	} else {
		activities, err = s.activityRepo.GetByDateRange(ctx, userID, startTime, now)
	}

	if err != nil {
		logger.Errorf("获取活动数据失败: %v", err)
		return nil, err
	}

	var anomalies []ActivityAnomaly

	// 按活动类型分组
	activitiesByType := make(map[string][]*models.DailyActivity)
	for _, act := range activities {
		activitiesByType[string(act.ActivityType)] = append(activitiesByType[string(act.ActivityType)], act)
	}

	// 按天分组
	activitiesByDay := make(map[string][]*models.DailyActivity)
	for _, act := range activities {
		dayKey := act.OccurredAt.Format("2006-01-02")
		activitiesByDay[dayKey] = append(activitiesByDay[dayKey], act)
	}

	// 检测缺失
	for actType, acts := range activitiesByType {
		// 检测今天是否缺失
		todayKey := now.Format("2006-01-02")
		if _, exists := activitiesByDay[todayKey]; !exists {
			// 检查是否有历史记录表明今天应该有
			expectedCount := s.calculateExpectedCount(acts, 1)
			if expectedCount > 0 {
				anomaly := ActivityAnomaly{
					AnomalyID:      fmt.Sprintf("anomaly-%s-missing-%s", actType, todayKey),
					UserID:         userID,
					ActivityType:   actType,
					AnomalyType:    "missing",
					Severity:       "medium",
					Description:    fmt.Sprintf("今天还没有记录「%s」活动", getActivityTypeDisplayName(models.ActivityType(actType))),
					ExpectedCount:  expectedCount,
					ActualCount:    0,
					DetectedAt:     now,
					Resolved:       false,
				}
				anomalies = append(anomalies, anomaly)
			}
		}

		// 检测频率下降
		anomaly := s.detectFrequencyDrop(actType, acts, activitiesByDay, userID)
		if anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}
	}

	// 检测时间异常
	for actType, acts := range activitiesByType {
		timeAnomaly := s.detectTimeAnomaly(actType, acts, userID)
		if timeAnomaly != nil {
			anomalies = append(anomalies, *timeAnomaly)
		}
	}

	// 按严重程度排序
	sort.Slice(anomalies, func(i, j int) bool {
		severityOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
		return severityOrder[anomalies[i].Severity] < severityOrder[anomalies[j].Severity]
	})

	return anomalies, nil
}

// calculateExpectedCount 计算预期数量
func (s *ActivityAnalysisServiceImpl) calculateExpectedCount(activities []*models.DailyActivity, days int) int {
	if len(activities) == 0 {
		return 0
	}

	// 简单的启发式：基于历史频率
	frequency := float64(len(activities)) / 30 // 假设30天周期
	return int(frequency * float64(days))
}

// detectFrequencyDrop 检测频率下降
func (s *ActivityAnalysisServiceImpl) detectFrequencyDrop(activityType string, activities []*models.DailyActivity, activitiesByDay map[string][]*models.DailyActivity, userID uint) *ActivityAnomaly {
	if len(activities) < 7 {
		return nil
	}

	now := time.Now()

	// 拆分数据：最近3天 vs 前7天
	var recent3, recent7 []*models.DailyActivity

	for dayKey, acts := range activitiesByDay {
		t, _ := time.Parse("2006-01-02", dayKey)
		daysAgo := int(now.Sub(t).Hours() / 24)

		if daysAgo <= 3 {
			recent3 = append(recent3, acts...)
		} else if daysAgo <= 10 {
			recent7 = append(recent7, acts...)
		}
	}

	// 计算频率
	recent3Rate := float64(len(recent3)) / 3
	recent7Rate := float64(len(recent7)) / 7

	// 如果最近3天的频率比之前下降超过50%
	if recent7Rate > 0 && recent3Rate < recent7Rate*0.5 {
		return &ActivityAnomaly{
			AnomalyID:     fmt.Sprintf("anomaly-%s-frequency-%d", activityType, now.Unix()),
			UserID:        userID,
			ActivityType:  activityType,
			AnomalyType:   "frequency_drop",
			Severity:      "low",
			Description:   fmt.Sprintf("「%s」活动频率下降，近期完成次数明显减少", getActivityTypeDisplayName(models.ActivityType(activityType))),
			ExpectedCount: int(recent7Rate * 3),
			ActualCount:   len(recent3),
			DetectedAt:    now,
			Resolved:      false,
		}
	}

	return nil
}

// detectTimeAnomaly 检测时间异常
func (s *ActivityAnalysisServiceImpl) detectTimeAnomaly(activityType string, activities []*models.DailyActivity, userID uint) *ActivityAnomaly {
	if len(activities) < 5 {
		return nil
	}

	// 计算平均时间
	var totalMinutes int
	for _, act := range activities {
		totalMinutes += act.OccurredAt.Hour()*60 + act.OccurredAt.Minute()
	}
	avgMinutes := totalMinutes / len(activities)

	// 检查最近一次活动时间是否严重偏离平均值
	recent := activities[0]
	recentMinutes := recent.OccurredAt.Hour()*60 + recent.OccurredAt.Minute()
	diff := math.Abs(float64(recentMinutes - avgMinutes))

	// 如果偏离超过3小时
	if diff > 180 {
		now := time.Now()
		return &ActivityAnomaly{
			AnomalyID:     fmt.Sprintf("anomaly-%s-time-%d", activityType, now.Unix()),
			UserID:        userID,
			ActivityType:  activityType,
			AnomalyType:   "late",
			Severity:      "low",
			Description:   fmt.Sprintf("「%s」执行时间与平时差异较大", getActivityTypeDisplayName(models.ActivityType(activityType))),
			ExpectedTime:  fmt.Sprintf("%02d:%02d", avgMinutes/60, avgMinutes%60),
			ActualTime:    recent.OccurredAt.Format("15:04"),
			DetectedAt:    now,
			Resolved:      false,
		}
	}

	return nil
}

// GenerateSuggestions 生成智能建议
func (s *ActivityAnalysisServiceImpl) GenerateSuggestions(ctx context.Context, userID uint) ([]ActivitySuggestion, error) {
	patterns, err := s.DetectPatterns(ctx, userID, "", 30)
	if err != nil {
		return nil, err
	}

	anomalies, err := s.DetectAnomalies(ctx, userID, "", 7)
	if err != nil {
		return nil, err
	}

	var suggestions []ActivitySuggestion

	// 基于模式生成建议
	for _, pattern := range patterns {
		if pattern.ConsistencyScore < 50 {
			suggestion := s.generateConsistencySuggestion(pattern)
			suggestions = append(suggestions, suggestion)
		}

		if pattern.Streak > 0 && pattern.Streak < 3 {
			suggestion := s.generateStreakSuggestion(pattern)
			suggestions = append(suggestions, suggestion)
		}
	}

	// 基于异常生成建议
	for _, anomaly := range anomalies {
		if !anomaly.Resolved {
			suggestion := s.generateAnomalySuggestion(anomaly)
			suggestions = append(suggestions, suggestion)
		}
	}

	// 生成总体洞察建议
	insightSuggestion := s.generateInsightSuggestion(patterns, anomalies)
	suggestions = append(suggestions, insightSuggestion)

	// 按优先级排序
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Priority < suggestions[j].Priority
	})

	// 只返回前5条建议
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions, nil
}

// generateConsistencySuggestion 生成一致性建议
func (s *ActivityAnalysisServiceImpl) generateConsistencySuggestion(pattern ActivityPattern) ActivitySuggestion {
	return ActivitySuggestion{
		SuggestionID:   fmt.Sprintf("suggestion-%s-consistency-%d", pattern.ActivityType, time.Now().Unix()),
		SuggestionType: "timing",
		Priority:       2,
		ActivityType:   pattern.ActivityType,
		Title:          fmt.Sprintf("提高「%s」的一致性", getActivityTypeDisplayName(models.ActivityType(pattern.ActivityType))),
		Description:    fmt.Sprintf("你最近一个月「%s」的一致性评分只有 %.1f 分", getActivityTypeDisplayName(models.ActivityType(pattern.ActivityType)), pattern.ConsistencyScore),
		Reason:         fmt.Sprintf("基于过去30天的数据分析，你只在 %.0f%% 的天数完成了该活动", pattern.Frequency*100),
		ActionItems: []string{
			fmt.Sprintf("设定固定的提醒时间（如每天 %s）", pattern.AverageTime),
			"尝试将该活动与已有的习惯绑定",
			"记录每次完成后的感受，增强正向反馈",
		},
		Confidence:  0.75,
		BasedOnDays: 30,
		CreatedAt:   time.Now(),
	}
}

// generateStreakSuggestion 生成连续性建议
func (s *ActivityAnalysisServiceImpl) generateStreakSuggestion(pattern ActivityPattern) ActivitySuggestion {
	return ActivitySuggestion{
		SuggestionID:   fmt.Sprintf("suggestion-%s-streak-%d", pattern.ActivityType, time.Now().Unix()),
		SuggestionType: "frequency",
		Priority:       3,
		ActivityType:   pattern.ActivityType,
		Title:          fmt.Sprintf("帮助「%s」形成连续习惯", getActivityTypeDisplayName(models.ActivityType(pattern.ActivityType))),
		Description:    fmt.Sprintf("你目前连续完成「%s」的天数是 %d 天，突破3天就能形成更强的习惯！", getActivityTypeDisplayName(models.ActivityType(pattern.ActivityType)), pattern.Streak),
		Reason:         fmt.Sprintf("研究表明，连续完成21天可以形成稳定的习惯"),
		ActionItems: []string{
			"设置多个时间点的提醒，避免错过",
			"降低难度，从简单版本开始",
			"完成后立即奖励自己",
		},
		Confidence:  0.8,
		BasedOnDays: 30,
		CreatedAt:   time.Now(),
	}
}

// generateAnomalySuggestion 生成异常建议
func (s *ActivityAnalysisServiceImpl) generateAnomalySuggestion(anomaly ActivityAnomaly) ActivitySuggestion {
	var suggestion ActivitySuggestion

	switch anomaly.AnomalyType {
	case "missing":
		suggestion = ActivitySuggestion{
			SuggestionID:   fmt.Sprintf("suggestion-%s-%d", anomaly.ActivityType, time.Now().Unix()),
			SuggestionType: "reminder",
			Priority:       1,
			ActivityType:   anomaly.ActivityType,
			Title:          fmt.Sprintf("别忘了「%s」", getActivityTypeDisplayName(models.ActivityType(anomaly.ActivityType))),
			Description:    anomaly.Description,
			Reason:         "系统检测到今天可能遗漏了这项活动",
			ActionItems:    []string{"现在完成它", "设置提醒明天按时进行"},
			Confidence:     0.85,
			BasedOnDays:    7,
			CreatedAt:      time.Now(),
		}
	case "frequency_drop":
		suggestion = ActivitySuggestion{
			SuggestionID:   fmt.Sprintf("suggestion-%s-freq-%d", anomaly.ActivityType, time.Now().Unix()),
			SuggestionType: "insight",
			Priority:       2,
			ActivityType:   anomaly.ActivityType,
			Title:          fmt.Sprintf("关注「%s」活动频率", getActivityTypeDisplayName(models.ActivityType(anomaly.ActivityType))),
			Description:    anomaly.Description,
			Reason:         "最近3天的完成频率相比之前明显下降",
			ActionItems: []string{
				"分析下降原因并调整策略",
				"重新评估目标是否合理",
				"寻求家人或朋友的监督和支持",
			},
			Confidence:  0.7,
			BasedOnDays: 7,
			CreatedAt:   time.Now(),
		}
	default:
		suggestion = ActivitySuggestion{
			SuggestionID:   fmt.Sprintf("suggestion-%s-default-%d", anomaly.ActivityType, time.Now().Unix()),
			SuggestionType: "insight",
			Priority:       3,
			ActivityType:   anomaly.ActivityType,
			Title:          fmt.Sprintf("关注「%s」执行情况", getActivityTypeDisplayName(models.ActivityType(anomaly.ActivityType))),
			Description:    anomaly.Description,
			ActionItems:    []string{"检查执行情况", "调整策略"},
			Confidence:     0.6,
			BasedOnDays:    7,
			CreatedAt:      time.Now(),
		}
	}

	return suggestion
}

// generateInsightSuggestion 生成洞察建议
func (s *ActivityAnalysisServiceImpl) generateInsightSuggestion(patterns []ActivityPattern, anomalies []ActivityAnomaly) ActivitySuggestion {
	// 计算整体评分
	avgScore := 0.0
	if len(patterns) > 0 {
		for _, p := range patterns {
			avgScore += p.ConsistencyScore
		}
		avgScore /= float64(len(patterns))
	}

	// 根据整体评分生成建议
	title := "继续加油！"
	description := "你的活动记录表现良好。"

	if avgScore < 50 {
		title = "提升空间很大"
		description = "你的活动记录还需要更多坚持，系统会持续关注并提醒你。"
	} else if avgScore < 70 {
		title = "表现不错"
		description = "你的活动记录比较稳定，可以尝试挑战更复杂的习惯。"
	} else {
		title = "非常优秀！"
		description = "你的活动记录非常稳定，已经形成了良好的习惯！"
	}

	return ActivitySuggestion{
		SuggestionID:   fmt.Sprintf("suggestion-insight-%d", time.Now().Unix()),
		SuggestionType: "insight",
		Priority:       5,
		ActivityType:   "",
		Title:          title,
		Description:    description,
		Reason:         fmt.Sprintf("基于对%d个活动类型的分析", len(patterns)),
		ActionItems:    []string{"保持当前节奏", "尝试建立新的好习惯"},
		Confidence:     0.9,
		BasedOnDays:    30,
		CreatedAt:      time.Now(),
	}
}

// AnalyzeHabitFormation 分析习惯形成情况
func (s *ActivityAnalysisServiceImpl) AnalyzeHabitFormation(ctx context.Context, userID uint, activityType string) (*HabitFormationReport, error) {
	patterns, err := s.DetectPatterns(ctx, userID, activityType, 30)
	if err != nil {
		return nil, err
	}

	if len(patterns) == 0 {
		return &HabitFormationReport{
			ActivityType:   activityType,
			TotalDays:      0,
			CompletionRate: 0,
			Stage:          "initiating",
			StageProgress:  0,
		}, nil
	}

	// 使用评分最高的模式
	pattern := patterns[0]

	// 确定阶段
	stage, stageProgress := s.determineHabitStage(pattern)

	// 找到最佳时间和日期
	bestDay := s.findBestDay(pattern.WeeklyDistribution)
	bestHour := s.findBestHour(pattern.HourlyDistribution)

	// 生成建议
	recommendations := s.generateHabitRecommendations(pattern, stage)

	report := &HabitFormationReport{
		ActivityType:     activityType,
		TotalDays:        int(pattern.LastRecorded.Sub(pattern.FirstRecorded).Hours()/24) + 1,
		CompletionRate:   pattern.Frequency,
		CurrentStreak:    pattern.Streak,
		LongestStreak:    pattern.LongestStreak,
		Stage:            stage,
		StageProgress:    stageProgress,
		BestDayOfWeek:    bestDay,
		BestTimeOfDay:    fmt.Sprintf("%02d:00", bestHour),
		ConsistencyScore: pattern.ConsistencyScore,
		Recommendations:  recommendations,
	}

	return report, nil
}

// determineHabitStage 确定习惯形成阶段
func (s *ActivityAnalysisServiceImpl) determineHabitStage(pattern ActivityPattern) (string, float64) {
	days := int(pattern.LastRecorded.Sub(pattern.FirstRecorded).Hours() / 24)
	if days < 1 {
		return "initiating", 0
	}

	// 习惯形成的阶段模型
	switch {
	case days < 7 || pattern.Frequency < 0.3:
		// 启动阶段
		progress := pattern.Frequency * 100 / 30
		return "initiating", math.Min(progress, 1.0)
	case days < 14 || pattern.Frequency < 0.5:
		// 形成阶段
		progress := 0.3 + pattern.Frequency*50/100
		return "forming", math.Min(progress, 1.0)
	case days < 21 || pattern.Frequency < 0.7:
		// 稳定阶段
		progress := 0.6 + pattern.ConsistencyScore*30/100
		return "established", math.Min(progress, 1.0)
	default:
		// 精通阶段
		progress := 0.8 + pattern.ConsistencyScore*20/100
		return "master", math.Min(progress, 1.0)
	}
}

// findBestDay 找到最佳日期
func (s *ActivityAnalysisServiceImpl) findBestDay(distribution map[string]float64) string {
	bestDay := "周一"
	bestValue := 0.0

	dayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	for day, value := range distribution {
		if value > bestValue {
			bestValue = value
			bestDay = day
		}
	}

	// 如果没有数据，返回默认
	if bestValue == 0 {
		return dayNames[time.Now().Weekday()]
	}

	return bestDay
}

// generateHabitRecommendations 生成习惯建议
func (s *ActivityAnalysisServiceImpl) generateHabitRecommendations(pattern ActivityPattern, stage string) []string {
	var recommendations []string

	switch stage {
	case "initiating":
		recommendations = []string{
			fmt.Sprintf("设定明确的目标：每天%s完成一次", getActivityTypeDisplayName(models.ActivityType(pattern.ActivityType))),
			"选择一个固定的时间点执行",
			"完成后立即记录，增强成就感",
			"不要中断，即使只完成一点点",
		}
	case "forming":
		recommendations = []string{
			"继续保持当前节奏",
			"尝试在最佳时间段执行（"+pattern.AverageTime+"）",
			"记录完成的感受，了解什么激励你",
			"设定小目标，如连续完成7天",
		}
	case "established":
		recommendations = []string{
			"你已经形成稳定习惯！",
			"可以尝试增加难度或频率",
			"分享你的经验给他人",
			"保持警惕，避免因忙碌而中断",
		}
	case "master":
		recommendations = []string{
			"你是习惯养成专家！",
			"考虑帮助他人建立类似的习惯",
			"尝试建立新的关联习惯",
			"回顾并优化你的习惯系统",
		}
	}

	return recommendations
}

// GetActivityInsights 获取综合洞察
func (s *ActivityAnalysisServiceImpl) GetActivityInsights(ctx context.Context, userID uint, days int) (*ActivityInsights, error) {
	if days <= 0 {
		days = 30
	}

	patterns, err := s.DetectPatterns(ctx, userID, "", days)
	if err != nil {
		return nil, err
	}

	anomalies, err := s.DetectAnomalies(ctx, userID, "", 7)
	if err != nil {
		return nil, err
	}

	suggestions, err := s.GenerateSuggestions(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 计算整体评分
	overallScore := 0.0
	if len(patterns) > 0 {
		for _, p := range patterns {
			overallScore += p.ConsistencyScore
		}
		overallScore /= float64(len(patterns))
	}

	// 生成摘要
	summary := s.generateInsightsSummary(patterns, anomalies, overallScore)

	insights := &ActivityInsights{
		UserID:         userID,
		Period:         fmt.Sprintf("最近%d天", days),
		MostConsistent: patterns[:min(3, len(patterns))],
		NeedsAttention: anomalies[:min(3, len(anomalies))],
		TopSuggestions: suggestions[:min(3, len(suggestions))],
		OverallScore:   overallScore,
		Summary:        summary,
		GeneratedAt:    time.Now(),
	}

	return insights, nil
}

// generateInsightsSummary 生成洞察摘要
func (s *ActivityAnalysisServiceImpl) generateInsightsSummary(patterns []ActivityPattern, anomalies []ActivityAnomaly, overallScore float64) string {
	var parts []string

	// 整体评价
	if overallScore >= 80 {
		parts = append(parts, "你的活动记录非常稳定，已经建立了良好的习惯！")
	} else if overallScore >= 60 {
		parts = append(parts, "你的活动记录表现不错，继续保持！")
	} else if overallScore >= 40 {
		parts = append(parts, "你的活动记录有提升空间，建议关注一致性问题。")
	} else {
		parts = append(parts, "你的活动记录需要更多关注和坚持。")
	}

	// 最佳表现
	if len(patterns) > 0 {
		best := patterns[0]
		parts = append(parts, fmt.Sprintf("最佳表现：「%s」一致性评分 %.1f 分", getActivityTypeDisplayName(models.ActivityType(best.ActivityType)), best.ConsistencyScore))
	}

	// 需要关注
	if len(anomalies) > 0 {
		parts = append(parts, fmt.Sprintf("需要关注：检测到 %d 个异常情况", len(anomalies)))
	}

	return strings.Join(parts, "\n")
}
