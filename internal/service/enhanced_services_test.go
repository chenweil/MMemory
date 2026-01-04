package service

import (
	"testing"
	"time"

	"mmemory/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// TestBaseService_Start 测试启动基础服务
func TestBaseService_Start(t *testing.T) {
	service := NewBaseService("test-service", ServiceTypeUser, "Test service for unit testing")

	err := service.Start()

	assert.NoError(t, err)
	assert.True(t, service.IsStarted())
}

// TestBaseService_Stop 测试停止基础服务
func TestBaseService_Stop(t *testing.T) {
	service := NewBaseService("test-service", ServiceTypeUser, "Test service for unit testing")

	// 先启动
	err := service.Start()
	assert.NoError(t, err)

	// 然后停止
	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.IsStarted())
}

// TestBaseService_StartStop 测试启动和停止
func TestBaseService_StartStop(t *testing.T) {
	service := NewBaseService("test-service", ServiceTypeReminder, "Test service for unit testing")

	// 启动
	err := service.Start()
	assert.NoError(t, err)
	assert.True(t, service.IsStarted())

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	// 停止
	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.IsStarted())
}

// TestBaseService_StartTwice 测试重复启动
func TestBaseService_StartTwice(t *testing.T) {
	service := NewBaseService("test-service", ServiceTypeScheduler, "Test service for unit testing")

	// 第一次启动
	err := service.Start()
	assert.NoError(t, err)

	// 第二次启动应该失败
	err = service.Start()
	assert.Error(t, err)
}

// TestBaseService_GetMetadata 测试获取元数据
func TestBaseService_GetMetadata(t *testing.T) {
	service := NewBaseService("test-service", ServiceTypeNotification, "Test service for unit testing")

	metadata := service.GetMetadata()

	assert.Equal(t, "test-service", metadata.Name)
	assert.Equal(t, ServiceTypeNotification, metadata.Type)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "Test service for unit testing", metadata.Description)
}

// 初始化日志
func init() {
	logger.Init("info", "text", "stdout", "")
}