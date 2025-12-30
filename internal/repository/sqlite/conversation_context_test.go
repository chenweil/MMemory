package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
)

// setupTestDBForConversationContext 创建测试数据库
func setupTestDBForConversationContext(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.User{}, &models.ConversationContext{})
	require.NoError(t, err)

	return db
}

// createTestUserForConversationContext 创建测试用户
func createTestUserForConversationContext(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// TestConversationContextRepository_GetByUserID 测试根据用户ID获取上下文
func TestConversationContextRepository_GetByUserID(t *testing.T) {
	db := setupTestDBForConversationContext(t)
	repo := NewConversationContextRepository(db)
	ctx := context.Background()

	user := createTestUserForConversationContext(t, db)

	// 创建测试上下文
	expiresAt := time.Now().Add(24 * time.Hour)
	contextModel := &models.ConversationContext{
		UserID:       user.ID,
		SessionID:    "test-session-123",
		State:        "waiting_for_time",
		Intent:       "create_reminder",
		Channel:      "telegram",
		Locale:       "zh-CN",
		MessagesJSON: `[]`,
		TTLSeconds:   86400,
		LastActivity: time.Now(),
		ExpiresAt:    &expiresAt,
	}
	err := repo.CreateOrUpdate(ctx, contextModel)
	require.NoError(t, err)

	t.Run("获取存在的上下文", func(t *testing.T) {
		retrievedContext, err := repo.GetByUserID(ctx, user.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedContext)
		assert.Equal(t, contextModel.ID, retrievedContext.ID)
		assert.Equal(t, contextModel.UserID, retrievedContext.UserID)
		assert.Equal(t, contextModel.SessionID, retrievedContext.SessionID)
	})

	t.Run("获取不存在的上下文", func(t *testing.T) {
		retrievedContext, err := repo.GetByUserID(ctx, 9999)
		assert.NoError(t, err)
		assert.Nil(t, retrievedContext)
	})
}

// TestConversationContextRepository_CreateOrUpdate 测试创建或更新上下文
func TestConversationContextRepository_CreateOrUpdate(t *testing.T) {
	t.Run("创建新上下文", func(t *testing.T) {
		db := setupTestDBForConversationContext(t)
		repo := NewConversationContextRepository(db)
		ctx := context.Background()
		user := createTestUserForConversationContext(t, db)

		expiresAt := time.Now().Add(24 * time.Hour)
		contextModel := &models.ConversationContext{
			UserID:       user.ID,
			SessionID:    "test-session-123",
			State:        "waiting_for_time",
			Intent:       "create_reminder",
			Channel:      "telegram",
			Locale:       "zh-CN",
			MessagesJSON: `[]`,
			TTLSeconds:   86400,
			LastActivity: time.Now(),
			ExpiresAt:    &expiresAt,
		}

		err := repo.CreateOrUpdate(ctx, contextModel)
		assert.NoError(t, err)
		assert.NotZero(t, contextModel.ID)
	})

	t.Run("更新现有上下文", func(t *testing.T) {
		db := setupTestDBForConversationContext(t)
		repo := NewConversationContextRepository(db)
		ctx := context.Background()
		user := createTestUserForConversationContext(t, db)

		// 先创建上下文
		expiresAt := time.Now().Add(24 * time.Hour)
		contextModel := &models.ConversationContext{
			UserID:       user.ID,
			SessionID:    "test-session-456",
			State:        "waiting_for_time",
			Intent:       "create_reminder",
			Channel:      "telegram",
			Locale:       "zh-CN",
			MessagesJSON: `[]`,
			TTLSeconds:   86400,
			LastActivity: time.Now(),
			ExpiresAt:    &expiresAt,
		}
		err := repo.CreateOrUpdate(ctx, contextModel)
		require.NoError(t, err)

		// 更新上下文
		contextModel.State = "waiting_for_confirmation"
		contextModel.Intent = "confirm_reminder"
		contextModel.LastActivity = time.Now()

		err = repo.CreateOrUpdate(ctx, contextModel)
		assert.NoError(t, err)

		// 验证更新
		retrievedContext, err := repo.GetByUserID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, "waiting_for_confirmation", retrievedContext.State)
		assert.Equal(t, "confirm_reminder", retrievedContext.Intent)
	})

	t.Run("创建nil上下文", func(t *testing.T) {
		db := setupTestDBForConversationContext(t)
		repo := NewConversationContextRepository(db)
		ctx := context.Background()

		err := repo.CreateOrUpdate(ctx, nil)
		assert.NoError(t, err) // 应该不返回错误
	})

	t.Run("批量创建和更新", func(t *testing.T) {
		db := setupTestDBForConversationContext(t)
		repo := NewConversationContextRepository(db)
		ctx := context.Background()

		// 创建多个用户的上下文
		for i := 0; i < 3; i++ {
			expiresAt := time.Now().Add(24 * time.Hour)
			contextModel := &models.ConversationContext{
				UserID:       uint(1000 + i),
				SessionID:    "test-session-789",
				State:        "waiting_for_time",
				Intent:       "create_reminder",
				Channel:      "telegram",
				Locale:       "zh-CN",
				MessagesJSON: `[]`,
				TTLSeconds:   86400,
				LastActivity: time.Now(),
				ExpiresAt:    &expiresAt,
			}
			err := repo.CreateOrUpdate(ctx, contextModel)
			assert.NoError(t, err)
		}
	})
}

// TestConversationContextRepository_DeleteByUserID 测试根据用户ID删除上下文
func TestConversationContextRepository_DeleteByUserID(t *testing.T) {
	db := setupTestDBForConversationContext(t)
	repo := NewConversationContextRepository(db)
	ctx := context.Background()

	user := createTestUserForConversationContext(t, db)

	// 创建测试上下文
	expiresAt := time.Now().Add(24 * time.Hour)
	contextModel := &models.ConversationContext{
		UserID:       user.ID,
		SessionID:    "test-session-123",
		State:        "waiting_for_time",
		Intent:       "create_reminder",
		Channel:      "telegram",
		Locale:       "zh-CN",
		MessagesJSON: `[]`,
		TTLSeconds:   86400,
		LastActivity: time.Now(),
		ExpiresAt:    &expiresAt,
	}
	err := repo.CreateOrUpdate(ctx, contextModel)
	require.NoError(t, err)

	t.Run("删除存在的上下文", func(t *testing.T) {
		err := repo.DeleteByUserID(ctx, user.ID)
		assert.NoError(t, err)

		// 验证已删除
		retrievedContext, err := repo.GetByUserID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Nil(t, retrievedContext)
	})

	t.Run("删除不存在的上下文", func(t *testing.T) {
		// GORM的Delete方法不返回错误当记录不存在时
		err := repo.DeleteByUserID(ctx, 9999)
		assert.NoError(t, err)
	})
}

// TestConversationContextRepository_CleanupExpired 测试清理过期上下文
func TestConversationContextRepository_CleanupExpired(t *testing.T) {
	db := setupTestDBForConversationContext(t)
	repo := NewConversationContextRepository(db)
	ctx := context.Background()

	user := createTestUserForConversationContext(t, db)

	// 创建已过期的上下文
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredContext := &models.ConversationContext{
		UserID:       user.ID,
		SessionID:    "test-session-expired",
		State:        "waiting_for_time",
		Intent:       "create_reminder",
		Channel:      "telegram",
		Locale:       "zh-CN",
		MessagesJSON: `[]`,
		TTLSeconds:   86400,
		LastActivity: time.Now(),
		ExpiresAt:    &expiredTime,
	}
	err := repo.CreateOrUpdate(ctx, expiredContext)
	require.NoError(t, err)

	// 创建未过期的上下文
	validTime := time.Now().Add(24 * time.Hour)
	validContext := &models.ConversationContext{
		UserID:       uint(2000),
		SessionID:    "test-session-valid",
		State:        "waiting_for_location",
		Intent:       "get_weather",
		Channel:      "telegram",
		Locale:       "zh-CN",
		MessagesJSON: `[]`,
		TTLSeconds:   86400,
		LastActivity: time.Now(),
		ExpiresAt:    &validTime,
	}
	err = repo.CreateOrUpdate(ctx, validContext)
	require.NoError(t, err)

	t.Run("清理过期上下文", func(t *testing.T) {
		err := repo.CleanupExpired(ctx, time.Now())
		assert.NoError(t, err)

		// 验证过期上下文已删除
		expiredRetrieved, err := repo.GetByUserID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Nil(t, expiredRetrieved)

		// 验证未过期上下文仍然存在
		validRetrieved, err := repo.GetByUserID(ctx, 2000)
		assert.NoError(t, err)
		assert.NotNil(t, validRetrieved)
	})

	t.Run("清理无过期时间的上下文", func(t *testing.T) {
		// 创建没有过期时间的上下文
		noExpiryContext := &models.ConversationContext{
			UserID:       uint(3000),
			SessionID:    "test-session-no-expiry",
			State:        "waiting_for_time",
			Intent:       "create_reminder",
			Channel:      "telegram",
			Locale:       "zh-CN",
			MessagesJSON: `[]`,
			TTLSeconds:   86400,
			LastActivity: time.Now(),
			ExpiresAt:    nil, // nil表示无过期时间
		}
		err := repo.CreateOrUpdate(ctx, noExpiryContext)
		require.NoError(t, err)

		// 清理过期上下文
		err = repo.CleanupExpired(ctx, time.Now())
		assert.NoError(t, err)

		// 验证无过期时间的上下文仍然存在
		noExpiryRetrieved, err := repo.GetByUserID(ctx, 3000)
		assert.NoError(t, err)
		assert.NotNil(t, noExpiryRetrieved)
	})

	t.Run("清理指定时间之前的上下文", func(t *testing.T) {
		// 创建多个不同过期时间的上下文
		oldTime := time.Now().Add(-2 * time.Hour)
		oldContext := &models.ConversationContext{
			UserID:       uint(4000),
			SessionID:    "test-session-old",
			State:        "waiting_for_time",
			Intent:       "create_reminder",
			Channel:      "telegram",
			Locale:       "zh-CN",
			MessagesJSON: `[]`,
			TTLSeconds:   86400,
			LastActivity: time.Now(),
			ExpiresAt:    &oldTime,
		}
		err := repo.CreateOrUpdate(ctx, oldContext)
		require.NoError(t, err)

		recentTime := time.Now().Add(-30 * time.Minute)
		recentContext := &models.ConversationContext{
			UserID:       uint(5000),
			SessionID:    "test-session-recent",
			State:        "waiting_for_time",
			Intent:       "create_reminder",
			Channel:      "telegram",
			Locale:       "zh-CN",
			MessagesJSON: `[]`,
			TTLSeconds:   86400,
			LastActivity: time.Now(),
			ExpiresAt:    &recentTime,
		}
		err = repo.CreateOrUpdate(ctx, recentContext)
		require.NoError(t, err)

		// 清理1小时之前的上下文
		cutoffTime := time.Now().Add(-1 * time.Hour)
		err = repo.CleanupExpired(ctx, cutoffTime)
		assert.NoError(t, err)

		// 验证旧的上下文已删除
		oldRetrieved, err := repo.GetByUserID(ctx, 4000)
		assert.NoError(t, err)
		assert.Nil(t, oldRetrieved)

		// 验证最近的上下文仍然存在（因为未超过1小时）
		recentRetrieved, err := repo.GetByUserID(ctx, 5000)
		assert.NoError(t, err)
		assert.NotNil(t, recentRetrieved)
	})
}