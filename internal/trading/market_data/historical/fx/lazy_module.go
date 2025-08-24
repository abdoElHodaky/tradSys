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
	provideLazyHistoricalDataLoader,
	provideLazyHistoricalDataAnalyzer,
	
	// Register lifecycle hooks
	fx.Invoke(registerLazyHistoricalDataHooks),
)

// provideLazyHistoricalDataService provides a lazily loaded historical data service
func provideLazyHistoricalDataService(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"historical-data-service",
		func(config *historical.HistoricalDataConfig, logger *zap.Logger) (*historical.HistoricalDataService, error) {
			logger.Info("Lazily initializing historical data service")
			return historical.NewHistoricalDataService(config, logger)
		},
		logger,
		metrics,
	)
}

// provideLazyHistoricalDataLoader provides a lazily loaded historical data loader
func provideLazyHistoricalDataLoader(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"historical-data-loader",
		func(config *historical.HistoricalDataConfig, logger *zap.Logger) (*historical.HistoricalDataLoader, error) {
			logger.Info("Lazily initializing historical data loader")
			return historical.NewHistoricalDataLoader(config, logger)
		},
		logger,
		metrics,
	)
}

// provideLazyHistoricalDataAnalyzer provides a lazily loaded historical data analyzer
func provideLazyHistoricalDataAnalyzer(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"historical-data-analyzer",
		func(logger *zap.Logger) (*historical.HistoricalDataAnalyzer, error) {
			logger.Info("Lazily initializing historical data analyzer")
			return historical.NewHistoricalDataAnalyzer(logger)
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
	historicalDataLoaderProvider *lazy.LazyProvider,
	historicalDataAnalyzerProvider *lazy.LazyProvider,
) {
	logger.Info("Registering lazy historical data component hooks")
}

// GetHistoricalDataService gets the historical data service, initializing it if necessary
func GetHistoricalDataService(provider *lazy.LazyProvider) (*historical.HistoricalDataService, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*historical.HistoricalDataService), nil
}

// GetHistoricalDataLoader gets the historical data loader, initializing it if necessary
func GetHistoricalDataLoader(provider *lazy.LazyProvider) (*historical.HistoricalDataLoader, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*historical.HistoricalDataLoader), nil
}

// GetHistoricalDataAnalyzer gets the historical data analyzer, initializing it if necessary
func GetHistoricalDataAnalyzer(provider *lazy.LazyProvider) (*historical.HistoricalDataAnalyzer, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*historical.HistoricalDataAnalyzer), nil
}

