package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	require.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, "closed", cb.StateString())
	assert.Equal(t, 0, cb.FailureCount())
	assert.Equal(t, 0, cb.SuccessCount())
	assert.False(t, cb.IsOpen())
	assert.True(t, cb.IsClosed())
	assert.False(t, cb.IsHalfOpen())
}

func TestCircuitBreaker_CustomConfig(t *testing.T) {
	config := &Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          10 * time.Second,
		HalfOpenMaxCalls: 5,
	}

	cb := NewCircuitBreaker(config)

	require.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_Execute(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	// Успешное выполнение
	err := cb.Execute(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, 0, cb.FailureCount())

	// Выполнение с ошибкой
	err = cb.Execute(func() error { return errors.New("test error") })
	assert.Error(t, err)
	assert.Equal(t, 1, cb.FailureCount())
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	config := &Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 3,
	}

	cb := NewCircuitBreaker(config)

	// Начальное состояние: Closed
	assert.Equal(t, StateClosed, cb.State())

	// Две ошибки -> Open
	cb.RecordResult(false)
	cb.RecordResult(false)
	assert.Equal(t, StateOpen, cb.State())
	assert.True(t, cb.IsOpen())

	// Ждём таймаут
	time.Sleep(150 * time.Millisecond)

	// Попытка выполнения -> HalfOpen
	err := cb.CanExecute()
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())

	// Два успеха -> Closed
	cb.RecordResult(true)
	cb.RecordResult(true)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_OpenState(t *testing.T) {
	config := &Config{
		FailureThreshold: 1,
		Timeout:          1 * time.Second,
	}

	cb := NewCircuitBreaker(config)

	// Одна ошибка -> Open
	cb.RecordResult(false)
	assert.Equal(t, StateOpen, cb.State())

	// Выполнение запрещено
	err := cb.CanExecute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Execute тоже запрещён
	err = cb.Execute(func() error { return nil })
	assert.Error(t, err)
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	config := &Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	// Open
	cb.RecordResult(false)
	assert.Equal(t, StateOpen, cb.State())

	// Ждём таймаут
	time.Sleep(100 * time.Millisecond)

	// Первый вызов -> HalfOpen
	err := cb.CanExecute()
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())

	// Второй вызов разрешён (лимит 2)
	err = cb.CanExecute()
	assert.NoError(t, err)

	// Успех -> Closed
	cb.RecordResult(true)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_HalfOpenLimit(t *testing.T) {
	config := &Config{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	// Open
	cb.RecordResult(false)
	assert.Equal(t, StateOpen, cb.State())
	
	time.Sleep(100 * time.Millisecond)

	// Первый вызов -> переход в HalfOpen, halfOpenCalls=1
	err1 := cb.CanExecute()
	assert.NoError(t, err1)
	assert.Equal(t, StateHalfOpen, cb.State())
	t.Logf("After first call: state=%s, halfOpenCalls=%d", cb.State(), cb.HalfOpenCalls())
	
	// Второй вызов, halfOpenCalls=2
	err2 := cb.CanExecute()
	assert.NoError(t, err2)
	t.Logf("After second call: state=%s, halfOpenCalls=%d", cb.State(), cb.HalfOpenCalls())
	
	// Третий вызов запрещён (лимит 2)
	err3 := cb.CanExecute()
	t.Logf("Third call: err=%v, state=%s, halfOpenCalls=%d", err3, cb.State(), cb.HalfOpenCalls())
	assert.Error(t, err3)
}

func TestCircuitBreaker_Callback(t *testing.T) {
	cb := NewCircuitBreaker(&Config{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
	})

	stateChanges := make([]State, 0)
	cb.OnStateChange(func(oldState, newState State) {
		stateChanges = append(stateChanges, newState)
	})

	// Open
	cb.RecordResult(false)
	time.Sleep(100 * time.Millisecond)

	// HalfOpen
	cb.CanExecute()

	// Ждём немного
	time.Sleep(10 * time.Millisecond)

	assert.GreaterOrEqual(t, len(stateChanges), 1)
	assert.Contains(t, stateChanges, StateOpen)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(&Config{
		FailureThreshold: 1,
		Timeout:          1 * time.Second,
	})

	// Open
	cb.RecordResult(false)
	assert.Equal(t, StateOpen, cb.State())
	assert.Equal(t, 1, cb.FailureCount())

	// Reset
	cb.Reset()

	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, cb.FailureCount())
	assert.Equal(t, 0, cb.SuccessCount())
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	cb.RecordResult(false)
	// Не записываем успех чтобы не сбросить failureCount

	stats := cb.GetStats()

	assert.Equal(t, "closed", stats.State)
	assert.Equal(t, 1, stats.FailureCount)
	assert.False(t, stats.LastFailureTime.IsZero())
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func() {
			cb.State()
			cb.IsOpen()
			cb.IsClosed()
			cb.IsHalfOpen()
			cb.GetStats()
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 3, config.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.HalfOpenMaxCalls)
}
