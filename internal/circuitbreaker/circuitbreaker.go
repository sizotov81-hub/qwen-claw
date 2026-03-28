// Package circuitbreaker предоставляет Circuit Breaker для микросервисов
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State состояние Circuit Breaker
type State int

const (
	StateClosed State = iota // Нормальное состояние
	StateOpen                // Цепь разомкнута (ошибки)
	StateHalfOpen            // Полуоткрытое состояние (проверка)
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config конфигурация Circuit Breaker
type Config struct {
	FailureThreshold int           // Порог ошибок для размыкания
	SuccessThreshold int           // Порог успехов для замыкания
	Timeout          time.Duration // Таймаут в открытом состоянии
	HalfOpenMaxCalls int           // Максимум вызовов в полуоткрытом состоянии
}

// DefaultConfig конфигурация по умолчанию
func DefaultConfig() *Config {
	return &Config{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// CircuitBreaker Circuit Breaker
type CircuitBreaker struct {
	mu sync.RWMutex

	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	halfOpenCalls   int

	config *Config

	onStateChange func(State, State) // Callback при смене состояния
}

// NewCircuitBreaker создаёт новый Circuit Breaker
func NewCircuitBreaker(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}

	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
	}
}

// OnStateChange устанавливает callback при смене состояния
func (cb *CircuitBreaker) OnStateChange(fn func(oldState, newState State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// Execute выполняет функцию с Circuit Breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if err := cb.CanExecute(); err != nil {
		return err
	}

	err := fn()
	cb.RecordResult(err == nil)

	return err
}

// CanExecute проверяет можно ли выполнить вызов
func (cb *CircuitBreaker) CanExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Проверяем не истёк ли таймаут
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.halfOpenCalls = 1 // Первый вызов в HalfOpen
			return nil
		}
		return errors.New("circuit breaker is open")

	case StateHalfOpen:
		// Проверяем лимит вызовов
		if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			return errors.New("circuit breaker half-open limit reached")
		}
		cb.halfOpenCalls++
		return nil

	default:
		return errors.New("unknown circuit breaker state")
	}
}

// RecordResult записывает результат вызова
func (cb *CircuitBreaker) RecordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess обрабатывает успешный вызов
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.changeState(StateClosed)
			cb.successCount = 0
			cb.failureCount = 0
		}
	}
}

// onFailure обрабатывает неудачный вызов
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.changeState(StateOpen)
		}

	case StateHalfOpen:
		cb.changeState(StateOpen)
	}
}

// changeState изменяет состояние
func (cb *CircuitBreaker) changeState(newState State) {
	oldState := cb.state
	cb.state = newState

	if cb.onStateChange != nil {
		cb.onStateChange(oldState, newState)
	}
}

// State возвращает текущее состояние
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// StateString возвращает строковое представление состояния
func (cb *CircuitBreaker) StateString() string {
	return cb.State().String()
}

// FailureCount возвращает счётчик ошибок
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// SuccessCount возвращает счётчик успехов
func (cb *CircuitBreaker) SuccessCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.successCount
}

// Reset сбрасывает Circuit Breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.changeState(StateClosed)
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenCalls = 0
}

// IsOpen проверяет разомкнута ли цепь
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.State() == StateOpen
}

// IsClosed проверяет замкнута ли цепь
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.State() == StateClosed
}

// IsHalfOpen проверяет полуоткрыто ли состояние
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.State() == StateHalfOpen
}

// HalfOpenCalls возвращает счётчик вызовов в HalfOpen состоянии
func (cb *CircuitBreaker) HalfOpenCalls() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.halfOpenCalls
}

// Stats статистика Circuit Breaker
type Stats struct {
	State           string
	FailureCount    int
	SuccessCount    int
	HalfOpenCalls   int
	LastFailureTime time.Time
}

// GetStats возвращает статистику
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:           cb.state.String(),
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		HalfOpenCalls:   cb.halfOpenCalls,
		LastFailureTime: cb.lastFailureTime,
	}
}
