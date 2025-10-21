package gateway

import (
	"net/http"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// MiddlewareParams contains the parameters for creating middleware
type MiddlewareParams struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

// Middleware provides API Gateway middleware functions
type Middleware struct {
	logger         *zap.Logger
	config         *config.Config
	ipLimiters     map[string]*rate.Limiter
	ipLimitersMu   sync.Mutex
	pathLimiters   map[string]*rate.Limiter
	pathLimitersMu sync.Mutex
	circuitBreakers map[string]*CircuitBreaker
	cbMu           sync.Mutex
}

// NewMiddleware creates a new middleware provider with fx dependency injection
func NewMiddleware(p MiddlewareParams) *Middleware {
	return &Middleware{
		logger:         p.Logger,
		config:         p.Config,
		ipLimiters:     make(map[string]*rate.Limiter),
		pathLimiters:   make(map[string]*rate.Limiter),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// RateLimitByIP returns a middleware that rate limits requests by IP address
func (m *Middleware) RateLimitByIP(rps float64, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		m.ipLimitersMu.Lock()
		limiter, exists := m.ipLimiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(rps), burst)
			m.ipLimiters[ip] = limiter
		}
		m.ipLimitersMu.Unlock()

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// RateLimitByPath returns a middleware that rate limits requests by path
func (m *Middleware) RateLimitByPath(rps float64, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		
		m.pathLimitersMu.Lock()
		limiter, exists := m.pathLimiters[path]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(rps), burst)
			m.pathLimiters[path] = limiter
		}
		m.pathLimitersMu.Unlock()

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded for this endpoint",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// CircuitBreaker represents a circuit breaker for API calls
type CircuitBreaker struct {
	name           string
	failureThreshold int
	resetTimeout   time.Duration
	failures       int
	lastFailure    time.Time
	state          string // "closed", "open", "half-open"
	mutex          sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:           name,
		failureThreshold: failureThreshold,
		resetTimeout:   resetTimeout,
		state:          "closed",
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(f func() error) error {
	cb.mutex.Lock()
	
	// Check if circuit is open
	if cb.state == "open" {
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half-open"
		} else {
			cb.mutex.Unlock()
			return ErrCircuitOpen
		}
	}
	
	cb.mutex.Unlock()
	
	// Execute the function
	err := f()
	
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	// Handle the result
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		
		// Check if failure threshold is reached
		if cb.state == "closed" && cb.failures >= cb.failureThreshold {
			cb.state = "open"
		} else if cb.state == "half-open" {
			cb.state = "open"
		}
		
		return err
	}
	
	// Reset on success
	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failures = 0
	}
	
	return nil
}

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = &CircuitOpenError{message: "Circuit breaker is open"}

// CircuitOpenError represents a circuit breaker open error
type CircuitOpenError struct {
	message string
}

// Error returns the error message
func (e *CircuitOpenError) Error() string {
	return e.message
}

// CircuitBreakerMiddleware returns a middleware that implements the circuit breaker pattern
func (m *Middleware) CircuitBreakerMiddleware(name string, failureThreshold int, resetTimeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or create circuit breaker
		m.cbMu.Lock()
		cb, exists := m.circuitBreakers[name]
		if !exists {
			cb = NewCircuitBreaker(name, failureThreshold, resetTimeout)
			m.circuitBreakers[name] = cb
		}
		m.cbMu.Unlock()
		
		// Execute the request with circuit breaker
		err := cb.Execute(func() error {
			// Store the original response writer
			originalWriter := c.Writer
			
			// Create a custom response writer to capture the status code
			blw := &bodyLogWriter{ResponseWriter: originalWriter}
			c.Writer = blw
			
			// Process the request
			c.Next()
			
			// Check if the response status code indicates a failure
			if blw.Status() >= 500 {
				return &CircuitOpenError{message: "Server error"}
			}
			
			return nil
		})
		
		// If circuit is open, return service unavailable
		if err == ErrCircuitOpen {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable",
			})
		}
	}
}

// bodyLogWriter is a custom response writer that captures the status code
type bodyLogWriter struct {
	gin.ResponseWriter
	status int
}

// WriteHeader captures the status code
func (w *bodyLogWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Status returns the status code
func (w *bodyLogWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// SecurityHeaders returns a middleware that adds security headers
func (m *Middleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		c.Next()
	}
}

