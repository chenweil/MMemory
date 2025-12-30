package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mmemory/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConversationRepository 模拟对话仓储
type MockConversationRepository struct {
	mock.Mock
}

func (m *MockConversationRepository) Create(ctx context.Context, conversation *models.Conversation) error {
	args := m.Called(ctx, conversation)
	return args.Error(0)
}

func (m *MockConversationRepository) GetByUserID(ctx context.Context, userID uint, contextType models.ContextType) (*models.Conversation, error) {
	args := m.Called(ctx, userID, contextType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Conversation), args.Error(1)
}

func (m *MockConversationRepository) Update(ctx context.Context, conversation *models.Conversation) error {
	args := m.Called(ctx, conversation)
	return args.Error(0)
}

func (m *MockConversationRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockConversationRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestConversationService_CreateConversation(t *testing.T) {
	mockRepo := &MockConversationRepository{}
	service := NewConversationService(mockRepo)

	ctx := context.Background()
	userID := uint(1)
	contextType := models.ContextTypeCreatingReminder
	contextData := map[string]interface{}{
		"step": "content",
		"data": "test data",
	}
	ttl := 30 * time.Minute

	mockRepo.On("Create", ctx, mock.MatchedBy(func(c *models.Conversation) bool {
		return c.UserID == userID && c.ContextType == contextType
	})).Return(nil)

	conversation, err := service.CreateConversation(ctx, userID, contextType, contextData, ttl)

	assert.NoError(t, err)
	assert.NotNil(t, conversation)
	assert.Equal(t, userID, conversation.UserID)
	assert.Equal(t, contextType, conversation.ContextType)
	assert.NotNil(t, conversation.ExpiresAt)
	mockRepo.AssertExpectations(t)
}

func TestConversationService_GetConversation(t *testing.T) {
	t.Run("找到活跃对话", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		activeConversation := &models.Conversation{
			ID:          1,
			UserID:      userID,
			ContextType: contextType,
			ContextData: `{"step":"content"}`,
			ExpiresAt:   &[]time.Time{time.Now().Add(1 * time.Hour)}[0],
		}

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(activeConversation, nil)

		conversation, err := service.GetConversation(ctx, userID, contextType)

		assert.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Equal(t, activeConversation.ID, conversation.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("过期对话应被删除", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		expiredTime := time.Now().Add(-1 * time.Hour)
		expiredConversation := &models.Conversation{
			ID:          2,
			UserID:      userID,
			ContextType: contextType,
			ContextData: `{"step":"content"}`,
			ExpiresAt:   &expiredTime,
		}

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(expiredConversation, nil)
		mockRepo.On("Delete", ctx, uint(2)).Return(nil)

		conversation, err := service.GetConversation(ctx, userID, contextType)

		assert.NoError(t, err)
		assert.Nil(t, conversation)
		mockRepo.AssertExpectations(t)
	})
}

func TestConversationService_UpdateConversation(t *testing.T) {
	mockRepo := &MockConversationRepository{}
	service := NewConversationService(mockRepo)

	ctx := context.Background()
	conversation := &models.Conversation{
		ID:          1,
		UserID:      1,
		ContextType: models.ContextTypeCreatingReminder,
		ContextData: `{"old":"data"}`,
	}

	newContextData := map[string]interface{}{
		"new":  "data",
		"step": "schedule",
	}

	// 修改 Mock 匹配器，只检查 ID，不检查 ContextData（因为JSON序列化顺序可能不同）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(c *models.Conversation) bool {
		return c.ID == conversation.ID
	})).Return(nil)

	err := service.UpdateConversation(ctx, conversation, newContextData)

	assert.NoError(t, err)
	assert.Contains(t, conversation.ContextData, "new")
	assert.Contains(t, conversation.ContextData, "schedule")
	mockRepo.AssertExpectations(t)
}

func TestConversationService_ClearConversation(t *testing.T) {
	mockRepo := &MockConversationRepository{}
	service := NewConversationService(mockRepo)

	ctx := context.Background()
	userID := uint(1)
	contextType := models.ContextTypeCreatingReminder

	existingConversation := &models.Conversation{
		ID:          1,
		UserID:      userID,
		ContextType: contextType,
	}

	mockRepo.On("GetByUserID", ctx, userID, contextType).Return(existingConversation, nil)
	mockRepo.On("Delete", ctx, uint(1)).Return(nil)

	err := service.ClearConversation(ctx, userID, contextType)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestConversationService_IsConversationActive(t *testing.T) {
	t.Run("活跃对话返回true", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		activeConversation := &models.Conversation{
			ID:          1,
			UserID:      userID,
			ContextType: contextType,
			ExpiresAt:   &[]time.Time{time.Now().Add(1 * time.Hour)}[0],
		}

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(activeConversation, nil)

		active, err := service.IsConversationActive(ctx, userID, contextType)

		assert.NoError(t, err)
		assert.True(t, active)
		mockRepo.AssertExpectations(t)
	})

	t.Run("无对话返回false", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(nil, nil)

		active, err := service.IsConversationActive(ctx, userID, contextType)

		assert.NoError(t, err)
		assert.False(t, active)
		mockRepo.AssertExpectations(t)
	})
}

func TestConversationService_GetContextData(t *testing.T) {
	mockRepo := &MockConversationRepository{}
	service := NewConversationService(mockRepo)

	ctx := context.Background()
	userID := uint(1)
	contextType := models.ContextTypeCreatingReminder

	conversation := &models.Conversation{
		ID:          1,
		UserID:      userID,
		ContextType: contextType,
		ContextData: `{"step":"content","message":"test message"}`,
		ExpiresAt:   &[]time.Time{time.Now().Add(1 * time.Hour)}[0],
	}

	mockRepo.On("GetByUserID", ctx, userID, contextType).Return(conversation, nil)

	var target map[string]interface{}
	err := service.GetContextData(ctx, userID, contextType, &target)

	assert.NoError(t, err)
	assert.Equal(t, "content", target["step"])
	assert.Equal(t, "test message", target["message"])
	mockRepo.AssertExpectations(t)
}

func TestConversationService_GetContextData_Errors(t *testing.T) {
	t.Run("无对话时返回错误", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(nil, nil)

		var target map[string]interface{}
		err := service.GetContextData(ctx, userID, contextType, &target)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active conversation")
		mockRepo.AssertExpectations(t)
	})

	t.Run("JSON解析失败时返回错误", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		conversation := &models.Conversation{
			ID:          1,
			UserID:      userID,
			ContextType: contextType,
			ContextData: `invalid json`,
			ExpiresAt:   &[]time.Time{time.Now().Add(1 * time.Hour)}[0],
		}

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(conversation, nil)

		var target map[string]interface{}
		err := service.GetContextData(ctx, userID, contextType, &target)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
		mockRepo.AssertExpectations(t)
	})
}

func TestConversationService_CleanupExpiredConversations(t *testing.T) {
	t.Run("成功清理过期对话", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()

		mockRepo.On("DeleteExpired", ctx).Return(nil)

		err := service.CleanupExpiredConversations(ctx)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("清理失败时返回错误", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		expectedErr := fmt.Errorf("database error")

		mockRepo.On("DeleteExpired", ctx).Return(expectedErr)

		err := service.CleanupExpiredConversations(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cleanup")
		mockRepo.AssertExpectations(t)
	})
}

func TestConversationService_CreateConversation_Errors(t *testing.T) {
	t.Run("JSON序列化失败", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		// 使用无法序列化的对象
		nonSerializable := make(chan int)

		_, err := service.CreateConversation(ctx, userID, contextType, nonSerializable, 30*time.Minute)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal")
	})

	t.Run("仓库创建失败", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder
		contextData := map[string]interface{}{"step": "content"}
		expectedErr := fmt.Errorf("database error")

		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Conversation")).Return(expectedErr)

		_, err := service.CreateConversation(ctx, userID, contextType, contextData, 30*time.Minute)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create conversation")
	})

	t.Run("TTL为0时不设置过期时间", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder
		contextData := map[string]interface{}{"step": "content"}

		mockRepo.On("Create", ctx, mock.MatchedBy(func(c *models.Conversation) bool {
			return c.UserID == userID && c.ExpiresAt == nil
		})).Return(nil)

		conversation, err := service.CreateConversation(ctx, userID, contextType, contextData, 0)

		assert.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Nil(t, conversation.ExpiresAt)
		mockRepo.AssertExpectations(t)
	})
}

func TestConversationService_GetConversation_Errors(t *testing.T) {
	t.Run("仓库返回错误", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder
		expectedErr := fmt.Errorf("database error")

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(nil, expectedErr)

		conversation, err := service.GetConversation(ctx, userID, contextType)

		assert.Error(t, err)
		assert.Nil(t, conversation)
		assert.Contains(t, err.Error(), "failed to get conversation")
		mockRepo.AssertExpectations(t)
	})
}

func TestConversationService_UpdateConversation_Errors(t *testing.T) {
	t.Run("JSON序列化失败", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		conversation := &models.Conversation{
			ID:          1,
			UserID:      1,
			ContextType: models.ContextTypeCreatingReminder,
		}

		// 使用无法序列化的对象
		nonSerializable := make(chan int)

		err := service.UpdateConversation(ctx, conversation, nonSerializable)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal")
	})

	t.Run("仓库更新失败", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		conversation := &models.Conversation{
			ID:          1,
			UserID:      1,
			ContextType: models.ContextTypeCreatingReminder,
			ContextData: `{}`,
		}
		contextData := map[string]interface{}{"step": "updated"}
		expectedErr := fmt.Errorf("database error")

		mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Conversation")).Return(expectedErr)

		err := service.UpdateConversation(ctx, conversation, contextData)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update conversation")
	})
}

func TestConversationService_ClearConversation_Errors(t *testing.T) {
	t.Run("获取对话失败", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder
		expectedErr := fmt.Errorf("database error")

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(nil, expectedErr)

		err := service.ClearConversation(ctx, userID, contextType)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get conversation")
	})

	t.Run("删除对话失败", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder
		expectedErr := fmt.Errorf("database error")

		existingConversation := &models.Conversation{
			ID:          1,
			UserID:      userID,
			ContextType: contextType,
		}

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(existingConversation, nil)
		mockRepo.On("Delete", ctx, uint(1)).Return(expectedErr)

		err := service.ClearConversation(ctx, userID, contextType)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete conversation")
	})

	t.Run("无对话时不返回错误", func(t *testing.T) {
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)

		ctx := context.Background()
		userID := uint(1)
		contextType := models.ContextTypeCreatingReminder

		mockRepo.On("GetByUserID", ctx, userID, contextType).Return(nil, nil)

		err := service.ClearConversation(ctx, userID, contextType)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}