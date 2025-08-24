package lazy

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/zap"
)

// MarketDataServiceProvider provides lazy loading for market data services
type MarketDataServiceProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
}

// NewMarketDataServiceProvider creates a new provider for market data service
func NewMarketDataServiceProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	config market_data.ServiceConfig,
) *MarketDataServiceProvider {
	return &MarketDataServiceProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"market-data-service",
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing market data service")
				startTime := time.Now()
				
				// This is typically an expensive operation
				service, err := market_data.NewMarketDataService(config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Market data service initialized",
					zap.Duration("duration", time.Since(startTime)))
				
				return service, nil
			},
			logger,
			metrics,
			lazy.WithPriority(15), // Higher priority (lower number)
			lazy.WithTimeout(60*time.Second),
			lazy.WithMemoryEstimate(100*1024*1024), // 100MB estimate
		),
		logger: logger,
	}
}

// Get returns the market data service, initializing it if necessary
func (p *MarketDataServiceProvider) Get() (*market_data.MarketDataService, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*market_data.MarketDataService), nil
}

// GetWithContext returns the market data service with context timeout
func (p *MarketDataServiceProvider) GetWithContext(ctx context.Context) (*market_data.MarketDataService, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(*market_data.MarketDataService), nil
}

// IsInitialized returns whether the market data service has been initialized
func (p *MarketDataServiceProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// DataSourceProvider provides lazy loading for market data sources
type DataSourceProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
	sourceType   string
}

// NewDataSourceProvider creates a new provider for market data sources
func NewDataSourceProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	sourceType string,
	config market_data.DataSourceConfig,
) *DataSourceProvider {
	return &DataSourceProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"data-source-"+sourceType,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing data source", 
					zap.String("type", sourceType))
				startTime := time.Now()
				
				// This is typically an expensive operation
				source, err := market_data.NewDataSource(sourceType, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Data source initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("type", sourceType))
				
				return source, nil
			},
			logger,
			metrics,
			lazy.WithPriority(25), // Medium priority
			lazy.WithTimeout(30*time.Second),
			lazy.WithMemoryEstimate(30*1024*1024), // 30MB estimate
		),
		logger:     logger,
		sourceType: sourceType,
	}
}

// Get returns the data source, initializing it if necessary
func (p *DataSourceProvider) Get() (market_data.DataSource, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(market_data.DataSource), nil
}

// GetWithContext returns the data source with context timeout
func (p *DataSourceProvider) GetWithContext(ctx context.Context) (market_data.DataSource, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(market_data.DataSource), nil
}

// IsInitialized returns whether the data source has been initialized
func (p *DataSourceProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetSourceType returns the source type
func (p *DataSourceProvider) GetSourceType() string {
	return p.sourceType
}

// IndicatorProviderFactory creates indicator providers
type IndicatorProviderFactory struct {
	logger  *zap.Logger
	metrics *lazy.AdaptiveMetrics
	config  market_data.IndicatorConfig
}

// NewIndicatorProviderFactory creates a new indicator provider factory
func NewIndicatorProviderFactory(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	config market_data.IndicatorConfig,
) *IndicatorProviderFactory {
	return &IndicatorProviderFactory{
		logger:  logger,
		metrics: metrics,
		config:  config,
	}
}

// CreateProvider creates an indicator provider for a specific indicator type
func (f *IndicatorProviderFactory) CreateProvider(indicatorType string) *IndicatorProvider {
	return NewIndicatorProvider(f.logger, f.metrics, indicatorType, f.config)
}

// IndicatorProvider provides lazy loading for market data indicators
type IndicatorProvider struct {
	lazyProvider   *lazy.EnhancedLazyProvider
	logger         *zap.Logger
	indicatorType  string
}

// NewIndicatorProvider creates a new provider for market data indicators
func NewIndicatorProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	indicatorType string,
	config market_data.IndicatorConfig,
) *IndicatorProvider {
	return &IndicatorProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"indicator-"+indicatorType,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing indicator", 
					zap.String("type", indicatorType))
				startTime := time.Now()
				
				// This is typically an expensive operation
				indicator, err := market_data.NewIndicator(indicatorType, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Indicator initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("type", indicatorType))
				
				return indicator, nil
			},
			logger,
			metrics,
			lazy.WithPriority(35), // Lower priority
			lazy.WithTimeout(15*time.Second),
			lazy.WithMemoryEstimate(10*1024*1024), // 10MB estimate
		),
		logger:        logger,
		indicatorType: indicatorType,
	}
}

// Get returns the indicator, initializing it if necessary
func (p *IndicatorProvider) Get() (market_data.Indicator, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(market_data.Indicator), nil
}

// GetWithContext returns the indicator with context timeout
func (p *IndicatorProvider) GetWithContext(ctx context.Context) (market_data.Indicator, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(market_data.Indicator), nil
}

// IsInitialized returns whether the indicator has been initialized
func (p *IndicatorProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetIndicatorType returns the indicator type
func (p *IndicatorProvider) GetIndicatorType() string {
	return p.indicatorType
}

// DataTransformerProvider provides lazy loading for market data transformers
type DataTransformerProvider struct {
	lazyProvider    *lazy.EnhancedLazyProvider
	logger          *zap.Logger
	transformerType string
}

// NewDataTransformerProvider creates a new provider for market data transformers
func NewDataTransformerProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	transformerType string,
	config market_data.TransformerConfig,
) *DataTransformerProvider {
	return &DataTransformerProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"transformer-"+transformerType,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing data transformer", 
					zap.String("type", transformerType))
				startTime := time.Now()
				
				// This is typically an expensive operation
				transformer, err := market_data.NewDataTransformer(transformerType, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Data transformer initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("type", transformerType))
				
				return transformer, nil
			},
			logger,
			metrics,
			lazy.WithPriority(30), // Medium priority
			lazy.WithTimeout(20*time.Second),
			lazy.WithMemoryEstimate(15*1024*1024), // 15MB estimate
		),
		logger:          logger,
		transformerType: transformerType,
	}
}

// Get returns the data transformer, initializing it if necessary
func (p *DataTransformerProvider) Get() (market_data.DataTransformer, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(market_data.DataTransformer), nil
}

// GetWithContext returns the data transformer with context timeout
func (p *DataTransformerProvider) GetWithContext(ctx context.Context) (market_data.DataTransformer, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(market_data.DataTransformer), nil
}

// IsInitialized returns whether the data transformer has been initialized
func (p *DataTransformerProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetTransformerType returns the transformer type
func (p *DataTransformerProvider) GetTransformerType() string {
	return p.transformerType
}

