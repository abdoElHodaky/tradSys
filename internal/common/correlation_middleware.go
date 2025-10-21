package common

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// CorrelationIDHeader is the header name for correlation ID
	CorrelationIDHeader = "X-Correlation-ID"
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey = "correlation_id"
)

// CorrelationMiddleware provides correlation ID middleware for request tracing
type CorrelationMiddleware struct {
	logger *zap.Logger
}

// NewCorrelationMiddleware creates a new correlation middleware
func NewCorrelationMiddleware(logger *zap.Logger) *CorrelationMiddleware {
	return &CorrelationMiddleware{
		logger: logger,
	}
}

// Handler returns a Gin middleware that adds correlation IDs to requests
func (m *CorrelationMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get correlation ID from header or generate a new one
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Add correlation ID to response header
		c.Header(CorrelationIDHeader, correlationID)

		// Add correlation ID to context
		ctx := context.WithValue(c.Request.Context(), CorrelationIDKey, correlationID)
		c.Request = c.Request.WithContext(ctx)

		// Add correlation ID to Gin context for easy access
		c.Set(CorrelationIDKey, correlationID)

		// Log the request with correlation ID
		m.logger.Info("Request started",
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")))

		// Process request
		c.Next()

		// Log the response with correlation ID
		m.logger.Info("Request completed",
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int("response_size", c.Writer.Size()))
	}
}

// GetCorrelationID extracts correlation ID from Gin context
func GetCorrelationID(c *gin.Context) string {
	if correlationID, exists := c.Get(CorrelationIDKey); exists {
		if id, ok := correlationID.(string); ok {
			return id
		}
	}
	return ""
}

// GetCorrelationIDFromContext extracts correlation ID from context
func GetCorrelationIDFromContext(ctx context.Context) string {
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			return id
		}
	}
	return ""
}

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// LogWithCorrelation creates a logger with correlation ID field
func LogWithCorrelation(logger *zap.Logger, correlationID string) *zap.Logger {
	if correlationID != "" {
		return logger.With(zap.String("correlation_id", correlationID))
	}
	return logger
}

// LogWithCorrelationFromContext creates a logger with correlation ID from context
func LogWithCorrelationFromContext(logger *zap.Logger, ctx context.Context) *zap.Logger {
	correlationID := GetCorrelationIDFromContext(ctx)
	return LogWithCorrelation(logger, correlationID)
}

// LogWithCorrelationFromGin creates a logger with correlation ID from Gin context
func LogWithCorrelationFromGin(logger *zap.Logger, c *gin.Context) *zap.Logger {
	correlationID := GetCorrelationID(c)
	return LogWithCorrelation(logger, correlationID)
}
