package mitigation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err       error
	Temporary bool
	Timeout   bool
}

// Error implements the error interface
func (e *RetryableError) Error() string {
	if e.Temporary {
		return fmt.Sprintf("temporary error: %s", e.Err.Error())
	}
	if e.Timeout {
		return fmt.Sprintf("timeout error: %s", e.Err.Error())
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for RetryableError
	var retryErr *RetryableError
	if errors.As(err, &retryErr) {
		return retryErr.Temporary || retryErr.Timeout
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Temporary() || netErr.Timeout()
	}

	// Add other error types that should be retried
	return false
}

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
	Logger      *zap.Logger
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
		Logger:      zap.NewNop(),
	}
}

// Retry executes the given function with exponential backoff
func Retry(ctx context.Context, config RetryConfig, operation func() error) error {
	var err error
	wait := config.InitialWait

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the operation
		err = operation()
		if err == nil {
			return nil
		}

		// Check if the error is retryable
		if !IsRetryable(err) {
			config.Logger.Debug("Non-retryable error encountered",
				zap.Error(err),
				zap.Int("attempt", attempt),
			)
			return err
		}

		// Check if we've reached max retries
		if attempt == config.MaxRetries {
			config.Logger.Debug("Max retries reached",
				zap.Error(err),
				zap.Int("maxRetries", config.MaxRetries),
			)
			return fmt.Errorf("max retries reached: %w", err)
		}

		// Check if context is cancelled
		if ctx.Err() != nil {
			config.Logger.Debug("Context cancelled during retry",
				zap.Error(ctx.Err()),
			)
			return ctx.Err()
		}

		// Log retry attempt
		config.Logger.Debug("Retrying operation",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
		)

		// Wait before next retry with exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
			// Calculate next wait time with exponential backoff
			wait = time.Duration(float64(wait) * config.Multiplier)
			if wait > config.MaxWait {
				wait = config.MaxWait
			}
		}
	}

	return err
}

// RetryWithResult executes the given function with exponential backoff and returns a result
func RetryWithResult[T any](ctx context.Context, config RetryConfig, operation func() (T, error)) (T, error) {
	var result T
	var err error
	wait := config.InitialWait

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the operation
		result, err = operation()
		if err == nil {
			return result, nil
		}

		// Check if the error is retryable
		if !IsRetryable(err) {
			config.Logger.Debug("Non-retryable error encountered",
				zap.Error(err),
				zap.Int("attempt", attempt),
			)
			return result, err
		}

		// Check if we've reached max retries
		if attempt == config.MaxRetries {
			config.Logger.Debug("Max retries reached",
				zap.Error(err),
				zap.Int("maxRetries", config.MaxRetries),
			)
			return result, fmt.Errorf("max retries reached: %w", err)
		}

		// Check if context is cancelled
		if ctx.Err() != nil {
			config.Logger.Debug("Context cancelled during retry",
				zap.Error(ctx.Err()),
			)
			return result, ctx.Err()
		}

		// Log retry attempt
		config.Logger.Debug("Retrying operation",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
		)

		// Wait before next retry with exponential backoff
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(wait):
			// Calculate next wait time with exponential backoff
			wait = time.Duration(float64(wait) * config.Multiplier)
			if wait > config.MaxWait {
				wait = config.MaxWait
			}
		}
	}

	return result, err
}

