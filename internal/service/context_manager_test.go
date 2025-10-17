package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gormSqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
	repoSqlite "mmemory/internal/repository/sqlite"
)

func newTestContextManager(t *testing.T) *ContextManager {
	t.Helper()

	db, err := gorm.Open(gormSqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	if err := db.AutoMigrate(
		&models.ConversationContext{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := repoSqlite.NewConversationContextRepository(db)

	manager := NewContextManager(
		repo,
		&DefaultEntityExtractor{},
		&DefaultIntentTracker{},
		ContextManagerConfig{
			MaxMessages: 3,
			DefaultTTL:  10 * time.Minute,
		},
	)

	manager.nowFunc = func() time.Time {
		return time.Date(2025, 10, 14, 12, 0, 0, 0, time.UTC)
	}

	return manager
}

func TestContextManager_ProcessMessage_CreateNew(t *testing.T) {
	manager := newTestContextManager(t)

	state, err := manager.ProcessMessage(context.Background(), ProcessMessageInput{
		UserID:  1,
		Message: "每天晚上提醒我读书",
		Role:    "user",
	})
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, uint(1), state.UserID)
	assert.Equal(t, "daily", state.Entities["recurrence"].Value)
	assert.Equal(t, "每天晚上提醒我读书", state.Entities["task"].Value)
	assert.Len(t, state.Messages, 1)
	assert.Equal(t, "create_reminder", state.Intent)
}

func TestContextManager_ProcessMessage_MaxMessages(t *testing.T) {
	manager := newTestContextManager(t)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := manager.ProcessMessage(ctx, ProcessMessageInput{
			UserID:  2,
			Message: "message",
			Role:    "user",
		})
		assert.NoError(t, err)
	}

	state, err := manager.GetContext(ctx, 2)
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Len(t, state.Messages, 3)
}

func TestContextManager_UpdateState(t *testing.T) {
	manager := newTestContextManager(t)

	ctx := context.Background()
	_, err := manager.ProcessMessage(ctx, ProcessMessageInput{
		UserID:  3,
		Message: "提醒我明天开会",
		Role:    "user",
	})
	assert.NoError(t, err)

	err = manager.UpdateContextState(ctx, UpdateContextStateInput{
		UserID: 3,
		State:  "awaiting_confirmation",
		Intent: "confirm_reminder",
		Entities: map[string]models.ConversationEntity{
			"datetime": {
				Name:  "datetime",
				Value: "2025-10-15T10:00:00Z",
			},
		},
	})
	assert.NoError(t, err)

	state, err := manager.GetContext(ctx, 3)
	assert.NoError(t, err)
	assert.Equal(t, "awaiting_confirmation", state.State)
	assert.Equal(t, "confirm_reminder", state.Intent)
	assert.Equal(t, "2025-10-15T10:00:00Z", state.Entities["datetime"].Value)
}

func TestContextManager_CleanupExpired(t *testing.T) {
	manager := newTestContextManager(t)

	ctx := context.Background()

	// Process a message to create context
	_, err := manager.ProcessMessage(ctx, ProcessMessageInput{
		UserID:  4,
		Message: "提醒我学习",
		Role:    "user",
	})
	assert.NoError(t, err)

	// Advance time beyond TTL and cleanup
	manager.nowFunc = func() time.Time {
		return time.Date(2025, 10, 14, 12, 11, 0, 0, time.UTC)
	}

	err = manager.CleanupExpired(ctx)
	assert.NoError(t, err)

	state, err := manager.GetContext(ctx, 4)
	assert.NoError(t, err)
	assert.Nil(t, state)
}
