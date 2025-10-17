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
)

// ReminderSuggestion 提醒建议结构
type ReminderSuggestion struct {
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	SuggestedSchedule string  `json:"suggested_schedule"`
	Reason            string  `json:"reason"`
	Score             float64 `json:"score"`
}

// SuggestionServiceConfig 建议服务配置
type SuggestionServiceConfig struct {
	AnalysisWindow time.Duration
	MaxSuggestions int
}

// reminderSuggestionServiceImpl 提醒建议服务实现
type reminderSuggestionServiceImpl struct {
	reminderRepo    interfaces.ReminderRepository
	reminderLogRepo interfaces.ReminderLogRepository
	contextManager  ContextManagerService

	analysisWindow time.Duration
	maxSuggestions int
	nowFunc        func() time.Time
}

// behaviorProfile 用户行为画像
type behaviorProfile struct {
	Reminders           []*models.Reminder
	ActiveHourCounts    map[int]int
	ActiveHours         []int
	ReminderSummaries   map[uint]*reminderSummary
	AverageDelayMinutes float64
	TotalCompleted      int
	TotalInteractions   int
}

type reminderSummary struct {
	Reminder            *models.Reminder
	Total               int
	Completed           int
	Skipped             int
	AverageDelayMinutes float64
}

// NewReminderSuggestionService 创建提醒建议服务
func NewReminderSuggestionService(
	reminderRepo interfaces.ReminderRepository,
	reminderLogRepo interfaces.ReminderLogRepository,
	contextManager ContextManagerService,
	config SuggestionServiceConfig,
) ReminderSuggestionService {
	window := config.AnalysisWindow
	if window <= 0 {
		window = 30 * 24 * time.Hour
	}

	maxSuggestions := config.MaxSuggestions
	if maxSuggestions <= 0 {
		maxSuggestions = 3
	}

	return &reminderSuggestionServiceImpl{
		reminderRepo:    reminderRepo,
		reminderLogRepo: reminderLogRepo,
		contextManager:  contextManager,
		analysisWindow:  window,
		maxSuggestions:  maxSuggestions,
		nowFunc:         time.Now,
	}
}

// GenerateSuggestions 生成提醒建议
func (s *reminderSuggestionServiceImpl) GenerateSuggestions(
	ctx context.Context,
	userID uint,
	contextState *models.ConversationContextState,
) ([]ReminderSuggestion, error) {
	if userID == 0 {
		return nil, fmt.Errorf("userID is required")
	}

	if contextState == nil && s.contextManager != nil {
		state, err := s.contextManager.GetContext(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("load context failed: %w", err)
		}
		contextState = state
	}

	profile, err := s.analyzeBehavior(ctx, userID)
	if err != nil {
		return nil, err
	}

	suggestions := s.buildSuggestions(profile, contextState)

	sort.SliceStable(suggestions, func(i, j int) bool {
		if suggestions[i].Score == suggestions[j].Score {
			return suggestions[i].Title < suggestions[j].Title
		}
		return suggestions[i].Score > suggestions[j].Score
	})

	if len(suggestions) > s.maxSuggestions {
		suggestions = suggestions[:s.maxSuggestions]
	}

	return suggestions, nil
}

func (s *reminderSuggestionServiceImpl) analyzeBehavior(ctx context.Context, userID uint) (*behaviorProfile, error) {
	since := s.nowFunc().Add(-s.analysisWindow)

	reminders, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load reminders failed: %w", err)
	}

	logs, err := s.reminderLogRepo.GetUserLogs(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("load reminder logs failed: %w", err)
	}

	profile := &behaviorProfile{
		Reminders:         reminders,
		ActiveHourCounts:  make(map[int]int),
		ReminderSummaries: make(map[uint]*reminderSummary),
	}

	for _, reminder := range reminders {
		profile.ReminderSummaries[reminder.ID] = &reminderSummary{
			Reminder: reminder,
		}
	}

	var totalDelay float64
	var delaySamples int

	for _, log := range logs {
		if log == nil {
			continue
		}

		summary, exists := profile.ReminderSummaries[log.ReminderID]
		if !exists {
			continue
		}

		summary.Total++
		profile.TotalInteractions++

		if log.IsCompleted() {
			summary.Completed++
			profile.TotalCompleted++

			hour := log.ScheduledTime.Hour()
			if log.ResponseTime != nil {
				hour = log.ResponseTime.Hour()
			}
			profile.ActiveHourCounts[hour]++
		} else if log.Status == models.ReminderStatusSkipped {
			summary.Skipped++
		}

		if log.ResponseTime != nil {
			delay := log.ResponseTime.Sub(log.ScheduledTime).Minutes()
			if delay > 0 {
				totalDelay += delay
				delaySamples++
			}
		}
	}

	if delaySamples > 0 {
		profile.AverageDelayMinutes = totalDelay / float64(delaySamples)
	}

	for _, summary := range profile.ReminderSummaries {
		if summary.Total > 0 && summary.Completed > 0 {
			completedLogs := make([]*models.ReminderLog, 0, len(logs))
			for _, log := range logs {
				if log != nil && log.ReminderID == summary.Reminder.ID && log.ResponseTime != nil && log.IsCompleted() {
					completedLogs = append(completedLogs, log)
				}
			}

			if len(completedLogs) > 0 {
				var delaySum float64
				var delayCount int
				for _, log := range completedLogs {
					delay := log.ResponseTime.Sub(log.ScheduledTime).Minutes()
					if delay > 0 {
						delaySum += delay
						delayCount++
					}
				}
				if delayCount > 0 {
					summary.AverageDelayMinutes = delaySum / float64(delayCount)
				}
			}
		}
	}

	profile.ActiveHours = rankHours(profile.ActiveHourCounts)

	return profile, nil
}

func (s *reminderSuggestionServiceImpl) buildSuggestions(profile *behaviorProfile, contextState *models.ConversationContextState) []ReminderSuggestion {
	if profile == nil {
		return nil
	}

	suggestions := make([]ReminderSuggestion, 0)
	topHour := defaultPreferredHour(profile.ActiveHours)
	topHourText := fmt.Sprintf("%02d:00", topHour)

	contextTask := extractEntityValue(contextState, "task")
	contextRecurrence := extractEntityValue(contextState, "recurrence")

	if contextTask != "" && !hasSimilarReminder(profile.Reminders, contextTask) {
		recurrenceText := "每天"
		if contextRecurrence != "" {
			recurrenceText = normalizeRecurrence(contextRecurrence)
		}

		suggestions = append(suggestions, ReminderSuggestion{
			Title:             fmt.Sprintf("为「%s」建立固定提醒", contextTask),
			Description:       fmt.Sprintf("结合最近的对话，看起来「%s」是你想坚持的任务。", contextTask),
			SuggestedSchedule: fmt.Sprintf("%s %s 开始", recurrenceText, topHourText),
			Reason:            fmt.Sprintf("你通常会在 %s 前后更活跃，适合安排「%s」。", topHourText, contextTask),
			Score:             0.9,
		})
	}

	for _, summary := range profile.ReminderSummaries {
		if summary.Total < 3 {
			continue
		}
		completionRate := float64(summary.Completed) / float64(summary.Total)
		if completionRate >= 0.6 {
			continue
		}
		title := summary.Reminder.Title
		reason := fmt.Sprintf("最近完成率约 %.0f%%，建议尝试调整到 %s 开始，以贴合你的活跃时段。", completionRate*100, topHourText)
		if summary.AverageDelayMinutes > 0 {
			reason += fmt.Sprintf(" 你通常会延后 %.0f 分钟才完成。", summary.AverageDelayMinutes)
		}

		suggestions = append(suggestions, ReminderSuggestion{
			Title:             fmt.Sprintf("优化「%s」的提醒时间", title),
			Description:       fmt.Sprintf("将提醒时间调整到你更容易达成的时间段，提升完成率。"),
			SuggestedSchedule: fmt.Sprintf("建议改到 %s 并预留缓冲", topHourText),
			Reason:            reason,
			Score:             0.75,
		})
	}

	if profile.AverageDelayMinutes >= 30 {
		suggestions = append(suggestions, ReminderSuggestion{
			Title:             "延迟缓冲优化",
			Description:       "近期提醒经常被延迟完成，可以调整提醒时间或增加提前提示。",
			SuggestedSchedule: fmt.Sprintf("将提醒时间后移 %d 分钟，或增加一个提前提醒。", int(math.Round(profile.AverageDelayMinutes))),
			Reason:            fmt.Sprintf("平均会延后 %.0f 分钟才完成提醒。", profile.AverageDelayMinutes),
			Score:             0.65,
		})
	}

	if len(suggestions) == 0 && len(profile.Reminders) > 0 {
		topReminder := selectTopReminder(profile.ReminderSummaries)
		if topReminder != nil {
			suggestions = append(suggestions, ReminderSuggestion{
				Title:             fmt.Sprintf("继续巩固「%s」", topReminder.Reminder.Title),
				Description:       "保持高完成率的习惯，同时尝试在同一时间段扩展其他小目标。",
				SuggestedSchedule: fmt.Sprintf("维持在 %s，并配对一个 5 分钟的整理任务。", topHourText),
				Reason:            fmt.Sprintf("该提醒完成率约 %.0f%%，说明该时间段对你非常适合。", completionRate(topReminder)*100),
				Score:             0.55,
			})
		}
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, ReminderSuggestion{
			Title:             "尝试一个 5 分钟复盘提醒",
			Description:       "在晚间安排一个简短复盘，用于记录今日亮点和明日待办。",
			SuggestedSchedule: fmt.Sprintf("每天 %s", topHourText),
			Reason:            "晚间是专注度较高的时间段，适合进行总结和计划。",
			Score:             0.5,
		})
	}

	return suggestions
}

func rankHours(counts map[int]int) []int {
	type hourCount struct {
		Hour  int
		Count int
	}

	items := make([]hourCount, 0, len(counts))
	for hour, count := range counts {
		items = append(items, hourCount{Hour: hour, Count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Hour < items[j].Hour
		}
		return items[i].Count > items[j].Count
	})

	result := make([]int, 0, len(items))
	for _, item := range items {
		result = append(result, item.Hour)
	}

	return result
}

func defaultPreferredHour(hours []int) int {
	if len(hours) > 0 {
		return hours[0]
	}
	return 20
}

func hasSimilarReminder(reminders []*models.Reminder, taskName string) bool {
	if taskName == "" {
		return false
	}
	lowerTask := strings.ToLower(taskName)
	for _, reminder := range reminders {
		if reminder == nil {
			continue
		}
		title := strings.ToLower(reminder.Title)
		desc := strings.ToLower(reminder.Description)
		if strings.Contains(title, lowerTask) || strings.Contains(desc, lowerTask) {
			return true
		}
	}
	return false
}

func extractEntityValue(contextState *models.ConversationContextState, key string) string {
	if contextState == nil || contextState.Entities == nil {
		return ""
	}
	entity, ok := contextState.Entities[key]
	if !ok {
		return ""
	}

	switch value := entity.Value.(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	default:
		return ""
	}
}

func normalizeRecurrence(recurrence string) string {
	lower := strings.ToLower(recurrence)
	switch lower {
	case "weekly", "周", "一周一次":
		return "每周"
	case "workday", "weekday", "工作日":
		return "工作日"
	case "daily", "每天", "每日":
		return "每天"
	default:
		return recurrence
	}
}

func selectTopReminder(summaries map[uint]*reminderSummary) *reminderSummary {
	var best *reminderSummary
	var bestScore float64

	for _, summary := range summaries {
		if summary == nil || summary.Total == 0 {
			continue
		}
		score := completionRate(summary)
		if score > bestScore {
			best = summary
			bestScore = score
		}
	}

	return best
}

func completionRate(summary *reminderSummary) float64 {
	if summary == nil || summary.Total == 0 {
		return 0
	}
	return float64(summary.Completed) / float64(summary.Total)
}
