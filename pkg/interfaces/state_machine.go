// Package interfaces provides state machine abstractions for standardized state management
package interfaces

import "context"

// State represents a state in a state machine
type State interface {
	// Name returns the name of the state
	Name() string
	
	// String returns a string representation of the state
	String() string
}

// Transition represents a state transition
type Transition struct {
	From    State
	To      State
	Event   string
	Context context.Context
}

// StateTransitionHandler handles state transitions
type StateTransitionHandler func(transition Transition) error

// StateMachine defines the interface for state machine implementations
type StateMachine interface {
	// GetCurrentState returns the current state
	GetCurrentState() State
	
	// CanTransition checks if a transition from current state to target state is allowed
	CanTransition(to State) bool
	
	// Transition attempts to transition to a new state
	Transition(to State, event string, ctx context.Context) error
	
	// GetValidTransitions returns all valid transitions from the current state
	GetValidTransitions() []State
	
	// AddTransitionHandler adds a handler for state transitions
	AddTransitionHandler(handler StateTransitionHandler)
	
	// RemoveTransitionHandler removes a transition handler
	RemoveTransitionHandler(handler StateTransitionHandler)
}

// CircuitBreakerState represents standardized circuit breaker states
type CircuitBreakerState interface {
	State
	
	// IsRequestAllowed checks if requests should be allowed in this state
	IsRequestAllowed() bool
	
	// ShouldRecordMetrics checks if metrics should be recorded in this state
	ShouldRecordMetrics() bool
}

// CircuitBreakerStateMachine extends StateMachine for circuit breaker specific functionality
type CircuitBreakerStateMachine interface {
	StateMachine
	
	// RecordSuccess records a successful operation
	RecordSuccess(ctx context.Context) error
	
	// RecordFailure records a failed operation
	RecordFailure(ctx context.Context) error
	
	// GetFailureCount returns the current failure count
	GetFailureCount() int64
	
	// GetSuccessCount returns the current success count
	GetSuccessCount() int64
	
	// Reset resets the circuit breaker to its initial state
	Reset(ctx context.Context) error
}
