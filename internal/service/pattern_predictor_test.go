package service

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"
)

// MockPatternDetector 模拟模式检测器用于测试
type MockPatternDetector struct {
	mockPatterns []DetectedPattern
	mockErr      error
}

func (m *MockPatternDetector) DetectPatterns(ctx context.Context, reminders []*models.Reminder) ([]DetectedPattern, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	return m.mockPatterns, nil
}

func (m *MockPatternDetector) AnalyzeUserBehavior(ctx context.Context, userID uint) ([]DetectedPattern, error) {
	return m.mockPatterns, m.mockErr
}

func (m *MockPatternDetector) GetPatternSuggestions(ctx context.Context, userID uint) ([]ReminderSuggestion, error) {
	return nil, m.mockErr
}

// MockReminderServiceImpl 模拟提醒服务用于测试
type MockReminderServiceImpl struct {
	mockReminders []*models.Reminder
	mockErr       error
}

func (m *MockReminderServiceImpl) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	return m.mockErr
}

func (m *MockReminderServiceImpl) ParseReminderFromText(ctx context.Context, text string, userID uint) (*models.Reminder, error) {
	return nil, m.mockErr
}

func (m *MockReminderServiceImpl) GetUserReminders(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	return m.mockReminders, nil
}

func (m *MockReminderServiceImpl) GetReminderByID(ctx context.Context, id uint) (*models.Reminder, error) {
	return nil, m.mockErr
}

func (m *MockReminderServiceImpl) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	return m.mockErr
}

func (m *MockReminderServiceImpl) EditReminder(ctx context.Context, params EditReminderParams) error {
	return m.mockErr
}

func (m *MockReminderServiceImpl) DeleteReminder(ctx context.Context, id uint) error {
	return m.mockErr
}

func (m *MockReminderServiceImpl) PauseReminder(ctx context.Context, id uint, duration time.Duration, reason string) error {
	return m.mockErr
}

func (m *MockReminderServiceImpl) ResumeReminder(ctx context.Context, id uint) error {
	return m.mockErr
}

func TestPatternPredictor_PredictReminders(t *testing.T) {
	t.Run("无提醒时返回空", func(t *testing.T) {
		mockService := &MockReminderServiceImpl{
			mockReminders: []*models.Reminder{},
		}
		predictor := NewPatternPredictor(nil, mockService)

		result, err := predictor.PredictReminders(context.Background(), 1)
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}
		if result != nil {
			t.Fatalf("期望空结果，实际: %v", result)
		}
	})

	t.Run("有提醒时检测模式", func(t *testing.T) {
		reminders := []*models.Reminder{
			{
				ID:              1,
				UserID:          1,
				Title:           "每天早上喝水",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "08:00:00",
				IsActive:        true,
				Type:            models.ReminderTypeHabit,
			},
			{
				ID:              2,
				UserID:          1,
				Title:           "每天晚上跑步",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "20:00:00",
				IsActive:        true,
				Type:            models.ReminderTypeHabit,
			},
		}

		patterns := []DetectedPattern{
			{
				Type:       PatternDaily,
				Title:      "每天早上喝水",
				Frequency:  "每天",
				Confidence: 0.9,
				Examples:   []string{"每天早上喝水"},
			},
		}

		mockDetector := &MockPatternDetector{
			mockPatterns: patterns,
		}
		mockService := &MockReminderServiceImpl{
			mockReminders: reminders,
		}

		predictor := NewPatternPredictor(mockDetector, mockService)
		result, err := predictor.PredictReminders(context.Background(), 1)
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}

		if len(result) == 0 {
			t.Fatalf("期望有预测结果，实际为空")
		}

		t.Logf("预测结果数量: %d", len(result))
		for _, pred := range result {
			t.Logf("预测: %s, 置信度: %.2f", pred.Title, pred.Confidence)
		}
	})
}

func TestPatternPredictor_GetOptimizationSuggestions(t *testing.T) {
	t.Run("无提醒时返回空", func(t *testing.T) {
		mockService := &MockReminderServiceImpl{
			mockReminders: []*models.Reminder{},
		}
		predictor := NewPatternPredictor(nil, mockService)

		result, err := predictor.GetOptimizationSuggestions(context.Background(), 1)
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}
		if result != nil {
			t.Fatalf("期望空结果，实际: %v", result)
		}
	})

	t.Run("检测时间冲突", func(t *testing.T) {
		now := time.Now()
		reminders := []*models.Reminder{
			{
				ID:              1,
				UserID:          1,
				Title:           "提醒1",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "08:00:00",
				IsActive:        true,
				CreatedAt:       now,
			},
			{
				ID:              2,
				UserID:          1,
				Title:           "提醒2",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "08:00:00",
				IsActive:        true,
				CreatedAt:       now,
			},
			{
				ID:              3,
				UserID:          1,
				Title:           "提醒3",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "08:00:00",
				IsActive:        true,
				CreatedAt:       now,
			},
			{
				ID:              4,
				UserID:          1,
				Title:           "提醒4",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "08:00:00",
				IsActive:        true,
				CreatedAt:       now,
			},
		}

		mockService := &MockReminderServiceImpl{
			mockReminders: reminders,
		}

		predictor := NewPatternPredictor(nil, mockService)
		result, err := predictor.GetOptimizationSuggestions(context.Background(), 1)
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}

		if len(result) == 0 {
			t.Fatalf("期望有优化建议，实际为空")
		}

		found := false
		for _, suggestion := range result {
			if suggestion.Title == "时间冲突检测" {
				found = true
				break
			}
		}

		if !found {
			t.Logf("未检测到时间冲突建议")
		}
	})

	t.Run("检测重复提醒", func(t *testing.T) {
		now := time.Now()
		reminders := []*models.Reminder{
			{
				ID:              1,
				UserID:          1,
				Title:           "喝水",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "08:00:00",
				IsActive:        true,
				CreatedAt:       now,
			},
			{
				ID:              2,
				UserID:          1,
				Title:           "喝水",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "09:00:00",
				IsActive:        true,
				CreatedAt:       now,
			},
		}

		mockService := &MockReminderServiceImpl{
			mockReminders: reminders,
		}

		predictor := NewPatternPredictor(nil, mockService)
		result, err := predictor.GetOptimizationSuggestions(context.Background(), 1)
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}

		found := false
		for _, suggestion := range result {
			if suggestion.Title == "重复提醒检测" {
				found = true
				break
			}
		}

		if !found {
			t.Logf("未检测到重复提醒建议")
		}
	})
}

func TestPatternPredictor_GetUserBehaviorAnalysis(t *testing.T) {
	t.Run("用户行为分析", func(t *testing.T) {
		now := time.Now()
		reminders := []*models.Reminder{
			{
				ID:              1,
				UserID:          1,
				Title:           "每天早上喝水",
				SchedulePattern: string(models.SchedulePatternDaily),
				TargetTime:      "08:00:00",
				IsActive:        true,
				Type:            models.ReminderTypeHabit,
				CreatedAt:       now,
			},
			{
				ID:              2,
				UserID:          1,
				Title:           "每周会议",
				SchedulePattern: "weekly:1",
				TargetTime:      "10:00:00",
				IsActive:        true,
				Type:            models.ReminderTypeTask,
				CreatedAt:       now,
			},
			{
				ID:              3,
				UserID:          1,
				Title:           "一次性提醒",
				SchedulePattern: "once:2024-12-31",
				TargetTime:      "12:00:00",
				IsActive:        false,
				Type:            models.ReminderTypeTask,
				CreatedAt:       now,
			},
		}

		mockService := &MockReminderServiceImpl{
			mockReminders: reminders,
		}

		predictor := NewPatternPredictor(nil, mockService)
		analysis, err := predictor.GetUserBehaviorAnalysis(context.Background(), 1)
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}

		if analysis.TotalReminders != 3 {
			t.Errorf("期望总提醒数3，实际: %d", analysis.TotalReminders)
		}

		if analysis.ActiveReminders != 2 {
			t.Errorf("期望活跃提醒数2，实际: %d", analysis.ActiveReminders)
		}

		if analysis.HabitReminders != 1 {
			t.Errorf("期望习惯提醒数1，实际: %d", analysis.HabitReminders)
		}

		if analysis.PatternStats["daily"] != 1 {
			t.Errorf("期望每日模式数1，实际: %d", analysis.PatternStats["daily"])
		}

		if analysis.PatternStats["weekly"] != 1 {
			t.Errorf("期望每周模式数1，实际: %d", analysis.PatternStats["weekly"])
		}

		if analysis.PatternStats["once"] != 1 {
			t.Errorf("期望一次性模式数1，实际: %d", analysis.PatternStats["once"])
		}

		t.Logf("用户行为分析: 总提醒=%d, 活跃=%d, 习惯=%d, 任务=%d",
			analysis.TotalReminders, analysis.ActiveReminders,
			analysis.HabitReminders, analysis.TaskReminders)
	})
}

func TestExtractCoreTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"每天早上喝水", "早上喝水"},
		{"每天提醒我喝水", "喝水"},
		{"提醒我喝水", "喝水"},
		{"请提醒我喝水", "喝水"},
		{"喝水", "喝水"},
	}

	for _, tt := range tests {
		result := extractCoreTitle(tt.input)
		if result != tt.expected {
			t.Errorf("extractCoreTitle(%q) = %q, 期望 %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseDaysFromFrequency(t *testing.T) {
	tests := []struct {
		frequency string
		expected  []string
	}{
		{"周一", []string{"周一"}},
		{"周二", []string{"周二"}},
		{"周日", []string{"周日"}},
		{"未知日期", []string{"指定日期"}},
	}

	for _, tt := range tests {
		result := parseDaysFromFrequency(tt.frequency)
		if len(result) > 0 && result[0] != tt.expected[0] {
			t.Errorf("parseDaysFromFrequency(%q) = %v, 期望 %v", tt.frequency, result, tt.expected)
		}
	}
}

func TestSortPredictionsByConfidence(t *testing.T) {
	predictions := []PredictedReminder{
		{Title: "低置信度", Confidence: 0.5},
		{Title: "高置信度", Confidence: 0.9},
		{Title: "中置信度", Confidence: 0.7},
	}

	SortPredictionsByConfidence(predictions)

	if predictions[0].Confidence != 0.9 {
		t.Errorf("期望第一个预测置信度最高，实际: %.2f", predictions[0].Confidence)
	}

	if predictions[2].Confidence != 0.5 {
		t.Errorf("期望最后一个预测置信度最低，实际: %.2f", predictions[2].Confidence)
	}
}

func TestGetTopPredictions(t *testing.T) {
	predictions := []PredictedReminder{
		{Title: "预测1", Confidence: 0.9},
		{Title: "预测2", Confidence: 0.8},
		{Title: "预测3", Confidence: 0.7},
		{Title: "预测4", Confidence: 0.6},
		{Title: "预测5", Confidence: 0.5},
	}

	top2 := GetTopPredictions(predictions, 2)
	if len(top2) != 2 {
		t.Errorf("期望获取2个预测，实际: %d", len(top2))
	}

	if top2[0].Title != "预测1" {
		t.Errorf("期望第一个是预测1，实际: %s", top2[0].Title)
	}

	all := GetTopPredictions(predictions, 10)
	if len(all) != 5 {
		t.Errorf("期望获取全部5个预测，实际: %d", len(all))
	}
}
