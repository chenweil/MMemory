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

// setupTestDBForConversation 创建测试数据库
func setupTestDBForConversation(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.User{}, &models.Conversation{})
	require.NoError(t, err)

	return db
}

// createTestUserForConversation 创建测试用户
func createTestUserForConversation(t *testing.T, db *gorm.DB) *models.User {
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

// TestConversationRepository_Create 测试创建对话
func TestConversationRepository_Create(t *testing.T) {
	db := setupTestDBForConversation(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := createTestUserForConversation(t, db)

	t.Run("创建成功", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		conversation := &models.Conversation{
			UserID:      user.ID,
			ContextType: models.ContextTypeCreatingReminder,
			ContextData: `{"step":"waiting_for_time"}`,
			ExpiresAt:   &expiresAt,
		}

		err := repo.Create(ctx, conversation)
		assert.NoError(t, err)
		assert.NotZero(t, conversation.ID)
	})

	t.Run("创建多个对话", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			expiresAt := time.Now().Add(24 * time.Hour)
			conversation := &models.Conversation{
				UserID:      user.ID,
				ContextType: models.ContextTypeCreatingReminder,
				ContextData: `{"step":"waiting_for_time"}`,
				ExpiresAt:   &expiresAt,
			}
			err := repo.Create(ctx, conversation)
			assert.NoError(t, err)
		}
	})
}

// TestConversationRepository_GetByUserID 测试根据用户ID获取对话
func TestConversationRepository_GetByUserID(t *testing.T) {
	db := setupTestDBForConversation(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := createTestUserForConversation(t, db)

	// 创建测试对话
	expiresAt := time.Now().Add(24 * time.Hour)
	conversation := &models.Conversation{
		UserID:      user.ID,
		ContextType: models.ContextTypeCreatingReminder,
		ContextData: `{"step":"waiting_for_time"}`,
		ExpiresAt:   &expiresAt,
	}
	err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	t.Run("获取存在的对话", func(t *testing.T) {
		retrievedConversation, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeCreatingReminder)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedConversation)
		assert.Equal(t, conversation.ID, retrievedConversation.ID)
		assert.Equal(t, conversation.ContextData, retrievedConversation.ContextData)
	})

	t.Run("获取不存在的对话", func(t *testing.T) {
		retrievedConversation, err := repo.GetByUserID(ctx, 9999, models.ContextTypeCreatingReminder)
		assert.NoError(t, err)
		assert.Nil(t, retrievedConversation)
	})

	t.Run("获取不同类型的对话", func(t *testing.T) {
		// 创建不同类型的对话
		chatExpiresAt := time.Now().Add(24 * time.Hour)
		chatConversation := &models.Conversation{
			UserID:      user.ID,
			ContextType: models.ContextTypeChat,
			ContextData: `{"message":"hello"}`,
			ExpiresAt:   &chatExpiresAt,
		}
		err := repo.Create(ctx, chatConversation)
		require.NoError(t, err)

		// 获取聊天类型的对话
		retrievedConversation, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeChat)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedConversation)
		assert.Equal(t, models.ContextTypeChat, retrievedConversation.ContextType)
	})
}

// TestConversationRepository_Update 测试更新对话
func TestConversationRepository_Update(t *testing.T) {
	db := setupTestDBForConversation(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := createTestUserForConversation(t, db)

	expiresAt := time.Now().Add(24 * time.Hour)
	conversation := &models.Conversation{
		UserID:      user.ID,
		ContextType: models.ContextTypeCreatingReminder,
		ContextData: `{"step":"waiting_for_time"}`,
		ExpiresAt:   &expiresAt,
	}
	err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	t.Run("更新对话内容", func(t *testing.T) {
		conversation.ContextData = `{"step":"waiting_for_confirmation"}`
		err := repo.Update(ctx, conversation)
		assert.NoError(t, err)

		// 验证更新
		updatedConversation, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeCreatingReminder)
		assert.NoError(t, err)
		assert.Equal(t, `{"step":"waiting_for_confirmation"}`, updatedConversation.ContextData)
	})

	t.Run("更新过期时间", func(t *testing.T) {
		newExpiresAt := time.Now().Add(48 * time.Hour)
		conversation.ExpiresAt = &newExpiresAt
		err := repo.Update(ctx, conversation)
		assert.NoError(t, err)

		updatedConversation, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeCreatingReminder)
		assert.NoError(t, err)
		assert.WithinDuration(t, newExpiresAt, *updatedConversation.ExpiresAt, time.Second)
	})
}

// TestConversationRepository_Delete 测试删除对话
func TestConversationRepository_Delete(t *testing.T) {
	db := setupTestDBForConversation(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := createTestUserForConversation(t, db)

	expiresAt := time.Now().Add(24 * time.Hour)
	conversation := &models.Conversation{
		UserID:      user.ID,
		ContextType: models.ContextTypeCreatingReminder,
		ContextData: `{"step":"waiting_for_time"}`,
		ExpiresAt:   &expiresAt,
	}
	err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	t.Run("删除存在的对话", func(t *testing.T) {
		err := repo.Delete(ctx, conversation.ID)
		assert.NoError(t, err)

		// 验证已删除
		retrievedConversation, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeCreatingReminder)
		assert.NoError(t, err)
		assert.Nil(t, retrievedConversation)
	})

	t.Run("删除不存在的对话", func(t *testing.T) {
		// GORM的Delete方法不返回错误当记录不存在时
		err := repo.Delete(ctx, 9999)
		assert.NoError(t, err)
	})
}

// TestConversationRepository_DeleteExpired 测试删除过期对话
func TestConversationRepository_DeleteExpired(t *testing.T) {
	db := setupTestDBForConversation(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := createTestUserForConversation(t, db)

	// 创建已过期的对话
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredConversation := &models.Conversation{
		UserID:      user.ID,
		ContextType: models.ContextTypeCreatingReminder,
		ContextData: `{"step":"waiting_for_time"}`,
		ExpiresAt:   &expiredTime,
	}
	err := repo.Create(ctx, expiredConversation)
	require.NoError(t, err)

	// 创建未过期的对话
	validTime := time.Now().Add(24 * time.Hour)
	validConversation := &models.Conversation{
		UserID:      user.ID,
		ContextType: models.ContextTypeChat,
		ContextData: `{"message":"hello"}`,
		ExpiresAt:   &validTime,
	}
	err = repo.Create(ctx, validConversation)
	require.NoError(t, err)

	t.Run("删除过期对话", func(t *testing.T) {
		err := repo.DeleteExpired(ctx)
		assert.NoError(t, err)

		// 验证过期对话已删除
		expiredRetrieved, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeCreatingReminder)
		assert.NoError(t, err)
		assert.Nil(t, expiredRetrieved)

		// 验证未过期对话仍然存在
		validRetrieved, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeChat)
		assert.NoError(t, err)
		assert.NotNil(t, validRetrieved)
	})

	t.Run("删除无过期时间的对话", func(t *testing.T) {
		// 创建没有过期时间的对话
		noExpiryConversation := &models.Conversation{
			UserID:      user.ID,
			ContextType: models.ContextTypeCreatingReminder,
			ContextData: `{"step":"waiting_for_time"}`,
			ExpiresAt:   nil, // nil表示无过期时间
		}
		err := repo.Create(ctx, noExpiryConversation)
		require.NoError(t, err)

		// 删除过期对话
		err = repo.DeleteExpired(ctx)
		assert.NoError(t, err)

		// 验证无过期时间的对话仍然存在
		noExpiryRetrieved, err := repo.GetByUserID(ctx, user.ID, models.ContextTypeCreatingReminder)
		assert.NoError(t, err)
		assert.NotNil(t, noExpiryRetrieved)
	})
}