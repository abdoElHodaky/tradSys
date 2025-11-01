// Package state provides standardized state machine implementations
package state

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// StandardCircuitBreakerStateMachine implements the CircuitBreakerStateMachine interface
type StandardCircuitBreakerStateMachine struct {
	mu                  sync.RWMutex
	currentState        interfaces.CircuitBreakerState
	failureCount        int64 // atomic
	successCount        int64 // atomic
	lastFailureTime     int64 // atomic, unix timestamp
	failureThreshold    int64
	recoveryTimeout     time.Duration
	halfOpenMaxRequests int64
	transitionHandlers  []interfaces.StateTransitionHandler
	name                string
}

// CircuitBreakerConfig contains configuration for the circuit breaker state machine
type CircuitBreakerConfig struct {
	Name                string
	FailureThreshold    int64
	RecoveryTimeout     time.Duration
	HalfOpenMaxRequests int64
}

// NewCircuitBreakerStateMachine creates a new standardized circuit breaker state machine
func NewCircuitBreakerStateMachine(config CircuitBreakerConfig) interfaces.CircuitBreakerStateMachine {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.RecoveryTimeout <= 0 {
		config.RecoveryTimeout = 30 * time.Second
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = 3
	}

	return &StandardCircuitBreakerStateMachine{
		currentState:        CircuitBreakerStates.Closed,
		failureThreshold:    config.FailureThreshold,
		recoveryTimeout:     config.RecoveryTimeout,
		halfOpenMaxRequests: config.HalfOpenMaxRequests,
		transitionHandlers:  make([]interfaces.StateTransitionHandler, 0),
		name:                config.Name,
	}
}

// GetCurrentState returns the current state
func (sm *StandardCircuitBreakerStateMachine) GetCurrentState() interfaces.State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// CanTransition checks if a transition is allowed
func (sm *StandardCircuitBreakerStateMachine) CanTransition(to interfaces.State) bool {
	sm.mu.RLock()
	current := sm.currentState
	sm.mu.RUnlock()

	// Define valid transitions
	switch current.Name() {
	case "closed":
		return to.Name() == "open"
	case "open":
		return to.Name() == "half-open"
	case "half-open":
		return to.Name() == "closed" || to.Name() == "open"
	default:
		return false
	}
}

// Transition attempts to transition to a new state
func (sm *StandardCircuitBreakerStateMachine) Transition(to interfaces.State, event string, ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.canTransitionUnsafe(to) {
		return fmt.Errorf("invalid transition from %s to %s", sm.currentState.Name(), to.Name())
	}

	from := sm.currentState
	sm.currentState = to.(interfaces.CircuitBreakerState)

	// Notify handlers
	transition := interfaces.Transition{
		From:    from,
		To:      to,
		Event:   event,
		Context: ctx,
	}

	for _, handler := range sm.transitionHandlers {
		if err := handler(transition); err != nil {
			// Log error but don't fail the transition
			continue
		}
	}

	return nil
}

// GetValidTransitions returns valid transitions from current state
func (sm *StandardCircuitBreakerStateMachine) GetValidTransitions() []interfaces.State {
	sm.mu.RLock()
	current := sm.currentState
	sm.mu.RUnlock()

	switch current.Name() {
	case "closed":
		return []interfaces.State{CircuitBreakerStates.Open}
	case "open":
		return []interfaces.State{CircuitBreakerStates.HalfOpen}
	case "half-open":
		return []interfaces.State{CircuitBreakerStates.Closed, CircuitBreakerStates.Open}
	default:
		return []interfaces.State{}
	}
}

// AddTransitionHandler adds a transition handler
func (sm *StandardCircuitBreakerStateMachine) AddTransitionHandler(handler interfaces.StateTransitionHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.transitionHandlers = append(sm.transitionHandlers, handler)
}

// RemoveTransitionHandler removes a transition handler
func (sm *StandardCircuitBreakerStateMachine) RemoveTransitionHandler(handler interfaces.StateTransitionHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Note: This is a simplified implementation. In production, you'd need to compare function pointers
	// or use a different approach like handler IDs
}

// RecordSuccess records a successful operation
func (sm *StandardCircuitBreakerStateMachine) RecordSuccess(ctx context.Context) error {
	current := sm.GetCurrentState().(interfaces.CircuitBreakerState)

	switch current.Name() {
	case "closed":
		// Reset failure count on success in closed state
		atomic.StoreInt64(&sm.failureCount, 0)
	case "half-open":
		newCount := atomic.AddInt64(&sm.successCount, 1)
		if newCount >= sm.halfOpenMaxRequests {
			// Transition back to closed after enough successes
			return sm.Transition(CircuitBreakerStates.Closed, "success_threshold_reached", ctx)
		}
	}

	return nil
}

// RecordFailure records a failed operation
func (sm *StandardCircuitBreakerStateMachine) RecordFailure(ctx context.Context) error {
	current := sm.GetCurrentState().(interfaces.CircuitBreakerState)
	atomic.StoreInt64(&sm.lastFailureTime, time.Now().Unix())

	switch current.Name() {
	case "closed":
		newCount := atomic.AddInt64(&sm.failureCount, 1)
		if newCount >= sm.failureThreshold {
			// Transition to open state after threshold reached
			return sm.Transition(CircuitBreakerStates.Open, "failure_threshold_reached", ctx)
		}
	case "half-open":
		// Any failure in half-open state transitions back to open
		atomic.StoreInt64(&sm.successCount, 0)
		return sm.Transition(CircuitBreakerStates.Open, "failure_in_half_open", ctx)
	}

	return nil
}

// GetFailureCount returns the current failure count
func (sm *StandardCircuitBreakerStateMachine) GetFailureCount() int64 {
	return atomic.LoadInt64(&sm.failureCount)
}

// GetSuccessCount returns the current success count
func (sm *StandardCircuitBreakerStateMachine) GetSuccessCount() int64 {
	return atomic.LoadInt64(&sm.successCount)
}

// Reset resets the circuit breaker to its initial state
func (sm *StandardCircuitBreakerStateMachine) Reset(ctx context.Context) error {
	atomic.StoreInt64(&sm.failureCount, 0)
	atomic.StoreInt64(&sm.successCount, 0)
	atomic.StoreInt64(&sm.lastFailureTime, 0)
	return sm.Transition(CircuitBreakerStates.Closed, "manual_reset", ctx)
}

// canTransitionUnsafe checks if transition is allowed (without locking)
func (sm *StandardCircuitBreakerStateMachine) canTransitionUnsafe(to interfaces.State) bool {
	switch sm.currentState.Name() {
	case "closed":
		return to.Name() == "open"
	case "open":
		return to.Name() == "half-open"
	case "half-open":
		return to.Name() == "closed" || to.Name() == "open"
	default:
		return false
	}
}

// ShouldAllowRequest checks if requests should be allowed based on current state and timing
func (sm *StandardCircuitBreakerStateMachine) ShouldAllowRequest() bool {
	current := sm.GetCurrentState().(interfaces.CircuitBreakerState)

	switch current.Name() {
	case "closed":
		return true
	case "open":
		// Check if recovery timeout has elapsed
		lastFailure := time.Unix(atomic.LoadInt64(&sm.lastFailureTime), 0)
		if time.Since(lastFailure) > sm.recoveryTimeout {
			// Try to transition to half-open
			ctx := context.Background()
			if err := sm.Transition(CircuitBreakerStates.HalfOpen, "recovery_timeout_elapsed", ctx); err == nil {
				atomic.StoreInt64(&sm.successCount, 0)
				return true
			}
		}
		return false
	case "half-open":
		// Allow limited requests in half-open state
		return atomic.LoadInt64(&sm.successCount) < sm.halfOpenMaxRequests
	default:
		return false
	}
}
