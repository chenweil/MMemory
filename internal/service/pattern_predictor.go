package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/logger"

	"github.com/sirupsen/logrus"
)

// PatternPredictor 模式预测器
type PatternPredictor struct {
	patternDetector PatternDetector
	reminderService ReminderService
	log             *logrus.Logger
}

// NewPatternPredictor 创建模式预测器
func NewPatternPredictor(patternDetector PatternDetector, reminderService ReminderService) *PatternPredictor {
	return &PatternPredictor{
		patternDetector: patternDetector,
		reminderService: reminderService,
		log:             logger.GetLogger(),
	}
}

// PredictedReminder 预测的提醒
type PredictedReminder struct {
	Title           string          `json:"title"`
	PredictedTime   string          `json:"predicted_time"`
	PredictedDays   []string        `json:"predicted_days"`
	Confidence      float64         `json:"confidence"`
	PatternType     PatternType     `json:"pattern_type"`
	BasedOnPatterns []DetectedPattern `json:"based_on_patterns"`
	SuggestedAction string          `json:"suggested_action"`
}

// PredictReminders 预测用户可能需要的提醒
func (pp *PatternPredictor) PredictReminders(ctx context.Context, userID uint) ([]PredictedReminder, error) {
	// 获取用户提醒
	reminders, err := pp.reminderService.GetUserReminders(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(reminders) == 0 {
		return nil, nil
	}

	// 检测模式
	patterns, err := pp.patternDetector.DetectPatterns(ctx, reminders)
	if err != nil {
		return nil, err
	}

	// 基于模式生成预测
	return pp.generatePredictions(reminders, patterns), nil
}

// generatePredictions 基于检测到的模式生成预测
func (pp *PatternPredictor) generatePredictions(reminders []*models.Reminder, patterns []DetectedPattern) []PredictedReminder {
	var predictions []PredictedReminder

	// 按模式分组提醒
	patternGroups := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		patternGroups[reminder.Title] = append(patternGroups[reminder.Title], reminder)
	}

	// 基于每日模式生成预测
	dailyPatterns := filterPatternsByType(patterns, PatternDaily)
	for _, pattern := range dailyPatterns {
		prediction := PredictedReminder{
			Title:           fmt.Sprintf("建议: 每天 %s", extractCoreTitle(pattern.Title)),
			PredictedTime:   pattern.Frequency,
			PredictedDays:   []string{"每天"},
			Confidence:      pattern.Confidence,
			PatternType:     PatternDaily,
			BasedOnPatterns: []DetectedPattern{pattern},
			SuggestedAction: "创建每日提醒",
		}
		predictions = append(predictions, prediction)
	}

	// 基于每周模式生成预测
	weeklyPatterns := filterPatternsByType(patterns, PatternWeekly)
	for _, pattern := range weeklyPatterns {
		prediction := PredictedReminder{
			Title:           fmt.Sprintf("建议: %s", pattern.Title),
			PredictedTime:   pattern.Frequency,
			PredictedDays:   parseDaysFromFrequency(pattern.Frequency),
			Confidence:      pattern.Confidence,
			PatternType:     PatternWeekly,
			BasedOnPatterns: []DetectedPattern{pattern},
			SuggestedAction: "创建每周提醒",
		}
		predictions = append(predictions, prediction)
	}

	// 基于间隔模式生成预测
	intervalPatterns := filterPatternsByType(patterns, PatternInterval)
	for _, pattern := range intervalPatterns {
		prediction := PredictedReminder{
			Title:           fmt.Sprintf("建议: 每%s %s", pattern.Frequency, pattern.Title),
			PredictedTime:   "待定",
			PredictedDays:   []string{},
			Confidence:      pattern.Confidence,
			PatternType:     PatternInterval,
			BasedOnPatterns: []DetectedPattern{pattern},
			SuggestedAction: "创建周期提醒",
		}
		predictions = append(predictions, prediction)
	}

	return predictions
}

// GetOptimizationSuggestions 获取优化建议
func (pp *PatternPredictor) GetOptimizationSuggestions(ctx context.Context, userID uint) ([]ReminderSuggestion, error) {
	reminders, err := pp.reminderService.GetUserReminders(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(reminders) == 0 {
		return nil, nil
	}

	var suggestions []ReminderSuggestion

	// 检测时间冲突
	conflictSuggestion := pp.detectTimeConflicts(reminders)
	if conflictSuggestion != nil {
		suggestions = append(suggestions, *conflictSuggestion)
	}

	// 检测重复提醒
	duplicateSuggestion := pp.detectDuplicates(reminders)
	if duplicateSuggestion != nil {
		suggestions = append(suggestions, *duplicateSuggestion)
	}

	// 检测不规律的提醒
	irregularSuggestion := pp.detectIrregularReminders(reminders)
	if irregularSuggestion != nil {
		suggestions = append(suggestions, *irregularSuggestion)
	}

	// 检测可以合并的提醒
	mergeSuggestion := pp.detectMergeableReminders(reminders)
	if mergeSuggestion != nil {
		suggestions = append(suggestions, *mergeSuggestion)
	}

	return suggestions, nil
}

// detectTimeConflicts 检测时间冲突
func (pp *PatternPredictor) detectTimeConflicts(reminders []*models.Reminder) *ReminderSuggestion {
	timeMap := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		if reminder.IsActive {
			timeMap[reminder.TargetTime] = append(timeMap[reminder.TargetTime], reminder)
		}
	}

	for time, list := range timeMap {
		if len(list) >= 3 {
			return &ReminderSuggestion{
				Title:             "时间冲突检测",
				Description:       fmt.Sprintf("您在 %s 有 %d 个提醒，可能过于密集。建议重新安排时间以提高完成率。", time, len(list)),
				SuggestedSchedule: fmt.Sprintf("重新安排 %d 个提醒的时间", len(list)),
				Reason:            "多个提醒在同一时间点可能导致遗漏",
				Score:             0.8,
			}
		}
	}
	return nil
}

// detectDuplicates 检测重复提醒
func (pp *PatternPredictor) detectDuplicates(reminders []*models.Reminder) *ReminderSuggestion {
	titleMap := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		if reminder.IsActive {
			titleMap[reminder.Title] = append(titleMap[reminder.Title], reminder)
		}
	}

	for title, list := range titleMap {
		if len(list) >= 2 {
			return &ReminderSuggestion{
				Title:             "重复提醒检测",
				Description:       fmt.Sprintf("您有 %d 个标题相似的提醒：'%s'。建议合并为一个重复提醒。", len(list), title),
				SuggestedSchedule: fmt.Sprintf("合并 %d 个提醒", len(list)),
				Reason:            "重复提醒可能导致提醒疲劳",
				Score:             0.9,
			}
		}
	}
	return nil
}

// detectIrregularReminders 检测不规律的提醒
func (pp *PatternPredictor) detectIrregularReminders(reminders []*models.Reminder) *ReminderSuggestion {
	onceReminders := make([]*models.Reminder, 0)
	for _, reminder := range reminders {
		if reminder.IsOnce() {
			onceReminders = append(onceReminders, reminder)
		}
	}

	if len(onceReminders) >= 5 {
		// 检查是否有相似的标题
		titleGroups := groupRemindersByTitle(onceReminders)
		for title, group := range titleGroups {
			if len(group) >= 3 {
				return &ReminderSuggestion{
					Title:             "规律性建议",
					Description:       fmt.Sprintf("您有 %d 个一次性的'%s'提醒。建议创建固定周期提醒以保持习惯。", len(group), title),
					SuggestedSchedule: "转换为周期提醒",
					Reason:            "一次性提醒难以形成习惯，周期提醒更有效",
					Score:             0.7,
				}
			}
		}
	}
	return nil
}

// detectMergeableReminders 检测可以合并的提醒
func (pp *PatternPredictor) detectMergeableReminders(reminders []*models.Reminder) *ReminderSuggestion {
	// 检测同一天内多个相关提醒
	dayGroups := make(map[string][]*models.Reminder)
	for _, reminder := range reminders {
		if reminder.IsActive {
			day := reminder.CreatedAt.Format("2006-01-02")
			dayGroups[day] = append(dayGroups[day], reminder)
		}
	}

	for day, list := range dayGroups {
		if len(list) >= 4 {
			return &ReminderSuggestion{
				Title:             "合并提醒建议",
				Description:       fmt.Sprintf("您在 %s 创建了 %d 个提醒。建议评估是否可以合并或简化。", day, len(list)),
				SuggestedSchedule: "简化提醒设置",
				Reason:            "过多的提醒可能导致执行困难",
				Score:             0.6,
			}
		}
	}
	return nil
}

// GetUserBehaviorAnalysis 获取用户行为分析
func (pp *PatternPredictor) GetUserBehaviorAnalysis(ctx context.Context, userID uint) (*UserBehaviorAnalysis, error) {
	reminders, err := pp.reminderService.GetUserReminders(ctx, userID)
	if err != nil {
		return nil, err
	}

	analysis := &UserBehaviorAnalysis{
		UserID:          userID,
		AnalysisDate:    time.Now(),
		TotalReminders:  len(reminders),
		ActiveReminders: 0,
		HabitReminders:  0,
		TaskReminders:   0,
		PatternStats:    make(map[string]int),
	}

	activeCount := 0
	habitCount := 0
	taskCount := 0

	for _, reminder := range reminders {
		if reminder.IsActive {
			activeCount++
		}

		if reminder.Type == models.ReminderTypeHabit {
			habitCount++
		} else {
			taskCount++
		}

		// 统计模式类型
		if reminder.IsDaily() {
			analysis.PatternStats["daily"]++
		} else if reminder.IsWeekly() {
			analysis.PatternStats["weekly"]++
		} else if reminder.IsOnce() {
			analysis.PatternStats["once"]++
		}
	}

	analysis.ActiveReminders = activeCount
	analysis.HabitReminders = habitCount
	analysis.TaskReminders = taskCount

	// 计算习惯养成率
	if habitCount > 0 {
		analysis.HabitFormationRate = float64(habitCount) / float64(len(reminders))
	}

	return analysis, nil
}

// UserBehaviorAnalysis 用户行为分析
type UserBehaviorAnalysis struct {
	UserID            uint              `json:"user_id"`
	AnalysisDate      time.Time         `json:"analysis_date"`
	TotalReminders    int               `json:"total_reminders"`
	ActiveReminders   int               `json:"active_reminders"`
	HabitReminders    int               `json:"habit_reminders"`
	TaskReminders     int               `json:"task_reminders"`
	PatternStats      map[string]int    `json:"pattern_stats"`
	HabitFormationRate float64          `json:"habit_formation_rate"`
}

// filterPatternsByType 按类型过滤模式
func filterPatternsByType(patterns []DetectedPattern, patternType PatternType) []DetectedPattern {
	var filtered []DetectedPattern
	for _, p := range patterns {
		if p.Type == patternType {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// extractCoreTitle 提取核心标题
func extractCoreTitle(title string) string {
	// 移除常见前缀
	title = removePrefix(title, "每天")
	title = removePrefix(title, "每天提醒我")
	title = removePrefix(title, "提醒我")
	title = removePrefix(title, "请提醒我")
	return title
}

// removePrefix 移除前缀
func removePrefix(s, prefix string) string {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// parseDaysFromFrequency 从频率解析天数
func parseDaysFromFrequency(frequency string) []string {
	days := make([]string, 0)
	dayMap := map[int]string{
		0: "周日", 1: "周一", 2: "周二", 3: "周三",
		4: "周四", 5: "周五", 6: "周六",
	}

	// 简单解析
	for i := 0; i <= 6; i++ {
		if containsDayString(frequency, dayMap[i]) {
			days = append(days, dayMap[i])
		}
	}

	if len(days) == 0 {
		days = append(days, "指定日期")
	}

	return days
}

// containsDayString 检查频率字符串是否包含某天
func containsDayString(frequency, day string) bool {
	return len(frequency) >= len(day) && (frequency == day || containsString(frequency, day))
}

// containsString 检查子字符串
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SortPredictionsByConfidence 按置信度排序预测
func SortPredictionsByConfidence(predictions []PredictedReminder) {
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Confidence > predictions[j].Confidence
	})
}

// GetTopPredictions 获取置信度最高的预测
func GetTopPredictions(predictions []PredictedReminder, limit int) []PredictedReminder {
	if len(predictions) <= limit {
		return predictions
	}
	return predictions[:limit]
}
