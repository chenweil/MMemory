package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceRegistry 测试服务注册中心
func TestServiceRegistry(t *testing.T) {
	// 创建新的注册中心实例，避免影响全局
	registry := NewServiceRegistry()
	ctx := context.Background()

	t.Run("服务注册和获取", func(t *testing.T) {
		// 创建模拟服务
		mockService := &struct {
		*BaseService
		healthy bool
	}{
			BaseService: NewBaseService("MockService", ServiceTypeUser, "测试服务"),
		}

		// 注册服务
		err := registry.Register(mockService)
		require.NoError(t, err)

		// 获取服务
		retrievedService, err := registry.Get(ServiceTypeUser)
		require.NoError(t, err)
		assert.Equal(t, mockService, retrievedService)
	})

	t.Run("重复注册应该失败", func(t *testing.T) {
		mockService2 := &struct {
		*BaseService
		healthy bool
	}{
			BaseService: NewBaseService("MockService2", ServiceTypeUser, "测试服务2"),
		}

		err := registry.Register(mockService2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "已存在")
	})

	t.Run("获取不存在的服务应该失败", func(t *testing.T) {
		_, err := registry.Get(ServiceTypeScheduler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不存在")
	})

	t.Run("服务注销", func(t *testing.T) {
		err := registry.Unregister(ServiceTypeUser)
		require.NoError(t, err)

		_, err = registry.Get(ServiceTypeUser)
		assert.Error(t, err)
	})

	t.Run("服务启动和停止", func(t *testing.T) {
		mockService := &struct {
		*BaseService
		healthy bool
	}{
			BaseService: NewBaseService("MockService", ServiceTypeReminder, "测试提醒服务"),
		}

		err := registry.Register(mockService)
		require.NoError(t, err)

		// 启动所有服务
		err = registry.StartAll(ctx)
		require.NoError(t, err)
		assert.True(t, mockService.IsStarted())

		// 停止所有服务
		err = registry.StopAll(ctx)
		require.NoError(t, err)
		assert.False(t, mockService.IsStarted())
	})

	t.Run("健康检查", func(t *testing.T) {
		mockService := &struct {
		*BaseService
		healthy bool
	}{
			BaseService: NewBaseService("MockService", ServiceTypeNotification, "测试通知服务"),
		}

		err := registry.Register(mockService)
		require.NoError(t, err)

		// 启动服务
		err = registry.StartAll(ctx)
		require.NoError(t, err)

		// 健康检查
		results := registry.HealthCheck(ctx)
		assert.Contains(t, results, ServiceTypeNotification)
		assert.NoError(t, results[ServiceTypeNotification])
	})

	t.Run("事件监听器", func(t *testing.T) {
		eventReceived := false
		var receivedEvent ServiceEvent

		// 添加事件监听器
		registry.AddEventListener(func(event ServiceEvent) {
			eventReceived = true
			receivedEvent = event
		})

		mockService := &struct {
		*BaseService
		healthy bool
	}{
			BaseService: NewBaseService("MockService", ServiceTypeReminderLog, "测试记录服务"),
		}

		// 注册服务应该触发事件
		err := registry.Register(mockService)
		require.NoError(t, err)

		// 等待事件处理
		time.Sleep(100 * time.Millisecond)

		assert.True(t, eventReceived)
		assert.Equal(t, ServiceEventRegistered, receivedEvent.Type)
		assert.Equal(t, mockService, receivedEvent.Service)
	})
}

// TestServiceError 测试服务错误处理
func TestServiceError(t *testing.T) {
	t.Run("创建服务错误", func(t *testing.T) {
		err := NewError("TEST_ERROR", "测试错误消息")
		assert.Equal(t, "TEST_ERROR", err.Code)
		assert.Equal(t, "测试错误消息", err.Message)
		assert.Equal(t, ErrorLevelError, err.Level)
	})

	t.Run("服务错误链", func(t *testing.T) {
		originalErr := fmt.Errorf("原始错误")
		serviceErr := NewError("CHAIN_ERROR", "链式错误").
			WithService("TestService").
			WithOperation("TestOperation").
			WithCause(originalErr).
			WithDetail("key", "value")

		assert.Equal(t, "CHAIN_ERROR", serviceErr.Code)
		assert.Equal(t, "TestService", serviceErr.Service)
		assert.Equal(t, "TestOperation", serviceErr.Operation)
		assert.Equal(t, originalErr, serviceErr.Cause)
		assert.Equal(t, "value", serviceErr.Details["key"])
	})

	t.Run("错误包装", func(t *testing.T) {
		ctx := context.Background()
		originalErr := fmt.Errorf("数据库连接失败")
		
		wrappedErr := WrapError(ctx, originalErr, "TestService", "ConnectDB")
		assert.NotNil(t, wrappedErr)
		assert.Equal(t, "TestService", wrappedErr.Service)
		assert.Equal(t, "ConnectDB", wrappedErr.Operation)
		assert.Equal(t, originalErr, wrappedErr.Cause)
	})
}

// TestDatabaseOptimizer 测试数据库优化器
func TestDatabaseOptimizer(t *testing.T) {
	// 这里使用模拟的数据库连接
	// 实际测试中应该使用真实的测试数据库
	t.Run("性能分析", func(t *testing.T) {
		// 创建优化器（使用模拟组件）
		optimizer := &DatabaseOptimizer{
			queryMetrics: &QueryMetrics{},
		}

		ctx := context.Background()
		report, err := optimizer.AnalyzePerformance(ctx)
		
		// 由于我们使用的是模拟数据，应该能成功生成报告
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.NotNil(t, report.IndexAnalysis)
		assert.NotNil(t, report.QueryAnalysis)
		assert.NotNil(t, report.TableAnalysis)
		assert.NotEmpty(t, report.Recommendations)
	})

	t.Run("索引分析", func(t *testing.T) {
		optimizer := &DatabaseOptimizer{}
		
		analysis, err := optimizer.analyzeIndexes(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.NotEmpty(t, analysis.MissingIndexes)
		
		// 验证推荐的索引
		hasReminderIndex := false
		for _, index := range analysis.MissingIndexes {
			if index.TableName == "reminders" && testSliceContains(index.Columns, "user_id") {
				hasReminderIndex = true
				break
			}
		}
		assert.True(t, hasReminderIndex, "应该推荐reminders表的user_id索引")
	})

	t.Run("SQL生成", func(t *testing.T) {
		optimizer := &DatabaseOptimizer{}
		
		missingIndex := MissingIndex{
			TableName: "test_table",
			Columns:   []string{"col1", "col2"},
		}
		
		sql := optimizer.generateCreateIndexSQL(missingIndex)
		assert.Contains(t, sql, "CREATE INDEX")
		assert.Contains(t, sql, "test_table")
		assert.Contains(t, sql, "col1")
		assert.Contains(t, sql, "col2")
	})
}

// TestTransactionManager 测试事务管理器
func TestTransactionManager(t *testing.T) {
	t.Run("并发操作管理", func(t *testing.T) {
		// 测试并发操作管理器
		concurrency := NewConcurrentOperationManager(2) // 最多2个并发
		
		results := make([]int, 5)
		errors := concurrency.ExecuteBatch(context.Background(), []func() error{
			func() error { results[0] = 1; return nil },
			func() error { results[1] = 2; return nil },
			func() error { results[2] = 3; return nil },
			func() error { results[3] = 4; return nil },
			func() error { results[4] = 5; return nil },
		})
		
		// 验证所有操作都成功完成
		for i, err := range errors {
			assert.NoError(t, err, "操作 %d 应该成功", i)
		}
		
		// 验证结果
		for i, result := range results {
			assert.Equal(t, i+1, result)
		}
	})

}

// 辅助函数
func testSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}