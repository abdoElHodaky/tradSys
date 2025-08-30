package resilience

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CircuitBreakerParams contains parameters for creating a circuit breaker factory
type CircuitBreakerParams struct {
	fx.In

	Logger *zap.Logger
}

// ProvideCircuitBreakerFactory provides a circuit breaker factory
func ProvideCircuitBreakerFactory(p CircuitBreakerParams) *CircuitBreakerFactory {
	return NewCircuitBreakerFactory(CircuitBreakerFactoryParams{
		Logger: p.Logger,
	})
}

// Module provides the resilience module
var Module = fx.Options(
	fx.Provide(ProvideCircuitBreakerFactory),
	fx.Invoke(func(lc fx.Lifecycle, circuitBreaker *CircuitBreakerFactory, logger *zap.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info("Starting resilience components")
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("Stopping resilience components")
				
				// Log circuit breaker metrics for common circuit breakers
				for _, name := range []string{"strategy-market-data", "strategy-order-update", "strategy-start", "strategy-stop"} {
					// Get metrics from the factory instead of individual circuit breakers
					metrics := circuitBreaker.GetMetrics()
					
					logger.Info("Circuit breaker metrics",
						zap.String("name", name),
						zap.Int64("executions", metrics.GetExecutionCount(name)),
						zap.Int64("successes", metrics.GetSuccessCount(name)),
						zap.Int64("failures", metrics.GetFailureCount(name)),
						zap.Float64("success_rate", metrics.GetSuccessRate(name)),
						zap.Duration("avg_execution_time", metrics.GetAverageExecutionTime(name)))
				}
				
				return nil
			},
		})
	}),
)

