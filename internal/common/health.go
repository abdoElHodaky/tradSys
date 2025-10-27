package common

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthHandler provides health check functionality
type HealthHandler struct {
	serviceName string
	version     string
	logger      *zap.Logger
	startTime   time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(serviceName, version string, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		serviceName: serviceName,
		version:     version,
		logger:      logger,
		startTime:   time.Now(),
	}
}

// RegisterRoutes registers health check routes
func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	health := router.Group("/health")
	{
		health.GET("/", h.healthCheck)
		health.GET("/ready", h.readinessCheck)
		health.GET("/live", h.livenessCheck)
	}
}

// healthCheck returns the general health status
func (h *HealthHandler) healthCheck(c *gin.Context) {
	uptime := time.Since(h.startTime)

	response := gin.H{
		"status":    "healthy",
		"service":   h.serviceName,
		"version":   h.version,
		"uptime":    uptime.String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// readinessCheck returns readiness status
func (h *HealthHandler) readinessCheck(c *gin.Context) {
	// In a real implementation, you would check dependencies here
	// (database connections, external services, etc.)

	response := gin.H{
		"status":    "ready",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// livenessCheck returns liveness status
func (h *HealthHandler) livenessCheck(c *gin.Context) {
	response := gin.H{
		"status":    "alive",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}
