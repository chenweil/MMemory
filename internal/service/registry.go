package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ServiceType 服务类型枚举
type ServiceType string

const (
	ServiceTypeUser         ServiceType = "user"
	ServiceTypeReminder     ServiceType = "reminder"
	ServiceTypeReminderLog  ServiceType = "reminder_log"
	ServiceTypeScheduler    ServiceType = "scheduler"
	ServiceTypeNotification ServiceType = "notification"
)

// ServiceMetadata 服务元数据
type ServiceMetadata struct {
	Name        string
	Type        ServiceType
	Version     string
	Description string
	HealthCheck func() error
}

// ServiceInstance 服务实例接口
type ServiceInstance interface {
	GetMetadata() ServiceMetadata
	Start() error
	Stop() error
	IsHealthy() bool
}

// ServiceRegistry 服务注册中心
type ServiceRegistry struct {
	mu        sync.RWMutex
	services  map[ServiceType]ServiceInstance
	listeners []ServiceEventListener
}

// ServiceEvent 服务事件
type ServiceEvent struct {
	Type      ServiceEventType
	Service   ServiceInstance
	Timestamp int64
	Error     error
}

// ServiceEventType 服务事件类型
type ServiceEventType string

const (
	ServiceEventRegistered   ServiceEventType = "registered"
	ServiceEventUnregistered ServiceEventType = "unregistered"
	ServiceEventStarted      ServiceEventType = "started"
	ServiceEventStopped      ServiceEventType = "stopped"
	ServiceEventHealthCheck  ServiceEventType = "health_check"
	ServiceEventError        ServiceEventType = "error"
)

// ServiceEventListener 服务事件监听器
type ServiceEventListener func(event ServiceEvent)

// NewServiceRegistry 创建服务注册中心
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services:  make(map[ServiceType]ServiceInstance),
		listeners: make([]ServiceEventListener, 0),
	}
}

// Register 注册服务
func (r *ServiceRegistry) Register(service ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := service.GetMetadata()
	if _, exists := r.services[metadata.Type]; exists {
		return fmt.Errorf("服务类型 %s 已存在", metadata.Type)
	}

	r.services[metadata.Type] = service
	fmt.Printf("✅ 服务注册成功: %s (%s)\n", metadata.Name, metadata.Type)

	// 发送注册事件
	r.publishEvent(ServiceEvent{
		Type:      ServiceEventRegistered,
		Service:   service,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// Unregister 注销服务
func (r *ServiceRegistry) Unregister(serviceType ServiceType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exists := r.services[serviceType]
	if !exists {
		return fmt.Errorf("服务类型 %s 不存在", serviceType)
	}

	delete(r.services, serviceType)
	fmt.Printf("🗑️ 服务注销成功: %s\n", serviceType)

	// 发送注销事件
	r.publishEvent(ServiceEvent{
		Type:      ServiceEventUnregistered,
		Service:   service,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// Get 获取服务实例
func (r *ServiceRegistry) Get(serviceType ServiceType) (ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[serviceType]
	if !exists {
		return nil, fmt.Errorf("服务类型 %s 不存在", serviceType)
	}

	return service, nil
}

// GetUserService 获取用户服务
func (r *ServiceRegistry) GetUserService() (interface{}, error) {
	service, err := r.Get(ServiceTypeUser)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// GetReminderService 获取提醒服务
func (r *ServiceRegistry) GetReminderService() (interface{}, error) {
	service, err := r.Get(ServiceTypeReminder)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// GetSchedulerService 获取调度服务
func (r *ServiceRegistry) GetSchedulerService() (interface{}, error) {
	service, err := r.Get(ServiceTypeScheduler)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// GetNotificationService 获取通知服务
func (r *ServiceRegistry) GetNotificationService() (interface{}, error) {
	service, err := r.Get(ServiceTypeNotification)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// GetReminderLogService 获取提醒记录服务
func (r *ServiceRegistry) GetReminderLogService() (interface{}, error) {
	service, err := r.Get(ServiceTypeReminderLog)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// StartAll 启动所有服务
func (r *ServiceRegistry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fmt.Println("🚀 启动所有服务...")
	
	for serviceType, service := range r.services {
		if err := service.Start(); err != nil {
			r.publishEvent(ServiceEvent{
				Type:      ServiceEventError,
				Service:   service,
				Timestamp: time.Now().Unix(),
				Error:     err,
			})
			return fmt.Errorf("启动服务 %s 失败: %w", serviceType, err)
		}

		fmt.Printf("✅ 服务启动成功: %s\n", serviceType)
		
		r.publishEvent(ServiceEvent{
			Type:      ServiceEventStarted,
			Service:   service,
			Timestamp: time.Now().Unix(),
		})
	}

	fmt.Println("✅ 所有服务启动完成")
	return nil
}

// StopAll 停止所有服务
func (r *ServiceRegistry) StopAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fmt.Println("🛑 停止所有服务...")
	
	for serviceType, service := range r.services {
		if err := service.Stop(); err != nil {
			fmt.Printf("停止服务 %s 失败: %v\n", serviceType, err)
			continue
		}

		fmt.Printf("✅ 服务停止成功: %s\n", serviceType)

		r.publishEvent(ServiceEvent{
			Type:      ServiceEventStopped,
			Service:   service,
			Timestamp: time.Now().Unix(),
		})
	}

	fmt.Println("✅ 所有服务停止完成")
	return nil
}

// HealthCheck 健康检查
func (r *ServiceRegistry) HealthCheck(ctx context.Context) map[ServiceType]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[ServiceType]error)
	
	for serviceType, service := range r.services {
		metadata := service.GetMetadata()
		var err error
		
		if metadata.HealthCheck != nil {
			err = metadata.HealthCheck()
		} else {
			err = r.basicHealthCheck(service)
		}
		
		results[serviceType] = err
		
		r.publishEvent(ServiceEvent{
			Type:      ServiceEventHealthCheck,
			Service:   service,
			Timestamp: time.Now().Unix(),
			Error:     err,
		})
	}

	return results
}

// basicHealthCheck 基础健康检查
func (r *ServiceRegistry) basicHealthCheck(service ServiceInstance) error {
	if !service.IsHealthy() {
		return fmt.Errorf("服务健康检查失败")
	}
	return nil
}

// AddEventListener 添加事件监听器
func (r *ServiceRegistry) AddEventListener(listener ServiceEventListener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.listeners = append(r.listeners, listener)
}

// publishEvent 发布事件
func (r *ServiceRegistry) publishEvent(event ServiceEvent) {
	for _, listener := range r.listeners {
		go func(l ServiceEventListener) {
			defer func() {
				if r := recover(); r != nil {
					// 避免监听器panic影响主流程
					fmt.Printf("服务事件监听器 panic: %v\n", r)
				}
			}()
			l(event)
		}(listener)
	}
}

// GetServiceStats 获取服务统计信息
func (r *ServiceRegistry) GetServiceStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"total_services": len(r.services),
		"service_types":  make([]string, 0, len(r.services)),
	}

	for serviceType := range r.services {
		stats["service_types"] = append(stats["service_types"].([]string), string(serviceType))
	}

	return stats
}

// GlobalServiceRegistry 全局服务注册中心实例
var GlobalServiceRegistry = NewServiceRegistry()