package architecture

import (
	"context"
	"errors"
	"sync"
	"time"
)

// RateLimiter implements rate limiting for API calls and other operations
type RateLimiter struct {
	name      string
	limit     int
	interval  time.Duration
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

// RateLimiterOptions contains options for creating a rate limiter
type RateLimiterOptions struct {
	Name     string
	Limit    int
	Interval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(options RateLimiterOptions) *RateLimiter {
	if options.Limit <= 0 {
		options.Limit = 100
	}
	if options.Interval <= 0 {
		options.Interval = time.Second
	}

	return &RateLimiter{
		name:      options.Name,
		limit:     options.Limit,
		interval:  options.Interval,
		tokens:    options.Limit,
		lastReset: time.Now(),
	}
}

// Allow checks if a request should be allowed based on the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Reset tokens if interval has passed
	if now.Sub(rl.lastReset) >= rl.interval {
		rl.tokens = rl.limit
		rl.lastReset = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// Wait waits until a token is available or the context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		// Calculate time until next reset
		rl.mu.Lock()
		timeUntilReset := rl.interval - time.Since(rl.lastReset)
		rl.mu.Unlock()

		if timeUntilReset <= 0 {
			continue
		}

		// Wait for the shorter of timeUntilReset or context cancellation
		select {
		case <-time.After(timeUntilReset):
			// Continue and try again
		case <-ctx.Done():
			return errors.New("rate limit wait aborted due to context cancellation")
		}
	}
}

// RemainingTokens returns the number of remaining tokens
func (rl *RateLimiter) RemainingTokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Reset tokens if interval has passed
	if now.Sub(rl.lastReset) >= rl.interval {
		rl.tokens = rl.limit
		rl.lastReset = now
	}

	return rl.tokens
}

// ResetIn returns the duration until the rate limiter resets
func (rl *RateLimiter) ResetIn() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	timeUntilReset := rl.interval - time.Since(rl.lastReset)
	if timeUntilReset < 0 {
		return 0
	}

	return timeUntilReset
}
