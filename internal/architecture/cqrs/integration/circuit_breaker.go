package integration

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/eventbus"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"go.uber.org/zap"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// ClosedState indicates the circuit is closed and operating normally
	ClosedState CircuitBreakerState = iota
	
	// OpenState indicates the circuit is open and failing fast
	OpenState
	
	// HalfOpenState indicates the circuit is half-open and testing if it can close
	HalfOpenState
)

// CircuitBreaker provides circuit breaking functionality for event buses
type CircuitBreaker struct {
	logger *zap.Logger
	
	// Configuration
	failureThreshold   int
	resetTimeout       time.Duration
	halfOpenMaxCalls   int
	
	// State
	state              CircuitBreakerState
	failures           int
	halfOpenSuccesses  int
	halfOpenFailures   int
	lastStateChange    time.Time
	
	// Statistics
	totalCalls         int64
	successfulCalls    int64
	failedCalls        int64
	shortCircuitedCalls int64
	
	// Synchronization
	mu                 sync.RWMutex
}

// CircuitBreakerConfig contains configuration for the CircuitBreaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int
	
	// ResetTimeout is the time to wait before attempting to close the circuit
	ResetTimeout time.Duration
	
	// HalfOpenMaxCalls is the maximum number of calls to allow in half-open state
	HalfOpenMaxCalls int
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     10 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(logger *zap.Logger, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		logger:             logger,
		failureThreshold:   config.FailureThreshold,
		resetTimeout:       config.ResetTimeout,
		halfOpenMaxCalls:   config.HalfOpenMaxCalls,
		state:              ClosedState,
		failures:           0,
		halfOpenSuccesses:  0,
		halfOpenFailures:   0,
		lastStateChange:    time.Now(),
		totalCalls:         0,
		successfulCalls:    0,
		failedCalls:        0,
		shortCircuitedCalls: 0,
	}
}

// Execute executes a function with circuit breaking
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	cb.mu.Lock()
	
	// Update statistics
	cb.totalCalls++
	
	// Check if the circuit is open
	if cb.state == OpenState {
		// Check if it's time to transition to half-open
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			cb.transitionToHalfOpen()
		} else {
			// Circuit is still open, fail fast
			cb.shortCircuitedCalls++
			cb.mu.Unlock()
			return errors.New("circuit breaker is open")
		}
	}
	
	// If we're in half-open state, check if we've reached the maximum calls
	if cb.state == HalfOpenState {
		if cb.halfOpenSuccesses+cb.halfOpenFailures >= cb.halfOpenMaxCalls {
			// We've reached the maximum calls, decide whether to close or open
			if cb.halfOpenFailures > 0 {
				// We had failures, keep the circuit open
				cb.transitionToOpen()
				cb.shortCircuitedCalls++
				cb.mu.Unlock()
				return errors.New("circuit breaker is open")
			} else {
				// All calls were successful, close the circuit
				cb.transitionToClosed()
			}
		}
	}
	
	// At this point, we're either in closed state or half-open state with capacity
	cb.mu.Unlock()
	
	// Execute the function
	err := fn(ctx)
	
	// Update the circuit state based on the result
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if err != nil {
		// The call failed
		cb.failedCalls++
		
		if cb.state == ClosedState {
			// Increment the failure counter
			cb.failures++
			
			// Check if we've reached the failure threshold
			if cb.failures >= cb.failureThreshold {
				cb.transitionToOpen()
			}
		} else if cb.state == HalfOpenState {
			// Increment the half-open failure counter
			cb.halfOpenFailures++
			
			// Check if we've reached the maximum calls
			if cb.halfOpenSuccesses+cb.halfOpenFailures >= cb.halfOpenMaxCalls {
				// We've reached the maximum calls, decide whether to close or open
				if cb.halfOpenFailures > 0 {
					// We had failures, keep the circuit open
					cb.transitionToOpen()
				} else {
					// All calls were successful, close the circuit
					cb.transitionToClosed()
				}
			}
		}
	} else {
		// The call succeeded
		cb.successfulCalls++
		
		if cb.state == ClosedState {
			// Reset the failure counter
			cb.failures = 0
		} else if cb.state == HalfOpenState {
			// Increment the half-open success counter
			cb.halfOpenSuccesses++
			
			// Check if we've reached the maximum calls
			if cb.halfOpenSuccesses+cb.halfOpenFailures >= cb.halfOpenMaxCalls {
				// We've reached the maximum calls, decide whether to close or open
				if cb.halfOpenFailures > 0 {
					// We had failures, keep the circuit open
					cb.transitionToOpen()
				} else {
					// All calls were successful, close the circuit
					cb.transitionToClosed()
				}
			}
		}
	}
	
	return err
}

// transitionToOpen transitions the circuit to the open state
func (cb *CircuitBreaker) transitionToOpen() {
	cb.state = OpenState
	cb.lastStateChange = time.Now()
	cb.logger.Warn("Circuit breaker transitioned to open state")
}

// transitionToHalfOpen transitions the circuit to the half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.state = HalfOpenState
	cb.lastStateChange = time.Now()
	cb.halfOpenSuccesses = 0
	cb.halfOpenFailures = 0
	cb.logger.Info("Circuit breaker transitioned to half-open state")
}

// transitionToClosed transitions the circuit to the closed state
func (cb *CircuitBreaker) transitionToClosed() {
	cb.state = ClosedState
	cb.lastStateChange = time.Now()
	cb.failures = 0
	cb.logger.Info("Circuit breaker transitioned to closed state")
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return cb.state
}

// GetStatistics returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStatistics() (totalCalls, successfulCalls, failedCalls, shortCircuitedCalls int64) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return cb.totalCalls, cb.successfulCalls, cb.failedCalls, cb.shortCircuitedCalls
}

// LogStatistics logs statistics about the circuit breaker
func (cb *CircuitBreaker) LogStatistics() {
	totalCalls, successfulCalls, failedCalls, shortCircuitedCalls := cb.GetStatistics()
	
	cb.logger.Info("Circuit breaker statistics",
		zap.Int64("total_calls", totalCalls),
		zap.Int64("successful_calls", successfulCalls),
		zap.Int64("failed_calls", failedCalls),
		zap.Int64("short_circuited_calls", shortCircuitedCalls),
		zap.String("state", cb.stateToString()),
	)
}

// stateToString converts the circuit breaker state to a string
func (cb *CircuitBreaker) stateToString() string {
	switch cb.GetState() {
	case ClosedState:
		return "closed"
	case OpenState:
		return "open"
	case HalfOpenState:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerEventBusDecorator decorates an event bus with circuit breaking
type CircuitBreakerEventBusDecorator struct {
	eventBus eventbus.EventBus
	breaker  *CircuitBreaker
	logger   *zap.Logger
}

// NewCircuitBreakerEventBusDecorator creates a new circuit breaker event bus decorator
func NewCircuitBreakerEventBusDecorator(
	eventBus eventbus.EventBus,
	breaker *CircuitBreaker,
	logger *zap.Logger,
) *CircuitBreakerEventBusDecorator {
	return &CircuitBreakerEventBusDecorator{
		eventBus: eventBus,
		breaker:  breaker,
		logger:   logger,
	}
}

// PublishEvent publishes an event with circuit breaking
func (d *CircuitBreakerEventBusDecorator) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	return d.breaker.Execute(ctx, func(ctx context.Context) error {
		return d.eventBus.PublishEvent(ctx, event)
	})
}

// PublishEvents publishes multiple events with circuit breaking
func (d *CircuitBreakerEventBusDecorator) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	return d.breaker.Execute(ctx, func(ctx context.Context) error {
		return d.eventBus.PublishEvents(ctx, events)
	})
}

// Subscribe subscribes to all events
func (d *CircuitBreakerEventBusDecorator) Subscribe(handler eventsourcing.EventHandler) error {
	return d.eventBus.Subscribe(handler)
}

// SubscribeToType subscribes to events of a specific type
func (d *CircuitBreakerEventBusDecorator) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToType(eventType, handler)
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (d *CircuitBreakerEventBusDecorator) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToAggregate(aggregateType, handler)
}

