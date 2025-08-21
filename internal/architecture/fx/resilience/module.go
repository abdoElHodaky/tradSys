package resilience

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the resilience components
var Module = fx.Options(
	// Provide the circuit breaker factory
	fx.Provide(NewCircuitBreakerFactory),
	
	// Register lifecycle hooks
	fx.Invoke(registerHooks),
)

// registerHooks registers lifecycle hooks for the resilience components
func registerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	circuitBreaker *CircuitBreakerFactory,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx fx.Context) error {
			logger.Info("Starting resilience components")
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping resilience components")
			
			// Log circuit breaker metrics
			for _, name := range []string{"strategy-market-data", "strategy-order-update", "strategy-start", "strategy-stop"} {
				logger.Info("Circuit breaker metrics",
					zap.String("name", name),
					zap.Int64("executions", circuitBreaker.GetMetrics().GetExecutionCount(name)),
					zap.Int64("successes", circuitBreaker.GetMetrics().GetSuccessCount(name)),
					zap.Int64("failures", circuitBreaker.GetMetrics().GetFailureCount(name)),
					zap.Float64("success_rate", circuitBreaker.GetMetrics().GetSuccessRate(name)),
					zap.Duration("avg_execution_time", circuitBreaker.GetMetrics().GetAverageExecutionTime(name)),
					zap.Int64("fallbacks", circuitBreaker.GetMetrics().GetFallbackCount(name)),
					zap.Float64("fallback_success_rate", circuitBreaker.GetMetrics().GetFallbackSuccessRate(name)))
			}
			
			return nil
		},
	})
}

