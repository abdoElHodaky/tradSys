package lazy

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
	"go.uber.org/zap"
)

// MatchingEngineProvider provides lazy loading for matching engine
type MatchingEngineProvider struct {
	lazyProvider *lazy.LazyProvider
	logger       *zap.Logger
}

// NewMatchingEngineProvider creates a new provider
func NewMatchingEngineProvider(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	config matching.EngineConfig,
) *MatchingEngineProvider {
	return &MatchingEngineProvider{
		lazyProvider: lazy.NewLazyProvider(
			"matching-engine",
			func(logger *zap.Logger) (*matching.Engine, error) {
				logger.Info("Initializing matching engine")
				startTime := time.Now()
				
				// This is typically an expensive operation
				engine, err := matching.NewEngine(config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Matching engine initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.Int("markets", len(config.Markets)),
				)
				
				return engine, nil
			},
			logger,
			metrics,
		),
		logger: logger,
	}
}

// Get returns the matching engine, initializing it if necessary
func (p *MatchingEngineProvider) Get() (*matching.Engine, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*matching.Engine), nil
}

// GetWithContext returns the matching engine with context timeout
func (p *MatchingEngineProvider) GetWithContext(ctx context.Context) (*matching.Engine, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		engine *matching.Engine
		err    error
	})

	// Get the engine in a goroutine
	go func() {
		instance, err := p.lazyProvider.Get()
		if err != nil {
			resultCh <- struct {
				engine *matching.Engine
				err    error
			}{nil, err}
			return
		}

		resultCh <- struct {
			engine *matching.Engine
			err    error
		}{instance.(*matching.Engine), nil}
	}()

	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.engine, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsInitialized returns whether the engine has been initialized
func (p *MatchingEngineProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// OrderBookProvider provides lazy loading for order book
type OrderBookProvider struct {
	lazyProvider *lazy.LazyProvider
	logger       *zap.Logger
	symbol       string
}

// NewOrderBookProvider creates a new provider
func NewOrderBookProvider(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	symbol string,
	config matching.OrderBookConfig,
) *OrderBookProvider {
	return &OrderBookProvider{
		lazyProvider: lazy.NewLazyProvider(
			"order-book-"+symbol,
			func(logger *zap.Logger) (*matching.OrderBook, error) {
				logger.Info("Initializing order book", zap.String("symbol", symbol))
				startTime := time.Now()
				
				// This is typically an expensive operation
				orderBook, err := matching.NewOrderBook(symbol, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Order book initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("symbol", symbol),
				)
				
				return orderBook, nil
			},
			logger,
			metrics,
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
	// Create a channel for the result
	resultCh := make(chan struct {
		orderBook *matching.OrderBook
		err       error
	})

	// Get the order book in a goroutine
	go func() {
		instance, err := p.lazyProvider.Get()
		if err != nil {
			resultCh <- struct {
				orderBook *matching.OrderBook
				err       error
			}{nil, err}
			return
		}

		resultCh <- struct {
			orderBook *matching.OrderBook
			err       error
		}{instance.(*matching.OrderBook), nil}
	}()

	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.orderBook, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsInitialized returns whether the order book has been initialized
func (p *OrderBookProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetSymbol returns the symbol for this order book
func (p *OrderBookProvider) GetSymbol() string {
	return p.symbol
}

// MatchingAlgorithmProvider provides lazy loading for matching algorithm
type MatchingAlgorithmProvider struct {
	lazyProvider *lazy.LazyProvider
	logger       *zap.Logger
	algorithmType string
}

// NewMatchingAlgorithmProvider creates a new provider
func NewMatchingAlgorithmProvider(
	logger *zap.Logger,
	metrics *lazy.LazyLoadingMetrics,
	algorithmType string,
	config matching.AlgorithmConfig,
) *MatchingAlgorithmProvider {
	return &MatchingAlgorithmProvider{
		lazyProvider: lazy.NewLazyProvider(
			"matching-algorithm-"+algorithmType,
			func(logger *zap.Logger) (matching.MatchingAlgorithm, error) {
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
					zap.String("type", algorithmType),
				)
				
				return algorithm, nil
			},
			logger,
			metrics,
		),
		logger: logger,
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

// IsInitialized returns whether the algorithm has been initialized
func (p *MatchingAlgorithmProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetAlgorithmType returns the algorithm type
func (p *MatchingAlgorithmProvider) GetAlgorithmType() string {
	return p.algorithmType
}

