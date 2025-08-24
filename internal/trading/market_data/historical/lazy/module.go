package lazy

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides fx options for lazy loading historical data components
var Module = fx.Options(
	fx.Provide(
		NewHistoricalDataServiceProvider,
		NewTimeSeriesAnalyzerProvider,
		NewBacktestDataProviderProvider,
	),
)

// ProvideHistoricalDataService provides a historical data service via lazy loading
func ProvideHistoricalDataService(
	provider *HistoricalDataServiceProvider,
	logger *zap.Logger,
) (*historical.HistoricalDataService, error) {
	logger.Debug("Providing historical data service via lazy loading")
	return provider.Get()
}

// ProvideTimeSeriesAnalyzer provides a time series analyzer via lazy loading
func ProvideTimeSeriesAnalyzer(
	provider *TimeSeriesAnalyzerProvider,
	logger *zap.Logger,
) (*historical.TimeSeriesAnalyzer, error) {
	logger.Debug("Providing time series analyzer via lazy loading")
	return provider.Get()
}

// ProvideBacktestDataProvider provides a backtest data provider via lazy loading
func ProvideBacktestDataProvider(
	provider *BacktestDataProviderProvider,
	logger *zap.Logger,
) (*historical.BacktestDataProvider, error) {
	logger.Debug("Providing backtest data provider via lazy loading")
	return provider.Get()
}

// WithLazyHistoricalDataService provides an fx option for lazy loading historical data service
func WithLazyHistoricalDataService() fx.Option {
	return fx.Provide(ProvideHistoricalDataService)
}

// WithLazyTimeSeriesAnalyzer provides an fx option for lazy loading time series analyzer
func WithLazyTimeSeriesAnalyzer() fx.Option {
	return fx.Provide(ProvideTimeSeriesAnalyzer)
}

// WithLazyBacktestDataProvider provides an fx option for lazy loading backtest data provider
func WithLazyBacktestDataProvider() fx.Option {
	return fx.Provide(ProvideBacktestDataProvider)
}

// WithAllLazyHistoricalComponents provides an fx option for lazy loading all historical data components
func WithAllLazyHistoricalComponents() fx.Option {
	return fx.Options(
		WithLazyHistoricalDataService(),
		WithLazyTimeSeriesAnalyzer(),
		WithLazyBacktestDataProvider(),
	)
}

// RegisterMetrics registers metrics for lazy loading historical data components
func RegisterMetrics(metrics *lazy.LazyLoadingMetrics) {
	// Register component names for metrics tracking
	metrics.RegisterComponent("historical-data-service")
	metrics.RegisterComponent("time-series-analyzer")
	metrics.RegisterComponent("backtest-data-provider")
}

