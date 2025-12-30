package ai

import (
	"sync"
	"time"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	failureThreshold int
	successThreshold int
	timeout          time.Duration

	state            CircuitBreakerState
	failures         int
	successes        int
	lastFailureTime  time.Time
	mu               sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            StateClosed,
	}
}

// CanRequest 是否允许请求
func (cb *CircuitBreaker) CanRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// 如果是开启状态，检查是否超时
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// 超时后进入半开状态
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	}

	return true
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = StateClosed
		}
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// GetState 获取状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}