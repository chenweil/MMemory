package sqlite

import (
	"context"
	"time"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type conversationArchiveRepository struct {
	db *gorm.DB
}

// NewConversationArchiveRepository 创建存档仓库
func NewConversationArchiveRepository(db *gorm.DB) interfaces.ConversationArchiveRepository {
	return &conversationArchiveRepository{db: db}
}

// Create 创建存档
func (r *conversationArchiveRepository) Create(ctx context.Context, archive *models.ConversationArchive) error {
	return r.db.WithContext(ctx).Create(archive).Error
}

// GetByUserID 获取用户存档
func (r *conversationArchiveRepository) GetByUserID(ctx context.Context, userID uint, limit int, offset int) ([]*models.ConversationArchive, error) {
	var archives []*models.ConversationArchive
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&archives).Error
	return archives, err
}

// GetByID 根据ID获取
func (r *conversationArchiveRepository) GetByID(ctx context.Context, id uint) (*models.ConversationArchive, error) {
	var archive models.ConversationArchive
	err := r.db.WithContext(ctx).First(&archive, id).Error
	return &archive, err
}

// Delete 删除存档
func (r *conversationArchiveRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ConversationArchive{}, id).Error
}

// DeleteExpired 删除过期存档
func (r *conversationArchiveRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&models.ConversationArchive{})
	return result.RowsAffected, result.Error
}

// CountByUserID 统计数量
func (r *conversationArchiveRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ConversationArchive{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}
