package durability

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// DurabilityManager coordinates all durability features
type DurabilityManager struct {
	ErrorHandler   *ErrorHandler
	HealthMonitor  *HealthMonitor
	Metrics        *Metrics
	logger         *zap.Logger
}

// NewDurabilityManager creates a new durability manager
func NewDurabilityManager(logger *zap.Logger) *DurabilityManager {
	return &DurabilityManager{
		ErrorHandler:  NewErrorHandler(logger),
		HealthMonitor: NewHealthMonitor(logger),
		Metrics:       NewMetrics(),
		logger:        logger,
	}
}

// InitializeSystemComponents registers all system components for monitoring
func (dm *DurabilityManager) InitializeSystemComponents() {
	// Register core trading components
	dm.HealthMonitor.RegisterComponent("order_matching", dm.orderMatchingHealthCheck, 10*time.Second)
	dm.HealthMonitor.RegisterComponent("risk_manager", dm.riskManagerHealthCheck, 15*time.Second)
	dm.HealthMonitor.RegisterComponent("market_data", dm.marketDataHealthCheck, 5*time.Second)
	dm.HealthMonitor.RegisterComponent("websocket_gateway", dm.websocketHealthCheck, 30*time.Second)
	dm.HealthMonitor.RegisterComponent("database", dm.databaseHealthCheck, 20*time.Second)
	dm.HealthMonitor.RegisterComponent("external_apis", dm.externalAPIHealthCheck, 30*time.Second)
	
	dm.logger.Info("Durability system initialized with health monitoring for all components")
}

// Health check implementations for each component
func (dm *DurabilityManager) orderMatchingHealthCheck(ctx context.Context) error {
	// Check if order matching engine is responsive
	// This would typically involve checking queue sizes, processing times, etc.
	return nil // Placeholder - implement actual health check
}

func (dm *DurabilityManager) riskManagerHealthCheck(ctx context.Context) error {
	// Check if risk manager is functioning properly
	// This would check risk calculation latency, rule evaluation, etc.
	return nil // Placeholder - implement actual health check
}

func (dm *DurabilityManager) marketDataHealthCheck(ctx context.Context) error {
	// Check if market data feeds are active and current
	// This would check data freshness, feed connectivity, etc.
	return nil // Placeholder - implement actual health check
}

func (dm *DurabilityManager) websocketHealthCheck(ctx context.Context) error {
	// Check if WebSocket connections are healthy
	// This would check connection counts, message throughput, etc.
	return nil // Placeholder - implement actual health check
}

func (dm *DurabilityManager) databaseHealthCheck(ctx context.Context) error {
	// Check database connectivity and performance
	// This would check connection pool, query response times, etc.
	return nil // Placeholder - implement actual health check
}

func (dm *DurabilityManager) externalAPIHealthCheck(ctx context.Context) error {
	// Check external API connectivity (Binance, etc.)
	// This would check API response times, rate limits, etc.
	return nil // Placeholder - implement actual health check
}

// ExecuteWithDurability wraps an operation with full durability features
func (dm *DurabilityManager) ExecuteWithDurability(
	ctx context.Context,
	operationName string,
	operation func() error,
	retryConfig RetryConfig,
	circuitBreaker *CircuitBreaker,
	timeout time.Duration,
) error {
	start := time.Now()
	
	// Wrap with timeout
	timedOperation := func(ctx context.Context) error {
		return TimeoutWrapper(ctx, timeout, func(timeoutCtx context.Context) error {
			// Execute with circuit breaker if provided
			if circuitBreaker != nil {
				return circuitBreaker.Execute(timeoutCtx, operation)
			}
			return operation()
		})
	}
	
	// Execute with retry
	err := RetryWithBackoff(ctx, retryConfig, func() error {
		return timedOperation(ctx)
	}, dm.logger)
	
	// Record metrics
	latency := time.Since(start)
	if err == nil {
		dm.Metrics.UpdateOrderMetrics(1, latency)
	} else {
		// Handle error through error handler
		tradingErr := WrapError(
			"OPERATION_FAILURE",
			fmt.Sprintf("Operation %s failed", operationName),
			SeverityMedium,
			err,
			map[string]interface{}{
				"operation": operationName,
				"latency":   latency,
			},
		)
		dm.ErrorHandler.HandleError(ctx, tradingErr)
	}
	
	return err
}

// GetSystemStatus returns comprehensive system status
func (dm *DurabilityManager) GetSystemStatus() SystemStatus {
	overallHealth := dm.HealthMonitor.GetOverallHealth()
	components := dm.HealthMonitor.GetAllComponents()
	metrics := dm.Metrics.GetSnapshot()
	
	return SystemStatus{
		OverallHealth: overallHealth,
		Components:    components,
		Metrics:       metrics,
		Timestamp:     time.Now(),
	}
}

// SystemStatus represents the overall system status
type SystemStatus struct {
	OverallHealth HealthStatus
	Components    map[string]*ComponentHealth
	Metrics       Metrics
	Timestamp     time.Time
}

// CreateCircuitBreakerForComponent creates a circuit breaker for a specific component
func (dm *DurabilityManager) CreateCircuitBreakerForComponent(componentName string) *CircuitBreaker {
	var failureThreshold int
	var recoveryTimeout time.Duration
	
	// Configure circuit breaker based on component type
	switch componentName {
	case "order_matching":
		failureThreshold = 5
		recoveryTimeout = 30 * time.Second
	case "risk_manager":
		failureThreshold = 3
		recoveryTimeout = 60 * time.Second
	case "market_data":
		failureThreshold = 10
		recoveryTimeout = 15 * time.Second
	case "websocket_gateway":
		failureThreshold = 5
		recoveryTimeout = 30 * time.Second
	case "external_apis":
		failureThreshold = 3
		recoveryTimeout = 120 * time.Second
	default:
		failureThreshold = 5
		recoveryTimeout = 60 * time.Second
	}
	
	return NewCircuitBreaker(failureThreshold, recoveryTimeout, dm.logger)
}

// Stop gracefully shuts down the durability manager
func (dm *DurabilityManager) Stop() {
	dm.HealthMonitor.Stop()
	dm.logger.Info("Durability manager stopped")
}
