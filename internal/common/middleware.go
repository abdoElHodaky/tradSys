package common

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// Handler returns the gin middleware handler
func (cm *CorrelationMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or generate correlation ID
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Set correlation ID in context and response header
		c.Set("correlation_id", correlationID)
		c.Header("X-Correlation-ID", correlationID)

		// Log request with correlation ID
		cm.logger.Info("Request received",
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}

