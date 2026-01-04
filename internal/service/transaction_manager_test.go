package service

import (
	"context"
	"errors"
	"testing"

	"mmemory/pkg/logger"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestTransactionManager_ExecuteInTransaction_Success 测试成功执行事务
func TestTransactionManager_ExecuteInTransaction_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建测试表
	err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, value TEXT)").Error
	assert.NoError(t, err)

	tm := NewTransactionManager(db)
	ctx := context.Background()

	executed := false
	err = tm.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		executed = true
		return tx.Exec("INSERT INTO test_table (id, value) VALUES (1, 'test')").Error
	})

	assert.NoError(t, err)
	assert.True(t, executed)

	// 验证数据已插入
	var count int64
	db.Table("test_table").Count(&count)
	assert.Equal(t, int64(1), count)
}

// TestTransactionManager_ExecuteInTransaction_Rollback 测试事务回滚
func TestTransactionManager_ExecuteInTransaction_Rollback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建测试表
	err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, value TEXT UNIQUE)").Error
	assert.NoError(t, err)

	// 插入初始数据
	err = db.Exec("INSERT INTO test_table (id, value) VALUES (1, 'existing')").Error
	assert.NoError(t, err)

	tm := NewTransactionManager(db)
	ctx := context.Background()

	// 尝试插入重复数据，应该失败并回滚
	err = tm.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		return tx.Exec("INSERT INTO test_table (id, value) VALUES (1, 'duplicate')").Error
	})

	assert.Error(t, err)

	// 验证数据没有变化
	var count int64
	db.Table("test_table").Where("value = ?", "duplicate").Count(&count)
	assert.Equal(t, int64(0), count)
}

// TestTransactionManager_ExecuteWithRetry_Success 测试重试成功
func TestTransactionManager_ExecuteWithRetry_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建测试表
	err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, value TEXT)").Error
	assert.NoError(t, err)

	tm := NewTransactionManager(db)
	ctx := context.Background()

	attempts := 0
	err = tm.ExecuteWithRetry(ctx, func(tx *gorm.DB) error {
		attempts++
		if attempts < 2 {
			return errors.New("database is locked")
		}
		return tx.Exec("INSERT INTO test_table (id, value) VALUES (1, 'test')").Error
	}, 3)

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)

	// 验证数据已插入
	var count int64
	db.Table("test_table").Count(&count)
	assert.Equal(t, int64(1), count)
}

// TestTransactionManager_ExecuteWithRetry_MaxRetries 测试达到最大重试次数
func TestTransactionManager_ExecuteWithRetry_MaxRetries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	tm := NewTransactionManager(db)
	ctx := context.Background()

	err = tm.ExecuteWithRetry(ctx, func(tx *gorm.DB) error {
		return errors.New("database is locked")
	}, 3)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已达到最大重试次数")
}

// TestTransactionManager_shouldRetry 测试重试判断逻辑
func TestTransactionManager_shouldRetry(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	tm := NewTransactionManager(db)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"Deadlock error", errors.New("deadlock detected"), true},
		{"Lock error", errors.New("lock timeout"), true},
		{"Network error", errors.New("network connection lost"), true},
		{"Connection error", errors.New("database connection closed"), true},
		{"Other error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.shouldRetry(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConcurrentOperationManager_New 测试创建并发操作管理器
func TestConcurrentOperationManager_New(t *testing.T) {
	manager := NewConcurrentOperationManager(5)

	assert.NotNil(t, manager)
	assert.Equal(t, 5, manager.maxConcurrent)
	assert.NotNil(t, manager.semaphore)
}

// TestConcurrentOperationManager_Execute 测试执行并发操作
func TestConcurrentOperationManager_Execute(t *testing.T) {
	manager := NewConcurrentOperationManager(2)
	ctx := context.Background()

	count := 0
	err := manager.Execute(ctx, func() error {
		count++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

// 初始化日志
func init() {
	logger.Init("info", "text", "stdout", "")
}