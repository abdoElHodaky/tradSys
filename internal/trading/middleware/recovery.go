package middleware

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	
	"github.com/abdoElHodaky/tradSys/internal/trading/metrics"
)

// HFTRecoveryConfig contains recovery middleware configuration
type HFTRecoveryConfig struct {
	EnableStackTrace bool `yaml:"enable_stack_trace" default:"false"`
	EnableLogging    bool `yaml:"enable_logging" default:"true"`
	MaxStackSize     int  `yaml:"max_stack_size" default:"4096"`
}

// HFTRecoveryMiddleware provides optimized panic recovery for HFT endpoints
func HFTRecoveryMiddleware() gin.HandlerFunc {
	return HFTRecoveryMiddlewareWithConfig(&HFTRecoveryConfig{
		EnableStackTrace: false, // Disable for performance in production
		EnableLogging:    true,
		MaxStackSize:     4096,
	})
}

// HFTRecoveryMiddlewareWithConfig creates recovery middleware with custom configuration
func HFTRecoveryMiddlewareWithConfig(config *HFTRecoveryConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Record error in metrics
				if metrics.GlobalMetrics != nil {
					metrics.GlobalMetrics.RecordError()
				}
				
				// Log the panic if enabled
				if config.EnableLogging {
					logPanic(err, config)
				}
				
				// Return error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":     "internal_server_error",
					"message":   "An internal error occurred",
					"timestamp": time.Now().Unix(),
				})
			}
		}()
		
		c.Next()
	}
}

// logPanic logs panic information
func logPanic(err interface{}, config *HFTRecoveryConfig) {
	// In a real implementation, this would use a proper logger
	fmt.Printf("[PANIC] %v\n", err)
	
	if config.EnableStackTrace {
		// Get stack trace
		stack := make([]byte, config.MaxStackSize)
		length := runtime.Stack(stack, false)
		fmt.Printf("[STACK] %s\n", stack[:length])
	}
}

// HFTLoggerMiddleware provides minimal logging for HFT endpoints
func HFTLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		
		// Process request
		c.Next()
		
		// Calculate latency
		latency := time.Since(start)
		
		// Log only errors and slow requests for HFT performance
		if c.Writer.Status() >= 400 || latency > 100*time.Millisecond {
			if raw != "" {
				path = path + "?" + raw
			}
			
			fmt.Printf("[HFT] %s %s %d %v\n",
				c.Request.Method,
				path,
				c.Writer.Status(),
				latency,
			)
		}
	}
}

// HFTRequestIDMiddleware adds request ID for tracing
func HFTRequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from header or generate one
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		
		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}



// HFTResponseTimeMiddleware adds response time header
func HFTResponseTimeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		// Calculate and set response time in microseconds
		duration := time.Since(start)
		microseconds := duration.Nanoseconds() / 1000
		c.Header("X-Response-Time", fmt.Sprintf("%dμs", microseconds))
	}
}
