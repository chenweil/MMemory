package ai

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAIError_Error 测试AIError的Error方法
func TestAIError_Error(t *testing.T) {
	t.Run("带原因的错误", func(t *testing.T) {
		cause := errors.New("original error")
		err := &AIError{
			Type:    ErrorTypeAPI,
			Message: "API call failed",
			Cause:   cause,
		}

		expected := "API_ERROR: API call failed (caused by: original error)"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("不带原因的错误", func(t *testing.T) {
		err := &AIError{
			Type:    ErrorTypeConfig,
			Message: "Invalid configuration",
			Cause:   nil,
		}

		expected := "CONFIG_ERROR: Invalid configuration"
		assert.Equal(t, expected, err.Error())
	})
}

// TestAIError_Unwrap 测试AIError的Unwrap方法
func TestAIError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &AIError{
		Type:    ErrorTypeNetwork,
		Message: "Network error occurred",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)
	assert.True(t, errors.Is(err, cause))
}

// TestNewAIError 测试NewAIError构造函数
func TestNewAIError(t *testing.T) {
	t.Run("创建新错误", func(t *testing.T) {
		cause := fmt.Errorf("test error")
		err := NewAIError(ErrorTypeValidation, "validation failed", cause)

		assert.NotNil(t, err)
		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "validation failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("创建不带原因的错误", func(t *testing.T) {
		err := NewAIError(ErrorTypeTimeout, "operation timed out", nil)

		assert.NotNil(t, err)
		assert.Equal(t, ErrorTypeTimeout, err.Type)
		assert.Equal(t, "operation timed out", err.Message)
		assert.Nil(t, err.Cause)
	})
}

// TestIsRetryableError 测试可重试错误检查
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "API超时错误-可重试",
			err:      ErrAPITimeout,
			expected: true,
		},
		{
			name:     "API限流错误-可重试",
			err:      ErrAPIRateLimit,
			expected: true,
		},
		{
			name:     "网络连接拒绝-可重试",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "网络超时-可重试",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "认证错误-不可重试",
			err:      ErrAPIAuth,
			expected: false,
		},
		{
			name:     "配置错误-不可重试",
			err:      ErrMissingAPIKey,
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsNetworkError 测试网络错误检查
func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection timeout",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "network unreachable",
			err:      errors.New("network unreachable"),
			expected: true,
		},
		{
			name:     "temporary failure",
			err:      errors.New("temporary failure in name resolution"),
			expected: true,
		},
		{
			name:     "dial tcp error",
			err:      errors.New("dial tcp: i/o timeout"),
			expected: true,
		},
		{
			name:     "非网络错误",
			err:      errors.New("invalid argument"),
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNetworkError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsAuthError 测试认证错误检查
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "认证错误",
			err:      ErrAPIAuth,
			expected: true,
		},
		{
			name:     "包装的认证错误",
			err:      fmt.Errorf("wrapped: %w", ErrAPIAuth),
			expected: true,
		},
		{
			name:     "其他错误",
			err:      ErrAPITimeout,
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsRateLimitError 测试限流错误检查
func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "限流错误",
			err:      ErrAPIRateLimit,
			expected: true,
		},
		{
			name:     "包装的限流错误",
			err:      fmt.Errorf("wrapped: %w", ErrAPIRateLimit),
			expected: true,
		},
		{
			name:     "其他错误",
			err:      ErrAPIAuth,
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRateLimitError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsModelError 测试模型错误检查
func TestIsModelError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "模型不存在错误",
			err:      ErrAPIModelNotFound,
			expected: true,
		},
		{
			name:     "包装的模型错误",
			err:      fmt.Errorf("wrapped: %w", ErrAPIModelNotFound),
			expected: true,
		},
		{
			name:     "其他错误",
			err:      ErrAPIAuth,
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsModelError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestContains 测试contains辅助函数
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "完全匹配",
			s:        "connection refused",
			substr:   "connection refused",
			expected: true,
		},
		{
			name:     "前缀匹配",
			s:        "connection refused error",
			substr:   "connection",
			expected: true,
		},
		{
			name:     "后缀匹配",
			s:        "tcp connection refused",
			substr:   "refused",
			expected: true,
		},
		{
			name:     "中间匹配",
			s:        "error: connection timeout occurred",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "不包含",
			s:        "some error",
			substr:   "network",
			expected: false,
		},
		{
			name:     "空字符串",
			s:        "test",
			substr:   "",
			expected: true,
		},
		{
			name:     "子串更长",
			s:        "abc",
			substr:   "abcdef",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
