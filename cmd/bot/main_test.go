package main

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	config := DefaultRetryConfig()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"第一次重试", 1, 1 * time.Second},
		{"第二次重试", 2, 2 * time.Second},
		{"第三次重试", 3, 4 * time.Second},
		{"第四次重试", 4, 8 * time.Second},
		{"第五次重试", 5, 16 * time.Second},
		{"第六次重试", 6, 32 * time.Second},
		{"第七次重试", 7, 60 * time.Second}, // 达到最大值
		{"第八次重试", 8, 60 * time.Second}, // 保持最大值
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBackoff(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"超时错误", &timeoutError{}, true},
		{"EOF错误", &eofError{}, true},
		{"连接重置错误", &connectionResetError{}, true},
		{"致命错误", &fatalError{}, false},
		{"无错误", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTransientError(tt.err)
			if result != tt.expected {
				t.Errorf("isTransientError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsFatalError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"未授权错误", &unauthorizedError{}, true},
		{"403错误", &forbiddenError{}, true},
		{"404错误", &notFoundError{}, true},
		{"临时错误", &timeoutError{}, false},
		{"无错误", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFatalError(tt.err)
			if result != tt.expected {
				t.Errorf("isFatalError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", config.MaxRetries)
	}
	if config.InitialDelay != 1*time.Second {
		t.Errorf("InitialDelay = %v, want 1s", config.InitialDelay)
	}
	if config.MaxDelay != 60*time.Second {
		t.Errorf("MaxDelay = %v, want 60s", config.MaxDelay)
	}
	if config.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor = %f, want 2.0", config.BackoffFactor)
	}
}

// 辅助错误类型用于测试
type timeoutError struct{}

func (e *timeoutError) Error() string { return "timeout" }

type eofError struct{}

func (e *eofError) Error() string { return "unexpected EOF" }

type connectionResetError struct{}

func (e *connectionResetError) Error() string { return "connection reset" }

type fatalError struct{}

func (e *fatalError) Error() string { return "some fatal error" }

type unauthorizedError struct{}

func (e *unauthorizedError) Error() string { return "Unauthorized" }

type forbiddenError struct{}

func (e *forbiddenError) Error() string { return "403" }

type notFoundError struct{}

func (e *notFoundError) Error() string { return "404" }