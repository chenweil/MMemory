package ai

import (
	"errors"
	"fmt"
)

// AI相关错误定义
var (
	// 配置错误
	ErrMissingAPIKey       = errors.New("openai api key is required")
	ErrMissingPrimaryModel = errors.New("primary model is required")
	ErrInvalidMaxTokens    = errors.New("max tokens must be greater than 0")
	ErrInvalidTemperature  = errors.New("temperature must be between 0 and 2")

	// API调用错误
	ErrAPITimeout        = errors.New("openai api timeout")
	ErrAPIRateLimit      = errors.New("openai api rate limit exceeded")
	ErrAPIAuth           = errors.New("openai api authentication failed")
	ErrAPIQuotaExceeded  = errors.New("openai api quota exceeded")
	ErrAPIModelNotFound  = errors.New("openai model not found")
	ErrAPIInvalidRequest = errors.New("openai api invalid request")

	// 解析错误
	ErrInvalidJSONResponse = errors.New("invalid json response from ai")
	ErrMissingRequiredField = errors.New("missing required field in ai response")
	ErrInvalidIntent       = errors.New("invalid intent in ai response")
	ErrLowConfidence       = errors.New("ai response confidence too low")

	// 降级错误
	ErrAllParsersFailed = errors.New("all parsers failed to parse message")
	ErrNoFallbackAvailable = errors.New("no fallback parser available")
)

// AIError AI错误类型
type AIError struct {
	Type    string
	Message string
	Cause   error
}

func (e *AIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AIError) Unwrap() error {
	return e.Cause
}

// NewAIError 创建AI错误
func NewAIError(errorType, message string, cause error) *AIError {
	return &AIError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// 错误类型常量
const (
	ErrorTypeConfig     = "CONFIG_ERROR"
	ErrorTypeAPI        = "API_ERROR"
	ErrorTypeParsing    = "PARSING_ERROR"
	ErrorTypeFallback   = "FALLBACK_ERROR"
	ErrorTypeValidation = "VALIDATION_ERROR"
	ErrorTypeTimeout    = "TIMEOUT_ERROR"
	ErrorTypeNetwork    = "NETWORK_ERROR"
)

// IsRetryableError 检查是否为可重试的错误
func IsRetryableError(err error) bool {
	switch {
	case errors.Is(err, ErrAPITimeout):
		return true
	case errors.Is(err, ErrAPIRateLimit):
		return true
	case IsNetworkError(err):
		return true
	default:
		return false
	}
}

// IsNetworkError 检查是否为网络错误
func IsNetworkError(err error) bool {
	// 简化实现，实际使用时可以更详细地检查网络错误类型
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	networkErrors := []string{
		"connection refused",
		"connection timeout",
		"network unreachable",
		"temporary failure",
		"dial tcp",
	}
	
	for _, netErr := range networkErrors {
		if contains(errStr, netErr) {
			return true
		}
	}
	
	return false
}

// IsAuthError 检查是否为认证错误
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAPIAuth)
}

// IsRateLimitError 检查是否为限流错误
func IsRateLimitError(err error) bool {
	return errors.Is(err, ErrAPIRateLimit)
}

// IsModelError 检查是否为模型错误
func IsModelError(err error) bool {
	return errors.Is(err, ErrAPIModelNotFound)
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 len(s) > len(substr)*2 && s[len(s)/2-len(substr)/2:len(s)/2+len(substr)/2+len(substr)%2] == substr))
}