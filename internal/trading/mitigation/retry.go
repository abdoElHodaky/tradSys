package mitigation

import (
	"context"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

// RetryConfig represents the configuration for a retry mechanism
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffFactor is the factor by which the backoff increases
	BackoffFactor float64
	// Jitter is the maximum jitter factor (0.0 to 1.0)
	Jitter float64
}

// DefaultRetryConfig returns a default configuration for a retry mechanism
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0.2,
	}
}

// RetryStrategy defines the interface for retry strategies
type RetryStrategy interface {
	// ShouldRetry determines if a retry should be attempted based on the error and attempt number
	ShouldRetry(err error, attempt int) bool
	// NextBackoff calculates the next backoff duration
	NextBackoff(attempt int) time.Duration
}

// ExponentialBackoffStrategy implements exponential backoff with jitter
type ExponentialBackoffStrategy struct {
	config RetryConfig
	logger *zap.Logger
}

// NewExponentialBackoffStrategy creates a new exponential backoff strategy
func NewExponentialBackoffStrategy(config RetryConfig, logger *zap.Logger) *ExponentialBackoffStrategy {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &ExponentialBackoffStrategy{
		config: config,
		logger: logger.With(zap.String("component", "retry_strategy")),
	}
}

// ShouldRetry determines if a retry should be attempted
func (s *ExponentialBackoffStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.config.MaxRetries {
		return false
	}

	// Check if the error is retryable
	if IsRetryableError(err) {
		return true
	}

	return false
}

// NextBackoff calculates the next backoff duration
func (s *ExponentialBackoffStrategy) NextBackoff(attempt int) time.Duration {
	// Calculate base backoff with exponential increase
	backoff := float64(s.config.InitialBackoff) * math.Pow(s.config.BackoffFactor, float64(attempt))
	
	// Apply maximum backoff limit
	if backoff > float64(s.config.MaxBackoff) {
		backoff = float64(s.config.MaxBackoff)
	}
	
	// Apply jitter
	if s.config.Jitter > 0 {
		jitter := rand.Float64() * s.config.Jitter * backoff
		backoff = backoff + jitter
	}
	
	return time.Duration(backoff)
}

// Retry executes the given function with retry logic
func Retry(ctx context.Context, fn func(ctx context.Context) error, strategy RetryStrategy, logger *zap.Logger) error {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	
	var err error
	attempt := 0
	
	for {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}
		
		// Execute the function
		err = fn(ctx)
		
		// If no error or we shouldn't retry, return the result
		if err == nil || !strategy.ShouldRetry(err, attempt) {
			return err
		}
		
		// Calculate backoff duration
		backoff := strategy.NextBackoff(attempt)
		
		logger.Debug("Retrying operation",
			zap.Int("attempt", attempt+1),
			zap.Duration("backoff", backoff),
			zap.Error(err))
		
		// Wait for backoff duration or context cancellation
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
		
		attempt++
	}
}

// RetryWithFallback executes the given function with retry logic and falls back to a fallback function if all retries fail
func RetryWithFallback(
	ctx context.Context,
	fn func(ctx context.Context) error,
	fallback func(ctx context.Context, err error) error,
	strategy RetryStrategy,
	logger *zap.Logger,
) error {
	err := Retry(ctx, fn, strategy, logger)
	
	// If the operation failed after all retries, execute the fallback
	if err != nil && fallback != nil {
		logger.Debug("Executing fallback after retry failure", zap.Error(err))
		return fallback(ctx, err)
	}
	
	return err
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	// Check for specific error types that are retryable
	switch err.(type) {
	case TemporaryError, TimeoutError:
		return true
	}
	
	// Check if the error implements the Temporary interface
	if temp, ok := err.(interface{ Temporary() bool }); ok && temp.Temporary() {
		return true
	}
	
	// Check if the error implements the Timeout interface
	if timeout, ok := err.(interface{ Timeout() bool }); ok && timeout.Timeout() {
		return true
	}
	
	return false
}

// TemporaryError represents a temporary error that can be retried
type TemporaryError struct {
	Err error
}

// Error returns the error message
func (e TemporaryError) Error() string {
	if e.Err != nil {
		return "temporary error: " + e.Err.Error()
	}
	return "temporary error"
}

// Temporary returns true to indicate this is a temporary error
func (e TemporaryError) Temporary() bool {
	return true
}

// TimeoutError represents a timeout error that can be retried
type TimeoutError struct {
	Err error
}

// Error returns the error message
func (e TimeoutError) Error() string {
	if e.Err != nil {
		return "timeout error: " + e.Err.Error()
	}
	return "timeout error"
}

// Timeout returns true to indicate this is a timeout error
func (e TimeoutError) Timeout() bool {
	return true
}

