package fx

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ArchitectureCircuitBreakerModule provides the architecture circuit breaker components
var ArchitectureCircuitBreakerModule = fx.Options(
	// Provide the circuit breaker factory
	fx.Provide(NewArchitectureCircuitBreakerFactory),

	// Register lifecycle hooks
	fx.Invoke(registerArchitectureCircuitBreakerHooks),
)

// ArchitectureCircuitBreakerConfig contains configuration for circuit breakers
type ArchitectureCircuitBreakerConfig struct {
	// DefaultFailureThreshold is the default number of failures before opening the circuit
	DefaultFailureThreshold int64

	// DefaultResetTimeout is the default time to wait before attempting to close the circuit
	DefaultResetTimeout time.Duration

	// DefaultHalfOpenMaxRequests is the default maximum number of requests to allow in half-open state
	DefaultHalfOpenMaxRequests int64
}

// DefaultArchitectureCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultArchitectureCircuitBreakerConfig() ArchitectureCircuitBreakerConfig {
	return ArchitectureCircuitBreakerConfig{
		DefaultFailureThreshold:    5,
		DefaultResetTimeout:        30 * time.Second,
		DefaultHalfOpenMaxRequests: 1,
	}
}

// ArchitectureCircuitBreakerFactory creates circuit breakers
type ArchitectureCircuitBreakerFactory struct {
	logger *zap.Logger
	config ArchitectureCircuitBreakerConfig
}

// NewArchitectureCircuitBreakerFactory creates a new circuit breaker factory
func NewArchitectureCircuitBreakerFactory(logger *zap.Logger) *ArchitectureCircuitBreakerFactory {
	return &ArchitectureCircuitBreakerFactory{
		logger: logger,
		config: DefaultArchitectureCircuitBreakerConfig(),
	}
}

// CreateCircuitBreaker creates a new circuit breaker with the given name
func (f *ArchitectureCircuitBreakerFactory) CreateCircuitBreaker(name string) *architecture.CircuitBreaker {
	return architecture.NewCircuitBreaker(architecture.CircuitBreakerOptions{
		Name:                name,
		FailureThreshold:    f.config.DefaultFailureThreshold,
		ResetTimeout:        f.config.DefaultResetTimeout,
		HalfOpenMaxRequests: f.config.DefaultHalfOpenMaxRequests,
		OnStateChange: func(name string, from, to architecture.CircuitBreakerState) {
			f.logger.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.Int("from", int(from)),
				zap.Int("to", int(to)),
			)
		},
	})
}

// CreateCustomCircuitBreaker creates a new circuit breaker with custom options
func (f *ArchitectureCircuitBreakerFactory) CreateCustomCircuitBreaker(options architecture.CircuitBreakerOptions) *architecture.CircuitBreaker {
	// Ensure the OnStateChange function is set
	if options.OnStateChange == nil {
		options.OnStateChange = func(name string, from, to architecture.CircuitBreakerState) {
			f.logger.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.Int("from", int(from)),
				zap.Int("to", int(to)),
			)
		}
	}

	return architecture.NewCircuitBreaker(options)
}

// registerArchitectureCircuitBreakerHooks registers lifecycle hooks for the circuit breaker components
func registerArchitectureCircuitBreakerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting architecture circuit breaker components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping architecture circuit breaker components")
			return nil
		},
	})
}
