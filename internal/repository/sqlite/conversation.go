package sqlite

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) interfaces.ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) Create(ctx context.Context, conversation *models.Conversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

func (r *conversationRepository) GetByUserID(ctx context.Context, userID uint, contextType models.ContextType) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND context_type = ?", userID, contextType).
		First(&conversation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conversation, nil
}

func (r *conversationRepository) Update(ctx context.Context, conversation *models.Conversation) error {
	return r.db.WithContext(ctx).Save(conversation).Error
}

func (r *conversationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Conversation{}, id).Error
}

func (r *conversationRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Delete(&models.Conversation{}).Error
}