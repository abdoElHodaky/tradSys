package durability

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthStatus represents the health status of a component
type HealthStatus int

const (
	HealthStatusHealthy HealthStatus = iota
	HealthStatusDegraded
	HealthStatusUnhealthy
)

func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusDegraded:
		return "degraded"
	case HealthStatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// HealthCheck represents a health check function
type HealthCheck func(ctx context.Context) error

// ComponentHealth tracks the health of a system component
type ComponentHealth struct {
	Name           string
	Status         HealthStatus
	LastCheck      time.Time
	LastError      error
	CheckInterval  time.Duration
	FailureCount   int
	SuccessCount   int
}

// HealthMonitor monitors the health of system components
type HealthMonitor struct {
	mu         sync.RWMutex
	components map[string]*ComponentHealth
	checks     map[string]HealthCheck
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(logger *zap.Logger) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthMonitor{
		components: make(map[string]*ComponentHealth),
		checks:     make(map[string]HealthCheck),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// RegisterComponent registers a component for health monitoring
func (hm *HealthMonitor) RegisterComponent(name string, check HealthCheck, interval time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.components[name] = &ComponentHealth{
		Name:          name,
		Status:        HealthStatusHealthy,
		CheckInterval: interval,
	}
	hm.checks[name] = check

	// Start monitoring this component
	go hm.monitorComponent(name)
}

// monitorComponent runs health checks for a specific component
func (hm *HealthMonitor) monitorComponent(name string) {
	ticker := time.NewTicker(hm.getCheckInterval(name))
	defer ticker.Stop()

	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.performHealthCheck(name)
		}
	}
}

func (hm *HealthMonitor) getCheckInterval(name string) time.Duration {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	if comp, exists := hm.components[name]; exists {
		return comp.CheckInterval
	}
	return 30 * time.Second // default
}

// performHealthCheck executes a health check for a component
func (hm *HealthMonitor) performHealthCheck(name string) {
	hm.mu.RLock()
	check, exists := hm.checks[name]
	hm.mu.RUnlock()

	if !exists {
		return
	}

	ctx, cancel := context.WithTimeout(hm.ctx, 10*time.Second)
	defer cancel()

	err := check(ctx)
	hm.updateComponentHealth(name, err)
}

// updateComponentHealth updates the health status of a component
func (hm *HealthMonitor) updateComponentHealth(name string, err error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	comp, exists := hm.components[name]
	if !exists {
		return
	}

	comp.LastCheck = time.Now()
	comp.LastError = err

	if err == nil {
		comp.SuccessCount++
		// Recover from degraded/unhealthy state
		if comp.Status != HealthStatusHealthy {
			comp.Status = HealthStatusHealthy
			hm.logger.Info("Component recovered to healthy state",
				zap.String("component", name))
		}
	} else {
		comp.FailureCount++
		
		// Determine new status based on failure pattern
		if comp.FailureCount >= 5 {
			if comp.Status != HealthStatusUnhealthy {
				comp.Status = HealthStatusUnhealthy
				hm.logger.Error("Component marked as unhealthy",
					zap.String("component", name),
					zap.Int("failure_count", comp.FailureCount),
					zap.Error(err))
			}
		} else if comp.FailureCount >= 2 {
			if comp.Status != HealthStatusDegraded {
				comp.Status = HealthStatusDegraded
				hm.logger.Warn("Component marked as degraded",
					zap.String("component", name),
					zap.Int("failure_count", comp.FailureCount),
					zap.Error(err))
			}
		}
	}
}

// GetComponentHealth returns the health status of a component
func (hm *HealthMonitor) GetComponentHealth(name string) (*ComponentHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	comp, exists := hm.components[name]
	if !exists {
		return nil, false
	}
	// Return a copy to avoid race conditions
	return &ComponentHealth{
		Name:          comp.Name,
		Status:        comp.Status,
		LastCheck:     comp.LastCheck,
		LastError:     comp.LastError,
		CheckInterval: comp.CheckInterval,
		FailureCount:  comp.FailureCount,
		SuccessCount:  comp.SuccessCount,
	}, true
}

// GetOverallHealth returns the overall system health
func (hm *HealthMonitor) GetOverallHealth() HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if len(hm.components) == 0 {
		return HealthStatusHealthy
	}

	unhealthyCount := 0
	degradedCount := 0

	for _, comp := range hm.components {
		switch comp.Status {
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusDegraded:
			degradedCount++
		}
	}

	// If any component is unhealthy, system is unhealthy
	if unhealthyCount > 0 {
		return HealthStatusUnhealthy
	}

	// If any component is degraded, system is degraded
	if degradedCount > 0 {
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// GetAllComponents returns all component health statuses
func (hm *HealthMonitor) GetAllComponents() map[string]*ComponentHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]*ComponentHealth)
	for name, comp := range hm.components {
		result[name] = &ComponentHealth{
			Name:          comp.Name,
			Status:        comp.Status,
			LastCheck:     comp.LastCheck,
			LastError:     comp.LastError,
			CheckInterval: comp.CheckInterval,
			FailureCount:  comp.FailureCount,
			SuccessCount:  comp.SuccessCount,
		}
	}
	return result
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	hm.cancel()
}

// Metrics represents system metrics
type Metrics struct {
	mu                    sync.RWMutex
	OrdersProcessed       int64
	OrdersPerSecond       float64
	AverageLatency        time.Duration
	ErrorRate             float64
	ActiveConnections     int64
	MemoryUsage           int64
	CPUUsage              float64
	LastUpdated           time.Time
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		LastUpdated: time.Now(),
	}
}

// UpdateOrderMetrics updates order-related metrics
func (m *Metrics) UpdateOrderMetrics(processed int64, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.OrdersProcessed += processed
	
	// Calculate moving average for latency
	if m.AverageLatency == 0 {
		m.AverageLatency = latency
	} else {
		m.AverageLatency = (m.AverageLatency + latency) / 2
	}
	
	m.LastUpdated = time.Now()
}

// UpdateErrorRate updates the error rate
func (m *Metrics) UpdateErrorRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorRate = rate
	m.LastUpdated = time.Now()
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m
}
