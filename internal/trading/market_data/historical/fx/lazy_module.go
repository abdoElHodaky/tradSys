package fx

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyHistoricalDataModule provides lazily loaded historical data components
var LazyHistoricalDataModule = fx.Options(
	// Provide lazily loaded historical data components
	provideLazyHistoricalDataService,
	
	// Register lifecycle hooks
	fx.Invoke(registerLazyHistoricalDataHooks),
)

// provideLazyHistoricalDataService provides a lazily loaded historical data service
func provideLazyHistoricalDataService(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"historical-data-service",
		func(config *historical.Config, logger *zap.Logger) (*historical.Service, error) {
			logger.Info("Lazily initializing historical data service")
			return historical.NewService(*config, logger)
		},
		logger,
		metrics,
	)
}

// registerLazyHistoricalDataHooks registers lifecycle hooks for the lazy historical data components
func registerLazyHistoricalDataHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	historicalDataServiceProvider *lazy.LazyProvider,
) {
	logger.Info("Registering lazy historical data component hooks")
	
	// Register shutdown hook to clean up resources
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// Only clean up if the service was initialized
			if !historicalDataServiceProvider.IsInitialized() {
				return nil
			}
			
			// Get the service
			instance, err := historicalDataServiceProvider.Get()
			if err != nil {
				logger.Error("Failed to get historical data service during shutdown", zap.Error(err))
				return err
			}
			
			// Clean up resources
			service := instance.(*historical.Service)
			service.ClearCache()
			
			logger.Info("Historical data service resources cleaned up")
			return nil
		},
	})
}

// GetHistoricalDataService gets the historical data service, initializing it if necessary
func GetHistoricalDataService(provider *lazy.LazyProvider) (*historical.Service, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*historical.Service), nil
}

