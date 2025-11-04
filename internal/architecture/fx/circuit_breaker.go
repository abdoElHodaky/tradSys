package fx

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CircuitBreakerModule provides the circuit breaker components
var CircuitBreakerModule = fx.Options(
	// Provide the circuit breaker
	fx.Provide(NewCircuitBreaker),

	// Register lifecycle hooks
	fx.Invoke(registerCircuitBreakerHooks),
)

// CircuitBreakerConfig contains configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int

	// ResetTimeout is the time to wait before attempting to close the circuit
	ResetTimeout string

	// HalfOpenMaxCalls is the maximum number of calls to allow in half-open state
	HalfOpenMaxCalls int
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     "10s",
		HalfOpenMaxCalls: 3,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(logger *zap.Logger) *integration.CircuitBreaker {
	// Create the circuit breaker configuration
	config := integration.DefaultCircuitBreakerConfig()

	// Create the circuit breaker
	return integration.NewCircuitBreaker(logger, config)
}

// registerCircuitBreakerHooks registers lifecycle hooks for the circuit breaker
func registerCircuitBreakerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	breaker *integration.CircuitBreaker,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting circuit breaker")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping circuit breaker")
			breaker.LogStatistics()
			return nil
		},
	})
}
