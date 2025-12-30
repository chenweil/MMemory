package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

// activityVisualizationServiceImpl 活动可视化服务实现
type activityVisualizationServiceImpl struct {
	activityRepo interfaces.DailyActivityRepository
}

// NewActivityVisualizationService 创建活动可视化服务
func NewActivityVisualizationService(activityRepo interfaces.DailyActivityRepository) ActivityVisualizationService {
	return &activityVisualizationServiceImpl{
		activityRepo: activityRepo,
	}
}

// GetActivityTrendChart 获取活动趋势图表（ASCII）
func (s *activityVisualizationServiceImpl) GetActivityTrendChart(ctx context.Context, userID uint, activityType string, days int) (string, error) {
	if days <= 0 {
		days = 7
	}

	now := time.Now()
	startTime := now.AddDate(0, 0, -days)

	var activities []*models.DailyActivity
	var err error

	if activityType != "" && activityType != "all" {
		activities, err = s.activityRepo.GetByType(ctx, userID, models.ActivityType(activityType), days*10, 0)
	} else {
		activities, err = s.activityRepo.GetByDateRange(ctx, userID, startTime, now)
	}

	if err != nil {
		logger.Errorf("获取活动数据失败: %v", err)
		return "", fmt.Errorf("获取数据失败: %w", err)
	}

	// 按天聚合数据
	dailyData := make(map[string]int64)
	for _, act := range activities {
		dateKey := act.OccurredAt.Format("2006-01-02")
		dailyData[dateKey]++
	}

	// 构建图表
	return s.buildASCIIChart(dailyData, days, activityType), nil
}

// buildASCIIChart 构建 ASCII 柱状图
func (s *activityVisualizationServiceImpl) buildASCIIChart(dailyData map[string]int64, days int, activityType string) string {
	if len(dailyData) == 0 {
		return "📊 暂无活动数据\n\n尝试记录一些活动后再查看趋势吧！"
	}

	// 准备日期和数值
	type dateCount struct {
		date   string
		count  int64
		dayStr string // 简化日期如 "周一", "周二"
	}

	var dataList []dateCount
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateKey := date.Format("2006-01-02")
		count := dailyData[dateKey]

		dayName := date.Format("周一")
		switch date.Weekday() {
		case time.Sunday:
			dayName = "周日"
		case time.Monday:
			dayName = "周一"
		case time.Tuesday:
			dayName = "周二"
		case time.Wednesday:
			dayName = "周三"
		case time.Thursday:
			dayName = "周四"
		case time.Friday:
			dayName = "周五"
		case time.Saturday:
			dayName = "周六"
		}

		dataList = append(dataList, dateCount{
			date:   dateKey,
			count:  count,
			dayStr: dayName,
		})
	}

	// 找到最大值用于缩放
	var maxCount int64
	for _, d := range dataList {
		if d.count > maxCount {
			maxCount = d.count
		}
	}
	if maxCount == 0 {
		maxCount = 1
	}

	// 图表宽度
	chartWidth := 12
	scale := float64(maxCount) / float64(chartWidth)

	// 构建图表头部
	title := "📊 活动趋势"
	if activityType != "" && activityType != "all" {
		title += " - " + getActivityTypeDisplayName(models.ActivityType(activityType))
	}
	result := title + "\n"
	result += strings.Repeat("─", 30) + "\n"

	// 构建柱状图（从上到下）
	for i := chartWidth; i >= 1; i-- {
		line := ""
		for _, d := range dataList {
			value := float64(d.count)
			barHeight := int(value / scale)
			if barHeight >= i {
				line += "█  "
			} else if barHeight >= i-1 && i == 1 && value > 0 {
				// 显示小数值的点
				line += "▌  "
			} else {
				line += "   "
			}
		}
		result += line + "\n"
	}

	// 添加日期标签
	result += strings.Repeat("─", 30) + "\n"
	for _, d := range dataList {
		result += fmt.Sprintf("%s ", d.dayStr[:2])
	}
	result += "\n"

	// 添加数值标签
	for _, d := range dataList {
		result += fmt.Sprintf("%d  ", d.count)
	}
	result += "\n"

	// 添加统计摘要
	var total int64
	for _, d := range dataList {
		total += d.count
	}
	avg := float64(total) / float64(len(dataList))

	result += strings.Repeat("─", 30) + "\n"
	result += fmt.Sprintf("📈 总计: %d 次 | 日均: %.1f 次", total, avg)

	return result
}

// GetActivityHeatmap 获取活动热力图
func (s *activityVisualizationServiceImpl) GetActivityHeatmap(ctx context.Context, userID uint, activityType string, days int) (string, error) {
	if days <= 0 {
		days = 7
	}

	now := time.Now()
	startTime := now.AddDate(0, 0, -days)

	var activities []*models.DailyActivity
	var err error

	if activityType != "" && activityType != "all" {
		activities, err = s.activityRepo.GetByType(ctx, userID, models.ActivityType(activityType), days*10, 0)
	} else {
		activities, err = s.activityRepo.GetByDateRange(ctx, userID, startTime, now)
	}

	if err != nil {
		logger.Errorf("获取活动数据失败: %v", err)
		return "", fmt.Errorf("获取数据失败: %w", err)
	}

	// 按小时和星期聚合
	// heatmap[weekday][hour] = count
	heatmap := make(map[time.Weekday]map[int]int64) // weekday: 0-6, hour: 0-23
	for i := time.Sunday; i <= time.Saturday; i++ {
		heatmap[i] = make(map[int]int64)
	}

	for _, act := range activities {
		weekday := act.OccurredAt.Weekday()
		hour := act.OccurredAt.Hour()
		if heatmap[weekday] == nil {
			heatmap[weekday] = make(map[int]int64)
		}
		heatmap[weekday][hour]++
	}

	return s.buildHeatmap(heatmap, activityType), nil
}

// buildHeatmap 构建 ASCII 热力图
func (s *activityVisualizationServiceImpl) buildHeatmap(heatmap map[time.Weekday]map[int]int64, activityType string) string {
	if len(heatmap) == 0 {
		return "🌡️ 暂无活动热力数据\n\n活动记录越多，热力图越丰富！"
	}

	// 找到最大值
	var maxCount int64
	for _, hours := range heatmap {
		for _, count := range hours {
			if count > maxCount {
				maxCount = count
			}
		}
	}
	if maxCount == 0 {
		maxCount = 1
	}

	// 热力图字符（从低到高）
	heatChars := []string{"░", "▒", "▓", "█", "💯"}

	title := "🌡️ 活动热力图"
	if activityType != "" && activityType != "all" {
		title += " - " + getActivityTypeDisplayName(models.ActivityType(activityType))
	}
	result := title + "\n"
	result += strings.Repeat("─", 40) + "\n"

	// 小时标签
	hourLabels := "      "
	for h := 0; h < 24; h++ {
		if h%3 == 0 {
			hourLabels += fmt.Sprintf("%02d ", h)
		} else {
			hourLabels += "   "
		}
	}
	result += hourLabels + "\n"
	result += strings.Repeat("─", 40) + "\n"

	// 星期行
	weekdays := []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday}
	weekdayLabels := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	for i, w := range weekdays {
		row := weekdayLabels[i] + "  "
		hours := heatmap[w]
		if hours == nil {
			hours = make(map[int]int64)
		}

		for h := 0; h < 24; h++ {
			count := hours[h]
			var char string
			ratio := float64(count) / float64(maxCount)

			switch {
			case ratio == 0:
				char = "░"
			case ratio < 0.25:
				char = "░"
			case ratio < 0.5:
				char = "▒"
			case ratio < 0.75:
				char = "▓"
			default:
				char = "█"
			}

			row += char + " "
		}
		result += row + "\n"
	}

	result += strings.Repeat("─", 40) + "\n"
	result += "📊 颜色越深 = 活动越频繁\n"

	_ = heatChars // 使用变量避免编译警告
	return result
}

// GetActivityStatistics 获取活动统计数据
func (s *activityVisualizationServiceImpl) GetActivityStatistics(ctx context.Context, userID uint, timeRange string) (*ActivityStatistics, error) {
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
	case "这月", "本月":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 1, 0)
	case "最近30天":
		startTime = now.AddDate(0, 0, -30)
		endTime = now
	default:
		startTime = now.AddDate(0, 0, -7)
		endTime = now
	}

	// 获取统计数据
	stats, err := s.activityRepo.GetStatistics(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// 获取所有活动记录用于计算每日数据
	activities, err := s.activityRepo.GetByDateRange(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// 按天聚合
	byDay := make(map[string]int64)
	for _, act := range activities {
		dateKey := act.OccurredAt.Format("2006-01-02")
		byDay[dateKey]++
	}

	// 计算最活跃的日期和类型
	var mostActiveDay string
	var mostActiveDayCount int64
	var mostActiveType string
	var mostActiveTypeCount int64

	for date, count := range byDay {
		if count > mostActiveDayCount {
			mostActiveDayCount = count
			mostActiveDay = date
		}
	}

	for actType, count := range stats {
		if count > mostActiveTypeCount {
			mostActiveTypeCount = count
			mostActiveType = actType
		}
	}

	// 计算每日平均
	days := int(endTime.Sub(startTime).Hours() / 24)
	if days <= 0 {
		days = 1
	}

	var total int64
	for _, c := range stats {
		total += c
	}
	dailyAverage := float64(total) / float64(days)

	// 计算趋势
	var weeklyData []DailyData
	for i := days - 1; i >= 0; i-- {
		date := endTime.AddDate(0, 0, -i-days+1)
		dateKey := date.Format("2006-01-02")
		count := byDay[dateKey]

		dayData := DailyData{
			Date:   dateKey,
			Total:  count,
			ByType: make(map[string]int64),
		}

		// 统计当天的类型分布
		for _, a := range activities {
			if a.OccurredAt.Format("2006-01-02") == dateKey {
				dayData.ByType[string(a.ActivityType)]++
			}
		}

		weeklyData = append(weeklyData, dayData)
	}

	// 简单的趋势计算（比较前一半和后一半）
	trend := "stable"
	if len(weeklyData) >= 4 {
		mid := len(weeklyData) / 2
		var firstHalf, secondHalf int64
		for i, d := range weeklyData {
			if i < mid {
				firstHalf += d.Total
			} else {
				secondHalf += d.Total
			}
		}
		if secondHalf > firstHalf*12/10 {
			trend = "up"
		} else if firstHalf > secondHalf*12/10 {
			trend = "down"
		}
	}

	result := &ActivityStatistics{
		TotalActivities:    total,
		ByType:             stats,
		ByDay:              byDay,
		MostActiveDay:      mostActiveDay,
		MostActiveType:     mostActiveType,
		DailyAverage:       dailyAverage,
		Trend:              trend,
		WeeklyData:         weeklyData,
	}

	return result, nil
}

// GetCompletionRate 获取活动完成率
func (s *activityVisualizationServiceImpl) GetCompletionRate(ctx context.Context, userID uint, activityType string, days int) (*CompletionRate, error) {
	if days <= 0 {
		days = 7
	}

	now := time.Now()
	startTime := now.AddDate(0, 0, -days)

	var activities []*models.DailyActivity
	var err error

	if activityType != "" && activityType != "all" {
		activities, err = s.activityRepo.GetByType(ctx, userID, models.ActivityType(activityType), days*10, 0)
	} else {
		activities, err = s.activityRepo.GetByDateRange(ctx, userID, startTime, now)
	}

	if err != nil {
		return nil, err
	}

	total := int64(len(activities))
	if total == 0 {
		return &CompletionRate{
			ActivityType: activityType,
			TotalRecords: 0,
			Rate:         0,
			Trend:        "stable",
		}, nil
	}

	// 计算完成率（基于来源）
	var completed int64
	for _, act := range activities {
		// 如果来源是 conversation 或 reminder，认为是已完成的活动记录
		if act.Source == models.SourceConversation || act.Source == models.SourceReminder {
			completed++
		}
	}

	rate := float64(completed) / float64(total)

	// 计算趋势
	half := len(activities) / 2
	var firstHalf, secondHalf int64
	for i := range activities {
		if i < half {
			firstHalf++
		} else {
			secondHalf++
		}
	}

	trend := "stable"
	if secondHalf > firstHalf*12/10 && firstHalf > 0 {
		trend = "up"
	} else if firstHalf > secondHalf*12/10 && secondHalf > 0 {
		trend = "down"
	}

	return &CompletionRate{
		ActivityType:     activityType,
		TotalRecords:     total,
		CompletedRecords: completed,
		Rate:             rate,
		Trend:            trend,
	}, nil
}

// GetActivitySummary 获取活动综合摘要
func (s *activityVisualizationServiceImpl) GetActivitySummary(ctx context.Context, userID uint, timeRange string) (string, error) {
	stats, err := s.GetActivityStatistics(ctx, userID, timeRange)
	if err != nil {
		return "", err
	}

	// 获取趋势图表
	trendChart, err := s.GetActivityTrendChart(ctx, userID, "all", 7)
	if err != nil {
		trendChart = "无法生成趋势图表"
	}

	// 获取热力图
	heatmap, err := s.GetActivityHeatmap(ctx, userID, "all", 7)
	if err != nil {
		heatmap = "无法生成热力图"
	}

	// 构建摘要
	result := fmt.Sprintf("📊 <b>活动摘要 - %s</b>\n", timeRange)
	result += strings.Repeat("═", 30) + "\n\n"

	// 总体统计
	result += fmt.Sprintf("📈 <b>总体统计</b>\n")
	result += fmt.Sprintf("  总活动次数: %d 次\n", stats.TotalActivities)
	result += fmt.Sprintf("  日均活动: %.1f 次\n", stats.DailyAverage)
	result += fmt.Sprintf("  最活跃日: %s (%d 次)\n", FormatDate(stats.MostActiveDay), stats.ByDay[stats.MostActiveDay])
	result += fmt.Sprintf("  趋势: %s\n\n", FormatTrend(stats.Trend))

	// 按类型统计
	result += fmt.Sprintf("🏷️ <b>类型分布</b>\n")
	typeCounts := make([]struct {
		Type  string
		Count int64
	}, 0)
	for t, c := range stats.ByType {
		typeCounts = append(typeCounts, struct {
			Type  string
			Count int64
		}{t, c})
	}
	sort.Slice(typeCounts, func(i, j int) bool {
		return typeCounts[i].Count > typeCounts[j].Count
	})

	for _, tc := range typeCounts {
		if tc.Count > 0 {
			displayName := getActivityTypeDisplayName(models.ActivityType(tc.Type))
			bar := strings.Repeat("▓", int(tc.Count*20/stats.TotalActivities)+1)
			result += fmt.Sprintf("  %s %s %d次\n", displayName, bar, tc.Count)
		}
	}

	result += "\n"
	result += strings.Repeat("═", 30) + "\n\n"

	// 趋势图表
	result += "📈 <b>7天趋势</b>\n"
	result += trendChart + "\n\n"

	// 热力图
	result += "🌡️ <b>时间分布</b>\n"
	result += heatmap

	return result, nil
}

// FormatDate 格式化日期显示
func FormatDate(dateStr string) string {
	if dateStr == "" {
		return "无"
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("1月2日")
}

// FormatTrend 格式化趋势显示
func FormatTrend(trend string) string {
	switch trend {
	case "up":
		return "📈 上升 ↗"
	case "down":
		return "📉 下降 ↘"
	default:
		return "➡️ 稳定 →"
	}
}
