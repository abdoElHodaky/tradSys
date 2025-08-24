package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides fx components for lazy-loaded matching engine
var Module = fx.Options(
	fx.Provide(NewMatchingModule),
	fx.Provide(NewMatchingEngineProvider),
	fx.Provide(NewOrderBookRegistry),
	fx.Provide(NewMatchingAlgorithmRegistry),
	fx.Provide(NewExecutionHandlerRegistry),
)

// MatchingModule coordinates lazy loading of matching engine components
type MatchingModule struct {
	logger                 *zap.Logger
	metrics                *lazy.AdaptiveMetrics
	initManager            *lazy.InitializationManager
	contextPropagator      *lazy.ContextPropagator
	matchingEngineProvider *MatchingEngineProvider
	orderBookRegistry      *OrderBookRegistry
	algorithmRegistry      *MatchingAlgorithmRegistry
	handlerRegistry        *ExecutionHandlerRegistry
	resourceManager        *ResourceManager
}

// NewMatchingModule creates a new matching module
func NewMatchingModule(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	initManager *lazy.InitializationManager,
	contextPropagator *lazy.ContextPropagator,
	matchingEngineProvider *MatchingEngineProvider,
	orderBookRegistry *OrderBookRegistry,
	algorithmRegistry *MatchingAlgorithmRegistry,
	handlerRegistry *ExecutionHandlerRegistry,
) *MatchingModule {
	return &MatchingModule{
		logger:                 logger,
		metrics:                metrics,
		initManager:            initManager,
		contextPropagator:      contextPropagator,
		matchingEngineProvider: matchingEngineProvider,
		orderBookRegistry:      orderBookRegistry,
		algorithmRegistry:      algorithmRegistry,
		handlerRegistry:        handlerRegistry,
		resourceManager:        NewResourceManager(logger, metrics),
	}
}

// Initialize initializes the matching module
func (m *MatchingModule) Initialize(ctx context.Context) error {
	m.logger.Info("Initializing matching module")
	startTime := time.Now()
	
	// Register providers with initialization manager
	m.initManager.RegisterProvider(m.matchingEngineProvider.lazyProvider)
	
	// Register order book providers
	for _, provider := range m.orderBookRegistry.GetAllProviders() {
		m.initManager.RegisterProvider(provider.lazyProvider)
	}
	
	// Register algorithm providers
	for _, provider := range m.algorithmRegistry.GetAllProviders() {
		m.initManager.RegisterProvider(provider.lazyProvider)
	}
	
	// Register handler providers
	for _, provider := range m.handlerRegistry.GetAllProviders() {
		m.initManager.RegisterProvider(provider.lazyProvider)
	}
	
	// Start resource manager cleanup
	m.resourceManager.StartCleanup(ctx)
	
	// Warm up critical components
	warmupCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	m.initManager.WarmupComponents(warmupCtx)
	
	m.logger.Info("Matching module initialized",
		zap.Duration("duration", time.Since(startTime)))
	
	return nil
}

// GetMatchingEngine returns the matching engine
func (m *MatchingModule) GetMatchingEngine() (*matching.MatchingEngine, error) {
	return m.matchingEngineProvider.Get()
}

// GetMatchingEngineWithContext returns the matching engine with context
func (m *MatchingModule) GetMatchingEngineWithContext(ctx context.Context) (*matching.MatchingEngine, error) {
	return m.matchingEngineProvider.GetWithContext(ctx)
}

// GetOrderBook returns an order book by symbol
func (m *MatchingModule) GetOrderBook(symbol string) (*matching.OrderBook, error) {
	provider, err := m.orderBookRegistry.GetProvider(symbol)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetOrderBookWithContext returns an order book by symbol with context
func (m *MatchingModule) GetOrderBookWithContext(ctx context.Context, symbol string) (*matching.OrderBook, error) {
	provider, err := m.orderBookRegistry.GetProvider(symbol)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// GetMatchingAlgorithm returns a matching algorithm by type
func (m *MatchingModule) GetMatchingAlgorithm(algorithmType string) (matching.MatchingAlgorithm, error) {
	provider, err := m.algorithmRegistry.GetProvider(algorithmType)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetMatchingAlgorithmWithContext returns a matching algorithm by type with context
func (m *MatchingModule) GetMatchingAlgorithmWithContext(ctx context.Context, algorithmType string) (matching.MatchingAlgorithm, error) {
	provider, err := m.algorithmRegistry.GetProvider(algorithmType)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// GetExecutionHandler returns an execution handler by type
func (m *MatchingModule) GetExecutionHandler(handlerType string) (matching.ExecutionHandler, error) {
	provider, err := m.handlerRegistry.GetProvider(handlerType)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetExecutionHandlerWithContext returns an execution handler by type with context
func (m *MatchingModule) GetExecutionHandlerWithContext(ctx context.Context, handlerType string) (matching.ExecutionHandler, error) {
	provider, err := m.handlerRegistry.GetProvider(handlerType)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// Shutdown shuts down the matching module
func (m *MatchingModule) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down matching module")
	
	// Stop resource manager cleanup
	m.resourceManager.StopCleanup()
	
	return nil
}

// OrderBookRegistry manages order book providers
type OrderBookRegistry struct {
	logger    *zap.Logger
	metrics   *lazy.AdaptiveMetrics
	providers map[string]*OrderBookProvider
	mu        sync.RWMutex
}

// NewOrderBookRegistry creates a new order book registry
func NewOrderBookRegistry(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
) *OrderBookRegistry {
	return &OrderBookRegistry{
		logger:    logger,
		metrics:   metrics,
		providers: make(map[string]*OrderBookProvider),
	}
}

// RegisterProvider registers an order book provider
func (r *OrderBookRegistry) RegisterProvider(provider *OrderBookProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providers[provider.GetSymbol()] = provider
}

// GetProvider gets an order book provider by symbol
func (r *OrderBookRegistry) GetProvider(symbol string) (*OrderBookProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, ok := r.providers[symbol]
	if !ok {
		return nil, fmt.Errorf("order book provider not found: %s", symbol)
	}
	
	return provider, nil
}

// GetAllProviders gets all order book providers
func (r *OrderBookRegistry) GetAllProviders() []*OrderBookProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]*OrderBookProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	
	return providers
}

// MatchingAlgorithmRegistry manages matching algorithm providers
type MatchingAlgorithmRegistry struct {
	logger    *zap.Logger
	metrics   *lazy.AdaptiveMetrics
	providers map[string]*MatchingAlgorithmProvider
	mu        sync.RWMutex
}

// NewMatchingAlgorithmRegistry creates a new matching algorithm registry
func NewMatchingAlgorithmRegistry(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
) *MatchingAlgorithmRegistry {
	return &MatchingAlgorithmRegistry{
		logger:    logger,
		metrics:   metrics,
		providers: make(map[string]*MatchingAlgorithmProvider),
	}
}

// RegisterProvider registers a matching algorithm provider
func (r *MatchingAlgorithmRegistry) RegisterProvider(provider *MatchingAlgorithmProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providers[provider.GetAlgorithmType()] = provider
}

// GetProvider gets a matching algorithm provider by type
func (r *MatchingAlgorithmRegistry) GetProvider(algorithmType string) (*MatchingAlgorithmProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, ok := r.providers[algorithmType]
	if !ok {
		return nil, fmt.Errorf("matching algorithm provider not found: %s", algorithmType)
	}
	
	return provider, nil
}

// GetAllProviders gets all matching algorithm providers
func (r *MatchingAlgorithmRegistry) GetAllProviders() []*MatchingAlgorithmProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]*MatchingAlgorithmProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	
	return providers
}

// RegisterStandardAlgorithms registers standard matching algorithms
func (r *MatchingAlgorithmRegistry) RegisterStandardAlgorithms(config matching.AlgorithmConfig) {
	// Register standard matching algorithms
	standardAlgorithms := []string{
		"price-time",
		"pro-rata",
		"iceberg",
		"market-maker",
		"auction",
	}
	
	for _, algorithmType := range standardAlgorithms {
		provider := NewMatchingAlgorithmProvider(r.logger, r.metrics, algorithmType, config)
		r.RegisterProvider(provider)
	}
}

// ExecutionHandlerRegistry manages execution handler providers
type ExecutionHandlerRegistry struct {
	logger    *zap.Logger
	metrics   *lazy.AdaptiveMetrics
	providers map[string]*ExecutionHandlerProvider
	mu        sync.RWMutex
}

// NewExecutionHandlerRegistry creates a new execution handler registry
func NewExecutionHandlerRegistry(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
) *ExecutionHandlerRegistry {
	return &ExecutionHandlerRegistry{
		logger:    logger,
		metrics:   metrics,
		providers: make(map[string]*ExecutionHandlerProvider),
	}
}

// RegisterProvider registers an execution handler provider
func (r *ExecutionHandlerRegistry) RegisterProvider(provider *ExecutionHandlerProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providers[provider.GetHandlerType()] = provider
}

// GetProvider gets an execution handler provider by type
func (r *ExecutionHandlerRegistry) GetProvider(handlerType string) (*ExecutionHandlerProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, ok := r.providers[handlerType]
	if !ok {
		return nil, fmt.Errorf("execution handler provider not found: %s", handlerType)
	}
	
	return provider, nil
}

// GetAllProviders gets all execution handler providers
func (r *ExecutionHandlerRegistry) GetAllProviders() []*ExecutionHandlerProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]*ExecutionHandlerProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	
	return providers
}

// RegisterStandardHandlers registers standard execution handlers
func (r *ExecutionHandlerRegistry) RegisterStandardHandlers(config matching.ExecutionHandlerConfig) {
	// Register standard execution handlers
	standardHandlers := []string{
		"standard",
		"delayed",
		"batched",
		"conditional",
		"throttled",
	}
	
	for _, handlerType := range standardHandlers {
		provider := NewExecutionHandlerProvider(r.logger, r.metrics, handlerType, config)
		r.RegisterProvider(provider)
	}
}

