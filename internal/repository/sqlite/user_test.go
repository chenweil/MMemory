package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mmemory/internal/models"
)

// setupTestDBForUser 创建测试数据库
func setupTestDBForUser(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.User{})
	require.NoError(t, err)

	return db
}

// TestUserRepository_Create 测试创建用户
func TestUserRepository_Create(t *testing.T) {
	db := setupTestDBForUser(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("创建成功", func(t *testing.T) {
		user := &models.User{
			TelegramID:   123456789,
			Username:     "testuser",
			FirstName:    "Test",
			LastName:     "User",
			LanguageCode: "zh-CN",
		}

		err := repo.Create(ctx, user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
	})

	t.Run("创建多个用户", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			user := &models.User{
				TelegramID:   int64(1000000000 + i),
				Username:     "testuser",
				FirstName:    "Test",
				LastName:     "User",
				LanguageCode: "zh-CN",
			}
			err := repo.Create(ctx, user)
			assert.NoError(t, err)
		}
	})
}

// TestUserRepository_GetByTelegramID 测试根据Telegram ID获取用户
func TestUserRepository_GetByTelegramID(t *testing.T) {
	db := setupTestDBForUser(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	t.Run("获取存在的用户", func(t *testing.T) {
		retrievedUser, err := repo.GetByTelegramID(ctx, 123456789)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Username, retrievedUser.Username)
	})

	t.Run("获取不存在的用户", func(t *testing.T) {
		retrievedUser, err := repo.GetByTelegramID(ctx, 999999999)
		assert.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
}

// TestUserRepository_GetByID 测试根据ID获取用户
func TestUserRepository_GetByID(t *testing.T) {
	db := setupTestDBForUser(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	t.Run("获取存在的用户", func(t *testing.T) {
		retrievedUser, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Username, retrievedUser.Username)
	})

	t.Run("获取不存在的用户", func(t *testing.T) {
		retrievedUser, err := repo.GetByID(ctx, 9999)
		assert.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
}

// TestUserRepository_Update 测试更新用户
func TestUserRepository_Update(t *testing.T) {
	db := setupTestDBForUser(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	t.Run("更新用户信息", func(t *testing.T) {
		user.Username = "updateduser"
		user.FirstName = "Updated"
		user.LastName = "Name"

		err := repo.Update(ctx, user)
		assert.NoError(t, err)

		// 验证更新
		retrievedUser, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, "updateduser", retrievedUser.Username)
		assert.Equal(t, "Updated", retrievedUser.FirstName)
		assert.Equal(t, "Name", retrievedUser.LastName)
	})

	t.Run("更新语言代码", func(t *testing.T) {
		user.LanguageCode = "en-US"
		err := repo.Update(ctx, user)
		assert.NoError(t, err)

		retrievedUser, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, "en-US", retrievedUser.LanguageCode)
	})
}

// TestUserRepository_Delete 测试删除用户
func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDBForUser(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &models.User{
		TelegramID:   123456789,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "zh-CN",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	t.Run("删除存在的用户", func(t *testing.T) {
		err := repo.Delete(ctx, user.ID)
		assert.NoError(t, err)

		// 验证已删除
		retrievedUser, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})

	t.Run("删除不存在的用户", func(t *testing.T) {
		// GORM的Delete方法不返回错误当记录不存在时
		err := repo.Delete(ctx, 9999)
		assert.NoError(t, err)
	})
}

// TestUserRepository_Count 测试统计用户数量
func TestUserRepository_Count(t *testing.T) {
	db := setupTestDBForUser(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("空数据库计数", func(t *testing.T) {
		count, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("创建用户后计数", func(t *testing.T) {
		// 创建3个用户
		for i := 0; i < 3; i++ {
			user := &models.User{
				TelegramID:   int64(1000000000 + i),
				Username:     "testuser",
				FirstName:    "Test",
				LastName:     "User",
				LanguageCode: "zh-CN",
			}
			err := repo.Create(ctx, user)
			require.NoError(t, err)
		}

		count, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("删除用户后计数", func(t *testing.T) {
		// 获取所有用户
		var users []models.User
		err := db.Find(&users).Error
		require.NoError(t, err)

		if len(users) > 0 {
			// 删除一个用户
			err := repo.Delete(ctx, users[0].ID)
			require.NoError(t, err)

			count, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, int64(len(users)-1), count)
		}
	})
}