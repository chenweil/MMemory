package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCircuitBreaker_New 测试创建熔断器
func TestCircuitBreaker_New(t *testing.T) {
	cb := NewCircuitBreaker(5, 3, 10*time.Second)

	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 5, cb.failureThreshold)
	assert.Equal(t, 3, cb.successThreshold)
	assert.Equal(t, 10*time.Second, cb.timeout)
}

// TestCircuitBreaker_CanRequest_ClosedState 测试关闭状态允许请求
func TestCircuitBreaker_CanRequest_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(5, 3, 10*time.Second)

	assert.True(t, cb.CanRequest(), "关闭状态应该允许请求")
}

// TestCircuitBreaker_RecordSuccess 测试记录成功
func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(5, 3, 10*time.Second)

	// 初始状态
	assert.Equal(t, StateClosed, cb.GetState())

	// 记录成功
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.failures)
}

// TestCircuitBreaker_RecordFailure 测试记录失败
func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 10*time.Second)

	// 记录失败，未达到阈值
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 1, cb.failures)

	// 继续记录失败
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())
	assert.Equal(t, 3, cb.failures)
}

// TestCircuitBreaker_CanRequest_OpenState 测试开启状态拒绝请求
func TestCircuitBreaker_CanRequest_OpenState(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 1*time.Second)

	// 触发熔断
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())

	// 开启状态应该拒绝请求
	assert.False(t, cb.CanRequest(), "开启状态应该拒绝请求")
}

// TestCircuitBreaker_CanRequest_Timeout 测试超时后进入半开状态
func TestCircuitBreaker_CanRequest_Timeout(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 100*time.Millisecond)

	// 触发熔断
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	// 超时后应该允许请求（进入半开状态）
	assert.True(t, cb.CanRequest(), "超时后应该允许请求")
	assert.Equal(t, StateHalfOpen, cb.GetState())
}

// TestCircuitBreaker_HalfOpenRecovery 测试半开状态恢复
func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 100*time.Millisecond)

	// 触发熔断并等待超时
	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(150 * time.Millisecond)

	// 进入半开状态
	assert.True(t, cb.CanRequest())
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// 记录成功
	cb.RecordSuccess()
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// 再次成功，恢复关闭状态
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())
}

// TestCircuitBreaker_HalfOpenFailure 测试半开状态再次失败
func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 100*time.Millisecond)

	// 触发熔断并等待超时
	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(150 * time.Millisecond)

	// 进入半开状态
	assert.True(t, cb.CanRequest())
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// 半开状态下记录失败，回到开启状态
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())
}

// TestCircuitBreaker_ConcurrentAccess 测试并发访问安全
func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(100, 50, 10*time.Second)

	// 并发记录成功和失败
	done := make(chan bool)

	for i := 0; i < 50; i++ {
		go func() {
			cb.RecordSuccess()
			done <- true
		}()
		go func() {
			cb.RecordFailure()
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 不应该panic，状态应该一致
	assert.NotPanics(t, func() {
		cb.GetState()
		cb.CanRequest()
	})
}
