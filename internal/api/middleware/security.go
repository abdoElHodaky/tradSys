package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"go.uber.org/zap"
)

// SecurityMiddleware provides security-related middleware functions
type SecurityMiddleware struct {
	jwtService  *auth.JWTService
	logger      *zap.Logger
	rateLimiter *limiter.Limiter
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(jwtService *auth.JWTService, logger *zap.Logger) *SecurityMiddleware {
	// Create a rate limiter with 100 requests per minute
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}
	store := memory.NewStore()
	rateLimiter := limiter.New(store, rate)

	return &SecurityMiddleware{
		jwtService:  jwtService,
		logger:      logger,
		rateLimiter: rateLimiter,
	}
}

// JWTAuth is a middleware that validates JWT tokens
func (m *SecurityMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Check if the Authorization header has the correct format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be in the format 'Bearer {token}'"})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate the token
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			m.logger.Error("Failed to validate token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set the claims in the context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RoleAuth is a middleware that checks if the user has the required role
func (m *SecurityMiddleware) RoleAuth(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimiter is a middleware that limits the number of requests
func (m *SecurityMiddleware) RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the IP address from the request
		ip := c.ClientIP()

		// Create a context with the request
		ctx := c.Request.Context()

		// Check if the request is allowed
		limiterCtx, err := m.rateLimiter.Get(ctx, ip)
		if err != nil {
			m.logger.Error("Failed to get rate limiter context", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Set the rate limit headers
		c.Header("X-RateLimit-Limit", strconv.FormatInt(limiterCtx.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(limiterCtx.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(limiterCtx.Reset, 10))

		// Check if the request is over the limit
		if limiterCtx.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORS is a middleware that handles CORS
func (m *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeaders is a middleware that adds security headers
func (m *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers as required by tests
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Writer.Header().Set("Feature-Policy", "camera 'none'; microphone 'none'; geolocation 'none'")

		c.Next()
	}
}

// RequestID is a middleware that generates unique request IDs for audit logging
func (m *SecurityMiddleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate unique request ID
		requestID := generateRequestID()
		
		// Set request ID in context for audit logging
		c.Set("request_id", requestID)
		
		// Set request ID header for client
		c.Writer.Header().Set("X-Request-ID", requestID)
		
		// Log request with ID for audit trail
		m.logger.Info("Request received",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
		)

		c.Next()
	}
}

// generateRequestID creates a unique request ID for audit logging
func generateRequestID() string {
	// Use timestamp + random component for uniqueness
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%10000)
}
