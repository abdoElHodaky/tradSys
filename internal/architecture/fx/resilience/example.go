package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// ExampleUsage demonstrates how to use the circuit breaker
func ExampleUsage(logger *zap.Logger) {
	// Create a circuit breaker factory
	factory := NewCircuitBreakerFactory(CircuitBreakerParams{
		Logger: logger,
	})
	
	// Example 1: Basic usage
	result := factory.Execute("example", func() (interface{}, error) {
		// Simulate a successful operation
		return "success", nil
	})
	
	if result.Error != nil {
		logger.Error("Execution failed", zap.Error(result.Error))
	} else {
		logger.Info("Execution succeeded", zap.Any("result", result.Value))
	}
	
	// Example 2: With context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	result = factory.ExecuteWithContext(ctx, "example-with-context", func(ctx context.Context) (interface{}, error) {
		// Simulate a long-running operation
		select {
		case <-time.After(500 * time.Millisecond):
			return "success", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	
	if result.Error != nil {
		logger.Error("Execution with context failed", zap.Error(result.Error))
	} else {
		logger.Info("Execution with context succeeded", zap.Any("result", result.Value))
	}
	
	// Example 3: With fallback
	result = factory.ExecuteWithFallback(
		"example-with-fallback",
		func() (interface{}, error) {
			// Simulate a failing operation
			return nil, errors.New("operation failed")
		},
		func(err error) (interface{}, error) {
			// Fallback operation
			logger.Warn("Fallback triggered", zap.Error(err))
			return "fallback result", nil
		},
	)
	
	if result.Error != nil {
		logger.Error("Execution with fallback failed", zap.Error(result.Error))
	} else {
		logger.Info("Execution with fallback succeeded", zap.Any("result", result.Value))
	}
	
	// Example 4: With custom settings
	customSettings := gobreaker.Settings{
		Name:        "custom-example",
		MaxRequests: 3,
		Interval:    5 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}
	
	customCB := factory.GetCircuitBreakerWithSettings("custom-example", customSettings)
	
	// Use the custom circuit breaker
	result, err := customCB.Execute(func() (interface{}, error) {
		// Simulate a successful operation
		return "custom success", nil
	})
	
	if err != nil {
		logger.Error("Custom execution failed", zap.Error(err))
	} else {
		logger.Info("Custom execution succeeded", zap.Any("result", result))
	}
	
	// Example 5: Get metrics
	metrics := factory.GetMetrics()
	
	logger.Info("Circuit breaker metrics",
		zap.Int64("executions", metrics.GetExecutionCount("example")),
		zap.Int64("successes", metrics.GetSuccessCount("example")),
		zap.Int64("failures", metrics.GetFailureCount("example")),
		zap.Float64("success_rate", metrics.GetSuccessRate("example")),
		zap.Duration("avg_execution_time", metrics.GetAverageExecutionTime("example")),
		zap.Int64("fallbacks", metrics.GetFallbackCount("example-with-fallback")),
		zap.Float64("fallback_success_rate", metrics.GetFallbackSuccessRate("example-with-fallback")))
}

