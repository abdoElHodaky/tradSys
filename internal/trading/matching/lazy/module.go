package lazy

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides fx options for lazy loading matching components
var Module = fx.Options(
	fx.Provide(
		NewMatchingEngineProvider,
		NewOrderBookProviderFactory,
		NewMatchingAlgorithmProviderFactory,
	),
)

// OrderBookProviderFactory creates order book providers
type OrderBookProviderFactory struct {
	logger  *zap.Logger
	metrics *lazy.LazyLoadingMetrics
	config  matching.OrderBookConfig
}

// NewOrderBookProviderFactory creates a new order book provider factory
func NewOrderBookProviderFactory(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	config matching.OrderBookConfig,
) *OrderBookProviderFactory {
	return &OrderBookProviderFactory{
		logger:  logger,
		metrics: metrics,
		config:  config,
	}
}

// CreateProvider creates an order book provider for a symbol
func (f *OrderBookProviderFactory) CreateProvider(symbol string) *OrderBookProvider {
	return NewOrderBookProvider(f.logger, f.metrics, symbol, f.config)
}

// MatchingAlgorithmProviderFactory creates matching algorithm providers
type MatchingAlgorithmProviderFactory struct {
	logger  *zap.Logger
	metrics *lazy.LazyLoadingMetrics
	config  matching.AlgorithmConfig
}

// NewMatchingAlgorithmProviderFactory creates a new matching algorithm provider factory
func NewMatchingAlgorithmProviderFactory(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	config matching.AlgorithmConfig,
) *MatchingAlgorithmProviderFactory {
	return &MatchingAlgorithmProviderFactory{
		logger:  logger,
		metrics: metrics,
		config:  config,
	}
}

// CreateProvider creates a matching algorithm provider for an algorithm type
func (f *MatchingAlgorithmProviderFactory) CreateProvider(algorithmType string) *MatchingAlgorithmProvider {
	return NewMatchingAlgorithmProvider(f.logger, f.metrics, algorithmType, f.config)
}

// ProvideMatchingEngine provides a matching engine via lazy loading
func ProvideMatchingEngine(
	provider *MatchingEngineProvider,
	logger *zap.Logger,
) (*matching.Engine, error) {
	logger.Debug("Providing matching engine via lazy loading")
	return provider.Get()
}

// ProvideOrderBook provides an order book via lazy loading
func ProvideOrderBook(
	factory *OrderBookProviderFactory,
	symbol string,
	logger *zap.Logger,
) (*matching.OrderBook, error) {
	logger.Debug("Providing order book via lazy loading", zap.String("symbol", symbol))
	provider := factory.CreateProvider(symbol)
	return provider.Get()
}

// ProvideMatchingAlgorithm provides a matching algorithm via lazy loading
func ProvideMatchingAlgorithm(
	factory *MatchingAlgorithmProviderFactory,
	algorithmType string,
	logger *zap.Logger,
) (matching.MatchingAlgorithm, error) {
	logger.Debug("Providing matching algorithm via lazy loading", 
		zap.String("type", algorithmType))
	provider := factory.CreateProvider(algorithmType)
	return provider.Get()
}

// WithLazyMatchingEngine provides an fx option for lazy loading matching engine
func WithLazyMatchingEngine() fx.Option {
	return fx.Provide(ProvideMatchingEngine)
}

// WithLazyOrderBook provides an fx option for lazy loading order book
func WithLazyOrderBook(symbol string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   "order_book_" + symbol,
			Target: func(factory *OrderBookProviderFactory, logger *zap.Logger) (*matching.OrderBook, error) {
				return ProvideOrderBook(factory, symbol, logger)
			},
		},
	)
}

// WithLazyMatchingAlgorithm provides an fx option for lazy loading matching algorithm
func WithLazyMatchingAlgorithm(algorithmType string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   "matching_algorithm_" + algorithmType,
			Target: func(factory *MatchingAlgorithmProviderFactory, logger *zap.Logger) (matching.MatchingAlgorithm, error) {
				return ProvideMatchingAlgorithm(factory, algorithmType, logger)
			},
		},
	)
}

// RegisterMetrics registers metrics for lazy loading matching components
func RegisterMetrics(metrics *lazy.LazyLoadingMetrics) {
	// Register component names for metrics tracking
	metrics.RegisterComponent("matching-engine")
	
	// Common order book symbols
	symbols := []string{"BTC-USD", "ETH-USD", "XRP-USD", "LTC-USD", "BCH-USD"}
	for _, symbol := range symbols {
		metrics.RegisterComponent("order-book-" + symbol)
	}
	
	// Common algorithm types
	algorithmTypes := []string{"price-time", "pro-rata", "iceberg"}
	for _, algorithmType := range algorithmTypes {
		metrics.RegisterComponent("matching-algorithm-" + algorithmType)
	}
}

