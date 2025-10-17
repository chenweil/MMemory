package sqlite

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type conversationContextRepository struct {
	db *gorm.DB
}

// NewConversationContextRepository 创建上下文仓储实现
func NewConversationContextRepository(db *gorm.DB) interfaces.ConversationContextRepository {
	return &conversationContextRepository{db: db}
}

func (r *conversationContextRepository) GetByUserID(ctx context.Context, userID uint) (*models.ConversationContext, error) {
	var result models.ConversationContext
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (r *conversationContextRepository) CreateOrUpdate(ctx context.Context, ctxModel *models.ConversationContext) error {
	if ctxModel == nil {
		return nil
	}

	if ctxModel.ID == 0 {
		return r.db.WithContext(ctx).Create(ctxModel).Error
	}

	return r.db.WithContext(ctx).Save(ctxModel).Error
}

func (r *conversationContextRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.ConversationContext{}).Error
}

func (r *conversationContextRepository) CleanupExpired(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Delete(&models.ConversationContext{}).Error
}
