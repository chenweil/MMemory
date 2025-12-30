package sqlite

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mmemory/pkg/config"
)

// TestNewDatabase 测试创建数据库
func TestNewDatabase(t *testing.T) {
	t.Run("创建内存数据库", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			DSN:           ":memory:",
			MaxOpenConns:  10,
			MaxIdleConns:  5,
		}

		db, err := NewDatabase(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		assert.NotNil(t, db.GetDB())

		// 清理
		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("创建临时文件数据库", func(t *testing.T) {
		tmpFile := "/tmp/test_mmemory.db"
		defer os.Remove(tmpFile)

		cfg := &config.DatabaseConfig{
			DSN:           tmpFile,
			MaxOpenConns:  10,
			MaxIdleConns:  5,
		}

		db, err := NewDatabase(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		assert.NotNil(t, db.GetDB())

		// 清理
		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("无效的DSN", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			DSN:           "invalid://dsn",
			MaxOpenConns:  10,
			MaxIdleConns:  5,
		}

		db, err := NewDatabase(cfg)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("自定义连接池配置", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			DSN:           ":memory:",
			MaxOpenConns:  20,
			MaxIdleConns:  10,
		}

		db, err := NewDatabase(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// 验证连接池配置
		sqlDB, err := db.GetDB().DB()
		assert.NoError(t, err)
		assert.Equal(t, 20, sqlDB.Stats().MaxOpenConnections)

		// 清理
		err = db.Close()
		assert.NoError(t, err)
	})
}

// TestDatabase_AutoMigrate 测试自动迁移
func TestDatabase_AutoMigrate(t *testing.T) {
	cfg := &config.DatabaseConfig{
		DSN:           ":memory:",
		MaxOpenConns:  10,
		MaxIdleConns:  5,
	}

	db, err := NewDatabase(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	t.Run("验证表已创建", func(t *testing.T) {
		// AutoMigrate在NewDatabase中已经调用
		// 这里我们验证表确实存在
		gormDB := db.GetDB()

		// 检查users表
		err := gormDB.Exec("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Error
		assert.NoError(t, err)

		// 检查reminders表
		err = gormDB.Exec("SELECT name FROM sqlite_master WHERE type='table' AND name='reminders'").Error
		assert.NoError(t, err)

		// 检查reminder_logs表
		err = gormDB.Exec("SELECT name FROM sqlite_master WHERE type='table' AND name='reminder_logs'").Error
		assert.NoError(t, err)

		// 检查conversations表
		err = gormDB.Exec("SELECT name FROM sqlite_master WHERE type='table' AND name='conversations'").Error
		assert.NoError(t, err)

		// 检查conversation_contexts表
		err = gormDB.Exec("SELECT name FROM sqlite_master WHERE type='table' AND name='conversation_contexts'").Error
		assert.NoError(t, err)
	})

	// 清理
	err = db.Close()
	assert.NoError(t, err)
}

// TestDatabase_Close 测试关闭数据库
func TestDatabase_Close(t *testing.T) {
	cfg := &config.DatabaseConfig{
		DSN:           ":memory:",
		MaxOpenConns:  10,
		MaxIdleConns:  5,
	}

	db, err := NewDatabase(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	t.Run("正常关闭数据库", func(t *testing.T) {
		err := db.Close()
		assert.NoError(t, err)
	})

	t.Run("重复关闭数据库", func(t *testing.T) {
		// 第一次关闭
		err := db.Close()
		assert.NoError(t, err)

		// 第二次关闭（应该不会出错）
		err = db.Close()
		assert.NoError(t, err)
	})
}

// TestDatabase_GetDB 测试获取数据库连接
func TestDatabase_GetDB(t *testing.T) {
	cfg := &config.DatabaseConfig{
		DSN:           ":memory:",
		MaxOpenConns:  10,
		MaxIdleConns:  5,
	}

	db, err := NewDatabase(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	t.Run("获取数据库连接", func(t *testing.T) {
		gormDB := db.GetDB()
		assert.NotNil(t, gormDB)

		// 验证可以执行查询
		err := gormDB.Exec("SELECT 1").Error
		assert.NoError(t, err)
	})

	t.Run("多次获取连接", func(t *testing.T) {
		gormDB1 := db.GetDB()
		gormDB2 := db.GetDB()

		assert.NotNil(t, gormDB1)
		assert.NotNil(t, gormDB2)

		// 应该返回同一个连接
		assert.Equal(t, gormDB1, gormDB2)
	})

	// 清理
	err = db.Close()
	assert.NoError(t, err)
}

// TestDatabase_BeginTx 测试开始事务
func TestDatabase_BeginTx(t *testing.T) {
	cfg := &config.DatabaseConfig{
		DSN:           ":memory:",
		MaxOpenConns:  10,
		MaxIdleConns:  5,
	}

	db, err := NewDatabase(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	t.Run("开始事务", func(t *testing.T) {
		ctx := context.Background()
		tx, err := db.BeginTx(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, tx)

		// 验证事务可以执行查询
		err = tx.Exec("SELECT 1").Error
		assert.NoError(t, err)

		// 提交事务
		err = tx.Commit().Error
		assert.NoError(t, err)
	})

	t.Run("事务回滚", func(t *testing.T) {
		ctx := context.Background()
		tx, err := db.BeginTx(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, tx)

		// 执行一些操作
		err = tx.Exec("SELECT 1").Error
		assert.NoError(t, err)

		// 回滚事务
		err = tx.Rollback().Error
		assert.NoError(t, err)
	})

	t.Run("嵌套事务", func(t *testing.T) {
		ctx := context.Background()
		tx1, err := db.BeginTx(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, tx1)

		// 在事务内开始另一个事务
		tx2, err := db.BeginTx(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, tx2)

		// 提交两个事务
		err = tx2.Commit().Error
		assert.NoError(t, err)

		err = tx1.Commit().Error
		assert.NoError(t, err)
	})

	// 清理
	err = db.Close()
	assert.NoError(t, err)
}

// TestDatabase_Integration 测试数据库集成
func TestDatabase_Integration(t *testing.T) {
	cfg := &config.DatabaseConfig{
		DSN:           ":memory:",
		MaxOpenConns:  10,
		MaxIdleConns:  5,
	}

	db, err := NewDatabase(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	t.Run("完整的数据库操作流程", func(t *testing.T) {
		ctx := context.Background()

		// 开始事务
		tx, err := db.BeginTx(ctx)
		assert.NoError(t, err)

		// 创建用户
		err = tx.Exec(`
			INSERT INTO users (telegram_id, username, first_name, last_name, language_code, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, 123456789, "testuser", "Test", "User", "zh-CN").Error
		assert.NoError(t, err)

		// 查询用户
		var count int64
		err = tx.Raw("SELECT COUNT(*) FROM users WHERE telegram_id = ?", 123456789).Scan(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// 提交事务
		err = tx.Commit().Error
		assert.NoError(t, err)

		// 验证数据已提交
		gormDB := db.GetDB()
		err = gormDB.Raw("SELECT COUNT(*) FROM users WHERE telegram_id = ?", 123456789).Scan(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	// 清理
	err = db.Close()
	assert.NoError(t, err)
}