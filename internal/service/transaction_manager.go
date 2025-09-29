package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

// TransactionManager 事务管理器
type TransactionManager struct {
	db *gorm.DB
}

// NewTransactionManager 创建事务管理器
func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// ExecuteInTransaction 在事务中执行操作
func (tm *TransactionManager) ExecuteInTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 设置事务超时时间
		txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// 执行事务操作
		if err := fn(tx); err != nil {
			logger.Warnf("事务执行失败，将回滚: %v", err)
			return err
		}

		// 检查事务上下文是否超时
		select {
		case <-txCtx.Done():
			return fmt.Errorf("事务执行超时")
		default:
			return nil
		}
	})
}

// ExecuteWithRetry 带重试的事务执行
func (tm *TransactionManager) ExecuteWithRetry(ctx context.Context, fn func(tx *gorm.DB) error, maxRetries int) error {
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			logger.Infof("事务重试第 %d 次", attempt)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // 指数退避
		}

		err := tm.ExecuteInTransaction(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err
		
		// 判断是否需要重试
		if !tm.shouldRetry(err) {
			return err
		}
	}

	return fmt.Errorf("事务执行失败，已达到最大重试次数 %d: %w", maxRetries, lastErr)
}

// shouldRetry 判断是否应该重试
func (tm *TransactionManager) shouldRetry(err error) bool {
	errMsg := err.Error()
	
	// 死锁错误应该重试
	if contains(errMsg, "deadlock", "lock", "timeout") {
		return true
	}
	
	// 网络错误应该重试
	if contains(errMsg, "network", "connection", "timeout") {
		return true
	}
	
	// 数据库连接错误应该重试
	if contains(errMsg, "database", "connection", "closed") {
		return true
	}
	
	return false
}

// ConcurrentOperationManager 并发操作管理器
type ConcurrentOperationManager struct {
	maxConcurrent int
	semaphore     chan struct{}
}

// NewConcurrentOperationManager 创建并发操作管理器
func NewConcurrentOperationManager(maxConcurrent int) *ConcurrentOperationManager {
	return &ConcurrentOperationManager{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// Execute 执行并发操作
func (cm *ConcurrentOperationManager) Execute(ctx context.Context, fn func() error) error {
	// 获取信号量
	select {
	case cm.semaphore <- struct{}{}:
		defer func() { <-cm.semaphore }() // 释放信号量
	case <-ctx.Done():
		return ctx.Err()
	}

	return fn()
}

// ExecuteBatch 批量执行并发操作
func (cm *ConcurrentOperationManager) ExecuteBatch(ctx context.Context, operations []func() error) []error {
	errors := make([]error, len(operations))
	errChan := make(chan struct {
		index int
		err   error
	}, len(operations))

	// 启动所有操作
	for i, op := range operations {
		go func(index int, operation func() error) {
			err := cm.Execute(ctx, operation)
			errChan <- struct {
				index int
				err   error
			}{index, err}
		}(i, op)
	}

	// 收集结果
	for i := 0; i < len(operations); i++ {
		result := <-errChan
		errors[result.index] = result.err
	}

	return errors
}

// OptimizedReminderService 优化的提醒服务
type OptimizedReminderService struct {
	reminderRepo interfaces.ReminderRepository
	txManager    *TransactionManager
	concurrency  *ConcurrentOperationManager
}

// NewOptimizedReminderService 创建优化的提醒服务
func NewOptimizedReminderService(reminderRepo interfaces.ReminderRepository, db *gorm.DB) *OptimizedReminderService {
	return &OptimizedReminderService{
		reminderRepo: reminderRepo,
		txManager:    NewTransactionManager(db),
		concurrency:  NewConcurrentOperationManager(10), // 最多10个并发操作
	}
}

// CreateReminderWithLogs 创建提醒及相关记录（事务操作）
func (s *OptimizedReminderService) CreateReminderWithLogs(ctx context.Context, reminder *models.Reminder, logs []*models.ReminderLog) error {
	return s.txManager.ExecuteWithRetry(ctx, func(tx *gorm.DB) error {
		// 创建提醒
		if err := s.reminderRepo.Create(ctx, reminder); err != nil {
			return fmt.Errorf("创建提醒失败: %w", err)
		}

		// 批量创建提醒记录
		for _, log := range logs {
			log.ReminderID = reminder.ID
			if err := tx.WithContext(ctx).Create(log).Error; err != nil {
				return fmt.Errorf("创建提醒记录失败: %w", err)
			}
		}

		logger.Infof("✅ 事务性创建提醒完成: ID=%d, Logs=%d", reminder.ID, len(logs))
		return nil
	}, 3) // 最多重试3次
}

// BatchUpdateReminders 批量更新提醒（并发操作）
func (s *OptimizedReminderService) BatchUpdateReminders(ctx context.Context, reminders []*models.Reminder) error {
	operations := make([]func() error, len(reminders))
	
	for i, reminder := range reminders {
		r := reminder // 避免闭包问题
		operations[i] = func() error {
			if err := s.reminderRepo.Update(ctx, r); err != nil {
				return fmt.Errorf("更新提醒失败 (ID: %d): %w", r.ID, err)
			}
			return nil
		}
	}

	errors := s.concurrency.ExecuteBatch(ctx, operations)
	
	// 检查是否有错误
	hasError := false
	for i, err := range errors {
		if err != nil {
			logger.Errorf("批量更新提醒失败 [%d]: %v", i, err)
			hasError = true
		}
	}

	if hasError {
		return fmt.Errorf("批量更新提醒完成，部分操作失败")
	}

	logger.Infof("✅ 批量更新提醒完成: %d 条记录", len(reminders))
	return nil
}

// SafeDeleteReminder 安全删除提醒（带级联检查）
func (s *OptimizedReminderService) SafeDeleteReminder(ctx context.Context, reminderID uint) error {
	return s.txManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 检查是否存在相关的提醒记录
		var count int64
		err := tx.WithContext(ctx).
			Model(&models.ReminderLog{}).
			Where("reminder_id = ?", reminderID).
			Count(&count).Error
		
		if err != nil {
			return fmt.Errorf("检查提醒记录失败: %w", err)
		}

		if count > 0 {
			logger.Warnf("删除提醒存在关联记录 (ReminderID: %d, Logs: %d)", reminderID, count)
			// 可以选择软删除或级联删除
		}

		// 执行删除
		if err := s.reminderRepo.Delete(ctx, reminderID); err != nil {
			return fmt.Errorf("删除提醒失败: %w", err)
		}

		logger.Infof("✅ 安全删除提醒完成: ID=%d", reminderID)
		return nil
	})
}


// toLower 转换为小写

// containsIgnoreCase 忽略大小写包含检查
