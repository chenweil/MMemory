package service

import (
	"context"
	"testing"

	"mmemory/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// MockServiceInstance Mock服务实例
type MockServiceInstance struct {
	name     string
	started  bool
	healthy  bool
	metadata ServiceMetadata
}

func NewMockServiceInstance(name string, serviceType ServiceType) *MockServiceInstance {
	return &MockServiceInstance{
		name:    name,
		healthy: true,
		metadata: ServiceMetadata{
			Name:        name,
			Type:        serviceType,
			Version:     "1.0.0",
			Description: "Mock service for testing",
		},
	}
}

func (m *MockServiceInstance) GetMetadata() ServiceMetadata {
	return m.metadata
}

func (m *MockServiceInstance) Start() error {
	m.started = true
	return nil
}

func (m *MockServiceInstance) Stop() error {
	m.started = false
	return nil
}

func (m *MockServiceInstance) IsHealthy() bool {
	return m.healthy
}

// TestServiceRegistry_Register 测试注册服务
func TestServiceRegistry_Register(t *testing.T) {
	registry := NewServiceRegistry()
	service := NewMockServiceInstance("test-service", ServiceTypeUser)

	err := registry.Register(service)

	assert.NoError(t, err)
	
	registeredService, exists := registry.services[ServiceTypeUser]
	assert.True(t, exists)
	assert.Equal(t, service, registeredService)
}

// TestServiceRegistry_RegisterDuplicate 测试注册重复服务
func TestServiceRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewServiceRegistry()
	service1 := NewMockServiceInstance("service1", ServiceTypeUser)
	service2 := NewMockServiceInstance("service2", ServiceTypeUser)

	// 注册第一个服务
	err := registry.Register(service1)
	assert.NoError(t, err)

	// 尝试注册相同类型的第二个服务
	err = registry.Register(service2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在")
}

// TestServiceRegistry_Unregister 测试注销服务
func TestServiceRegistry_Unregister(t *testing.T) {
	registry := NewServiceRegistry()
	service := NewMockServiceInstance("test-service", ServiceTypeUser)

	// 先注册
	registry.Register(service)

	// 然后注销
	err := registry.Unregister(ServiceTypeUser)

	assert.NoError(t, err)
	
	_, exists := registry.services[ServiceTypeUser]
	assert.False(t, exists)
}

// TestServiceRegistry_UnregisterNonExistent 测试注销不存在的服务
func TestServiceRegistry_UnregisterNonExistent(t *testing.T) {
	registry := NewServiceRegistry()

	err := registry.Unregister(ServiceTypeUser)

	assert.Error(t, err)
}

// TestServiceRegistry_Get 测试获取服务
func TestServiceRegistry_Get(t *testing.T) {
	registry := NewServiceRegistry()
	service := NewMockServiceInstance("test-service", ServiceTypeUser)

	registry.Register(service)

	retrievedService, err := registry.Get(ServiceTypeUser)

	assert.NoError(t, err)
	assert.Equal(t, service, retrievedService)
}

// TestServiceRegistry_GetNotFound 测试获取不存在的服务
func TestServiceRegistry_GetNotFound(t *testing.T) {
	registry := NewServiceRegistry()

	_, err := registry.Get(ServiceTypeUser)

	assert.Error(t, err)
}

// TestServiceRegistry_StartAll 测试启动所有服务
func TestServiceRegistry_StartAll(t *testing.T) {
	registry := NewServiceRegistry()
	
	service1 := NewMockServiceInstance("service1", ServiceTypeUser)
	service2 := NewMockServiceInstance("service2", ServiceTypeReminder)
	
	registry.Register(service1)
	registry.Register(service2)

	ctx := context.Background()
	err := registry.StartAll(ctx)

	assert.NoError(t, err)
	assert.True(t, service1.started)
	assert.True(t, service2.started)
}

// TestServiceRegistry_StopAll 测试停止所有服务
func TestServiceRegistry_StopAll(t *testing.T) {
	registry := NewServiceRegistry()
	
	service1 := NewMockServiceInstance("service1", ServiceTypeUser)
	service2 := NewMockServiceInstance("service2", ServiceTypeReminder)
	
	registry.Register(service1)
	registry.Register(service2)

	// 先启动
	ctx := context.Background()
	registry.StartAll(ctx)

	// 然后停止
	err := registry.StopAll(ctx)

	assert.NoError(t, err)
	assert.False(t, service1.started)
	assert.False(t, service2.started)
}

// TestServiceRegistry_AddEventListener 测试添加事件监听器
func TestServiceRegistry_AddEventListener(t *testing.T) {
	registry := NewServiceRegistry()

	listener := func(event ServiceEvent) {
		// 测试监听器
	}

	registry.AddEventListener(listener)

	assert.Len(t, registry.listeners, 1)
}

// TestServiceRegistry_HealthCheck 测试健康检查
func TestServiceRegistry_HealthCheck(t *testing.T) {
	registry := NewServiceRegistry()
	
	service := NewMockServiceInstance("test-service", ServiceTypeUser)
	registry.Register(service)

	ctx := context.Background()
	health := registry.HealthCheck(ctx)

	assert.NotNil(t, health)
	assert.Len(t, health, 1)
}

// TestServiceRegistry_GetServiceStats 测试获取统计信息
func TestServiceRegistry_GetServiceStats(t *testing.T) {
	registry := NewServiceRegistry()
	
	service := NewMockServiceInstance("test-service", ServiceTypeUser)
	registry.Register(service)

	stats := registry.GetServiceStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats["total_services"])
}

// 初始化日志
func init() {
	logger.Init("info", "text", "stdout", "")
}