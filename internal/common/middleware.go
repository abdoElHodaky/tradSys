package common

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CorrelationMiddleware provides request correlation functionality
type CorrelationMiddleware struct {
	logger *zap.Logger
}

// NewCorrelationMiddleware creates a new correlation middleware
func NewCorrelationMiddleware(logger *zap.Logger) *CorrelationMiddleware {
	return &CorrelationMiddleware{
		logger: logger,
	}
}

// Handler returns the middleware handler function
func (c *CorrelationMiddleware) Handler() gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		// Add correlation ID to context
		correlationID := ctx.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = generateCorrelationID()
		}

		ctx.Header("X-Correlation-ID", correlationID)
		ctx.Set("correlation_id", correlationID)

		c.logger.Info("Request received",
			zap.String("correlation_id", correlationID),
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.Request.URL.Path),
		)

		ctx.Next()
	})
}

// generateCorrelationID generates a unique correlation ID
func generateCorrelationID() string {
	// Simple implementation - in production you might want to use UUID
	return "corr-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[len(charset)/2] // Simple implementation
	}
	return string(b)
}
