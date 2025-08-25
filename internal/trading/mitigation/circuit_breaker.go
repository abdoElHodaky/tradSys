package mitigation

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// CircuitClosed means the circuit is closed and requests are allowed
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen means the circuit is open and requests are not allowed
	CircuitOpen
	// CircuitHalfOpen means the circuit is half-open and a limited number of requests are allowed
	CircuitHalfOpen
)

// CircuitBreakerConfig represents the configuration for a circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures that triggers the circuit to open
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open state that closes the circuit
	SuccessThreshold int
	// Timeout is the duration the circuit stays open before transitioning to half-open
	Timeout time.Duration
	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state
	HalfOpenMaxRequests int
}

// DefaultCircuitBreakerConfig returns a default configuration for a circuit breaker
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    3,
		Timeout:             10 * time.Second,
		HalfOpenMaxRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name              string
	state             CircuitBreakerState
	config            CircuitBreakerConfig
	failures          int
	successes         int
	halfOpenRequests  int
	lastStateChange   time.Time
	mutex             sync.RWMutex
	logger            *zap.Logger
	stateChangeHooks  []func(name string, from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a new circuit breaker with the given name and configuration
func NewCircuitBreaker(name string, config CircuitBreakerConfig, logger *zap.Logger) *CircuitBreaker {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &CircuitBreaker{
		name:            name,
		state:           CircuitClosed,
		config:          config,
		failures:        0,
		successes:       0,
		halfOpenRequests: 0,
		lastStateChange: time.Now(),
		logger:          logger.With(zap.String("component", "circuit_breaker"), zap.String("name", name)),
		stateChangeHooks: []func(name string, from, to CircuitBreakerState){},
	}
}

// AddStateChangeHook adds a hook that is called when the circuit breaker changes state
func (cb *CircuitBreaker) AddStateChangeHook(hook func(name string, from, to CircuitBreakerState)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.stateChangeHooks = append(cb.stateChangeHooks, hook)
}

// changeState changes the state of the circuit breaker
func (cb *CircuitBreaker) changeState(newState CircuitBreakerState) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Reset counters on state change
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0

	cb.logger.Info("Circuit breaker state changed",
		zap.String("from", stateToString(oldState)),
		zap.String("to", stateToString(newState)))

	// Call state change hooks
	for _, hook := range cb.stateChangeHooks {
		go hook(cb.name, oldState, newState)
	}
}

// stateToString converts a CircuitBreakerState to a string
func stateToString(state CircuitBreakerState) string {
	switch state {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	// Check if we should transition from open to half-open
	if cb.state == CircuitOpen && time.Since(cb.lastStateChange) > cb.config.Timeout {
		// Release the read lock and acquire a write lock
		cb.mutex.RUnlock()
		cb.changeState(CircuitHalfOpen)
		cb.mutex.RLock()
	}

	return cb.state
}

// Allow checks if a request should be allowed through the circuit breaker
func (cb *CircuitBreaker) Allow() bool {
	state := cb.State()

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		return false
	case CircuitHalfOpen:
		if cb.halfOpenRequests < cb.config.HalfOpenMaxRequests {
			cb.halfOpenRequests++
			return true
		}
		return false
	default:
		return false
	}
}

// Success records a successful operation
func (cb *CircuitBreaker) Success() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitClosed:
		// Reset failures on success in closed state
		cb.failures = 0
	case CircuitHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			// Release the lock before changing state
			cb.mutex.Unlock()
			cb.changeState(CircuitClosed)
			// Reacquire the lock
			cb.mutex.Lock()
		}
	}
}

// Failure records a failed operation
func (cb *CircuitBreaker) Failure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			// Release the lock before changing state
			cb.mutex.Unlock()
			cb.changeState(CircuitOpen)
			// Reacquire the lock
			cb.mutex.Lock()
		}
	case CircuitHalfOpen:
		// Any failure in half-open state opens the circuit again
		cb.mutex.Unlock()
		cb.changeState(CircuitOpen)
		cb.mutex.Lock()
	}
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if !cb.Allow() {
		return ErrCircuitOpen
	}

	err := fn(ctx)
	if err != nil {
		cb.Failure()
		return err
	}

	cb.Success()
	return nil
}

// Reset resets the circuit breaker to its initial closed state
func (cb *CircuitBreaker) Reset() {
	cb.changeState(CircuitClosed)
}

// ErrCircuitOpen is returned when the circuit is open
var ErrCircuitOpen = CircuitError{message: "circuit breaker is open"}

// CircuitError represents a circuit breaker error
type CircuitError struct {
	message string
}

// Error returns the error message
func (e CircuitError) Error() string {
	return e.message
}

