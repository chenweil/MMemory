package service

import (
	"context"
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