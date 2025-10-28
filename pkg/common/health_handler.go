package common

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthStatusType represents the health status of a service
type HealthStatusType string

const (
	HealthStatusHealthy   HealthStatusType = "healthy"
	HealthStatusUnhealthy HealthStatusType = "unhealthy"
	HealthStatusDegraded  HealthStatusType = "degraded"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatusType       `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    HealthStatusType  `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
	Service   string        `json:"service"`
	Version   string        `json:"version"`
	Uptime    time.Duration `json:"uptime"`
	Checks    []HealthCheck `json:"checks"`
}

// HealthChecker interface for implementing health checks
type HealthChecker interface {
	Name() string
	Check() HealthCheck
}

// HealthHandler provides health check endpoints
type HealthHandler struct {
	serviceName string
	version     string
	startTime   time.Time
	checkers    []HealthChecker
	logger      *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(serviceName, version string, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		serviceName: serviceName,
		version:     version,
		startTime:   time.Now(),
		checkers:    make([]HealthChecker, 0),
		logger:      logger,
	}
}

// AddChecker adds a health checker
func (h *HealthHandler) AddChecker(checker HealthChecker) {
	h.checkers = append(h.checkers, checker)
}

// RegisterRoutes registers health check routes
func (h *HealthHandler) RegisterRoutes(router gin.IRouter) {
	health := router.Group("/health")
	{
		health.GET("", h.HealthCheck)
		health.GET("/live", h.LivenessCheck)
		health.GET("/ready", h.ReadinessCheck)
	}
}

// HealthCheck returns the overall health status
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	correlationID := GetCorrelationID(c)
	logger := LogWithCorrelation(h.logger, correlationID)

	start := time.Now()
	checks := make([]HealthCheck, 0, len(h.checkers))
	overallStatus := HealthStatusHealthy

	// Run all health checks
	for _, checker := range h.checkers {
		check := checker.Check()
		checks = append(checks, check)

		// Determine overall status
		if check.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if check.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Service:   h.serviceName,
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
		Checks:    checks,
	}

	// Log health check
	logger.Info("Health check performed",
		zap.String("status", string(overallStatus)),
		zap.Duration("duration", time.Since(start)),
		zap.Int("checks_count", len(checks)))

	// Return appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == HealthStatusDegraded {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, response)
}

// LivenessCheck returns whether the service is alive
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	correlationID := GetCorrelationID(c)
	logger := LogWithCorrelation(h.logger, correlationID)

	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
		"service":   h.serviceName,
		"uptime":    time.Since(h.startTime),
	}

	logger.Debug("Liveness check performed")
	c.JSON(http.StatusOK, response)
}

// ReadinessCheck returns whether the service is ready to serve traffic
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	correlationID := GetCorrelationID(c)
	logger := LogWithCorrelation(h.logger, correlationID)

	// Check if all critical dependencies are ready
	ready := true
	failedChecks := make([]string, 0)

	for _, checker := range h.checkers {
		check := checker.Check()
		if check.Status == HealthStatusUnhealthy {
			ready = false
			failedChecks = append(failedChecks, check.Name)
		}
	}

	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now(),
		"service":   h.serviceName,
		"ready":     ready,
	}

	if !ready {
		response["failed_checks"] = failedChecks
		response["status"] = "not_ready"
	}

	logger.Debug("Readiness check performed", zap.Bool("ready", ready))

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	name string
	ping func() error
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(name string, ping func() error) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		name: name,
		ping: ping,
	}
}

// Name returns the checker name
func (d *DatabaseHealthChecker) Name() string {
	return d.name
}

// Check performs the database health check
func (d *DatabaseHealthChecker) Check() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        d.name,
		LastChecked: start,
	}

	if err := d.ping(); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = err.Error()
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Database connection is healthy"
	}

	check.Duration = time.Since(start)
	return check
}

// ServiceHealthChecker checks external service connectivity
type ServiceHealthChecker struct {
	name     string
	endpoint string
	timeout  time.Duration
	check    func() error
}

// NewServiceHealthChecker creates a new service health checker
func NewServiceHealthChecker(name, endpoint string, timeout time.Duration, check func() error) *ServiceHealthChecker {
	return &ServiceHealthChecker{
		name:     name,
		endpoint: endpoint,
		timeout:  timeout,
		check:    check,
	}
}

// Name returns the checker name
func (s *ServiceHealthChecker) Name() string {
	return s.name
}

// Check performs the service health check
func (s *ServiceHealthChecker) Check() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        s.name,
		LastChecked: start,
		Details: map[string]interface{}{
			"endpoint": s.endpoint,
			"timeout":  s.timeout,
		},
	}

	if err := s.check(); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = err.Error()
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Service is healthy"
	}

	check.Duration = time.Since(start)
	return check
}
