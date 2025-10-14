package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mmemory/internal/models"
)

// Mock repositories for monitoring tests
type mockUserRepositoryForMonitoring struct {
	users     map[int64]*models.User
	idCounter uint
}

func newMockUserRepositoryForMonitoring() *mockUserRepositoryForMonitoring {
	return &mockUserRepositoryForMonitoring{
		users:     make(map[int64]*models.User),
		idCounter: 1,
	}
}

func (m *mockUserRepositoryForMonitoring) Create(ctx context.Context, user *models.User) error {
	if user.ID == 0 {
		user.ID = m.idCounter
		m.idCounter++
	}
	m.users[user.TelegramID] = user
	return nil
}

func (m *mockUserRepositoryForMonitoring) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	if user, ok := m.users[telegramID]; ok {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockUserRepositoryForMonitoring) GetByID(ctx context.Context, id uint) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockUserRepositoryForMonitoring) Update(ctx context.Context, user *models.User) error {
	if _, exists := m.users[user.TelegramID]; exists {
		m.users[user.TelegramID] = user
		return nil
	}
	return fmt.Errorf("user not found")
}

func (m *mockUserRepositoryForMonitoring) Delete(ctx context.Context, id uint) error {
	for telegramID, user := range m.users {
		if user.ID == id {
			delete(m.users, telegramID)
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

func (m *mockUserRepositoryForMonitoring) Count(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

// Mock reminder repository for monitoring
type mockReminderRepositoryForMonitoring struct {
	*mockReminderRepository
	statusCounts map[models.ReminderStatStatus]int64
}

func newMockReminderRepositoryForMonitoring() *mockReminderRepositoryForMonitoring {
	return &mockReminderRepositoryForMonitoring{
		mockReminderRepository: newMockReminderRepository(),
		statusCounts: map[models.ReminderStatStatus]int64{
			models.ReminderStatStatusActive:    0,
			models.ReminderStatStatusCompleted: 0,
			models.ReminderStatStatusExpired:   0,
		},
	}
}

func (m *mockReminderRepositoryForMonitoring) CountByStatus(ctx context.Context, status models.ReminderStatStatus) (int64, error) {
	return m.statusCounts[status], nil
}

func (m *mockReminderRepositoryForMonitoring) SetStatusCount(status models.ReminderStatStatus, count int64) {
	m.statusCounts[status] = count
}

// TestMonitoringService_Start 测试监控服务启动
func TestMonitoringService_Start(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo).(*monitoringService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动监控服务
	if err := monitoring.Start(ctx); err != nil {
		t.Fatalf("Start() 失败: %v", err)
	}

	// 验证服务已启动
	if monitoring.updateTicker == nil {
		t.Fatal("期待 updateTicker 已初始化")
	}

	// 等待一小段时间确保goroutine启动
	time.Sleep(100 * time.Millisecond)

	// 停止服务
	if err := monitoring.Stop(); err != nil {
		t.Fatalf("Stop() 失败: %v", err)
	}
}

// TestMonitoringService_UpdateMetrics 测试指标更新
func TestMonitoringService_UpdateMetrics(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	// 添加一些测试数据
	mockUserRepo.Create(context.Background(), &models.User{TelegramID: 1, Username: "user1"})
	mockUserRepo.Create(context.Background(), &models.User{TelegramID: 2, Username: "user2"})

	mockReminderRepo.SetStatusCount(models.ReminderStatStatusActive, 10)
	mockReminderRepo.SetStatusCount(models.ReminderStatStatusCompleted, 50)
	mockReminderRepo.SetStatusCount(models.ReminderStatStatusExpired, 5)

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	ctx := context.Background()
	if err := monitoring.UpdateMetrics(ctx); err != nil {
		t.Fatalf("UpdateMetrics() 失败: %v", err)
	}

	// 注意：这里我们只验证没有报错，因为实际指标记录到prometheus，不易验证
	// 在真实环境中，可以通过prometheus的测试工具验证指标值
}

// TestMonitoringService_RecordReminderOperation 测试记录提醒操作
func TestMonitoringService_RecordReminderOperation(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	tests := []struct {
		name      string
		operation string
		status    bool
	}{
		{
			name:      "记录创建操作",
			operation: "created",
			status:    true,
		},
		{
			name:      "记录完成操作",
			operation: "completed",
			status:    true,
		},
		{
			name:      "记录跳过操作",
			operation: "skipped",
			status:    true,
		},
		{
			name:      "忽略失败操作",
			operation: "created",
			status:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这些方法不返回错误，只是记录指标
			monitoring.RecordReminderOperation(tt.operation, tt.status)
		})
	}
}

// TestMonitoringService_RecordDatabaseOperation 测试记录数据库操作
func TestMonitoringService_RecordDatabaseOperation(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	tests := []struct {
		name      string
		operation string
		duration  time.Duration
		err       error
	}{
		{
			name:      "成功的查询操作",
			operation: "select",
			duration:  50 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "失败的插入操作",
			operation: "insert",
			duration:  100 * time.Millisecond,
			err:       fmt.Errorf("insert error"),
		},
		{
			name:      "成功的更新操作",
			operation: "update",
			duration:  75 * time.Millisecond,
			err:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitoring.RecordDatabaseOperation(tt.operation, tt.duration, tt.err)
		})
	}
}

// TestMonitoringService_RecordNotificationSend 测试记录通知发送
func TestMonitoringService_RecordNotificationSend(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	tests := []struct {
		name             string
		notificationType string
		duration         time.Duration
		err              error
	}{
		{
			name:             "成功的提醒通知",
			notificationType: "reminder",
			duration:         200 * time.Millisecond,
			err:              nil,
		},
		{
			name:             "失败的跟进通知",
			notificationType: "followup",
			duration:         150 * time.Millisecond,
			err:              fmt.Errorf("send error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitoring.RecordNotificationSend(tt.notificationType, tt.duration, tt.err)
		})
	}
}

// TestMonitoringService_RecordBotMessage 测试记录Bot消息
func TestMonitoringService_RecordBotMessage(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	tests := []struct {
		name        string
		messageType string
		err         error
	}{
		{
			name:        "成功的文本消息",
			messageType: "text",
			err:         nil,
		},
		{
			name:        "失败的回调消息",
			messageType: "callback",
			err:         fmt.Errorf("callback error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitoring.RecordBotMessage(tt.messageType, tt.err)
		})
	}
}

// TestMonitoringService_RecordReminderParse 测试记录提醒解析
func TestMonitoringService_RecordReminderParse(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	tests := []struct {
		name       string
		parserType string
		duration   time.Duration
		err        error
	}{
		{
			name:       "成功的AI解析",
			parserType: "ai",
			duration:   500 * time.Millisecond,
			err:        nil,
		},
		{
			name:       "失败的正则解析",
			parserType: "regex",
			duration:   10 * time.Millisecond,
			err:        fmt.Errorf("parse error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitoring.RecordReminderParse(tt.parserType, tt.duration, tt.err)
		})
	}
}

// TestMonitoringService_Stop 测试停止监控服务
func TestMonitoringService_Stop(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo).(*monitoringService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 先启动
	if err := monitoring.Start(ctx); err != nil {
		t.Fatalf("Start() 失败: %v", err)
	}

	// 等待一小段时间
	time.Sleep(50 * time.Millisecond)

	// 停止服务
	if err := monitoring.Stop(); err != nil {
		t.Fatalf("Stop() 失败: %v", err)
	}

	// 验证ticker已停止
	// 注意：ticker.Stop()后ticker字段不会变成nil，只是停止触发
	// 我们主要验证stopChan已关闭
	select {
	case <-monitoring.stopChan:
		// stopChan已关闭，符合预期
	default:
		t.Fatal("期待 stopChan 已关闭")
	}
}

// TestMonitoringService_ConcurrentOperations 测试并发操作
func TestMonitoringService_ConcurrentOperations(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	// 并发记录多种操作
	done := make(chan bool, 4)

	go func() {
		for i := 0; i < 100; i++ {
			monitoring.RecordReminderOperation("created", true)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			monitoring.RecordDatabaseOperation("select", 50*time.Millisecond, nil)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			monitoring.RecordNotificationSend("reminder", 100*time.Millisecond, nil)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			monitoring.RecordBotMessage("text", nil)
		}
		done <- true
	}()

	// 等待所有goroutine完成
	for i := 0; i < 4; i++ {
		<-done
	}

	// 如果没有panic或死锁，测试通过
}

// TestMonitoringService_UpdateMetrics_EmptyData 测试空数据更新指标
func TestMonitoringService_UpdateMetrics_EmptyData(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo)

	ctx := context.Background()
	if err := monitoring.UpdateMetrics(ctx); err != nil {
		t.Fatalf("UpdateMetrics() 失败: %v", err)
	}

	// 验证空数据情况下不会报错
}

// TestMonitoringService_Uptime 测试系统运行时间计算
func TestMonitoringService_Uptime(t *testing.T) {
	mockUserRepo := newMockUserRepositoryForMonitoring()
	mockReminderRepo := newMockReminderRepositoryForMonitoring()
	mockLogRepo := newMockReminderLogRepository()

	monitoring := NewMonitoringService(mockUserRepo, mockReminderRepo, mockLogRepo).(*monitoringService)

	// 记录启动时间
	startTime := monitoring.startTime

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 验证运行时间至少100ms
	uptime := time.Since(startTime)
	if uptime < 100*time.Millisecond {
		t.Errorf("期待运行时间至少100ms，实际得到: %v", uptime)
	}
}
