package lazy

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
	"go.uber.org/zap"
)

// MatchingEngineProvider provides lazy loading for matching engines
type MatchingEngineProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
}

// NewMatchingEngineProvider creates a new provider for matching engines
func NewMatchingEngineProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	config matching.EngineConfig,
) *MatchingEngineProvider {
	return &MatchingEngineProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"matching-engine",
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing matching engine")
				startTime := time.Now()
				
				// This is typically an expensive operation
				engine, err := matching.NewMatchingEngine(config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Matching engine initialized",
					zap.Duration("duration", time.Since(startTime)))
				
				return engine, nil
			},
			logger,
			metrics,
			lazy.WithPriority(10), // Highest priority (lowest number)
			lazy.WithTimeout(60*time.Second),
			lazy.WithMemoryEstimate(200*1024*1024), // 200MB estimate
		),
		logger: logger,
	}
}

// Get returns the matching engine, initializing it if necessary
func (p *MatchingEngineProvider) Get() (*matching.MatchingEngine, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*matching.MatchingEngine), nil
}

// GetWithContext returns the matching engine with context timeout
func (p *MatchingEngineProvider) GetWithContext(ctx context.Context) (*matching.MatchingEngine, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(*matching.MatchingEngine), nil
}

// IsInitialized returns whether the matching engine has been initialized
func (p *MatchingEngineProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// OrderBookProvider provides lazy loading for order books
type OrderBookProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
	symbol       string
}

// NewOrderBookProvider creates a new provider for order books
func NewOrderBookProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	symbol string,
	config matching.OrderBookConfig,
) *OrderBookProvider {
	return &OrderBookProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"order-book-"+symbol,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing order book", 
					zap.String("symbol", symbol))
				startTime := time.Now()
				
				// This is typically an expensive operation
				orderBook, err := matching.NewOrderBook(symbol, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Order book initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("symbol", symbol))
				
				return orderBook, nil
			},
			logger,
			metrics,
			lazy.WithPriority(20), // High priority
			lazy.WithTimeout(30*time.Second),
			lazy.WithMemoryEstimate(50*1024*1024), // 50MB estimate
		),
		logger: logger,
		symbol: symbol,
	}
}

// Get returns the order book, initializing it if necessary
func (p *OrderBookProvider) Get() (*matching.OrderBook, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*matching.OrderBook), nil
}

// GetWithContext returns the order book with context timeout
func (p *OrderBookProvider) GetWithContext(ctx context.Context) (*matching.OrderBook, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(*matching.OrderBook), nil
}

// IsInitialized returns whether the order book has been initialized
func (p *OrderBookProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetSymbol returns the symbol
func (p *OrderBookProvider) GetSymbol() string {
	return p.symbol
}

// MatchingAlgorithmProvider provides lazy loading for matching algorithms
type MatchingAlgorithmProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
	algorithmType string
}

// NewMatchingAlgorithmProvider creates a new provider for matching algorithms
func NewMatchingAlgorithmProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	algorithmType string,
	config matching.AlgorithmConfig,
) *MatchingAlgorithmProvider {
	return &MatchingAlgorithmProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"matching-algorithm-"+algorithmType,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing matching algorithm", 
					zap.String("type", algorithmType))
				startTime := time.Now()
				
				// This is typically an expensive operation
				algorithm, err := matching.NewMatchingAlgorithm(algorithmType, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Matching algorithm initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("type", algorithmType))
				
				return algorithm, nil
			},
			logger,
			metrics,
			lazy.WithPriority(30), // Medium priority
			lazy.WithTimeout(20*time.Second),
			lazy.WithMemoryEstimate(20*1024*1024), // 20MB estimate
		),
		logger:        logger,
		algorithmType: algorithmType,
	}
}

// Get returns the matching algorithm, initializing it if necessary
func (p *MatchingAlgorithmProvider) Get() (matching.MatchingAlgorithm, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(matching.MatchingAlgorithm), nil
}

// GetWithContext returns the matching algorithm with context timeout
func (p *MatchingAlgorithmProvider) GetWithContext(ctx context.Context) (matching.MatchingAlgorithm, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(matching.MatchingAlgorithm), nil
}

// IsInitialized returns whether the matching algorithm has been initialized
func (p *MatchingAlgorithmProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetAlgorithmType returns the algorithm type
func (p *MatchingAlgorithmProvider) GetAlgorithmType() string {
	return p.algorithmType
}

// ExecutionHandlerProvider provides lazy loading for execution handlers
type ExecutionHandlerProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
	handlerType  string
}

// NewExecutionHandlerProvider creates a new provider for execution handlers
func NewExecutionHandlerProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	handlerType string,
	config matching.ExecutionHandlerConfig,
) *ExecutionHandlerProvider {
	return &ExecutionHandlerProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"execution-handler-"+handlerType,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing execution handler", 
					zap.String("type", handlerType))
				startTime := time.Now()
				
				// This is typically an expensive operation
				handler, err := matching.NewExecutionHandler(handlerType, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Execution handler initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("type", handlerType))
				
				return handler, nil
			},
			logger,
			metrics,
			lazy.WithPriority(25), // Medium-high priority
			lazy.WithTimeout(25*time.Second),
			lazy.WithMemoryEstimate(30*1024*1024), // 30MB estimate
		),
		logger:      logger,
		handlerType: handlerType,
	}
}

// Get returns the execution handler, initializing it if necessary
func (p *ExecutionHandlerProvider) Get() (matching.ExecutionHandler, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(matching.ExecutionHandler), nil
}

// GetWithContext returns the execution handler with context timeout
func (p *ExecutionHandlerProvider) GetWithContext(ctx context.Context) (matching.ExecutionHandler, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(matching.ExecutionHandler), nil
}

// IsInitialized returns whether the execution handler has been initialized
func (p *ExecutionHandlerProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetHandlerType returns the handler type
func (p *ExecutionHandlerProvider) GetHandlerType() string {
	return p.handlerType
}

