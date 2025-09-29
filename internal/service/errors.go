package service

import (
	"context"
	"fmt"
	"runtime/debug"

	"mmemory/pkg/logger"
)

// ErrorLevel 错误级别
type ErrorLevel string

const (
	ErrorLevelCritical ErrorLevel = "CRITICAL"
	ErrorLevelError    ErrorLevel = "ERROR"
	ErrorLevelWarning  ErrorLevel = "WARNING"
	ErrorLevelInfo     ErrorLevel = "INFO"
)

// ServiceError 服务错误结构
type ServiceError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Level      ErrorLevel             `json:"level"`
	Service    string                 `json:"service"`
	Operation  string                 `json:"operation"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Cause      error                  `json:"-"`
}

// Error 实现 error 接口
func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Level, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Level, e.Code, e.Message)
}

// Unwrap 实现 error unwrap 接口
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// NewServiceError 创建服务错误
func NewServiceError(code, message string, level ErrorLevel) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Level:   level,
		Details: make(map[string]interface{}),
	}
}

// NewCriticalError 创建关键错误
func NewCriticalError(code, message string) *ServiceError {
	err := NewServiceError(code, message, ErrorLevelCritical)
	err.StackTrace = string(debug.Stack())
	return err
}

// NewError 创建普通错误
func NewError(code, message string) *ServiceError {
	return NewServiceError(code, message, ErrorLevelError)
}

// NewWarning 创建警告
func NewWarning(code, message string) *ServiceError {
	return NewServiceError(code, message, ErrorLevelWarning)
}

// NewInfo 创建信息
func NewInfo(code, message string) *ServiceError {
	return NewServiceError(code, message, ErrorLevelInfo)
}

// WithService 设置服务名称
func (e *ServiceError) WithService(service string) *ServiceError {
	e.Service = service
	return e
}

// WithOperation 设置操作名称
func (e *ServiceError) WithOperation(operation string) *ServiceError {
	e.Operation = operation
	return e
}

// WithDetail 添加详情
func (e *ServiceError) WithDetail(key string, value interface{}) *ServiceError {
	e.Details[key] = value
	return e
}

// WithCause 设置原因
func (e *ServiceError) WithCause(cause error) *ServiceError {
	e.Cause = cause
	return e
}

// CommonErrorCodes 常见错误码
var CommonErrorCodes = struct {
	// 数据库相关错误
	DBConnectionError string
	DBQueryError      string
	DBNotFound        string
	DBDuplicateKey    string
	DBTransactionError string
	
	// 服务相关错误
	ServiceNotFound     string
	ServiceUnavailable  string
	ServiceTimeout      string
	ServiceRateLimit    string
	
	// 业务逻辑错误
	InvalidParameter string
	InvalidState     string
	OperationFailed  string
	ResourceNotFound string
	ResourceConflict string
	
	// 系统错误
	SystemError      string
	SystemBusy       string
	SystemMaintenance string
	
	// 外部依赖错误
	ExternalServiceError string
	NetworkError        string
	TimeoutError        string
}{
	// 数据库相关错误
	DBConnectionError:  "DB_CONNECTION_ERROR",
	DBQueryError:       "DB_QUERY_ERROR",
	DBNotFound:         "DB_NOT_FOUND",
	DBDuplicateKey:     "DB_DUPLICATE_KEY",
	DBTransactionError: "DB_TRANSACTION_ERROR",
	
	// 服务相关错误
	ServiceNotFound:    "SERVICE_NOT_FOUND",
	ServiceUnavailable: "SERVICE_UNAVAILABLE",
	ServiceTimeout:     "SERVICE_TIMEOUT",
	ServiceRateLimit:   "SERVICE_RATE_LIMIT",
	
	// 业务逻辑错误
	InvalidParameter: "INVALID_PARAMETER",
	InvalidState:     "INVALID_STATE",
	OperationFailed:  "OPERATION_FAILED",
	ResourceNotFound: "RESOURCE_NOT_FOUND",
	ResourceConflict: "RESOURCE_CONFLICT",
	
	// 系统错误
	SystemError:       "SYSTEM_ERROR",
	SystemBusy:        "SYSTEM_BUSY",
	SystemMaintenance: "SYSTEM_MAINTENANCE",
	
	// 外部依赖错误
	ExternalServiceError: "EXTERNAL_SERVICE_ERROR",
	NetworkError:         "NETWORK_ERROR",
	TimeoutError:         "TIMEOUT_ERROR",
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	serviceName string
	ctx         context.Context
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler(ctx context.Context, serviceName string) *ErrorHandler {
	return &ErrorHandler{
		serviceName: serviceName,
		ctx:         ctx,
	}
}

// HandleError 处理错误
func (h *ErrorHandler) HandleError(err error, operation string, details ...map[string]interface{}) {
	if err == nil {
		return
	}

	// 如果是 ServiceError，直接处理
	if serviceErr, ok := err.(*ServiceError); ok {
		h.logServiceError(serviceErr)
		return
	}

	// 转换普通错误为 ServiceError
	serviceErr := h.convertToServiceError(err, operation)
	if len(details) > 0 {
		for k, v := range details[0] {
			serviceErr.WithDetail(k, v)
		}
	}

	h.logServiceError(serviceErr)
}

// HandlePanic 处理 panic
func (h *ErrorHandler) HandlePanic(recover interface{}, operation string) *ServiceError {
	err := NewCriticalError(CommonErrorCodes.SystemError, fmt.Sprintf("panic in %s: %v", operation, recover))
	err.WithService(h.serviceName).WithOperation(operation)
	err.StackTrace = string(debug.Stack())
	
	logger.Errorf("🚨 服务发生严重错误: %v\n堆栈跟踪:\n%s", err, err.StackTrace)
	
	return err
}

// convertToServiceError 转换普通错误为 ServiceError
func (h *ErrorHandler) convertToServiceError(err error, operation string) *ServiceError {
	errMsg := err.Error()
	
	// 根据错误信息判断错误类型
	var serviceErr *ServiceError
	
	switch {
	case contains(errMsg, "connection", "timeout"):
		serviceErr = NewError(CommonErrorCodes.NetworkError, "网络连接错误")
	case contains(errMsg, "not found"):
		serviceErr = NewError(CommonErrorCodes.ResourceNotFound, "资源不存在")
	case contains(errMsg, "duplicate", "conflict"):
		serviceErr = NewError(CommonErrorCodes.ResourceConflict, "资源冲突")
	case contains(errMsg, "invalid", "validation"):
		serviceErr = NewError(CommonErrorCodes.InvalidParameter, "参数无效")
	default:
		serviceErr = NewError(CommonErrorCodes.OperationFailed, "操作失败")
	}
	
	return serviceErr.WithService(h.serviceName).WithOperation(operation).WithCause(err)
}

// logServiceError 记录服务错误
func (h *ErrorHandler) logServiceError(err *ServiceError) {
	errorMsg := err.Error()
	
	// 根据错误级别选择不同的日志级别
	switch err.Level {
	case ErrorLevelCritical:
		logger.Errorf("🚨 %s", errorMsg)
	case ErrorLevelError:
		logger.Errorf("❌ %s", errorMsg)
	case ErrorLevelWarning:
		logger.Warnf("⚠️ %s", errorMsg)
	case ErrorLevelInfo:
		logger.Infof("ℹ️ %s", errorMsg)
	}
	
	// 记录详细错误信息
	if len(err.Details) > 0 {
		logger.Errorf("错误详情: %+v", err.Details)
	}
	
	// 记录堆栈跟踪（如果有）
	if err.StackTrace != "" {
		logger.Errorf("堆栈跟踪:\n%s", err.StackTrace)
	}
}

// contains 检查字符串是否包含指定关键词
func contains(s string, substrs ...string) bool {
	sLower := toLower(s)
	for _, substr := range substrs {
		if containsIgnoreCase(sLower, toLower(substr)) {
			return true
		}
	}
	return false
}

// toLower 转换为小写
func toLower(s string) string {
	// 简单的ASCII小写转换
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// containsIgnoreCase 忽略大小写包含检查
func containsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// WrapError 包装错误
func WrapError(ctx context.Context, err error, service, operation string, details ...map[string]interface{}) *ServiceError {
	if err == nil {
		return nil
	}
	
	handler := NewErrorHandler(ctx, service)
	serviceErr := handler.convertToServiceError(err, operation)
	
	if len(details) > 0 {
		for k, v := range details[0] {
			serviceErr.WithDetail(k, v)
		}
	}
	
	return serviceErr
}