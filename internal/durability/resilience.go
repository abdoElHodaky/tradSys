package durability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultRetryConfig provides sensible defaults for trading operations
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitBreakerState
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	failureThreshold  int
	recoveryTimeout   time.Duration
	halfOpenMaxCalls  int
	logger            *zap.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration, logger *zap.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		state:             StateClosed,
		failureThreshold:  failureThreshold,
		recoveryTimeout:   recoveryTimeout,
		halfOpenMaxCalls:  3,
		logger:            logger,
	}
}

// Execute runs the operation with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	if !cb.canExecute() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := operation()
	cb.recordResult(err == nil)
	return err
}

func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.recoveryTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return cb.successCount < cb.halfOpenMaxCalls
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.successCount++
		if cb.state == StateHalfOpen && cb.successCount >= cb.halfOpenMaxCalls {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.logger.Info("Circuit breaker closed after successful recovery")
		}
	} else {
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		
		if cb.state == StateClosed && cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.logger.Warn("Circuit breaker opened due to failures",
				zap.Int("failure_count", cb.failureCount),
				zap.Int("threshold", cb.failureThreshold))
		} else if cb.state == StateHalfOpen {
			cb.state = StateOpen
			cb.logger.Warn("Circuit breaker reopened during half-open state")
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// RetryWithBackoff executes an operation with exponential backoff retry
func RetryWithBackoff(ctx context.Context, config RetryConfig, operation func() error, logger *zap.Logger) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			if attempt > 1 {
				logger.Info("Operation succeeded after retry",
					zap.Int("attempt", attempt),
					zap.Duration("total_delay", delay))
			}
			return nil
		}

		lastErr = err
		if attempt == config.MaxAttempts {
			break
		}

		logger.Warn("Operation failed, retrying",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", config.MaxAttempts),
			zap.Duration("delay", delay),
			zap.Error(err))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.BackoffFactor)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// TimeoutWrapper wraps an operation with a timeout
func TimeoutWrapper(ctx context.Context, timeout time.Duration, operation func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- operation(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
