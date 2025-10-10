package service

import (
	"context"
	"testing"

	"mmemory/internal/models"
)

// Mock UserRepository for testing
type mockUserRepository struct {
	users map[int64]*models.User
	idCounter uint
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[int64]*models.User),
		idCounter: 1,
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = m.idCounter
	m.idCounter++
	m.users[user.TelegramID] = user
	return nil
}

func (m *mockUserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	user := m.users[telegramID]
	return user, nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	if existingUser := m.users[user.TelegramID]; existingUser != nil {
		m.users[user.TelegramID] = user
		return nil
	}
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uint) error {
	for telegramID, user := range m.users {
		if user.ID == id {
			delete(m.users, telegramID)
			return nil
		}
	}
	return nil
}

func (m *mockUserRepository) Count(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func TestUserService_CreateUser(t *testing.T) {
	mockRepo := newMockUserRepository()
	userService := NewUserService(mockRepo)
	ctx := context.Background()

	tests := []struct {
		name    string
		user    *models.User
		wantErr bool
	}{
		{
			name: "成功创建用户",
			user: &models.User{
				TelegramID: 12345,
				Username:   "testuser",
				FirstName:  "Test",
				LastName:   "User",
			},
			wantErr: false,
		},
		{
			name: "TelegramID为空时失败",
			user: &models.User{
				Username:  "testuser",
				FirstName: "Test",
			},
			wantErr: true,
		},
		{
			name: "重复用户创建失败",
			user: &models.User{
				TelegramID: 12345, // 与第一个测试相同
				Username:   "duplicateuser",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := userService.CreateUser(ctx, tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserService_GetByTelegramID(t *testing.T) {
	mockRepo := newMockUserRepository()
	userService := NewUserService(mockRepo)
	ctx := context.Background()

	// 先创建一个用户
	testUser := &models.User{
		TelegramID: 67890,
		Username:   "testuser",
		FirstName:  "Test",
	}
	err := userService.CreateUser(ctx, testUser)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	tests := []struct {
		name       string
		telegramID int64
		wantUser   bool
		wantErr    bool
	}{
		{
			name:       "获取存在的用户",
			telegramID: 67890,
			wantUser:   true,
			wantErr:    false,
		},
		{
			name:       "获取不存在的用户",
			telegramID: 99999,
			wantUser:   false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := userService.GetByTelegramID(ctx, tt.telegramID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByTelegramID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (user != nil) != tt.wantUser {
				t.Errorf("GetByTelegramID() user = %v, wantUser %v", user != nil, tt.wantUser)
			}
			if user != nil && user.TelegramID != tt.telegramID {
				t.Errorf("GetByTelegramID() telegramID = %v, want %v", user.TelegramID, tt.telegramID)
			}
		})
	}
}