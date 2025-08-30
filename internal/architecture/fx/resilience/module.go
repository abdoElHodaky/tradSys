package resilience

import (
	"context"
	
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
	circuitBreaker CircuitBreakerFactory,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting resilience components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping resilience components")
			
			// Log circuit breaker metrics for common circuit breakers
			for _, name := range []string{"strategy-market-data", "strategy-order-update", "strategy-start", "strategy-stop"} {
				cb := circuitBreaker.GetCircuitBreaker(name)
				metrics := cb.GetMetrics()
				
				logger.Info("Circuit breaker metrics",
					zap.String("name", name),
					zap.Int("requests", metrics.Requests),
					zap.Int("successes", metrics.Successes),
					zap.Int("failures", metrics.Failures),
					zap.Int("timeouts", metrics.Timeouts),
					zap.Int("rejections", metrics.Rejections),
					zap.Float64("failure_rate", metrics.FailureRate))
			}
			
			return nil
		},
	})
}

