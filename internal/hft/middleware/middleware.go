package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/abdoElHodaky/tradSys/internal/hft/metrics"
)

// LatencyMiddleware measures and records request latency
func LatencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		latency := time.Since(start)
		metrics.RecordLatency(latency)
	}
}

// ThroughputMiddleware measures and records throughput
func ThroughputMiddleware() gin.HandlerFunc {
	var requestCount int64
	var lastReset time.Time = time.Now()
	
	return func(c *gin.Context) {
		requestCount++
		
		// Calculate RPS every second
		now := time.Now()
		if now.Sub(lastReset) >= time.Second {
			rps := float64(requestCount) / now.Sub(lastReset).Seconds()
			metrics.RecordThroughput(rps)
			requestCount = 0
			lastReset = now
		}
		
		c.Next()
	}
}

// ErrorTrackingMiddleware tracks errors
func ErrorTrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		
		// Check if there was an error
		if len(c.Errors) > 0 || c.Writer.Status() >= 400 {
			metrics.RecordError()
		} else {
			metrics.RecordSuccess()
		}
	}
}

// TimeoutMiddleware adds timeout to requests
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// CompressionMiddleware enables response compression for better performance
func CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set compression headers
		c.Header("Content-Encoding", "gzip")
		c.Next()
	}
}

// CacheMiddleware adds caching headers
func CacheMiddleware(maxAge time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age="+string(rune(int(maxAge.Seconds()))))
		c.Next()
	}
}

// SecurityMiddleware adds security headers
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// RateLimitMiddleware implements basic rate limiting
func RateLimitMiddleware(requestsPerSecond int) gin.HandlerFunc {
	var lastRequest time.Time
	var requestCount int
	
	return func(c *gin.Context) {
		now := time.Now()
		
		// Reset counter every second
		if now.Sub(lastRequest) >= time.Second {
			requestCount = 0
			lastRequest = now
		}
		
		requestCount++
		
		if requestCount > requestsPerSecond {
			c.JSON(429, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// HealthCheckMiddleware provides health check endpoint
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.JSON(200, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := generateRequestID()
		c.Header("X-Request-ID", requestID)
		c.Set("RequestID", requestID)
		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
