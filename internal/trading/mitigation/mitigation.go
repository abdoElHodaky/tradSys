package mitigation

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// MitigationConfig represents the configuration for the mitigation system
type MitigationConfig struct {
	// CircuitBreaker is the configuration for the circuit breaker
	CircuitBreaker CircuitBreakerConfig
	// RateLimiter is the configuration for the rate limiter
	RateLimiter RateLimiterConfig
	// Retry is the configuration for the retry mechanism
	Retry RetryConfig
	// Bulkhead is the configuration for the bulkhead
	Bulkhead BulkheadConfig
	// Timeout is the configuration for the timeout handler
	Timeout TimeoutConfig
	// Cache is the configuration for the cache
	Cache CacheConfig
}

// DefaultMitigationConfig returns a default configuration for the mitigation system
func DefaultMitigationConfig() MitigationConfig {
	return MitigationConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig(),
		RateLimiter:    DefaultRateLimiterConfig(),
		Retry:          DefaultRetryConfig(),
		Bulkhead:       DefaultBulkheadConfig(),
		Timeout:        DefaultTimeoutConfig(),
		Cache:          DefaultCacheConfig(),
	}
}

// MitigationSystem combines multiple mitigation patterns
type MitigationSystem struct {
	name          string
	config        MitigationConfig
	circuitBreaker *CircuitBreaker
	rateLimiter   *RateLimiter
	bulkhead      *Bulkhead
	timeoutHandler *TimeoutHandler
	cache         *Cache
	logger        *zap.Logger
}

// NewMitigationSystem creates a new mitigation system with the given name and configuration
func NewMitigationSystem(name string, config MitigationConfig, logger *zap.Logger) *MitigationSystem {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &MitigationSystem{
		name:          name,
		config:        config,
		circuitBreaker: NewCircuitBreaker(name+"-cb", config.CircuitBreaker, logger),
		rateLimiter:   NewRateLimiter(name+"-rl", config.RateLimiter, logger),
		bulkhead:      NewBulkhead(name+"-bh", config.Bulkhead, logger),
		timeoutHandler: NewTimeoutHandler(name+"-to", config.Timeout, logger),
		cache:         NewCache(name+"-cache", config.Cache, logger),
		logger:        logger.With(zap.String("component", "mitigation_system"), zap.String("name", name)),
	}
}

// Execute executes the given function with all mitigation patterns applied
func (m *MitigationSystem) Execute(ctx context.Context, operation string, fn func(ctx context.Context) error) error {
	// Create a retry strategy
	retryStrategy := NewExponentialBackoffStrategy(m.config.Retry, m.logger)
	
	// Execute with all patterns
	return Retry(ctx, func(retryCtx context.Context) error {
		// Check circuit breaker
		if !m.circuitBreaker.Allow() {
			return ErrCircuitOpen
		}
		
		// Check rate limiter
		if !m.rateLimiter.Allow() {
			return ErrRateLimited
		}
		
		// Execute with bulkhead
		return m.bulkhead.Execute(retryCtx, func(bulkheadCtx context.Context) error {
			// Execute with timeout
			err := m.timeoutHandler.Execute(bulkheadCtx, operation, fn)
			
			// Record result in circuit breaker
			if err != nil {
				m.circuitBreaker.Failure()
			} else {
				m.circuitBreaker.Success()
			}
			
			return err
		})
	}, retryStrategy, m.logger)
}

// ExecuteWithCache executes the given function with all mitigation patterns applied and caching
func (m *MitigationSystem) ExecuteWithCache(ctx context.Context, operation string, cacheKey string, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, found := m.cache.Get(cacheKey); found {
		return value, nil
	}
	
	// Create a wrapper function that returns a value
	var result interface{}
	err := m.Execute(ctx, operation, func(execCtx context.Context) error {
		var execErr error
		result, execErr = fn(execCtx)
		return execErr
	})
	
	// Cache the result if successful
	if err == nil {
		m.cache.Set(cacheKey, result)
	}
	
	return result, err
}

// CircuitBreaker returns the circuit breaker
func (m *MitigationSystem) CircuitBreaker() *CircuitBreaker {
	return m.circuitBreaker
}

// RateLimiter returns the rate limiter
func (m *MitigationSystem) RateLimiter() *RateLimiter {
	return m.rateLimiter
}

// Bulkhead returns the bulkhead
func (m *MitigationSystem) Bulkhead() *Bulkhead {
	return m.bulkhead
}

// TimeoutHandler returns the timeout handler
func (m *MitigationSystem) TimeoutHandler() *TimeoutHandler {
	return m.timeoutHandler
}

// Cache returns the cache
func (m *MitigationSystem) Cache() *Cache {
	return m.cache
}

// UpdateConfig updates the mitigation system configuration
func (m *MitigationSystem) UpdateConfig(config MitigationConfig) {
	m.config = config
	m.bulkhead.UpdateConfig(config.Bulkhead)
	
	// Update other components as needed
	m.logger.Info("Mitigation system configuration updated", zap.String("name", m.name))
}

// Reset resets all mitigation components
func (m *MitigationSystem) Reset() {
	m.circuitBreaker.Reset()
	m.rateLimiter.Reset()
	m.bulkhead.Reset()
	m.timeoutHandler.Reset()
	m.cache.Clear()
	
	m.logger.Info("Mitigation system reset", zap.String("name", m.name))
}

// Close closes all mitigation components
func (m *MitigationSystem) Close() {
	m.cache.Close()
	
	m.logger.Info("Mitigation system closed", zap.String("name", m.name))
}

