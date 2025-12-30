package service

import (
	"context"
	"fmt"

	"mmemory/internal/models"
	"mmemory/pkg/logger"
)

// PatternType 模式类型
type PatternType string

const (
	PatternDaily    PatternType = "daily"     // 每日模式
	PatternWeekly   PatternType = "weekly"    // 每周模式
	PatternMonthly  PatternType = "monthly"   // 每月模式
	PatternInterval PatternType = "interval"  // 间隔模式
	PatternCustom   PatternType = "custom"    // 自定义模式
)

// DetectedPattern 检测到的模式
type DetectedPattern struct {
	Type       PatternType `json:"type"`
	Title      string      `json:"title"`
	Frequency  string      `json:"frequency"`
	Confidence float64     `json:"confidence"`
	Examples   []string    `json:"examples"`
	Reason     string      `json:"reason"`
}

// PatternDetector 模式检测器接口
type PatternDetector interface {
	DetectPatterns(ctx context.Context, reminders []*models.Reminder) ([]DetectedPattern, error)
	AnalyzeUserBehavior(ctx context.Context, userID uint) ([]DetectedPattern, error)
}

// SimplePatternDetector 简单模式检测器实现
type SimplePatternDetector struct {
	reminderService ReminderService
}

// NewSimplePatternDetector 创建简单模式检测器
func NewSimplePatternDetector(reminderService ReminderService) *SimplePatternDetector {
	return &SimplePatternDetector{
		reminderService: reminderService,
	}
}

// DetectPatterns 检测模式
func (spd *SimplePatternDetector) DetectPatterns(ctx context.Context, reminders []*models.Reminder) ([]DetectedPattern, error) {
	if len(reminders) == 0 {
		return nil, nil
	}

	var patterns []DetectedPattern

	// 1. 检测每日模式
	dailyPatterns := spd.detectDailyPatterns(reminders)
	patterns = append(patterns, dailyPatterns...)

	// 2. 检测每周模式
	weeklyPatterns := spd.detectWeeklyPatterns(reminders)
	patterns = append(patterns, weeklyPatterns...)

	// 3. 检测间隔模式
	intervalPatterns := spd.detectIntervalPatterns(reminders)
	patterns = append(patterns, intervalPatterns...)

	// 4. 检测标题相似性模式
	titlePatterns := spd.detectTitlePatterns(reminders)
	patterns = append(patterns, titlePatterns...)

	return patterns, nil
}

// AnalyzeUserBehavior 分析用户行为模式
func (spd *SimplePatternDetector) AnalyzeUserBehavior(ctx context.Context, userID uint) ([]DetectedPattern, error) {
	// 获取用户的所有提醒
	reminders, err := spd.reminderService.GetUserReminders(ctx, userID)
	if err != nil {
		logger.Errorf("获取用户提醒失败: %v", err)
		return nil, err
	}

	if len(reminders) == 0 {
		return nil, nil
	}

	return spd.DetectPatterns(ctx, reminders)
}

// detectDailyPatterns 检测每日模式
func (spd *SimplePatternDetector) detectDailyPatterns(reminders []*models.Reminder) []DetectedPattern {
	var patterns []DetectedPattern

	// 按目标时间分组
	timeGroups := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		if reminder.IsDaily() {
			timeGroups[reminder.TargetTime] = append(timeGroups[reminder.TargetTime], reminder)
		}
	}

	for time, group := range timeGroups {
		if len(group) >= 2 {
			// 检测标题相似性
			similarity := calculateTitleSimilarity(group)
			if similarity > 0.15 { // 降低阈值以适应中文特性
				pattern := DetectedPattern{
					Type:      PatternDaily,
					Title:     fmt.Sprintf("%s类每日提醒", group[0].Title),
					Frequency: fmt.Sprintf("每天 %s", time[:5]),
					Confidence: similarity * 0.9,
					Examples:  getReminderTitles(group, 3),
					Reason:    fmt.Sprintf("检测到 %d 个相似的每日提醒，时间均为 %s", len(group), time[:5]),
				}
				patterns = append(patterns, pattern)
			}
		}
	}

	return patterns
}

// detectWeeklyPatterns 检测每周模式
func (spd *SimplePatternDetector) detectWeeklyPatterns(reminders []*models.Reminder) []DetectedPattern {
	var patterns []DetectedPattern

	// 按星期分组
	weekdayGroups := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		if reminder.IsWeekly() {
			weekdays := parseWeekdays(reminder.SchedulePattern)
			for _, weekday := range weekdays {
				time := fmt.Sprintf("%s-%s", weekday, reminder.TargetTime[:5])
				weekdayGroups[time] = append(weekdayGroups[time], reminder)
			}
		}
	}

	for time, group := range weekdayGroups {
		if len(group) >= 2 {
			similarity := calculateTitleSimilarity(group)
			if similarity > 0.08 { // 进一步降低阈值以适应中文字符特性
				pattern := DetectedPattern{
					Type:      PatternWeekly,
					Title:     fmt.Sprintf("%s类每周提醒", group[0].Title),
					Frequency: fmt.Sprintf("每周 %s", time),
					Confidence: similarity * 0.8,
					Examples:  getReminderTitles(group, 3),
					Reason:    fmt.Sprintf("检测到 %d 个相似的每周提醒，均为 %s", len(group), time),
				}
				patterns = append(patterns, pattern)
			}
		}
	}

	return patterns
}

// detectIntervalPatterns 检测间隔模式
func (spd *SimplePatternDetector) detectIntervalPatterns(reminders []*models.Reminder) []DetectedPattern {
	var patterns []DetectedPattern

	// 检测一次性提醒的间隔
	onceReminders := filterOnceReminders(reminders)
	if len(onceReminders) < 3 {
		return patterns
	}

	// 按标题分组
	titleGroups := groupRemindersByTitle(onceReminders)
	for title, group := range titleGroups {
		if len(group) < 3 {
			continue
		}

		// 计算时间间隔
		intervals := calculateIntervals(group)
		if len(intervals) == 0 {
			continue
		}

		// 计算平均间隔和标准差
		avgInterval, stdDev := calculateStatistics(intervals)
		if avgInterval == 0 {
			continue
		}

		// 如果标准差较小（间隔较规律），则认为存在间隔模式
		variationCoeff := stdDev / avgInterval
		if variationCoeff < 0.3 {
			confidence := (1 - variationCoeff) * 0.8
			if confidence > 0.5 {
				intervalText := formatInterval(avgInterval)
				pattern := DetectedPattern{
					Type:      PatternInterval,
					Title:     title,
					Frequency: fmt.Sprintf("每 %s", intervalText),
					Confidence: confidence,
					Examples:  getReminderTitles(group, 3),
					Reason:    fmt.Sprintf("检测到 %d 个相似提醒，平均间隔约 %s", len(group), intervalText),
				}
				patterns = append(patterns, pattern)
			}
		}
	}

	return patterns
}

// detectTitlePatterns 检测标题相似性模式
func (spd *SimplePatternDetector) detectTitlePatterns(reminders []*models.Reminder) []DetectedPattern {
	var patterns []DetectedPattern

	// 按标题关键词分组
	keywordGroups := groupRemindersByKeywords(reminders)
	for keyword, group := range keywordGroups {
		if len(group) >= 3 {
			timeConsistency := calculateTimeConsistency(group)
			if timeConsistency > 0.7 {
				confidence := timeConsistency * 0.7
				pattern := DetectedPattern{
					Type:      PatternCustom,
					Title:     fmt.Sprintf("%s相关提醒", keyword),
					Frequency: "不规律",
					Confidence: confidence,
					Examples:  getReminderTitles(group, 3),
					Reason:    fmt.Sprintf("检测到 %d 个包含关键词 \"%s\" 的提醒", len(group), keyword),
				}
				patterns = append(patterns, pattern)
			}
		}
	}

	return patterns
}

// calculateTitleSimilarity 计算标题相似性（支持中英文）
func calculateTitleSimilarity(reminders []*models.Reminder) float64 {
	if len(reminders) == 0 {
		return 0
	}

	if len(reminders) == 1 {
		return 1.0 // 单个提醒视为完全相似
	}

	// 提取所有标题的词频
	titleWordFreqs := make([]map[string]int, len(reminders))
	for i, reminder := range reminders {
		titleWordFreqs[i] = make(map[string]int)
		words := extractWords(reminder.Title)
		for _, word := range words {
			titleWordFreqs[i][word]++
		}
	}

	// 计算两两之间的相似度，取平均值
	totalSimilarity := 0.0
	pairCount := 0

	for i := 0; i < len(reminders); i++ {
		for j := i + 1; j < len(reminders); j++ {
			similarity := calculatePairSimilarity(titleWordFreqs[i], titleWordFreqs[j])
			totalSimilarity += similarity
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0
	}

	return totalSimilarity / float64(pairCount)
}

// calculatePairSimilarity 计算两个标题之间的相似度
func calculatePairSimilarity(words1, words2 map[string]int) float64 {
	if len(words1) == 0 || len(words2) == 0 {
		return 0
	}

	// 计算共同词汇
	commonWords := 0
	commonCount := 0

	// 找出共同词汇并计算最小出现次数
	for word, count1 := range words1 {
		if count2, exists := words2[word]; exists {
			commonWords++
			if count1 < count2 {
				commonCount += count1
			} else {
				commonCount += count2
			}
		}
	}

	if commonWords == 0 {
		return 0
	}

	// 计算总词汇数
	totalWords := len(words1) + len(words2) - commonWords

	// 使用改进的Jaccard相似度
	jaccard := float64(commonWords) / float64(totalWords)

	// 考虑词汇频率的相似度
	freqSimilarity := float64(commonCount) / float64(commonCount + (len(words1) + len(words2) - 2*commonCount))

	// 综合相似度（Jaccard权重0.7，频率权重0.3）
	return jaccard*0.7 + freqSimilarity*0.3
}

// extractWords 提取关键词
func extractWords(title string) []string {
	// 改进实现：支持中英文混合处理
	var words []string
	currentWord := ""

	// 定义分隔符
	separators := map[rune]bool{
		' ': true, '-': true, '_': true, '，': true, '。': true,
		'、': true, '；': true, ':': true, '：': true,
	}

	for _, char := range title {
		// 如果是分隔符，结束当前词
		if separators[char] {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
			continue
		}

		// 如果是英文字符或数字，添加到当前词
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			currentWord += string(char)
		} else if char >= 0x4E00 && char <= 0x9FFF { // 中文字符范围
			// 如果是中文字符，先结束当前英文字，然后单独添加中文字符
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
			words = append(words, string(char))
		} else {
			// 其他字符（如日文、韩文等），如果当前词不为空则先结束
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
			// 可以选择是否添加其他字符，这里选择添加为单独的词
			words = append(words, string(char))
		}
	}

	if currentWord != "" {
		words = append(words, currentWord)
	}

	return words
}

// parseWeekdays 解析星期几
func parseWeekdays(pattern string) []string {
	if len(pattern) <= 7 || pattern[:7] != "weekly:" {
		return nil
	}

	weekdayMap := map[string]string{
		"1": "周一", "2": "周二", "3": "周三", "4": "周四",
		"5": "周五", "6": "周六", "7": "周日",
	}

	weekdaysStr := pattern[7:]
	var weekdays []string
	for _, day := range weekdaysStr {
		dayStr := string(day)
		if dayName, ok := weekdayMap[dayStr]; ok {
			weekdays = append(weekdays, dayName)
		}
	}

	return weekdays
}

// filterOnceReminders 筛选一次性提醒
func filterOnceReminders(reminders []*models.Reminder) []*models.Reminder {
	var onceReminders []*models.Reminder
	for _, reminder := range reminders {
		if reminder.IsOnce() {
			onceReminders = append(onceReminders, reminder)
		}
	}
	return onceReminders
}

// groupRemindersByTitle 按标题分组
func groupRemindersByTitle(reminders []*models.Reminder) map[string][]*models.Reminder {
	groups := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		groups[reminder.Title] = append(groups[reminder.Title], reminder)
	}
	return groups
}

// groupRemindersByKeywords 按关键词分组
func groupRemindersByKeywords(reminders []*models.Reminder) map[string][]*models.Reminder {
	groups := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		words := extractWords(reminder.Title)
		for _, word := range words {
			if len(word) >= 2 {
				groups[word] = append(groups[word], reminder)
			}
		}
	}
	return groups
}

// calculateIntervals 计算时间间隔（天数）
func calculateIntervals(reminders []*models.Reminder) []float64 {
	if len(reminders) < 2 {
		return nil
	}

	var intervals []float64
	for i := 1; i < len(reminders); i++ {
		prevTime := reminders[i-1].CreatedAt
		currTime := reminders[i].CreatedAt
		interval := currTime.Sub(prevTime).Hours() / 24
		intervals = append(intervals, interval)
	}

	return intervals
}

// calculateStatistics 计算统计信息
func calculateStatistics(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// 计算平均值
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	avg := sum / float64(len(values))

	// 计算标准差
	variance := 0.0
	for _, v := range values {
		diff := v - avg
		variance += diff * diff
	}
	stdDev := variance / float64(len(values))
	stdDev = sqrt(stdDev)

	return avg, stdDev
}

// sqrt 简单平方根实现
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	// 牛顿法迭代
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// calculateTimeConsistency 计算时间一致性
func calculateTimeConsistency(reminders []*models.Reminder) float64 {
	if len(reminders) == 0 {
		return 0
	}

	timeMap := make(map[string]int)
	for _, reminder := range reminders {
		timeMap[reminder.TargetTime]++
	}

	// 计算最频繁的时间
	maxCount := 0
	totalCount := len(reminders)
	for _, count := range timeMap {
		if count > maxCount {
			maxCount = count
		}
	}

	return float64(maxCount) / float64(totalCount)
}

// getReminderTitles 获取提醒标题
func getReminderTitles(reminders []*models.Reminder, limit int) []string {
	var titles []string
	for i, reminder := range reminders {
		if i >= limit {
			break
		}
		titles = append(titles, reminder.Title)
	}
	return titles
}

// formatInterval 格式化间隔
func formatInterval(days float64) string {
	if days < 1 {
		return fmt.Sprintf("%.0f小时", days*24)
	} else if days < 30 {
		return fmt.Sprintf("%.0f天", days)
	} else {
		return fmt.Sprintf("%.0f个月", days/30)
	}
}
