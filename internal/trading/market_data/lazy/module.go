package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides fx components for lazy-loaded market data
var Module = fx.Options(
	fx.Provide(NewMarketDataModule),
	fx.Provide(NewMarketDataServiceProvider),
	fx.Provide(NewDataSourceRegistry),
	fx.Provide(NewIndicatorProviderFactory),
	fx.Provide(NewDataTransformerRegistry),
)

// MarketDataModule coordinates lazy loading of market data components
type MarketDataModule struct {
	logger                 *zap.Logger
	metrics                *lazy.AdaptiveMetrics
	initManager            *lazy.InitializationManager
	contextPropagator      *lazy.ContextPropagator
	marketDataProvider     *MarketDataServiceProvider
	dataSourceRegistry     *DataSourceRegistry
	indicatorFactory       *IndicatorProviderFactory
	transformerRegistry    *DataTransformerRegistry
	indicatorProviders     map[string]*IndicatorProvider
	indicatorProvidersMu   sync.RWMutex
}

// NewMarketDataModule creates a new market data module
func NewMarketDataModule(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	initManager *lazy.InitializationManager,
	contextPropagator *lazy.ContextPropagator,
	marketDataProvider *MarketDataServiceProvider,
	dataSourceRegistry *DataSourceRegistry,
	indicatorFactory *IndicatorProviderFactory,
	transformerRegistry *DataTransformerRegistry,
) *MarketDataModule {
	return &MarketDataModule{
		logger:              logger,
		metrics:             metrics,
		initManager:         initManager,
		contextPropagator:   contextPropagator,
		marketDataProvider:  marketDataProvider,
		dataSourceRegistry:  dataSourceRegistry,
		indicatorFactory:    indicatorFactory,
		transformerRegistry: transformerRegistry,
		indicatorProviders:  make(map[string]*IndicatorProvider),
	}
}

// Initialize initializes the market data module
func (m *MarketDataModule) Initialize(ctx context.Context) error {
	m.logger.Info("Initializing market data module")
	startTime := time.Now()
	
	// Register providers with initialization manager
	m.initManager.RegisterProvider(m.marketDataProvider.lazyProvider)
	
	// Register data source providers
	for _, provider := range m.dataSourceRegistry.GetAllProviders() {
		m.initManager.RegisterProvider(provider.lazyProvider)
	}
	
	// Register transformer providers
	for _, provider := range m.transformerRegistry.GetAllProviders() {
		m.initManager.RegisterProvider(provider.lazyProvider)
	}
	
	// Warm up critical components
	warmupCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	m.initManager.WarmupComponents(warmupCtx)
	
	m.logger.Info("Market data module initialized",
		zap.Duration("duration", time.Since(startTime)))
	
	return nil
}

// GetMarketDataService returns the market data service
func (m *MarketDataModule) GetMarketDataService() (*market_data.MarketDataService, error) {
	return m.marketDataProvider.Get()
}

// GetMarketDataServiceWithContext returns the market data service with context
func (m *MarketDataModule) GetMarketDataServiceWithContext(ctx context.Context) (*market_data.MarketDataService, error) {
	return m.marketDataProvider.GetWithContext(ctx)
}

// GetDataSource returns a data source by type
func (m *MarketDataModule) GetDataSource(sourceType string) (market_data.DataSource, error) {
	provider, err := m.dataSourceRegistry.GetProvider(sourceType)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetDataSourceWithContext returns a data source by type with context
func (m *MarketDataModule) GetDataSourceWithContext(ctx context.Context, sourceType string) (market_data.DataSource, error) {
	provider, err := m.dataSourceRegistry.GetProvider(sourceType)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// GetIndicator returns an indicator by type
func (m *MarketDataModule) GetIndicator(indicatorType string) (market_data.Indicator, error) {
	provider, err := m.getOrCreateIndicatorProvider(indicatorType)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetIndicatorWithContext returns an indicator by type with context
func (m *MarketDataModule) GetIndicatorWithContext(ctx context.Context, indicatorType string) (market_data.Indicator, error) {
	provider, err := m.getOrCreateIndicatorProvider(indicatorType)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// getOrCreateIndicatorProvider gets or creates an indicator provider
func (m *MarketDataModule) getOrCreateIndicatorProvider(indicatorType string) (*IndicatorProvider, error) {
	// Check if provider exists
	m.indicatorProvidersMu.RLock()
	provider, ok := m.indicatorProviders[indicatorType]
	m.indicatorProvidersMu.RUnlock()
	
	if ok {
		return provider, nil
	}
	
	// Create new provider
	m.indicatorProvidersMu.Lock()
	defer m.indicatorProvidersMu.Unlock()
	
	// Check again in case another goroutine created it
	provider, ok = m.indicatorProviders[indicatorType]
	if ok {
		return provider, nil
	}
	
	// Create new provider
	provider = m.indicatorFactory.CreateProvider(indicatorType)
	
	// Register with initialization manager
	m.initManager.RegisterProvider(provider.lazyProvider)
	
	// Store provider
	m.indicatorProviders[indicatorType] = provider
	
	return provider, nil
}

// GetDataTransformer returns a data transformer by type
func (m *MarketDataModule) GetDataTransformer(transformerType string) (market_data.DataTransformer, error) {
	provider, err := m.transformerRegistry.GetProvider(transformerType)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetDataTransformerWithContext returns a data transformer by type with context
func (m *MarketDataModule) GetDataTransformerWithContext(ctx context.Context, transformerType string) (market_data.DataTransformer, error) {
	provider, err := m.transformerRegistry.GetProvider(transformerType)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// DataSourceRegistry manages data source providers
type DataSourceRegistry struct {
	logger    *zap.Logger
	metrics   *lazy.AdaptiveMetrics
	providers map[string]*DataSourceProvider
	mu        sync.RWMutex
}

// NewDataSourceRegistry creates a new data source registry
func NewDataSourceRegistry(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
) *DataSourceRegistry {
	return &DataSourceRegistry{
		logger:    logger,
		metrics:   metrics,
		providers: make(map[string]*DataSourceProvider),
	}
}

// RegisterProvider registers a data source provider
func (r *DataSourceRegistry) RegisterProvider(provider *DataSourceProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providers[provider.GetSourceType()] = provider
}

// GetProvider gets a data source provider by type
func (r *DataSourceRegistry) GetProvider(sourceType string) (*DataSourceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, ok := r.providers[sourceType]
	if !ok {
		return nil, fmt.Errorf("data source provider not found: %s", sourceType)
	}
	
	return provider, nil
}

// GetAllProviders gets all data source providers
func (r *DataSourceRegistry) GetAllProviders() []*DataSourceProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]*DataSourceProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	
	return providers
}

// RegisterStandardSources registers standard data sources
func (r *DataSourceRegistry) RegisterStandardSources(config market_data.DataSourceConfig) {
	// Register standard data sources
	standardSources := []string{
		"realtime",
		"historical",
		"consolidated",
		"synthetic",
		"backtest",
	}
	
	for _, sourceType := range standardSources {
		provider := NewDataSourceProvider(r.logger, r.metrics, sourceType, config)
		r.RegisterProvider(provider)
	}
}

// DataTransformerRegistry manages data transformer providers
type DataTransformerRegistry struct {
	logger    *zap.Logger
	metrics   *lazy.AdaptiveMetrics
	providers map[string]*DataTransformerProvider
	mu        sync.RWMutex
}

// NewDataTransformerRegistry creates a new data transformer registry
func NewDataTransformerRegistry(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
) *DataTransformerRegistry {
	return &DataTransformerRegistry{
		logger:    logger,
		metrics:   metrics,
		providers: make(map[string]*DataTransformerProvider),
	}
}

// RegisterProvider registers a data transformer provider
func (r *DataTransformerRegistry) RegisterProvider(provider *DataTransformerProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providers[provider.GetTransformerType()] = provider
}

// GetProvider gets a data transformer provider by type
func (r *DataTransformerRegistry) GetProvider(transformerType string) (*DataTransformerProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, ok := r.providers[transformerType]
	if !ok {
		return nil, fmt.Errorf("data transformer provider not found: %s", transformerType)
	}
	
	return provider, nil
}

// GetAllProviders gets all data transformer providers
func (r *DataTransformerRegistry) GetAllProviders() []*DataTransformerProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]*DataTransformerProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	
	return providers
}

// RegisterStandardTransformers registers standard data transformers
func (r *DataTransformerRegistry) RegisterStandardTransformers(config market_data.TransformerConfig) {
	// Register standard data transformers
	standardTransformers := []string{
		"normalization",
		"aggregation",
		"filtering",
		"sampling",
		"interpolation",
	}
	
	for _, transformerType := range standardTransformers {
		provider := NewDataTransformerProvider(r.logger, r.metrics, transformerType, config)
		r.RegisterProvider(provider)
	}
}

