package mitigation

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiterConfig represents the configuration for a rate limiter
type RateLimiterConfig struct {
	// Rate is the maximum rate of events per second
	Rate float64
	// Burst is the maximum number of events that can happen in a burst
	Burst int
	// Timeout is the maximum time to wait for a token
	Timeout time.Duration
}

// DefaultRateLimiterConfig returns a default configuration for a rate limiter
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Rate:    100,
		Burst:   20,
		Timeout: 100 * time.Millisecond,
	}
}

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	name      string
	limiter   *rate.Limiter
	config    RateLimiterConfig
	logger    *zap.Logger
	metrics   *RateLimiterMetrics
	mutex     sync.RWMutex
}

// RateLimiterMetrics tracks metrics for the rate limiter
type RateLimiterMetrics struct {
	Allowed     int64
	Limited     int64
	WaitTime    time.Duration
	LastAllowed time.Time
	LastLimited time.Time
}

// NewRateLimiter creates a new rate limiter with the given name and configuration
func NewRateLimiter(name string, config RateLimiterConfig, logger *zap.Logger) *RateLimiter {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &RateLimiter{
		name:    name,
		limiter: rate.NewLimiter(rate.Limit(config.Rate), config.Burst),
		config:  config,
		logger:  logger.With(zap.String("component", "rate_limiter"), zap.String("name", name)),
		metrics: &RateLimiterMetrics{
			Allowed:     0,
			Limited:     0,
			WaitTime:    0,
			LastAllowed: time.Time{},
			LastLimited: time.Time{},
		},
	}
}

// Allow checks if an event should be allowed by the rate limiter
func (rl *RateLimiter) Allow() bool {
	allowed := rl.limiter.Allow()
	
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	if allowed {
		rl.metrics.Allowed++
		rl.metrics.LastAllowed = time.Now()
	} else {
		rl.metrics.Limited++
		rl.metrics.LastLimited = time.Now()
		rl.logger.Debug("Rate limited",
			zap.String("name", rl.name),
			zap.Int64("allowed", rl.metrics.Allowed),
			zap.Int64("limited", rl.metrics.Limited))
	}
	
	return allowed
}

// Wait waits for permission to proceed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	start := time.Now()
	
	// Create a context with timeout if not already set
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, rl.config.Timeout)
		defer cancel()
	}
	
	err := rl.limiter.Wait(ctx)
	
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	waitTime := time.Since(start)
	rl.metrics.WaitTime += waitTime
	
	if err != nil {
		rl.metrics.Limited++
		rl.metrics.LastLimited = time.Now()
		rl.logger.Debug("Rate limited (wait)",
			zap.String("name", rl.name),
			zap.Duration("wait_time", waitTime),
			zap.Error(err))
		return err
	}
	
	rl.metrics.Allowed++
	rl.metrics.LastAllowed = time.Now()
	return nil
}

// Reserve reserves a token and returns a Reservation that tells the caller how long to wait
func (rl *RateLimiter) Reserve() *rate.Reservation {
	return rl.limiter.Reserve()
}

// SetLimit updates the rate limit
func (rl *RateLimiter) SetLimit(newLimit float64) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.limiter.SetLimit(rate.Limit(newLimit))
	rl.config.Rate = newLimit
	rl.logger.Info("Rate limit updated", 
		zap.String("name", rl.name),
		zap.Float64("new_limit", newLimit))
}

// SetBurst updates the burst size
func (rl *RateLimiter) SetBurst(newBurst int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.limiter.SetBurst(newBurst)
	rl.config.Burst = newBurst
	rl.logger.Info("Burst size updated", 
		zap.String("name", rl.name),
		zap.Int("new_burst", newBurst))
}

// GetMetrics returns a copy of the current metrics
func (rl *RateLimiter) GetMetrics() RateLimiterMetrics {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	return RateLimiterMetrics{
		Allowed:     rl.metrics.Allowed,
		Limited:     rl.metrics.Limited,
		WaitTime:    rl.metrics.WaitTime,
		LastAllowed: rl.metrics.LastAllowed,
		LastLimited: rl.metrics.LastLimited,
	}
}

// Execute executes the given function with rate limiting
func (rl *RateLimiter) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := rl.Wait(ctx); err != nil {
		return ErrRateLimited
	}
	
	return fn(ctx)
}

// Reset resets the rate limiter metrics
func (rl *RateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.metrics.Allowed = 0
	rl.metrics.Limited = 0
	rl.metrics.WaitTime = 0
	rl.metrics.LastAllowed = time.Time{}
	rl.metrics.LastLimited = time.Time{}
	rl.logger.Info("Rate limiter metrics reset", zap.String("name", rl.name))
}

// ErrRateLimited is returned when a request is rate limited
var ErrRateLimited = RateLimitError{message: "rate limit exceeded"}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	message string
}

// Error returns the error message
func (e RateLimitError) Error() string {
	return e.message
}

