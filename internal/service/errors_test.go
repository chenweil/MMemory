package service

import (
	"errors"
	"testing"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name        string
		err         *ServiceError
		wantContain string
	}{
		{
			name: "带Cause的错误",
			err: &ServiceError{
				Code:    "TEST_ERROR",
				Message: "test message",
				Level:   ErrorLevelError,
				Cause:   errors.New("underlying error"),
			},
			wantContain: "caused by",
		},
		{
			name: "不带Cause的错误",
			err: &ServiceError{
				Code:    "TEST_ERROR",
				Message: "test message",
				Level:   ErrorLevelError,
			},
			wantContain: "TEST_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got == "" {
				t.Error("Error() returned empty string")
			}
			if len(tt.wantContain) > 0 {
				contains := false
				for i := 0; i <= len(got)-len(tt.wantContain); i++ {
					if got[i:i+len(tt.wantContain)] == tt.wantContain {
						contains = true
						break
					}
				}
				if !contains {
					t.Errorf("Error() = %v, want to contain %v", got, tt.wantContain)
				}
			}
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &ServiceError{
		Code:    "TEST_ERROR",
		Message: "test",
		Level:   ErrorLevelError,
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// 测试 errors.Is
	if !errors.Is(err, cause) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestNewServiceError(t *testing.T) {
	code := "TEST_CODE"
	message := "test message"

	err := NewServiceError(code, message, ErrorLevelError)

	if err.Code != code {
		t.Errorf("Code = %v, want %v", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	if err.Level != ErrorLevelError {
		t.Errorf("Level = %v, want ERROR", err.Level)
	}
}

func TestNewCriticalError(t *testing.T) {
	err := NewCriticalError("CRITICAL_CODE", "critical message")
	if err.Level != ErrorLevelCritical {
		t.Errorf("Level = %v, want CRITICAL", err.Level)
	}
	if err.StackTrace == "" {
		t.Error("StackTrace should not be empty for critical errors")
	}
}

func TestNewError(t *testing.T) {
	err := NewError("ERROR_CODE", "error message")
	if err.Level != ErrorLevelError {
		t.Errorf("Level = %v, want ERROR", err.Level)
	}
}

func TestNewWarning(t *testing.T) {
	err := NewWarning("WARN_CODE", "warning message")
	if err.Level != ErrorLevelWarning {
		t.Errorf("Level = %v, want WARNING", err.Level)
	}
}

func TestNewInfo(t *testing.T) {
	err := NewInfo("INFO_CODE", "info message")
	if err.Level != ErrorLevelInfo {
		t.Errorf("Level = %v, want INFO", err.Level)
	}
}

func TestServiceError_WithMethods(t *testing.T) {
	t.Run("WithService", func(t *testing.T) {
		err := NewServiceError("CODE", "message", ErrorLevelError)
		err = err.WithService("TestService")
		if err.Service != "TestService" {
			t.Errorf("Service = %v, want TestService", err.Service)
		}
	})

	t.Run("WithOperation", func(t *testing.T) {
		err := NewServiceError("CODE", "message", ErrorLevelError)
		err = err.WithOperation("TestOperation")
		if err.Operation != "TestOperation" {
			t.Errorf("Operation = %v, want TestOperation", err.Operation)
		}
	})

	t.Run("WithDetail", func(t *testing.T) {
		err := NewServiceError("CODE", "message", ErrorLevelError)
		err = err.WithDetail("key", "value")
		if len(err.Details) == 0 {
			t.Error("Details should not be empty")
		}
		if err.Details["key"] != "value" {
			t.Errorf("Details[key] = %v, want value", err.Details["key"])
		}
	})

	t.Run("WithCause", func(t *testing.T) {
		err := NewServiceError("CODE", "message", ErrorLevelError)
		cause := errors.New("cause")
		err = err.WithCause(cause)
		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}
	})
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substrs  []string
		expected bool
	}{
		{
			name:     "包含单个关键词",
			s:        "connection timeout error",
			substrs:  []string{"timeout"},
			expected: true,
		},
		{
			name:     "包含多个关键词之一",
			s:        "connection timeout error",
			substrs:  []string{"network", "timeout"},
			expected: true,
		},
		{
			name:     "不包含关键词",
			s:        "some error",
			substrs:  []string{"network"},
			expected: false,
		},
		{
			name:     "大小写不敏感",
			s:        "Connection Timeout Error",
			substrs:  []string{"timeout"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substrs...)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC", "abc"},
		{"Hello", "hello"},
		{"WORLD", "world"},
		{"123", "123"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toLower(tt.input)
		if result != tt.expected {
			t.Errorf("toLower(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "hello", true},
		{"hello world", "world", true},
		{"hello world", "xyz", false},
		{"hello world", "hello world!", false}, // substr longer than s
		{"", "", true},
	}

	for _, tt := range tests {
		result := containsIgnoreCase(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("containsIgnoreCase(%v, %v) = %v, want %v",
				tt.s, tt.substr, result, tt.expected)
		}
	}
}
