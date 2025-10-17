package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormSqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
	repoSqlite "mmemory/internal/repository/sqlite"
)

func setupSuggestionService(t *testing.T) (ReminderSuggestionService, *gorm.DB, func(time.Time)) {
	t.Helper()

	db, err := gorm.Open(gormSqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Reminder{}, &models.ReminderLog{}))

	reminderRepo := repoSqlite.NewReminderRepository(db)
	reminderLogRepo := repoSqlite.NewReminderLogRepository(db)

	service := NewReminderSuggestionService(
		reminderRepo,
		reminderLogRepo,
		nil,
		SuggestionServiceConfig{
			AnalysisWindow: 14 * 24 * time.Hour,
			MaxSuggestions: 5,
		},
	)

	// 提供一个设置时间的函数
	setNow := func(t time.Time) {
		if impl, ok := service.(*reminderSuggestionServiceImpl); ok {
			impl.nowFunc = func() time.Time { return t }
		}
	}

	// 设置默认时间
	setNow(time.Date(2025, 10, 14, 12, 0, 0, 0, time.UTC))

	return service, db, setNow
}

func TestReminderSuggestionService_WithContextTask(t *testing.T) {
	service, db, _ := setupSuggestionService(t)

	reminderRepo := repoSqlite.NewReminderRepository(db)
	reminder := &models.Reminder{
		UserID:          1,
		Title:           "晨跑",
		Description:     "早起跑步",
		Type:            models.ReminderTypeHabit,
		TargetTime:      "07:00:00",
		SchedulePattern: string(models.SchedulePatternDaily),
		IsActive:        true,
	}
	require.NoError(t, reminderRepo.Create(context.Background(), reminder))

	logRepo := repoSqlite.NewReminderLogRepository(db)
	now := time.Date(2025, 10, 14, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		scheduled := now.Add(-time.Duration(i+1) * 24 * time.Hour).Add(-2 * time.Hour)
		response := scheduled.Add(30 * time.Minute)
		require.NoError(t, logRepo.Create(context.Background(), &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: scheduled,
			Status:        models.ReminderStatusCompleted,
			ResponseTime:  &response,
		}))
	}

	contextState := &models.ConversationContextState{
		UserID:       1,
		Intent:       "suggestion_request",
		Entities:     map[string]models.ConversationEntity{"task": {Name: "task", Value: "晚间阅读"}},
		LastActivity: now,
	}

	suggestions, err := service.GenerateSuggestions(context.Background(), 1, contextState)
	require.NoError(t, err)
	require.NotEmpty(t, suggestions)

	assert.Contains(t, suggestions[0].Title, "晚间阅读")
	assert.Contains(t, suggestions[0].SuggestedSchedule, ":")
}

func TestReminderSuggestionService_LowCompletionReminder(t *testing.T) {
	service, db, _ := setupSuggestionService(t)
	ctx := context.Background()

	reminderRepo := repoSqlite.NewReminderRepository(db)
	reminder := &models.Reminder{
		UserID:          2,
		Title:           "周报整理",
		Description:     "准备周报",
		Type:            models.ReminderTypeTask,
		TargetTime:      "18:00:00",
		SchedulePattern: string(models.SchedulePatternWeekly),
		IsActive:        true,
	}
	require.NoError(t, reminderRepo.Create(ctx, reminder))

	logRepo := repoSqlite.NewReminderLogRepository(db)
	now := time.Date(2025, 10, 14, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		scheduled := now.Add(-time.Duration(i+1) * 48 * time.Hour)
		status := models.ReminderStatusCompleted
		if i%2 == 0 {
			status = models.ReminderStatusSkipped
		}
		var response *time.Time
		if status == models.ReminderStatusCompleted {
			resp := scheduled.Add(15 * time.Minute)
			response = &resp
		}
		require.NoError(t, logRepo.Create(ctx, &models.ReminderLog{
			ReminderID:    reminder.ID,
			ScheduledTime: scheduled,
			Status:        status,
			ResponseTime:  response,
		}))
	}

	suggestions, err := service.GenerateSuggestions(ctx, 2, nil)
	require.NoError(t, err)
	require.NotEmpty(t, suggestions)

	var found bool
	for _, suggestion := range suggestions {
		if strings.Contains(suggestion.Title, "周报整理") || strings.Contains(suggestion.Title, "优化") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected suggestion referencing low completion reminder")
}
