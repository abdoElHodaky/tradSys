// Package state provides standardized state implementations for circuit breakers
package state

import "github.com/abdoElHodaky/tradSys/pkg/interfaces"

// ClosedState represents the closed state of a circuit breaker
type ClosedState struct{}

// Name returns the state name
func (s *ClosedState) Name() string {
	return "closed"
}

// String returns a string representation
func (s *ClosedState) String() string {
	return "CLOSED"
}

// IsRequestAllowed returns true for closed state
func (s *ClosedState) IsRequestAllowed() bool {
	return true
}

// ShouldRecordMetrics returns true for closed state
func (s *ClosedState) ShouldRecordMetrics() bool {
	return true
}

// OpenState represents the open state of a circuit breaker
type OpenState struct{}

// Name returns the state name
func (s *OpenState) Name() string {
	return "open"
}

// String returns a string representation
func (s *OpenState) String() string {
	return "OPEN"
}

// IsRequestAllowed returns false for open state
func (s *OpenState) IsRequestAllowed() bool {
	return false
}

// ShouldRecordMetrics returns false for open state
func (s *OpenState) ShouldRecordMetrics() bool {
	return false
}

// HalfOpenState represents the half-open state of a circuit breaker
type HalfOpenState struct{}

// Name returns the state name
func (s *HalfOpenState) Name() string {
	return "half-open"
}

// String returns a string representation
func (s *HalfOpenState) String() string {
	return "HALF_OPEN"
}

// IsRequestAllowed returns true for half-open state (with limits)
func (s *HalfOpenState) IsRequestAllowed() bool {
	return true
}

// ShouldRecordMetrics returns true for half-open state
func (s *HalfOpenState) ShouldRecordMetrics() bool {
	return true
}

// CircuitBreakerStates provides singleton instances of circuit breaker states
var CircuitBreakerStates = struct {
	Closed   interfaces.CircuitBreakerState
	Open     interfaces.CircuitBreakerState
	HalfOpen interfaces.CircuitBreakerState
}{
	Closed:   &ClosedState{},
	Open:     &OpenState{},
	HalfOpen: &HalfOpenState{},
}
