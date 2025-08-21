# Circuit Breaker for Go with Fx Integration

This package provides a circuit breaker implementation for Go applications using Uber's Fx dependency injection framework. It is built on top of the [sony/gobreaker](https://github.com/sony/gobreaker) package and adds the following features:

- Integration with Uber's Fx dependency injection framework
- Comprehensive metrics collection
- Fallback support
- Context-aware execution
- Customizable settings
- Lifecycle management

## Usage

### Basic Usage

```go
// Create a circuit breaker factory
factory := resilience.NewCircuitBreakerFactory(resilience.CircuitBreakerParams{
    Logger: logger,
})

// Execute a function with circuit breaker protection
result := factory.Execute("example", func() (interface{}, error) {
    // Your code here
    return "success", nil
})

if result.Error != nil {
    logger.Error("Execution failed", zap.Error(result.Error))
} else {
    logger.Info("Execution succeeded", zap.Any("result", result.Value))
}
```

### With Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

result := factory.ExecuteWithContext(ctx, "example-with-context", func(ctx context.Context) (interface{}, error) {
    // Your code here with context
    select {
    case <-time.After(500 * time.Millisecond):
        return "success", nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
})
```

### With Fallback

```go
result := factory.ExecuteWithFallback(
    "example-with-fallback",
    func() (interface{}, error) {
        // Your code here
        return nil, errors.New("operation failed")
    },
    func(err error) (interface{}, error) {
        // Fallback operation
        logger.Warn("Fallback triggered", zap.Error(err))
        return "fallback result", nil
    },
)
```

### With Custom Settings

```go
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
    // Your code here
    return "custom success", nil
})
```

### Getting Metrics

```go
metrics := factory.GetMetrics()

logger.Info("Circuit breaker metrics",
    zap.Int64("executions", metrics.GetExecutionCount("example")),
    zap.Int64("successes", metrics.GetSuccessCount("example")),
    zap.Int64("failures", metrics.GetFailureCount("example")),
    zap.Float64("success_rate", metrics.GetSuccessRate("example")),
    zap.Duration("avg_execution_time", metrics.GetAverageExecutionTime("example")),
    zap.Int64("fallbacks", metrics.GetFallbackCount("example-with-fallback")),
    zap.Float64("fallback_success_rate", metrics.GetFallbackSuccessRate("example-with-fallback")))
```

## Fx Integration

To use the circuit breaker with Uber's Fx, you can use the provided module:

```go
app := fx.New(
    fx.Provide(
        // Provide a logger
        func() *zap.Logger {
            logger, _ := zap.NewDevelopment()
            return logger
        },
    ),
    
    // Include the resilience module
    resilience.Module,
    
    // Use the circuit breaker in your components
    fx.Invoke(func(cb *resilience.CircuitBreakerFactory) {
        // Use the circuit breaker
    }),
)
```

## Features

### Circuit Breaker States

The circuit breaker has three states:

- **Closed**: The circuit breaker is closed and allows requests to pass through.
- **Open**: The circuit breaker is open and blocks all requests, returning an error immediately.
- **Half-Open**: The circuit breaker allows a limited number of requests to pass through to test if the service is healthy again.

### Metrics

The circuit breaker collects the following metrics:

- **Executions**: The number of executions for a circuit breaker.
- **Successes**: The number of successful executions for a circuit breaker.
- **Failures**: The number of failed executions for a circuit breaker.
- **Success Rate**: The success rate for a circuit breaker.
- **Average Execution Time**: The average execution time for a circuit breaker.
- **Fallbacks**: The number of fallbacks for a circuit breaker.
- **Fallback Successes**: The number of successful fallbacks for a circuit breaker.
- **Fallback Failures**: The number of failed fallbacks for a circuit breaker.
- **Fallback Success Rate**: The fallback success rate for a circuit breaker.
- **Average Fallback Time**: The average fallback time for a circuit breaker.
- **State Changes**: The number of state changes for a circuit breaker.

### Lifecycle Management

The circuit breaker factory is integrated with Fx's lifecycle management. When the application stops, it logs the circuit breaker metrics for all circuit breakers.

## Dependencies

- [github.com/sony/gobreaker](https://github.com/sony/gobreaker): The underlying circuit breaker implementation.
- [go.uber.org/fx](https://github.com/uber-go/fx): Dependency injection framework.
- [go.uber.org/zap](https://github.com/uber-go/zap): Logging framework.

