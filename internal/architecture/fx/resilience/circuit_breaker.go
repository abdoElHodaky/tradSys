package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CircuitBreakerResult represents the result of a circuit breaker execution
type CircuitBreakerResult struct {
	Value interface{}
	Error error
}

// CircuitBreakerFactory creates and manages circuit breakers
type CircuitBreakerFactory struct {
	logger     *zap.Logger
	breakers   map[string]*gobreaker.CircuitBreaker
	settings   map[string]gobreaker.Settings
	mu         sync.RWMutex
	metrics    *CircuitBreakerMetrics
}

// CircuitBreakerParams contains parameters for creating a CircuitBreakerFactory
type CircuitBreakerParams struct {
	fx.In

	Logger *zap.Logger
}

// NewCircuitBreakerFactory creates a new CircuitBreakerFactory
func NewCircuitBreakerFactory(params CircuitBreakerParams) *CircuitBreakerFactory {
	metrics := NewCircuitBreakerMetrics()
	
	return &CircuitBreakerFactory{
		logger:   params.Logger,
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		settings: make(map[string]gobreaker.Settings),
		metrics:  metrics,
	}
}

// DefaultSettings returns the default circuit breaker settings
func DefaultSettings(name string, logger *zap.Logger, metrics *CircuitBreakerMetrics) gobreaker.Settings {
	return gobreaker.Settings{
		Name:        name,
		MaxRequests: 5,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()))
			
			metrics.RecordStateChange(name, from.String(), to.String())
		},
	}
}

// GetCircuitBreaker gets or creates a circuit breaker with the given name
func (f *CircuitBreakerFactory) GetCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	f.mu.RLock()
	cb, exists := f.breakers[name]
	f.mu.RUnlock()
	
	if exists {
		return cb
	}
	
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	if cb, exists = f.breakers[name]; exists {
		return cb
	}
	
	// Create a new circuit breaker with default settings
	settings := DefaultSettings(name, f.logger, f.metrics)
	cb = gobreaker.NewCircuitBreaker(settings)
	f.breakers[name] = cb
	f.settings[name] = settings
	
	return cb
}

// GetCircuitBreakerWithSettings gets or creates a circuit breaker with custom settings
func (f *CircuitBreakerFactory) GetCircuitBreakerWithSettings(name string, settings gobreaker.Settings) *gobreaker.CircuitBreaker {
	f.mu.RLock()
	cb, exists := f.breakers[name]
	f.mu.RUnlock()
	
	if exists {
		// Check if settings have changed
		f.mu.RLock()
		currentSettings := f.settings[name]
		f.mu.RUnlock()
		
		// If settings are the same, return the existing circuit breaker
		if currentSettings.MaxRequests == settings.MaxRequests &&
			currentSettings.Interval == settings.Interval &&
			currentSettings.Timeout == settings.Timeout {
			return cb
		}
		
		// Settings have changed, create a new circuit breaker
		f.mu.Lock()
		defer f.mu.Unlock()
		
		// Ensure OnStateChange is set to record metrics
		if settings.OnStateChange == nil {
			settings.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
				f.logger.Info("Circuit breaker state changed",
					zap.String("name", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()))
				
				f.metrics.RecordStateChange(name, from.String(), to.String())
			}
		}
		
		cb = gobreaker.NewCircuitBreaker(settings)
		f.breakers[name] = cb
		f.settings[name] = settings
		
		return cb
	}
	
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	if cb, exists = f.breakers[name]; exists {
		return cb
	}
	
	// Ensure OnStateChange is set to record metrics
	if settings.OnStateChange == nil {
		settings.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
			f.logger.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()))
			
			f.metrics.RecordStateChange(name, from.String(), to.String())
		}
	}
	
	// Create a new circuit breaker with custom settings
	cb = gobreaker.NewCircuitBreaker(settings)
	f.breakers[name] = cb
	f.settings[name] = settings
	
	return cb
}

// Execute executes a function with circuit breaker protection
func (f *CircuitBreakerFactory) Execute(name string, fn func() (interface{}, error)) CircuitBreakerResult {
	cb := f.GetCircuitBreaker(name)
	
	startTime := time.Now()
	result, err := cb.Execute(fn)
	duration := time.Since(startTime)
	
	f.metrics.RecordExecution(name, err == nil, duration)
	
	return CircuitBreakerResult{
		Value: result,
		Error: err,
	}
}

// ExecuteWithContext executes a function with circuit breaker protection and context
func (f *CircuitBreakerFactory) ExecuteWithContext(ctx context.Context, name string, fn func(ctx context.Context) (interface{}, error)) CircuitBreakerResult {
	cb := f.GetCircuitBreaker(name)
	
	startTime := time.Now()
	result, err := cb.Execute(func() (interface{}, error) {
		return fn(ctx)
	})
	duration := time.Since(startTime)
	
	f.metrics.RecordExecution(name, err == nil, duration)
	
	return CircuitBreakerResult{
		Value: result,
		Error: err,
	}
}

// ExecuteWithFallback executes a function with circuit breaker protection and fallback
func (f *CircuitBreakerFactory) ExecuteWithFallback(
	name string,
	fn func() (interface{}, error),
	fallback func(err error) (interface{}, error),
) CircuitBreakerResult {
	cb := f.GetCircuitBreaker(name)
	
	startTime := time.Now()
	result, err := cb.Execute(fn)
	duration := time.Since(startTime)
	
	f.metrics.RecordExecution(name, err == nil, duration)
	
	if err != nil && fallback != nil {
		fallbackStartTime := time.Now()
		fallbackResult, fallbackErr := fallback(err)
		fallbackDuration := time.Since(fallbackStartTime)
		
		f.metrics.RecordFallback(name, fallbackErr == nil, fallbackDuration)
		
		return CircuitBreakerResult{
			Value: fallbackResult,
			Error: fallbackErr,
		}
	}
	
	return CircuitBreakerResult{
		Value: result,
		Error: err,
	}
}

// GetState returns the current state of a circuit breaker
func (f *CircuitBreakerFactory) GetState(name string) gobreaker.State {
	f.mu.RLock()
	cb, exists := f.breakers[name]
	f.mu.RUnlock()
	
	if !exists {
		return gobreaker.StateClosed
	}
	
	return cb.State()
}

// GetMetrics returns the circuit breaker metrics
func (f *CircuitBreakerFactory) GetMetrics() *CircuitBreakerMetrics {
	return f.metrics
}

// Reset resets all circuit breakers
func (f *CircuitBreakerFactory) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.breakers = make(map[string]*gobreaker.CircuitBreaker)
	f.settings = make(map[string]gobreaker.Settings)
	f.metrics.Reset()
}

// CircuitBreakerMetrics collects metrics for circuit breakers
type CircuitBreakerMetrics struct {
	mu sync.RWMutex
	
	// Execution metrics
	executions     map[string]int64
	successes      map[string]int64
	failures       map[string]int64
	
	// Latency metrics
	executionTimes map[string][]time.Duration
	
	// Fallback metrics
	fallbacks           map[string]int64
	fallbackSuccesses   map[string]int64
	fallbackFailures    map[string]int64
	fallbackTimes       map[string][]time.Duration
	
	// State change metrics
	stateChanges map[string]map[string]map[string]int64 // name -> from -> to -> count
}

// NewCircuitBreakerMetrics creates a new CircuitBreakerMetrics
func NewCircuitBreakerMetrics() *CircuitBreakerMetrics {
	return &CircuitBreakerMetrics{
		executions:     make(map[string]int64),
		successes:      make(map[string]int64),
		failures:       make(map[string]int64),
		executionTimes: make(map[string][]time.Duration),
		fallbacks:      make(map[string]int64),
		fallbackSuccesses: make(map[string]int64),
		fallbackFailures:  make(map[string]int64),
		fallbackTimes:     make(map[string][]time.Duration),
		stateChanges:      make(map[string]map[string]map[string]int64),
	}
}

// RecordExecution records an execution of a circuit breaker
func (m *CircuitBreakerMetrics) RecordExecution(name string, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executions[name]++
	if success {
		m.successes[name]++
	} else {
		m.failures[name]++
	}
	
	if _, ok := m.executionTimes[name]; !ok {
		m.executionTimes[name] = make([]time.Duration, 0, 100)
	}
	
	m.executionTimes[name] = append(m.executionTimes[name], duration)
	
	// Keep only the last 100 execution times
	if len(m.executionTimes[name]) > 100 {
		m.executionTimes[name] = m.executionTimes[name][1:]
	}
}

// RecordFallback records a fallback execution of a circuit breaker
func (m *CircuitBreakerMetrics) RecordFallback(name string, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.fallbacks[name]++
	if success {
		m.fallbackSuccesses[name]++
	} else {
		m.fallbackFailures[name]++
	}
	
	if _, ok := m.fallbackTimes[name]; !ok {
		m.fallbackTimes[name] = make([]time.Duration, 0, 100)
	}
	
	m.fallbackTimes[name] = append(m.fallbackTimes[name], duration)
	
	// Keep only the last 100 fallback times
	if len(m.fallbackTimes[name]) > 100 {
		m.fallbackTimes[name] = m.fallbackTimes[name][1:]
	}
}

// RecordStateChange records a state change of a circuit breaker
func (m *CircuitBreakerMetrics) RecordStateChange(name, from, to string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.stateChanges[name]; !ok {
		m.stateChanges[name] = make(map[string]map[string]int64)
	}
	
	if _, ok := m.stateChanges[name][from]; !ok {
		m.stateChanges[name][from] = make(map[string]int64)
	}
	
	m.stateChanges[name][from][to]++
}

// GetExecutionCount returns the number of executions for a circuit breaker
func (m *CircuitBreakerMetrics) GetExecutionCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.executions[name]
}

// GetSuccessCount returns the number of successful executions for a circuit breaker
func (m *CircuitBreakerMetrics) GetSuccessCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.successes[name]
}

// GetFailureCount returns the number of failed executions for a circuit breaker
func (m *CircuitBreakerMetrics) GetFailureCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.failures[name]
}

// GetSuccessRate returns the success rate for a circuit breaker
func (m *CircuitBreakerMetrics) GetSuccessRate(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	executions := m.executions[name]
	if executions == 0 {
		return 0
	}
	
	return float64(m.successes[name]) / float64(executions)
}

// GetAverageExecutionTime returns the average execution time for a circuit breaker
func (m *CircuitBreakerMetrics) GetAverageExecutionTime(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	times, ok := m.executionTimes[name]
	if !ok || len(times) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	
	return sum / time.Duration(len(times))
}

// GetFallbackCount returns the number of fallbacks for a circuit breaker
func (m *CircuitBreakerMetrics) GetFallbackCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.fallbacks[name]
}

// GetFallbackSuccessCount returns the number of successful fallbacks for a circuit breaker
func (m *CircuitBreakerMetrics) GetFallbackSuccessCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.fallbackSuccesses[name]
}

// GetFallbackFailureCount returns the number of failed fallbacks for a circuit breaker
func (m *CircuitBreakerMetrics) GetFallbackFailureCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.fallbackFailures[name]
}

// GetFallbackSuccessRate returns the fallback success rate for a circuit breaker
func (m *CircuitBreakerMetrics) GetFallbackSuccessRate(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	fallbacks := m.fallbacks[name]
	if fallbacks == 0 {
		return 0
	}
	
	return float64(m.fallbackSuccesses[name]) / float64(fallbacks)
}

// GetAverageFallbackTime returns the average fallback time for a circuit breaker
func (m *CircuitBreakerMetrics) GetAverageFallbackTime(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	times, ok := m.fallbackTimes[name]
	if !ok || len(times) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	
	return sum / time.Duration(len(times))
}

// GetStateChangeCount returns the number of state changes for a circuit breaker
func (m *CircuitBreakerMetrics) GetStateChangeCount(name, from, to string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if _, ok := m.stateChanges[name]; !ok {
		return 0
	}
	
	if _, ok := m.stateChanges[name][from]; !ok {
		return 0
	}
	
	return m.stateChanges[name][from][to]
}

// Reset resets all metrics
func (m *CircuitBreakerMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executions = make(map[string]int64)
	m.successes = make(map[string]int64)
	m.failures = make(map[string]int64)
	m.executionTimes = make(map[string][]time.Duration)
	m.fallbacks = make(map[string]int64)
	m.fallbackSuccesses = make(map[string]int64)
	m.fallbackFailures = make(map[string]int64)
	m.fallbackTimes = make(map[string][]time.Duration)
	m.stateChanges = make(map[string]map[string]map[string]int64)
}

