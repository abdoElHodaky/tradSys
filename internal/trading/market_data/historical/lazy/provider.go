package lazy

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	"go.uber.org/zap"
)

// HistoricalDataServiceProvider provides lazy loading for historical data service
type HistoricalDataServiceProvider struct {
	lazyProvider *lazy.LazyProvider
	logger       *zap.Logger
}

// NewHistoricalDataServiceProvider creates a new provider
func NewHistoricalDataServiceProvider(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	config historical.Config,
) *HistoricalDataServiceProvider {
	return &HistoricalDataServiceProvider{
		lazyProvider: lazy.NewLazyProvider(
			"historical-data-service",
			func(logger *zap.Logger) (*historical.HistoricalDataService, error) {
				logger.Info("Initializing historical data service")
				return historical.NewHistoricalDataService(config, logger)
			},
			logger,
			metrics,
		),
		logger: logger,
	}
}

// Get returns the historical data service, initializing it if necessary
func (p *HistoricalDataServiceProvider) Get() (*historical.HistoricalDataService, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*historical.HistoricalDataService), nil
}

// GetWithContext returns the historical data service with context timeout
func (p *HistoricalDataServiceProvider) GetWithContext(ctx context.Context) (*historical.HistoricalDataService, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		service *historical.HistoricalDataService
		err     error
	})

	// Get the service in a goroutine
	go func() {
		instance, err := p.lazyProvider.Get()
		if err != nil {
			resultCh <- struct {
				service *historical.HistoricalDataService
				err     error
			}{nil, err}
			return
		}

		resultCh <- struct {
			service *historical.HistoricalDataService
			err     error
		}{instance.(*historical.HistoricalDataService), nil}
	}()

	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.service, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsInitialized returns whether the service has been initialized
func (p *HistoricalDataServiceProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// TimeSeriesAnalyzerProvider provides lazy loading for time series analyzer
type TimeSeriesAnalyzerProvider struct {
	lazyProvider *lazy.LazyProvider
	logger       *zap.Logger
}

// NewTimeSeriesAnalyzerProvider creates a new provider
func NewTimeSeriesAnalyzerProvider(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	config historical.AnalyzerConfig,
) *TimeSeriesAnalyzerProvider {
	return &TimeSeriesAnalyzerProvider{
		lazyProvider: lazy.NewLazyProvider(
			"time-series-analyzer",
			func(logger *zap.Logger) (*historical.TimeSeriesAnalyzer, error) {
				logger.Info("Initializing time series analyzer")
				return historical.NewTimeSeriesAnalyzer(config, logger)
			},
			logger,
			metrics,
		),
		logger: logger,
	}
}

// Get returns the time series analyzer, initializing it if necessary
func (p *TimeSeriesAnalyzerProvider) Get() (*historical.TimeSeriesAnalyzer, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*historical.TimeSeriesAnalyzer), nil
}

// IsInitialized returns whether the analyzer has been initialized
func (p *TimeSeriesAnalyzerProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// BacktestDataProviderProvider provides lazy loading for backtest data provider
type BacktestDataProviderProvider struct {
	lazyProvider *lazy.LazyProvider
	logger       *zap.Logger
}

// NewBacktestDataProviderProvider creates a new provider
func NewBacktestDataProviderProvider(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	config historical.BacktestConfig,
) *BacktestDataProviderProvider {
	return &BacktestDataProviderProvider{
		lazyProvider: lazy.NewLazyProvider(
			"backtest-data-provider",
			func(logger *zap.Logger) (*historical.BacktestDataProvider, error) {
				logger.Info("Initializing backtest data provider")
				startTime := time.Now()
				
				// This is typically an expensive operation that loads historical data
				provider, err := historical.NewBacktestDataProvider(config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Backtest data provider initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("start_date", config.StartDate),
					zap.String("end_date", config.EndDate),
					zap.Int("symbols", len(config.Symbols)),
				)
				
				return provider, nil
			},
			logger,
			metrics,
		),
		logger: logger,
	}
}

// Get returns the backtest data provider, initializing it if necessary
func (p *BacktestDataProviderProvider) Get() (*historical.BacktestDataProvider, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*historical.BacktestDataProvider), nil
}

// GetWithContext returns the backtest data provider with context timeout
func (p *BacktestDataProviderProvider) GetWithContext(ctx context.Context) (*historical.BacktestDataProvider, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		provider *historical.BacktestDataProvider
		err      error
	})

	// Get the provider in a goroutine
	go func() {
		instance, err := p.lazyProvider.Get()
		if err != nil {
			resultCh <- struct {
				provider *historical.BacktestDataProvider
				err      error
			}{nil, err}
			return
		}

		resultCh <- struct {
			provider *historical.BacktestDataProvider
			err      error
		}{instance.(*historical.BacktestDataProvider), nil}
	}()

	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.provider, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsInitialized returns whether the provider has been initialized
func (p *BacktestDataProviderProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

