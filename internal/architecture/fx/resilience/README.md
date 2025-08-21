# Resilience Package

This package provides resilience patterns for the TradSys platform, optimized to follow Fx benefits and best practices.

## Circuit Breaker

The circuit breaker pattern is implemented using the `github.com/sony/gobreaker` package, which is a well-established and maintained library for circuit breaking in Go.

### Key Features

- **Fx Integration**: Fully integrated with Uber's Fx dependency injection framework
- **Lifecycle Management**: Proper resource initialization and cleanup
- **Metrics Collection**: Built-in metrics collection for circuit breaker events
- **Context Support**: Context-aware circuit breaking
- **Fallback Support**: Support for fallback functions
- **Customizable**: Highly configurable circuit breaker behavior

### Usage

#### Basic Usage

```go
// In your Fx application
app := fx.New(
    // Include the resilience module
    resilience.Module,
    
    // Provide your services
    fx.Provide(
        NewMyService,
    ),
)

// In your service
type MyService struct {
    circuitBreaker *resilience.CircuitBreakerFactory
}

func NewMyService(circuitBreaker *resilience.CircuitBreakerFactory) *MyService {
    return &MyService{
        circuitBreaker: circuitBreaker,
    }
}

func (s *MyService) CallExternalService() (interface{}, error) {
    result := s.circuitBreaker.Execute("external-service", func() (interface{}, error) {
        // Call external service
        return callExternalService()
    })
    
    return result.Value, result.Error
}
```

#### With Fallback

```go
func (s *MyService) CallExternalServiceWithFallback() (interface{}, error) {
    result := s.circuitBreaker.ExecuteWithFallback(
        "external-service",
        func() (interface{}, error) {
            // Call external service
            return callExternalService()
        },
        func(err error) (interface{}, error) {
            // Fallback logic
            return getFromCache(), nil
        },
    )
    
    return result.Value, result.Error
}
```

#### With Context

```go
func (s *MyService) CallExternalServiceWithContext(ctx context.Context) (interface{}, error) {
    result := s.circuitBreaker.ExecuteContext(
        ctx,
        "external-service",
        func(ctx context.Context) (interface{}, error) {
            // Call external service with context
            return callExternalServiceWithContext(ctx)
        },
    )
    
    return result.Value, result.Error
}
```

#### Custom Configuration

```go
func (s *MyService) SetupCustomCircuitBreaker() {
    config := resilience.CircuitBreakerConfig{
        Name:        "custom-breaker",
        MaxRequests: 2,
        Interval:    time.Minute,
        Timeout:     10 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            // Trip when error rate is over 50% with at least 5 requests
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 5 && failureRatio >= 0.5
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            // Custom state change handler
            log.Printf("Circuit breaker %s changed from %s to %s", name, from, to)
        },
    }
    
    s.circuitBreaker.CreateCustomCircuitBreaker(config)
}
```

## Benefits Over Previous Implementation

1. **Better Dependency Injection**: Follows Fx's dependency injection pattern more closely
2. **Separation of Concerns**: Separates metrics collection from circuit breaker logic
3. **Context Support**: Adds support for context propagation
4. **Richer Results**: Returns more detailed results from circuit breaker executions
5. **Better Fallback Support**: Improved fallback function support
6. **More Configurable**: More configuration options for circuit breakers
7. **Better Metrics**: More detailed metrics collection
8. **Better Documentation**: More comprehensive documentation and examples

## Future Improvements

- Add support for more resilience patterns (retry, timeout, bulkhead, etc.)
- Add support for distributed circuit breaking
- Add support for more metrics backends (Prometheus, etc.)
- Add support for more sophisticated fallback strategies

