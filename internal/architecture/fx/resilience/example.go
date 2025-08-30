package resilience

import (
	"errors"
	"fmt"
	"time"
)

// CircuitBreakerResult represents the result of a circuit breaker execution
type CircuitBreakerResult interface{}

// CircuitBreakerFactory creates circuit breakers
type CircuitBreakerFactory interface {
	// GetCircuitBreaker gets a circuit breaker with default settings
	GetCircuitBreaker(name string) CircuitBreaker
	
	// GetCircuitBreakerWithSettings gets a circuit breaker with custom settings
	GetCircuitBreakerWithSettings(name string, settings CircuitBreakerSettings) CircuitBreaker
}

// CircuitBreaker represents a circuit breaker
type CircuitBreaker interface {
	// Execute executes a function with circuit breaker protection
	Execute(fn func() (interface{}, error)) (CircuitBreakerResult, error)
	
	// GetState gets the current state of the circuit breaker
	GetState() CircuitBreakerState
	
	// GetMetrics gets the metrics of the circuit breaker
	GetMetrics() CircuitBreakerMetrics
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	// CircuitBreakerStateClosed means the circuit breaker is closed (allowing requests)
	CircuitBreakerStateClosed CircuitBreakerState = "closed"
	
	// CircuitBreakerStateOpen means the circuit breaker is open (blocking requests)
	CircuitBreakerStateOpen CircuitBreakerState = "open"
	
	// CircuitBreakerStateHalfOpen means the circuit breaker is half-open (allowing a limited number of requests)
	CircuitBreakerStateHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreakerMetrics represents the metrics of a circuit breaker
type CircuitBreakerMetrics struct {
	// Requests is the total number of requests
	Requests int
	
	// Successes is the number of successful requests
	Successes int
	
	// Failures is the number of failed requests
	Failures int
	
	// Timeouts is the number of timed out requests
	Timeouts int
	
	// Rejections is the number of rejected requests
	Rejections int
	
	// FailureRate is the failure rate
	FailureRate float64
}

// CircuitBreakerSettings represents the settings of a circuit breaker
type CircuitBreakerSettings struct {
	// FailureThreshold is the threshold for failures
	FailureThreshold int
	
	// SuccessThreshold is the threshold for successes
	SuccessThreshold int
	
	// Timeout is the timeout for requests
	Timeout time.Duration
	
	// ResetTimeout is the timeout for resetting the circuit breaker
	ResetTimeout time.Duration
}

// DefaultCircuitBreakerSettings returns the default circuit breaker settings
func DefaultCircuitBreakerSettings() CircuitBreakerSettings {
	return CircuitBreakerSettings{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          1 * time.Second,
		ResetTimeout:     30 * time.Second,
	}
}

// ExampleUsage demonstrates how to use the circuit breaker
func ExampleUsage(factory CircuitBreakerFactory) {
	// Get a circuit breaker with default settings
	cb := factory.GetCircuitBreaker("example")
	
	// Execute a function with circuit breaker protection
	result, err := cb.Execute(func() (interface{}, error) {
		// Simulate a successful operation
		return "success", nil
	})
	
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	
	fmt.Println("Result:", result)
	
	// Execute a function that fails
	result, err = cb.Execute(func() (interface{}, error) {
		// Simulate a failed operation
		return nil, errors.New("operation failed")
	})
	
	if err != nil {
		fmt.Println("Error:", err)
	}
	
	// Get a circuit breaker with custom settings
	customSettings := CircuitBreakerSettings{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          500 * time.Millisecond,
		ResetTimeout:     10 * time.Second,
	}
	
	customCB := factory.GetCircuitBreakerWithSettings("custom-example", customSettings)
	
	// Use the custom circuit breaker
	result, err = customCB.Execute(func() (interface{}, error) {
		// Simulate a successful operation
		return "custom success", nil
	})
	
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	
	fmt.Println("Custom Result:", result)
}

