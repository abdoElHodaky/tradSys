package services

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/matching"
	"github.com/abdoElHodaky/tradSys/pkg/config"
	"github.com/abdoElHodaky/tradSys/pkg/errors"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// ServiceRegistry manages all services in the trading system
type ServiceRegistry struct {
	// Core services
	orderService      interfaces.OrderService
	tradeService      interfaces.TradeService
	marketDataService interfaces.MarketDataService
	positionService   interfaces.PositionService
	riskService       interfaces.RiskService
	matchingEngine    interfaces.MatchingEngine

	// Infrastructure services
	logger    interfaces.Logger
	metrics   interfaces.MetricsCollector
	publisher interfaces.EventPublisher
	config    *config.Config

	// Lifecycle management
	mu      sync.RWMutex
	started bool
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(
	cfg *config.Config,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	publisher interfaces.EventPublisher,
) *ServiceRegistry {
	return &ServiceRegistry{
		config:    cfg,
		logger:    logger,
		metrics:   metrics,
		publisher: publisher,
	}
}

// Initialize initializes all services with their dependencies
func (r *ServiceRegistry) Initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("Initializing service registry")

	// Initialize market data service first (no dependencies)
	r.marketDataService = NewMarketDataService(r.publisher, r.logger, r.metrics)

	// Initialize matching engine
	factory := matching.NewFactory(r.logger, r.publisher)
	engine, err := factory.CreateEngine(&r.config.Matching)
	if err != nil {
		return errors.Wrap(err, errors.ErrConfigurationError, "failed to create matching engine")
	}
	r.matchingEngine = engine

	// Initialize trade service (depends on position service)
	// Note: In a real implementation, you'd inject actual repository implementations
	r.tradeService = NewTradeServiceUnified(
		nil, // TradeRepository - would be injected
		r.positionService,
		r.publisher,
		r.logger,
		r.metrics,
	)

	r.logger.Info("Service registry initialized successfully")
	return nil
}

// Start starts all services
func (r *ServiceRegistry) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return errors.New(errors.ErrInvalidConfiguration, "service registry already started")
	}

	r.logger.Info("Starting all services")

	// Start market data service
	if r.marketDataService != nil {
		if mds, ok := r.marketDataService.(*MarketDataService); ok {
			if err := mds.Start(); err != nil {
				return errors.Wrap(err, errors.ErrServiceUnavailable, "failed to start market data service")
			}
		}
	}

	// Start matching engine
	if r.matchingEngine != nil {
		if err := r.matchingEngine.Start(ctx); err != nil {
			return errors.Wrap(err, errors.ErrServiceUnavailable, "failed to start matching engine")
		}
	}

	r.started = true
	r.logger.Info("All services started successfully")
	return nil
}

// Stop stops all services gracefully
func (r *ServiceRegistry) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return nil // Already stopped
	}

	r.logger.Info("Stopping all services")

	// Stop services in reverse order
	if r.matchingEngine != nil {
		if err := r.matchingEngine.Stop(ctx); err != nil {
			r.logger.Error("Failed to stop matching engine", "error", err)
		}
	}

	if r.marketDataService != nil {
		if mds, ok := r.marketDataService.(*MarketDataService); ok {
			if err := mds.Stop(); err != nil {
				r.logger.Error("Failed to stop market data service", "error", err)
			}
		}
	}

	r.started = false
	r.logger.Info("All services stopped")
	return nil
}

// GetOrderService returns the order service
func (r *ServiceRegistry) GetOrderService() interfaces.OrderService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.orderService
}

// GetTradeService returns the trade service
func (r *ServiceRegistry) GetTradeService() interfaces.TradeService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tradeService
}

// GetMarketDataService returns the market data service
func (r *ServiceRegistry) GetMarketDataService() interfaces.MarketDataService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.marketDataService
}

// GetPositionService returns the position service
func (r *ServiceRegistry) GetPositionService() interfaces.PositionService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.positionService
}

// GetRiskService returns the risk service
func (r *ServiceRegistry) GetRiskService() interfaces.RiskService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.riskService
}

// GetMatchingEngine returns the matching engine
func (r *ServiceRegistry) GetMatchingEngine() interfaces.MatchingEngine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.matchingEngine
}

// IsStarted returns whether the registry is started
func (r *ServiceRegistry) IsStarted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.started
}

// GetHealthStatus returns the health status of all services
func (r *ServiceRegistry) GetHealthStatus(ctx context.Context) *RegistryHealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := &RegistryHealthStatus{
		Overall:  interfaces.HealthStatusHealthy,
		Services: make(map[string]*interfaces.HealthStatus),
	}

	// Check each service if it implements HealthChecker
	services := map[string]interface{}{
		"order_service":       r.orderService,
		"trade_service":       r.tradeService,
		"market_data_service": r.marketDataService,
		"position_service":    r.positionService,
		"risk_service":        r.riskService,
		"matching_engine":     r.matchingEngine,
	}

	unhealthyCount := 0
	for name, service := range services {
		if service == nil {
			status.Services[name] = &interfaces.HealthStatus{
				Status:    interfaces.HealthStatusUnhealthy,
				Message:   "Service not initialized",
				Timestamp: ctx.Value("timestamp").(time.Time),
			}
			unhealthyCount++
			continue
		}

		if healthChecker, ok := service.(interfaces.HealthChecker); ok {
			if err := healthChecker.Check(ctx); err != nil {
				status.Services[name] = &interfaces.HealthStatus{
					Status:    interfaces.HealthStatusUnhealthy,
					Message:   err.Error(),
					Timestamp: ctx.Value("timestamp").(time.Time),
				}
				unhealthyCount++
			} else {
				status.Services[name] = &interfaces.HealthStatus{
					Status:    interfaces.HealthStatusHealthy,
					Message:   "Service is healthy",
					Timestamp: ctx.Value("timestamp").(time.Time),
				}
			}
		} else {
			// Service doesn't implement health checking, assume healthy if not nil
			status.Services[name] = &interfaces.HealthStatus{
				Status:    interfaces.HealthStatusHealthy,
				Message:   "Service is running (no health check)",
				Timestamp: ctx.Value("timestamp").(time.Time),
			}
		}
	}

	// Determine overall status
	if unhealthyCount > 0 {
		if unhealthyCount >= len(services)/2 {
			status.Overall = interfaces.HealthStatusUnhealthy
		} else {
			status.Overall = interfaces.HealthStatusDegraded
		}
	}

	return status
}

// GetServiceStatistics returns statistics about all services
func (r *ServiceRegistry) GetServiceStatistics() *ServiceStatistics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &ServiceStatistics{
		Started:        r.started,
		ServiceCount:   0,
		ServiceDetails: make(map[string]interface{}),
	}

	// Count initialized services
	services := []interface{}{
		r.orderService,
		r.tradeService,
		r.marketDataService,
		r.positionService,
		r.riskService,
		r.matchingEngine,
	}

	for _, service := range services {
		if service != nil {
			stats.ServiceCount++
		}
	}

	// Get detailed statistics from services that support it
	if r.matchingEngine != nil {
		stats.ServiceDetails["matching_engine"] = r.matchingEngine.GetMetrics()
	}

	if mds, ok := r.marketDataService.(*MarketDataService); ok {
		stats.ServiceDetails["market_data_service"] = mds.GetMarketDataStatistics()
	}

	return stats
}

// RegisterOrderService registers an order service (for dependency injection)
func (r *ServiceRegistry) RegisterOrderService(service interfaces.OrderService) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orderService = service
}

// RegisterPositionService registers a position service (for dependency injection)
func (r *ServiceRegistry) RegisterPositionService(service interfaces.PositionService) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.positionService = service
}

// RegisterRiskService registers a risk service (for dependency injection)
func (r *ServiceRegistry) RegisterRiskService(service interfaces.RiskService) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.riskService = service
}

// RegistryHealthStatus represents the health status of the entire registry
type RegistryHealthStatus struct {
	Overall  string                              `json:"overall"`
	Services map[string]*interfaces.HealthStatus `json:"services"`
}

// ServiceStatistics contains statistics about the service registry
type ServiceStatistics struct {
	Started        bool                   `json:"started"`
	ServiceCount   int                    `json:"service_count"`
	ServiceDetails map[string]interface{} `json:"service_details"`
}

// ServiceBuilder helps build services with proper dependency injection
type ServiceBuilder struct {
	registry *ServiceRegistry
}

// NewServiceBuilder creates a new service builder
func NewServiceBuilder(registry *ServiceRegistry) *ServiceBuilder {
	return &ServiceBuilder{
		registry: registry,
	}
}

// BuildOrderService builds an order service with all dependencies
func (b *ServiceBuilder) BuildOrderService(
	repository interfaces.OrderRepository,
	validator interfaces.OrderValidator,
) interfaces.OrderService {
	// This would be implemented with the actual OrderService
	// For now, return nil as we don't have the full implementation
	return nil
}

// BuildPositionService builds a position service with all dependencies
func (b *ServiceBuilder) BuildPositionService() interfaces.PositionService {
	// This would be implemented with the actual PositionService
	// For now, return nil as we don't have the full implementation
	return nil
}

// BuildRiskService builds a risk service with all dependencies
func (b *ServiceBuilder) BuildRiskService() interfaces.RiskService {
	// This would be implemented with the actual RiskService
	// For now, return nil as we don't have the full implementation
	return nil
}
