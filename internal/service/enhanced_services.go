package service

import (
	"context"
	"fmt"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

// BaseService 基础服务结构
type BaseService struct {
	name        string
	serviceType ServiceType
	metadata    ServiceMetadata
	started     bool
	healthy     bool
	startTime   time.Time
	stopTime    time.Time
}

// NewBaseService 创建基础服务
func NewBaseService(name string, serviceType ServiceType, description string) *BaseService {
	return &BaseService{
		name:        name,
		serviceType: serviceType,
		metadata: ServiceMetadata{
			Name:        name,
			Type:        serviceType,
			Version:     "1.0.0",
			Description: description,
		},
		healthy: true,
	}
}

// GetMetadata 获取服务元数据
func (s *BaseService) GetMetadata() ServiceMetadata {
	return s.metadata
}

// Start 启动服务
func (s *BaseService) Start() error {
	if s.started {
		return fmt.Errorf("服务 %s 已在运行中", s.name)
	}

	s.started = true
	s.healthy = true
	s.startTime = time.Now()
	
	// 避免日志nil指针，使用fmt打印
	fmt.Printf("🚀 服务启动成功: %s\n", s.name)
	return nil
}

// Stop 停止服务
func (s *BaseService) Stop() error {
	if !s.started {
		return fmt.Errorf("服务 %s 未在运行中", s.name)
	}

	s.started = false
	s.healthy = false
	s.stopTime = time.Now()
	
	// 避免日志nil指针，使用fmt打印
	fmt.Printf("🛑 服务停止成功: %s\n", s.name)
	return nil
}

// IsHealthy 检查服务健康状态
func (s *BaseService) IsHealthy() bool {
	return s.healthy && s.started
}

// IsStarted 检查服务是否已启动
func (s *BaseService) IsStarted() bool {
	return s.started
}

// SetHealthy 设置健康状态
func (s *BaseService) SetHealthy(healthy bool) {
	s.healthy = healthy
}

// GetName 获取服务名称
func (s *BaseService) GetName() string {
	return s.name
}

// GetStartTime 获取启动时间
func (s *BaseService) GetStartTime() time.Time {
	return s.startTime
}

// GetUptime 获取运行时长
func (s *BaseService) GetUptime() time.Duration {
	if s.startTime.IsZero() {
		return 0
	}
	
	if s.stopTime.After(s.startTime) {
		return s.stopTime.Sub(s.startTime)
	}
	
	return time.Since(s.startTime)
}

// EnhancedUserService 增强的用户服务
type EnhancedUserService struct {
	*BaseService
	userRepo interfaces.UserRepository
	errorHandler *ErrorHandler
}

// NewEnhancedUserService 创建增强的用户服务
func NewEnhancedUserService(userRepo interfaces.UserRepository) *EnhancedUserService {
	baseService := NewBaseService("UserService", ServiceTypeUser, "用户管理服务")
	return &EnhancedUserService{
		BaseService:  baseService,
		userRepo:     userRepo,
		errorHandler: NewErrorHandler(context.Background(), "UserService"),
	}
}

// Start 启动服务
func (s *EnhancedUserService) Start() error {
	if err := s.BaseService.Start(); err != nil {
		return err
	}
	
	// 执行额外的启动逻辑
	logger.Debug("🔄 用户服务初始化中...")
	
	// 可以在这里添加数据库连接检查、缓存初始化等
	return nil
}

// Stop 停止服务
func (s *EnhancedUserService) Stop() error {
	logger.Debug("🔄 用户服务清理中...")
	
	// 执行额外的清理逻辑
	
	return s.BaseService.Stop()
}

// CreateUser 创建用户（增强版）
func (s *EnhancedUserService) CreateUser(ctx context.Context, user *models.User) error {
	defer func() {
		if r := recover(); r != nil {
			err := s.errorHandler.HandlePanic(r, "CreateUser")
			logger.Errorf("创建用户时发生严重错误: %v", err)
		}
	}()

	if user.TelegramID == 0 {
		return NewError(CommonErrorCodes.InvalidParameter, "Telegram ID不能为空").
			WithService(s.GetName()).
			WithOperation("CreateUser")
	}

	// 检查用户是否已存在
	existingUser, err := s.userRepo.GetByTelegramID(ctx, user.TelegramID)
	if err != nil {
		s.errorHandler.HandleError(err, "GetByTelegramID", map[string]interface{}{
			"telegram_id": user.TelegramID,
		})
		return NewError(CommonErrorCodes.DBQueryError, "检查用户是否存在失败").
			WithService(s.GetName()).
			WithOperation("CreateUser").
			WithCause(err)
	}
	
	if existingUser != nil {
		return NewError(CommonErrorCodes.ResourceConflict, "用户已存在").
			WithService(s.GetName()).
			WithOperation("CreateUser").
			WithDetail("telegram_id", user.TelegramID)
	}

	// 设置默认值
	if user.Timezone == "" {
		user.Timezone = "Asia/Shanghai"
	}
	if user.LanguageCode == "" {
		user.LanguageCode = "zh-CN"
	}
	user.IsActive = true

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.errorHandler.HandleError(err, "Create", map[string]interface{}{
			"telegram_id": user.TelegramID,
		})
		return NewError(CommonErrorCodes.DBQueryError, "创建用户失败").
			WithService(s.GetName()).
			WithOperation("CreateUser").
			WithCause(err)
	}

	logger.Infof("✅ 用户创建成功: TelegramID=%d, Username=%s", user.TelegramID, user.Username)
	return nil
}

// GetByTelegramID 根据Telegram ID获取用户
func (s *EnhancedUserService) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	user, err := s.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		s.errorHandler.HandleError(err, "GetByTelegramID", map[string]interface{}{
			"telegram_id": telegramID,
		})
		return nil, NewError(CommonErrorCodes.DBQueryError, "获取用户失败").
			WithService(s.GetName()).
			WithOperation("GetByTelegramID").
			WithCause(err)
	}
	
	if user == nil {
		return nil, NewError(CommonErrorCodes.ResourceNotFound, "用户不存在").
			WithService(s.GetName()).
			WithOperation("GetByTelegramID").
			WithDetail("telegram_id", telegramID)
	}
	
	return user, nil
}

// GetByID 根据ID获取用户
func (s *EnhancedUserService) GetByID(ctx context.Context, id uint) (*models.User, error) {
	if id == 0 {
		return nil, NewError(CommonErrorCodes.InvalidParameter, "用户ID不能为空").
			WithService(s.GetName()).
			WithOperation("GetByID")
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.errorHandler.HandleError(err, "GetByID", map[string]interface{}{
			"user_id": id,
		})
		return nil, NewError(CommonErrorCodes.DBQueryError, "获取用户失败").
			WithService(s.GetName()).
			WithOperation("GetByID").
			WithCause(err)
	}
	
	if user == nil {
		return nil, NewError(CommonErrorCodes.ResourceNotFound, "用户不存在").
			WithService(s.GetName()).
			WithOperation("GetByID").
			WithDetail("user_id", id)
	}
	
	return user, nil
}

// UpdateUser 更新用户信息
func (s *EnhancedUserService) UpdateUser(ctx context.Context, user *models.User) error {
	if user.ID == 0 {
		return NewError(CommonErrorCodes.InvalidParameter, "用户ID不能为空").
			WithService(s.GetName()).
			WithOperation("UpdateUser")
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.errorHandler.HandleError(err, "Update", map[string]interface{}{
			"user_id": user.ID,
		})
		return NewError(CommonErrorCodes.DBQueryError, "更新用户失败").
			WithService(s.GetName()).
			WithOperation("UpdateUser").
			WithCause(err)
	}

	logger.Infof("✅ 用户更新成功: ID=%d, TelegramID=%d", user.ID, user.TelegramID)
	return nil
}

// GetHealthCheck 获取健康检查函数
func (s *EnhancedUserService) GetHealthCheck() func() error {
	return func() error {
		// 检查数据库连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		_, err := s.userRepo.GetByID(ctx, 1) // 尝试查询一个用户
		if err != nil {
			return fmt.Errorf("数据库连接检查失败: %w", err)
		}
		
		return nil
	}
}