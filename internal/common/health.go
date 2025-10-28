package common

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthHandler provides health check functionality
type HealthHandler struct {
	serviceName    string
	serviceVersion string
	logger         *zap.Logger
	startTime      time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(serviceName, serviceVersion string, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		logger:         logger,
		startTime:      time.Now(),
	}
}

// RegisterRoutes registers health check routes
func (hh *HealthHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/health", hh.healthCheck)
	router.GET("/health/ready", hh.readinessCheck)
	router.GET("/health/live", hh.livenessCheck)
}

// healthCheck returns basic health information
func (hh *HealthHandler) healthCheck(c *gin.Context) {
	uptime := time.Since(hh.startTime)
	
	response := gin.H{
		"status":         "healthy",
		"service":        hh.serviceName,
		"version":        hh.serviceVersion,
		"uptime":         uptime.String(),
		"timestamp":      time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}

// readinessCheck checks if the service is ready to serve requests
func (hh *HealthHandler) readinessCheck(c *gin.Context) {
	// Add any readiness checks here (database connections, etc.)
	response := gin.H{
		"status":    "ready",
		"service":   hh.serviceName,
		"timestamp": time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}

// livenessCheck checks if the service is alive
func (hh *HealthHandler) livenessCheck(c *gin.Context) {
	response := gin.H{
		"status":    "alive",
		"service":   hh.serviceName,
		"timestamp": time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}

