package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type conversationService struct {
	conversationRepo interfaces.ConversationRepository
}

// NewConversationService 创建对话服务
func NewConversationService(conversationRepo interfaces.ConversationRepository) ConversationService {
	return &conversationService{
		conversationRepo: conversationRepo,
	}
}

// CreateConversation 创建对话上下文
func (s *conversationService) CreateConversation(ctx context.Context, userID uint, contextType models.ContextType, contextData interface{}, ttl time.Duration) (*models.Conversation, error) {
	// 序列化上下文数据
	dataJSON, err := json.Marshal(contextData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context data: %w", err)
	}

	// 创建对话上下文
	conversation := &models.Conversation{
		UserID:      userID,
		ContextType: contextType,
		ContextData: string(dataJSON),
	}

	// 设置过期时间
	if ttl > 0 {
		conversation.SetExpiry(ttl)
	}

	// 保存到数据库
	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conversation, nil
}

// GetConversation 获取用户对话上下文
func (s *conversationService) GetConversation(ctx context.Context, userID uint, contextType models.ContextType) (*models.Conversation, error) {
	conversation, err := s.conversationRepo.GetByUserID(ctx, userID, contextType)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// 检查是否过期
	if conversation != nil && conversation.IsExpired() {
		// 删除过期对话
		_ = s.conversationRepo.Delete(ctx, conversation.ID)
		return nil, nil
	}

	return conversation, nil
}

// UpdateConversation 更新对话上下文
func (s *conversationService) UpdateConversation(ctx context.Context, conversation *models.Conversation, contextData interface{}) error {
	// 序列化新的上下文数据
	dataJSON, err := json.Marshal(contextData)
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	// 更新数据
	conversation.ContextData = string(dataJSON)
	
	// 更新过期时间（如果设置了TTL）
	if conversation.ExpiresAt != nil {
		conversation.SetExpiry(30 * time.Minute) // 默认延长30分钟
	}

	// 保存到数据库
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return nil
}

// ClearConversation 清除对话上下文
func (s *conversationService) ClearConversation(ctx context.Context, userID uint, contextType models.ContextType) error {
	conversation, err := s.conversationRepo.GetByUserID(ctx, userID, contextType)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conversation != nil {
		if err := s.conversationRepo.Delete(ctx, conversation.ID); err != nil {
			return fmt.Errorf("failed to delete conversation: %w", err)
		}
	}

	return nil
}

// IsConversationActive 检查对话是否活跃
func (s *conversationService) IsConversationActive(ctx context.Context, userID uint, contextType models.ContextType) (bool, error) {
	conversation, err := s.GetConversation(ctx, userID, contextType)
	if err != nil {
		return false, err
	}

	return conversation != nil, nil
}

// CleanupExpiredConversations 清理过期对话
func (s *conversationService) CleanupExpiredConversations(ctx context.Context) error {
	if err := s.conversationRepo.DeleteExpired(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired conversations: %w", err)
	}
	return nil
}

// GetContextData 获取上下文数据
func (s *conversationService) GetContextData(ctx context.Context, userID uint, contextType models.ContextType, target interface{}) error {
	conversation, err := s.GetConversation(ctx, userID, contextType)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conversation == nil {
		return fmt.Errorf("no active conversation found")
	}

	// 反序列化上下文数据
	if err := json.Unmarshal([]byte(conversation.ContextData), target); err != nil {
		return fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	return nil
}

// GetConversationHistory 获取用户对话历史（最近N天）
func (s *conversationService) GetConversationHistory(ctx context.Context, userID uint, days int) (string, error) {
	// 默认获取30天的历史
	if days <= 0 {
		days = 30
	}

	// TODO: 这里需要从数据库中获取历史消息
	// 目前 ConversationContext 模型存储了消息历史，但需要通过 ContextManagerService 来获取
	// 为了简化实现，我们返回一个空字符串，后续可以通过集成 ContextManagerService 来完善

	// 查询用户最近的对话上下文
	// 这里需要访问 ConversationContextRepository 来获取历史记录
	// 暂时返回空字符串，表示没有历史记录
	return "", nil
}