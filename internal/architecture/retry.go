package architecture

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// RetryOptions contains options for retry behavior
type RetryOptions struct {
	// MaxRetries is the maximum number of retries
	MaxRetries int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffFactor is the factor by which the backoff increases
	BackoffFactor float64
	// Jitter is the amount of randomness to add to the backoff (0.0-1.0)
	Jitter float64
	// RetryableErrors is a function that determines if an error is retryable
	RetryableErrors func(error) bool
}

// DefaultRetryOptions returns the default retry options
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0.2,
		RetryableErrors: func(err error) bool {
			// By default, retry all errors
			return err != nil
		},
	}
}

// Retry executes the given function with retries based on the provided options
func Retry(ctx context.Context, fn func() error, options RetryOptions) error {
	var err error
	
	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		// Execute the function
		err = fn()
		
		// If no error or error is not retryable, return immediately
		if err == nil || (options.RetryableErrors != nil && !options.RetryableErrors(err)) {
			return err
		}
		
		// If this was the last attempt, return the error
		if attempt == options.MaxRetries {
			return err
		}
		
		// Calculate backoff duration
		backoff := calculateBackoff(attempt, options)
		
		// Create a timer for the backoff
		timer := time.NewTimer(backoff)
		
		// Wait for either the backoff timer or context cancellation
		select {
		case <-timer.C:
			// Continue to the next attempt
		case <-ctx.Done():
			timer.Stop()
			return errors.New("retry aborted due to context cancellation")
		}
	}
	
	return err
}

// calculateBackoff calculates the backoff duration for a given attempt
func calculateBackoff(attempt int, options RetryOptions) time.Duration {
	// Calculate base backoff with exponential increase
	backoff := float64(options.InitialBackoff) * math.Pow(options.BackoffFactor, float64(attempt))
	
	// Apply maximum backoff limit
	if backoff > float64(options.MaxBackoff) {
		backoff = float64(options.MaxBackoff)
	}
	
	// Apply jitter
	if options.Jitter > 0 {
		jitter := options.Jitter * backoff
		backoff = backoff - (jitter / 2) + (rand.Float64() * jitter)
	}
	
	return time.Duration(backoff)
}

// RetryWithFallback executes the given function with retries and falls back to
// the fallback function if all retries fail
func RetryWithFallback(ctx context.Context, fn func() error, fallback func() error, options RetryOptions) error {
	err := Retry(ctx, fn, options)
	if err != nil && fallback != nil {
		return fallback()
	}
	return err
}

