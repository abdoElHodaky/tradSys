package resilience

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CircuitBreakerModule provides the optimized circuit breaker components
// following Fx's modular design pattern
var CircuitBreakerModule = fx.Options(
	// Provide the circuit breaker factory
	fx.Provide(NewCircuitBreakerFactory),
	
	// Provide the default circuit breaker configuration
	fx.Provide(NewDefaultCircuitBreakerConfig),
	
	// Provide the circuit breaker metrics collector
	fx.Provide(NewCircuitBreakerMetrics),
	
	// Register lifecycle hooks
	fx.Invoke(registerCircuitBreakerHooks),
)

// CircuitBreakerConfig contains configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// Name is the name of the circuit breaker
	Name string
	
	// MaxRequests is the maximum number of requests allowed to pass through
	// when the CircuitBreaker is half-open
	MaxRequests uint32
	
	// Interval is the cyclic period of the closed state
	// for CircuitBreaker to clear the internal Counts
	Interval time.Duration
	
	// Timeout is the period of the open state,
	// after which the state of CircuitBreaker becomes half-open
	Timeout time.Duration
	
	// ReadyToTrip is called with a copy of Counts whenever a request fails in the closed state
	// If ReadyToTrip returns true, CircuitBreaker will be placed into the open state
	ReadyToTrip func(counts gobreaker.Counts) bool
	
	// OnStateChange is called whenever the state of CircuitBreaker changes
	OnStateChange func(name string, from gobreaker.State, to gobreaker.State)
}

// NewDefaultCircuitBreakerConfig returns the default circuit breaker configuration
// This follows Fx's pattern of providing default configurations as injectable dependencies
func NewDefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:        "default",
		MaxRequests: 1,
		Interval:    0, // disabled
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip when the consecutive failures is more than 5
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: nil,
	}
}

// CircuitBreakerMetrics collects metrics for circuit breakers
// Following Fx's pattern of separating concerns
type CircuitBreakerMetrics struct {
	logger *zap.Logger
}

// NewCircuitBreakerMetrics creates a new circuit breaker metrics collector
func NewCircuitBreakerMetrics(logger *zap.Logger) *CircuitBreakerMetrics {
	return &CircuitBreakerMetrics{
		logger: logger,
	}
}

// RecordStateChange records a state change for a circuit breaker
func (m *CircuitBreakerMetrics) RecordStateChange(name string, from, to gobreaker.State) {
	m.logger.Info("Circuit breaker state changed",
		zap.String("name", name),
		zap.String("from", stateToString(from)),
		zap.String("to", stateToString(to)))
}

// RecordSuccess records a successful execution for a circuit breaker
func (m *CircuitBreakerMetrics) RecordSuccess(name string, duration time.Duration) {
	m.logger.Debug("Circuit breaker execution succeeded",
		zap.String("name", name),
		zap.Duration("duration", duration))
}

// RecordFailure records a failed execution for a circuit breaker
func (m *CircuitBreakerMetrics) RecordFailure(name string, duration time.Duration, err error) {
	m.logger.Debug("Circuit breaker execution failed",
		zap.String("name", name),
		zap.Duration("duration", duration),
		zap.Error(err))
}

// CircuitBreakerFactory creates and manages circuit breakers
// This is the main service that will be injected into other components
type CircuitBreakerFactory struct {
	logger    *zap.Logger
	metrics   *CircuitBreakerMetrics
	config    *CircuitBreakerConfig
	breakers  map[string]*gobreaker.CircuitBreaker
	configs   map[string]CircuitBreakerConfig
}

// CircuitBreakerResult wraps the result of a circuit breaker execution
// to provide more context about the execution
type CircuitBreakerResult struct {
	Value      interface{}
	Error      error
	Duration   time.Duration
	BreakerName string
	State      gobreaker.State
}

// NewCircuitBreakerFactory creates a new circuit breaker factory
// Following Fx's dependency injection pattern
func NewCircuitBreakerFactory(
	logger *zap.Logger,
	metrics *CircuitBreakerMetrics,
	config *CircuitBreakerConfig,
) *CircuitBreakerFactory {
	return &CircuitBreakerFactory{
		logger:   logger,
		metrics:  metrics,
		config:   config,
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		configs:  make(map[string]CircuitBreakerConfig),
	}
}

// CreateCircuitBreaker creates a new circuit breaker with the given name and default configuration
func (f *CircuitBreakerFactory) CreateCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	config := *f.config
	config.Name = name
	return f.CreateCustomCircuitBreaker(config)
}

// CreateCustomCircuitBreaker creates a new circuit breaker with custom configuration
func (f *CircuitBreakerFactory) CreateCustomCircuitBreaker(config CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	// Store the configuration
	f.configs[config.Name] = config
	
	// Create the circuit breaker settings
	settings := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: config.ReadyToTrip,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Record metrics
			f.metrics.RecordStateChange(name, from, to)
			
			// Call the user-defined state change handler if provided
			if config.OnStateChange != nil {
				config.OnStateChange(name, from, to)
			}
		},
	}
	
	// Create the circuit breaker
	breaker := gobreaker.NewCircuitBreaker(settings)
	
	// Store the circuit breaker
	f.breakers[config.Name] = breaker
	
	f.logger.Info("Created circuit breaker", zap.String("name", config.Name))
	
	return breaker
}

// GetCircuitBreaker gets a circuit breaker by name
func (f *CircuitBreakerFactory) GetCircuitBreaker(name string) (*gobreaker.CircuitBreaker, bool) {
	breaker, ok := f.breakers[name]
	return breaker, ok
}

// GetOrCreateCircuitBreaker gets a circuit breaker by name or creates it if it doesn't exist
func (f *CircuitBreakerFactory) GetOrCreateCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	breaker, ok := f.GetCircuitBreaker(name)
	if !ok {
		// Create a new circuit breaker if it doesn't exist
		breaker = f.CreateCircuitBreaker(name)
	}
	return breaker
}

// GetAllCircuitBreakers returns all circuit breakers
func (f *CircuitBreakerFactory) GetAllCircuitBreakers() map[string]*gobreaker.CircuitBreaker {
	return f.breakers
}

// Execute executes the given function with the specified circuit breaker
func (f *CircuitBreakerFactory) Execute(name string, fn func() (interface{}, error)) *CircuitBreakerResult {
	breaker := f.GetOrCreateCircuitBreaker(name)
	
	startTime := time.Now()
	value, err := breaker.Execute(fn)
	duration := time.Since(startTime)
	
	result := &CircuitBreakerResult{
		Value:       value,
		Error:       err,
		Duration:    duration,
		BreakerName: name,
		State:       breaker.State(),
	}
	
	// Record metrics
	if err != nil {
		f.metrics.RecordFailure(name, duration, err)
	} else {
		f.metrics.RecordSuccess(name, duration)
	}
	
	return result
}

// ExecuteWithFallback executes the given function with the specified circuit breaker and fallback
func (f *CircuitBreakerFactory) ExecuteWithFallback(
	name string,
	fn func() (interface{}, error),
	fallback func(error) (interface{}, error),
) *CircuitBreakerResult {
	result := f.Execute(name, fn)
	
	// Apply fallback if needed
	if result.Error != nil && fallback != nil {
		startTime := time.Now()
		value, err := fallback(result.Error)
		fallbackDuration := time.Since(startTime)
		
		// Update the result
		result.Value = value
		result.Error = err
		result.Duration += fallbackDuration
		
		f.logger.Debug("Circuit breaker fallback executed",
			zap.String("name", name),
			zap.Duration("fallback_duration", fallbackDuration),
			zap.Error(err))
	}
	
	return result
}

// ExecuteContext executes the given function with the specified circuit breaker and context
// This allows for context-aware circuit breaking, following Fx's context propagation pattern
func (f *CircuitBreakerFactory) ExecuteContext(
	ctx context.Context,
	name string,
	fn func(ctx context.Context) (interface{}, error),
) *CircuitBreakerResult {
	return f.Execute(name, func() (interface{}, error) {
		return fn(ctx)
	})
}

// ExecuteContextWithFallback executes the given function with context, circuit breaker, and fallback
func (f *CircuitBreakerFactory) ExecuteContextWithFallback(
	ctx context.Context,
	name string,
	fn func(ctx context.Context) (interface{}, error),
	fallback func(ctx context.Context, err error) (interface{}, error),
) *CircuitBreakerResult {
	result := f.ExecuteContext(ctx, name, fn)
	
	// Apply fallback if needed
	if result.Error != nil && fallback != nil {
		startTime := time.Now()
		value, err := fallback(ctx, result.Error)
		fallbackDuration := time.Since(startTime)
		
		// Update the result
		result.Value = value
		result.Error = err
		result.Duration += fallbackDuration
		
		f.logger.Debug("Circuit breaker fallback executed with context",
			zap.String("name", name),
			zap.Duration("fallback_duration", fallbackDuration),
			zap.Error(err))
	}
	
	return result
}

// registerCircuitBreakerHooks registers lifecycle hooks for the circuit breaker components
// This follows Fx's lifecycle management pattern
func registerCircuitBreakerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	factory *CircuitBreakerFactory,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting circuit breaker components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping circuit breaker components")
			
			// Log statistics for all circuit breakers
			for name, breaker := range factory.breakers {
				state := stateToString(breaker.State())
				logger.Info("Circuit breaker statistics",
					zap.String("name", name),
					zap.String("state", state))
			}
			
			return nil
		},
	})
}

// stateToString converts a circuit breaker state to a string
func stateToString(state gobreaker.State) string {
	switch state {
	case gobreaker.StateClosed:
		return "closed"
	case gobreaker.StateHalfOpen:
		return "half-open"
	case gobreaker.StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

