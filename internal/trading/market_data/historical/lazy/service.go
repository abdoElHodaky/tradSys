package lazy

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
)

// LazyHistoricalDataService is a lazy-loaded wrapper for the historical data service
type LazyHistoricalDataService struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Component name
	componentName string
	
	// Configuration
	config historical.Config
	
	// Logger
	logger *zap.Logger
}

// NewLazyHistoricalDataService creates a new lazy-loaded historical data service
func NewLazyHistoricalDataService(
	coordinator *coordination.ComponentCoordinator,
	config historical.Config,
	logger *zap.Logger,
) (*LazyHistoricalDataService, error) {
	componentName := "historical-data-service"
	
	// Create the provider function
	providerFn := func(log *zap.Logger) (interface{}, error) {
		return historical.NewService(config, log)
	}
	
	// Create the lazy provider
	provider := lazy.NewEnhancedLazyProvider(
		componentName,
		providerFn,
		logger,
		nil, // Metrics will be handled by the coordinator
		lazy.WithMemoryEstimate(config.MaxCacheSize),
		lazy.WithTimeout(30*time.Second),
		lazy.WithPriority(40), // Medium-high priority
	)
	
	// Register with the coordinator
	err := coordinator.RegisterComponent(
		componentName,
		"market-data",
		provider,
		[]string{}, // No dependencies
	)
	
	if err != nil {
		return nil, err
	}
	
	return &LazyHistoricalDataService{
		coordinator:   coordinator,
		componentName: componentName,
		config:        config,
		logger:        logger,
	}, nil
}

// GetHistoricalData gets historical market data
func (s *LazyHistoricalDataService) GetHistoricalData(
	ctx context.Context,
	symbol string,
	startTime time.Time,
	endTime time.Time,
	interval string,
) ([]*marketdata.MarketDataPoint, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(*historical.Service)
	if !ok {
		return nil, historical.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.GetHistoricalData(ctx, symbol, startTime, endTime, interval)
}

// GetLatestData gets the latest market data
func (s *LazyHistoricalDataService) GetLatestData(
	ctx context.Context,
	symbol string,
) (*marketdata.MarketDataPoint, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(*historical.Service)
	if !ok {
		return nil, historical.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.GetLatestData(ctx, symbol)
}

// ClearCache clears the cache
func (s *LazyHistoricalDataService) ClearCache(ctx context.Context) error {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(*historical.Service)
	if !ok {
		return historical.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.ClearCache(ctx)
}

// GetCacheStats gets cache statistics
func (s *LazyHistoricalDataService) GetCacheStats(ctx context.Context) (historical.CacheStats, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return historical.CacheStats{}, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(*historical.Service)
	if !ok {
		return historical.CacheStats{}, historical.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.GetCacheStats(ctx)
}

// Shutdown shuts down the service
func (s *LazyHistoricalDataService) Shutdown(ctx context.Context) error {
	return s.coordinator.ShutdownComponent(ctx, s.componentName)
}

