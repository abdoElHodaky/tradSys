package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	
	"github.com/abdoElHodaky/tradSys/internal/hft/metrics"
)

// Claims represents JWT claims for HFT authentication
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTPool provides pooling for JWT claims to reduce allocations
var jwtPool = sync.Pool{
	New: func() interface{} {
		return &Claims{}
	},
}

// HFTAuthConfig contains authentication configuration
type HFTAuthConfig struct {
	JWTSecret     string   `yaml:"jwt_secret"`
	SkipPaths     []string `yaml:"skip_paths"`
	RequiredPaths []string `yaml:"required_paths"`
}

// HFTAuthMiddleware provides high-performance JWT authentication
func HFTAuthMiddleware() gin.HandlerFunc {
	return HFTAuthMiddlewareWithConfig(&HFTAuthConfig{
		JWTSecret: "your-secret-key", // In production, load from config
		SkipPaths: []string{"/health", "/ready", "/metrics"},
	})
}

// HFTAuthMiddlewareWithConfig creates auth middleware with custom configuration
func HFTAuthMiddlewareWithConfig(config *HFTAuthConfig) gin.HandlerFunc {
	// Pre-compile skip paths for faster lookup
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}
	
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		
		// Get token from Authorization header
		token := c.GetHeader("Authorization")
		if len(token) < 7 || token[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Missing or invalid authorization header",
			})
			if metrics.GlobalMetrics != nil {
				metrics.GlobalMetrics.RecordError()
			}
			return
		}
		
		// Get claims from pool
		claims := jwtPool.Get().(*Claims)
		defer func() {
			// Reset claims before returning to pool
			claims.UserID = ""
			claims.Username = ""
			claims.RegisteredClaims = jwt.RegisteredClaims{}
			jwtPool.Put(claims)
		}()
		
		// Validate token
		if err := validateTokenFast(token[7:], claims, config.JWTSecret); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid token",
			})
			if metrics.GlobalMetrics != nil {
				metrics.GlobalMetrics.RecordError()
			}
			return
		}
		
		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		
		c.Next()
	}
}

// validateTokenFast performs fast JWT token validation with minimal allocations
func validateTokenFast(tokenString string, claims *Claims, secret string) error {
	// Parse token with claims
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	
	if err != nil {
		return err
	}
	
	if !token.Valid {
		return jwt.ErrTokenInvalidClaims
	}
	
	return nil
}

// HFTAPIKeyMiddleware provides API key authentication for high-frequency endpoints
func HFTAPIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "API key required",
			})
			if metrics.GlobalMetrics != nil {
				metrics.GlobalMetrics.RecordError()
			}
			return
		}
		
		// Validate API key (this would integrate with your API key service)
		if !validateAPIKeyFast(apiKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid API key",
			})
			if metrics.GlobalMetrics != nil {
				metrics.GlobalMetrics.RecordError()
			}
			return
		}
		
		// Set API key context
		c.Set("api_key", apiKey)
		
		c.Next()
	}
}

// validateAPIKeyFast performs fast API key validation
func validateAPIKeyFast(apiKey string) bool {
	// This would integrate with your API key validation service
	// For now, just check if it's not empty and has minimum length
	return len(apiKey) >= 32
}

// HFTRateLimitMiddleware provides rate limiting for HFT endpoints
func HFTRateLimitMiddleware(requestsPerSecond int) gin.HandlerFunc {
	// This would integrate with a rate limiting service like Redis
	// For now, just a placeholder
	return func(c *gin.Context) {
		// Rate limiting logic would go here
		c.Next()
	}
}

// HFTIPWhitelistMiddleware provides IP whitelisting for HFT endpoints
func HFTIPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	// Pre-compile allowed IPs for faster lookup
	allowedIPMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		allowedIPMap[ip] = true
	}
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if !allowedIPMap[clientIP] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "IP not whitelisted",
			})
			if metrics.GlobalMetrics != nil {
				metrics.GlobalMetrics.RecordError()
			}
			return
		}
		
		c.Next()
	}
}
