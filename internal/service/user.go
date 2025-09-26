package service

import (
	"context"
	"fmt"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

type userService struct {
	userRepo interfaces.UserRepository
}

func NewUserService(userRepo interfaces.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) CreateUser(ctx context.Context, user *models.User) error {
	if user.TelegramID == 0 {
		return fmt.Errorf("Telegram ID不能为空")
	}

	// 检查用户是否已存在
	existingUser, err := s.userRepo.GetByTelegramID(ctx, user.TelegramID)
	if err != nil {
		return fmt.Errorf("检查用户是否存在失败: %w", err)
	}
	if existingUser != nil {
		return fmt.Errorf("用户已存在")
	}

	return s.userRepo.Create(ctx, user)
}

func (s *userService) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	return s.userRepo.GetByTelegramID(ctx, telegramID)
}

func (s *userService) GetByID(ctx context.Context, id uint) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *userService) UpdateUser(ctx context.Context, user *models.User) error {
	if user.ID == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	return s.userRepo.Update(ctx, user)
}