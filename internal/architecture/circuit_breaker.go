package architecture

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int32

const (
	// CircuitClosed indicates the circuit is closed and requests are allowed
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen indicates the circuit is open and requests are not allowed
	CircuitOpen
	// CircuitHalfOpen indicates the circuit is half-open and testing if requests can be allowed
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern to prevent
// cascading failures in distributed systems
type CircuitBreaker struct {
	name                string
	state               int32 // atomic
	failureCount        int64 // atomic
	successCount        int64 // atomic
	lastFailure         int64 // atomic, unix timestamp
	failureThreshold    int64
	resetTimeout        time.Duration
	halfOpenMaxRequests int64
	mu                  sync.RWMutex
	onStateChange       func(name string, from, to CircuitBreakerState)
}

// CircuitBreakerOptions contains options for creating a circuit breaker
type CircuitBreakerOptions struct {
	Name                string
	FailureThreshold    int64
	ResetTimeout        time.Duration
	HalfOpenMaxRequests int64
	OnStateChange       func(name string, from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(options CircuitBreakerOptions) *CircuitBreaker {
	if options.FailureThreshold <= 0 {
		options.FailureThreshold = 5
	}
	if options.ResetTimeout <= 0 {
		options.ResetTimeout = 30 * time.Second
	}
	if options.HalfOpenMaxRequests <= 0 {
		options.HalfOpenMaxRequests = 1
	}
	if options.OnStateChange == nil {
		options.OnStateChange = func(name string, from, to CircuitBreakerState) {}
	}

	return &CircuitBreaker{
		name:                options.Name,
		state:               int32(CircuitClosed),
		failureThreshold:    options.FailureThreshold,
		resetTimeout:        options.ResetTimeout,
		halfOpenMaxRequests: options.HalfOpenMaxRequests,
		onStateChange:       options.OnStateChange,
	}
}

// Execute executes the given function if the circuit is closed or half-open
// and records the result
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return errors.New("circuit breaker is open")
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// allowRequest checks if a request should be allowed based on the circuit state
func (cb *CircuitBreaker) allowRequest() bool {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if the reset timeout has elapsed
		lastFailure := time.Unix(atomic.LoadInt64(&cb.lastFailure), 0)
		if time.Since(lastFailure) > cb.resetTimeout {
			// Transition to half-open state
			if cb.tryTransitionState(CircuitOpen, CircuitHalfOpen) {
				atomic.StoreInt64(&cb.successCount, 0)
				return true
			}
		}
		return false
	case CircuitHalfOpen:
		// Allow a limited number of requests in half-open state
		return atomic.LoadInt64(&cb.successCount) < cb.halfOpenMaxRequests
	default:
		return false
	}
}

// recordSuccess records a successful request
func (cb *CircuitBreaker) recordSuccess() {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case CircuitClosed:
		// Nothing to do in closed state
	case CircuitHalfOpen:
		newCount := atomic.AddInt64(&cb.successCount, 1)
		if newCount >= cb.halfOpenMaxRequests {
			// Transition back to closed state after enough successes
			cb.tryTransitionState(CircuitHalfOpen, CircuitClosed)
			atomic.StoreInt64(&cb.failureCount, 0)
		}
	}
}

// recordFailure records a failed request
func (cb *CircuitBreaker) recordFailure() {
	atomic.StoreInt64(&cb.lastFailure, time.Now().Unix())

	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case CircuitClosed:
		newCount := atomic.AddInt64(&cb.failureCount, 1)
		if newCount >= cb.failureThreshold {
			// Transition to open state after too many failures
			cb.tryTransitionState(CircuitClosed, CircuitOpen)
		}
	case CircuitHalfOpen:
		// Any failure in half-open state should reopen the circuit
		cb.tryTransitionState(CircuitHalfOpen, CircuitOpen)
	}
}

// tryTransitionState attempts to transition the circuit from one state to another
func (cb *CircuitBreaker) tryTransitionState(from, to CircuitBreakerState) bool {
	if atomic.CompareAndSwapInt32(&cb.state, int32(from), int32(to)) {
		cb.onStateChange(cb.name, from, to)
		return true
	}
	return false
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	return CircuitBreakerState(atomic.LoadInt32(&cb.state))
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	oldState := CircuitBreakerState(atomic.SwapInt32(&cb.state, int32(CircuitClosed)))
	if oldState != CircuitClosed {
		cb.onStateChange(cb.name, oldState, CircuitClosed)
	}
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt64(&cb.successCount, 0)
}
