package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"go.uber.org/zap"
)

// SecurityMiddleware contains security middleware functions
type SecurityMiddleware struct {
	logger *zap.Logger
	store  limiter.Store
	rate   limiter.Rate
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(logger *zap.Logger) *SecurityMiddleware {
	// Create a rate limiter with 100 requests per minute
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}

	// Create a memory store for rate limiting
	store := memory.NewStore()

	return &SecurityMiddleware{
		logger: logger,
		store:  store,
		rate:   rate,
	}
}

// RateLimiter is a middleware for rate limiting
func (m *SecurityMiddleware) RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get IP address
		ip := c.ClientIP()

		// Create limiter for this IP
		limiterCtx, err := limiter.New(m.store, m.rate)
		if err != nil {
			m.logger.Error("Failed to create rate limiter", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Get context for this IP
		limiterCtx.Key = ip

		// Try to get limiter from store
		context, err := m.store.Get(c.Request.Context(), limiterCtx.Key)
		if err != nil {
			m.logger.Error("Failed to get rate limiter from store", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Check if rate limit is exceeded
		if context.Reached {
			m.logger.Warn("Rate limit exceeded", zap.String("ip", ip))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			return
		}

		// Increment rate limiter
		if _, err := m.store.Increment(c.Request.Context(), limiterCtx.Key, 1); err != nil {
			m.logger.Error("Failed to increment rate limiter", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", "100")
		c.Header("X-RateLimit-Reset", "60")

		c.Next()
	}
}

// SecurityHeaders adds security headers to the response
func (m *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
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

// CORS adds CORS headers to the response
func (m *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add CORS headers
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestLogger logs request information
func (m *SecurityMiddleware) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Log request
		m.logger.Info("Request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

// RecoverPanic recovers from panics
func (m *SecurityMiddleware) RecoverPanic() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log error
				m.logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
				)

				// Return error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			}
		}()

		c.Next()
	}
}

// ContentTypeValidator validates the Content-Type header
func (m *SecurityMiddleware) ContentTypeValidator() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for GET, HEAD, OPTIONS requests
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Check Content-Type header
		contentType := c.GetHeader("Content-Type")
		if contentType == "" {
			c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type header is required"})
			return
		}

		// Check if Content-Type is application/json
		if !strings.Contains(contentType, "application/json") {
			c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
			return
		}

		c.Next()
	}
}

// RegisterMiddleware registers all security middleware
func (m *SecurityMiddleware) RegisterMiddleware(router *gin.Engine) {
	// Add middleware
	router.Use(m.RecoverPanic())
	router.Use(m.RequestLogger())
	router.Use(m.SecurityHeaders())
	router.Use(m.CORS())
	router.Use(m.RateLimiter())
	router.Use(m.ContentTypeValidator())
}
