package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/abdoElHodaky/tradSys/internal/trading/metrics"
)

// HFTCircuitBreakerConfig contains circuit breaker configuration
type HFTCircuitBreakerConfig struct {
	MaxFailures      int           `yaml:"max_failures" default:"5"`
	ResetTimeout     time.Duration `yaml:"reset_timeout" default:"30s"`
	FailureRatio     float64       `yaml:"failure_ratio" default:"0.5"`
	MinRequests      int           `yaml:"min_requests" default:"10"`
	HalfOpenMaxCalls int           `yaml:"half_open_max_calls" default:"3"`
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements a circuit breaker pattern for HFT
type CircuitBreaker struct {
	config        *HFTCircuitBreakerConfig
	state         CircuitBreakerState
	failures      int64
	requests      int64
	successes     int64
	lastFailTime  time.Time
	halfOpenCalls int64
	mu            sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *HFTCircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = &HFTCircuitBreakerConfig{
			MaxFailures:      5,
			ResetTimeout:     30 * time.Second,
			FailureRatio:     0.5,
			MinRequests:      10,
			HalfOpenMaxCalls: 3,
		}
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := fn()
	cb.recordResult(err == nil)
	return err
}

// allowRequest checks if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return time.Since(cb.lastFailTime) >= cb.config.ResetTimeout
	case StateHalfOpen:
		return atomic.LoadInt64(&cb.halfOpenCalls) < int64(cb.config.HalfOpenMaxCalls)
	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.requests, 1)

	if success {
		atomic.AddInt64(&cb.successes, 1)
		if cb.state == StateHalfOpen {
			cb.state = StateClosed
			atomic.StoreInt64(&cb.failures, 0)
			atomic.StoreInt64(&cb.halfOpenCalls, 0)
		}
	} else {
		atomic.AddInt64(&cb.failures, 1)
		cb.lastFailTime = time.Now()

		if cb.state == StateHalfOpen {
			cb.state = StateOpen
		} else if cb.shouldTrip() {
			cb.state = StateOpen
		}
	}

	if cb.state == StateHalfOpen {
		atomic.AddInt64(&cb.halfOpenCalls, 1)
	}
}

// shouldTrip determines if the circuit breaker should trip
func (cb *CircuitBreaker) shouldTrip() bool {
	requests := atomic.LoadInt64(&cb.requests)
	failures := atomic.LoadInt64(&cb.failures)

	if requests < int64(cb.config.MinRequests) {
		return false
	}

	failureRatio := float64(failures) / float64(requests)
	return failureRatio >= cb.config.FailureRatio || failures >= int64(cb.config.MaxFailures)
}

// HFTCircuitBreakerMiddleware provides circuit breaker middleware
func HFTCircuitBreakerMiddleware(config *HFTCircuitBreakerConfig) gin.HandlerFunc {
	cb := NewCircuitBreaker(config)

	return func(c *gin.Context) {
		err := cb.Execute(func() error {
			c.Next()

			// Consider 5xx status codes as failures
			if c.Writer.Status() >= 500 {
				return fmt.Errorf("server error: %d", c.Writer.Status())
			}
			return nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_unavailable",
				"message": "Circuit breaker is open",
			})
		}
	}
}

// HFTRateLimiterConfig contains rate limiter configuration
type HFTRateLimiterConfig struct {
	RequestsPerSecond int           `yaml:"requests_per_second" default:"1000"`
	BurstSize         int           `yaml:"burst_size" default:"100"`
	WindowSize        time.Duration `yaml:"window_size" default:"1s"`
	KeyFunc           func(*gin.Context) string
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	capacity   int64
	tokens     int64
	refillRate int64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate
	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// HFTRateLimiterMiddleware provides rate limiting middleware
func HFTRateLimiterMiddleware(config *HFTRateLimiterConfig) gin.HandlerFunc {
	if config == nil {
		config = &HFTRateLimiterConfig{
			RequestsPerSecond: 1000,
			BurstSize:         100,
			WindowSize:        time.Second,
		}
	}

	if config.KeyFunc == nil {
		config.KeyFunc = func(c *gin.Context) string {
			return c.ClientIP()
		}
	}

	buckets := sync.Map{}

	return func(c *gin.Context) {
		key := config.KeyFunc(c)

		bucketInterface, _ := buckets.LoadOrStore(key, NewTokenBucket(
			int64(config.BurstSize),
			int64(config.RequestsPerSecond),
		))

		bucket := bucketInterface.(*TokenBucket)

		if !bucket.Allow() {
			c.Header("X-Rate-Limit-Limit", strconv.Itoa(config.RequestsPerSecond))
			c.Header("X-Rate-Limit-Remaining", "0")
			c.Header("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Add(config.WindowSize).Unix(), 10))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests",
			})
			return
		}

		c.Header("X-Rate-Limit-Limit", strconv.Itoa(config.RequestsPerSecond))
		c.Header("X-Rate-Limit-Remaining", strconv.FormatInt(bucket.tokens, 10))

		c.Next()
	}
}

// HFTTimeoutConfig contains timeout configuration
type HFTTimeoutConfig struct {
	RequestTimeout time.Duration `yaml:"request_timeout" default:"5s"`
	WriteTimeout   time.Duration `yaml:"write_timeout" default:"10s"`
}

// HFTTimeoutMiddleware provides request timeout middleware
func HFTTimeoutMiddleware(config *HFTTimeoutConfig) gin.HandlerFunc {
	if config == nil {
		config = &HFTTimeoutConfig{
			RequestTimeout: 5 * time.Second,
			WriteTimeout:   10 * time.Second,
		}
	}

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), config.RequestTimeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		go func() {
			c.Next()
			close(finished)
		}()

		select {
		case <-finished:
			// Request completed normally
		case <-ctx.Done():
			// Request timed out
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"error":   "request_timeout",
				"message": "Request timed out",
			})
		}
	}
}

// HFTMetricsMiddleware provides detailed metrics collection
func HFTMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		// Record metrics
		if metrics.GlobalMetrics != nil {
			metrics.GlobalMetrics.RecordHTTPRequest(method, path, status, duration)
		}

		// Add response headers
		c.Header("X-Response-Time", fmt.Sprintf("%.3fms", float64(duration.Nanoseconds())/1e6))
		c.Header("X-Request-ID", c.GetString("request_id"))
	}
}

// HFTCompressionConfig contains compression configuration
type HFTCompressionConfig struct {
	Level         int      `yaml:"level" default:"1"`         // 1-9, 1 = fastest
	MinLength     int      `yaml:"min_length" default:"1024"` // Minimum response size to compress
	ExcludedTypes []string `yaml:"excluded_types"`            // MIME types to exclude
	ExcludedPaths []string `yaml:"excluded_paths"`            // Paths to exclude
}

// HFTCompressionMiddleware provides optimized compression
func HFTCompressionMiddleware(config *HFTCompressionConfig) gin.HandlerFunc {
	if config == nil {
		config = &HFTCompressionConfig{
			Level:     1, // Fastest compression for HFT
			MinLength: 1024,
			ExcludedTypes: []string{
				"image/", "video/", "audio/", // Already compressed
				"application/octet-stream", // Binary data
			},
			ExcludedPaths: []string{
				"/ws", "/websocket", // WebSocket endpoints
			},
		}
	}

	// Pre-compile excluded paths and types for faster lookup
	excludedPaths := make(map[string]bool)
	for _, path := range config.ExcludedPaths {
		excludedPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip compression for excluded paths
		if excludedPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Check if client accepts compression
		acceptEncoding := c.GetHeader("Accept-Encoding")
		if acceptEncoding == "" {
			c.Next()
			return
		}

		// For HFT, we prioritize speed over compression ratio
		// Use minimal compression level
		c.Header("Vary", "Accept-Encoding")

		c.Next()
	}
}

// HFTCacheConfig contains caching configuration
type HFTCacheConfig struct {
	DefaultTTL   time.Duration `yaml:"default_ttl" default:"5m"`
	MaxSize      int           `yaml:"max_size" default:"1000"`
	EnableETag   bool          `yaml:"enable_etag" default:"true"`
	CacheControl string        `yaml:"cache_control" default:"public, max-age=300"`
}

// HFTCacheMiddleware provides response caching
func HFTCacheMiddleware(config *HFTCacheConfig) gin.HandlerFunc {
	if config == nil {
		config = &HFTCacheConfig{
			DefaultTTL:   5 * time.Minute,
			MaxSize:      1000,
			EnableETag:   true,
			CacheControl: "public, max-age=300",
		}
	}

	return func(c *gin.Context) {
		// Set cache headers
		c.Header("Cache-Control", config.CacheControl)

		if config.EnableETag {
			// Generate ETag based on request path and query
			etag := fmt.Sprintf(`"%x"`, time.Now().UnixNano())
			c.Header("ETag", etag)

			// Check If-None-Match header
			if match := c.GetHeader("If-None-Match"); match == etag {
				c.AbortWithStatus(http.StatusNotModified)
				return
			}
		}

		c.Next()
	}
}

// min returns the minimum of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
