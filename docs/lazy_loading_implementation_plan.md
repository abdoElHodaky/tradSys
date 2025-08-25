# Lazy and Dynamic Loading Implementation Plan

This document outlines the detailed implementation plan for adding lazy and dynamic loading capabilities to high-priority components in the tradSys platform.

## 1. Risk Reporter Implementation

### Current Implementation
The Risk Reporter (`internal/risk/risk_reporter.go`) is currently initialized at startup and runs on a fixed schedule, consuming resources even when reports are not being generated or viewed.

### Lazy Loading Implementation

#### 1.1 Create Lazy Module

```go
// internal/risk/fx/lazy_module.go
package fx

import (
    "context"
    "github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
    "github.com/abdoElHodaky/tradSys/internal/risk"
    "go.uber.org/fx"
    "go.uber.org/zap"
)

// LazyRiskReporterModule provides a lazily loaded risk reporter
var LazyRiskReporterModule = fx.Options(
    // Provide lazily loaded risk reporter
    provideLazyRiskReporter,
    
    // Register lifecycle hooks
    fx.Invoke(registerLazyRiskReporterHooks),
)

// provideLazyRiskReporter provides a lazily loaded risk reporter
func provideLazyRiskReporter(
    logger *zap.Logger,
    metrics *lazy.LazyLoadingMetrics,
) *lazy.LazyProvider {
    return lazy.NewLazyProvider(
        "risk-reporter",
        func(exposureTracker *risk.ExposureTracker, logger *zap.Logger) (*risk.RiskReporter, error) {
            logger.Info("Lazily initializing risk reporter")
            return risk.NewRiskReporter(logger, exposureTracker), nil
        },
        logger,
        metrics,
    )
}

// registerLazyRiskReporterHooks registers lifecycle hooks for the lazy risk reporter
func registerLazyRiskReporterHooks(
    lc fx.Lifecycle,
    logger *zap.Logger,
    riskReporterProvider *lazy.LazyProvider,
) {
    logger.Info("Registering lazy risk reporter hooks")
    
    // Register shutdown hook to clean up resources
    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            // Only clean up if the reporter was initialized
            if !riskReporterProvider.IsInitialized() {
                return nil
            }
            
            // Get the reporter
            instance, err := riskReporterProvider.Get()
            if err != nil {
                logger.Error("Failed to get risk reporter during shutdown", zap.Error(err))
                return err
            }
            
            // Stop the reporter
            reporter := instance.(*risk.RiskReporter)
            return reporter.Stop(ctx)
        },
    })
}

// GetRiskReporter gets the risk reporter, initializing it if necessary
func GetRiskReporter(provider *lazy.LazyProvider) (*risk.RiskReporter, error) {
    instance, err := provider.Get()
    if err != nil {
        return nil, err
    }
    return instance.(*risk.RiskReporter), nil
}
```

#### 1.2 Modify Risk Reporter Service

```go
// internal/risk/service.go (modified)
func (s *Service) GenerateRiskReport(ctx context.Context, accountID string) (*risk.RiskReport, error) {
    // Get the risk reporter lazily
    reporter, err := riskfx.GetRiskReporter(s.riskReporterProvider)
    if err != nil {
        return nil, fmt.Errorf("failed to get risk reporter: %w", err)
    }
    
    // Generate report
    return reporter.GenerateReport(accountID), nil
}
```

#### 1.3 Register in Application Module

```go
// internal/app/app.go (modified)
var Module = fx.Options(
    // Provide lazy loading metrics
    fx.Provide(lazy.NewLazyLoadingMetrics),
    
    // Register lazy modules
    riskfx.LazyRiskReporterModule,
    
    // ... other modules
)
```

## 2. Strategy Optimizer Implementation

### Current Implementation
The Strategy Optimizer (`internal/strategy/optimization/optimizer.go`) is a resource-intensive component that is only used when optimizing trading strategies, which is an infrequent operation.

### Lazy Loading Implementation

#### 2.1 Create Lazy Module

```go
// internal/strategy/optimization/fx/lazy_module.go
package fx

import (
    "context"
    "github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
    "github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
    "github.com/abdoElHodaky/tradSys/internal/strategy"
    "github.com/abdoElHodaky/tradSys/internal/strategy/optimization"
    "go.uber.org/fx"
    "go.uber.org/zap"
)

// LazyStrategyOptimizerModule provides a lazily loaded strategy optimizer
var LazyStrategyOptimizerModule = fx.Options(
    // Provide lazily loaded strategy optimizer
    provideLazyStrategyOptimizer,
    
    // Register lifecycle hooks
    fx.Invoke(registerLazyStrategyOptimizerHooks),
)

// provideLazyStrategyOptimizer provides a lazily loaded strategy optimizer
func provideLazyStrategyOptimizer(
    logger *zap.Logger,
    metrics *lazy.LazyLoadingMetrics,
) *lazy.LazyProvider {
    return lazy.NewLazyProvider(
        "strategy-optimizer",
        func(
            factory strategy.StrategyFactory,
            evaluator *optimization.StrategyEvaluator,
            workerPool *workerpool.WorkerPoolFactory,
            logger *zap.Logger,
        ) (*optimization.StrategyOptimizer, error) {
            logger.Info("Lazily initializing strategy optimizer")
            return optimization.NewStrategyOptimizer(
                factory,
                evaluator,
                workerPool,
                logger,
            ), nil
        },
        logger,
        metrics,
    )
}

// registerLazyStrategyOptimizerHooks registers lifecycle hooks for the lazy strategy optimizer
func registerLazyStrategyOptimizerHooks(
    lc fx.Lifecycle,
    logger *zap.Logger,
    strategyOptimizerProvider *lazy.LazyProvider,
) {
    logger.Info("Registering lazy strategy optimizer hooks")
    
    // No specific cleanup needed for the optimizer, but we could add resource cleanup here if needed
}

// GetStrategyOptimizer gets the strategy optimizer, initializing it if necessary
func GetStrategyOptimizer(provider *lazy.LazyProvider) (*optimization.StrategyOptimizer, error) {
    instance, err := provider.Get()
    if err != nil {
        return nil, err
    }
    return instance.(*optimization.StrategyOptimizer), nil
}
```

#### 2.2 Modify Strategy Service

```go
// internal/strategy/service.go (modified)
func (s *Service) OptimizeStrategy(ctx context.Context, config optimization.OptimizationConfig) (*optimization.OptimizationResult, error) {
    // Get the strategy optimizer lazily
    optimizer, err := optimizationfx.GetStrategyOptimizer(s.strategyOptimizerProvider)
    if err != nil {
        return nil, fmt.Errorf("failed to get strategy optimizer: %w", err)
    }
    
    // Optimize strategy
    return optimizer.Optimize(ctx, config)
}
```

#### 2.3 Register in Application Module

```go
// internal/app/app.go (modified)
var Module = fx.Options(
    // Provide lazy loading metrics
    fx.Provide(lazy.NewLazyLoadingMetrics),
    
    // Register lazy modules
    riskfx.LazyRiskReporterModule,
    optimizationfx.LazyStrategyOptimizerModule,
    
    // ... other modules
)
```

## 3. Order Matching Engine Implementation

### Current Implementation
The Order Matching Engine (`internal/trading/order_matching/engine.go`) creates order books for all symbols at startup, consuming significant memory even for symbols that are not actively traded.

### Lazy Loading Implementation

#### 3.1 Modify Order Matching Engine

```go
// internal/trading/order_matching/engine.go (modified)
package order_matching

import (
    "sync"
    "time"
    
    "github.com/google/uuid"
    "go.uber.org/zap"
)

// Engine represents an order matching engine
type Engine struct {
    // OrderBooks is a map of symbol to order book
    OrderBooks sync.Map // Changed from map to sync.Map for thread safety
    
    // Mutex for thread safety
    mu sync.RWMutex
    
    // Logger
    logger *zap.Logger
    
    // Trade channel
    TradeChannel chan *Trade
}

// NewEngine creates a new order matching engine
func NewEngine(logger *zap.Logger) *Engine {
    return &Engine{
        OrderBooks:   sync.Map{}, // Changed from map to sync.Map
        logger:       logger,
        TradeChannel: make(chan *Trade, 1000),
    }
}

// GetOrderBook gets an order book for a symbol, creating it if it doesn't exist
func (e *Engine) GetOrderBook(symbol string) *OrderBook {
    // Try to get the order book from the map
    if orderBook, exists := e.OrderBooks.Load(symbol); exists {
        return orderBook.(*OrderBook)
    }
    
    // Create a new order book if it doesn't exist
    e.mu.Lock()
    defer e.mu.Unlock()
    
    // Check again in case another goroutine created it while we were waiting for the lock
    if orderBook, exists := e.OrderBooks.Load(symbol); exists {
        return orderBook.(*OrderBook)
    }
    
    // Create a new order book
    orderBook := NewOrderBook(symbol, e.logger)
    e.OrderBooks.Store(symbol, orderBook)
    
    e.logger.Info("Created order book for symbol", zap.String("symbol", symbol))
    
    return orderBook
}

// PlaceOrder places an order
func (e *Engine) PlaceOrder(order *Order) ([]*Trade, error) {
    // Get or create the order book for the symbol
    orderBook := e.GetOrderBook(order.Symbol)
    
    // Add the order to the order book
    trades, err := orderBook.AddOrder(order)
    if err != nil {
        return nil, err
    }
    
    // Send trades to trade channel
    for _, trade := range trades {
        select {
        case e.TradeChannel <- trade:
        default:
            e.logger.Warn("Trade channel full, dropping trade",
                zap.String("trade_id", trade.ID),
                zap.String("symbol", trade.Symbol),
                zap.Float64("price", trade.Price),
                zap.Float64("quantity", trade.Quantity))
        }
    }
    
    return trades, nil
}

// CleanupUnusedOrderBooks removes order books that haven't been used for a while
func (e *Engine) CleanupUnusedOrderBooks(maxAge time.Duration) {
    e.OrderBooks.Range(func(key, value interface{}) bool {
        symbol := key.(string)
        orderBook := value.(*OrderBook)
        
        // Check if the order book has been used recently
        if time.Since(orderBook.LastAccessed) > maxAge {
            e.logger.Info("Removing unused order book",
                zap.String("symbol", symbol),
                zap.Time("last_accessed", orderBook.LastAccessed))
            
            e.OrderBooks.Delete(symbol)
        }
        
        return true
    })
}
```

#### 3.2 Modify OrderBook to Track Last Access

```go
// internal/trading/order_matching/engine.go (modified)
// OrderBook represents an order book for a symbol
type OrderBook struct {
    // Symbol is the trading symbol
    Symbol string
    
    // Bids is the buy orders
    Bids *OrderHeap
    
    // Asks is the sell orders
    Asks *OrderHeap
    
    // Orders is a map of order ID to order
    Orders map[string]*Order
    
    // StopBids is the stop buy orders
    StopBids *OrderHeap
    
    // StopAsks is the stop sell orders
    StopAsks *OrderHeap
    
    // LastPrice is the last traded price
    LastPrice float64
    
    // LastAccessed is the time the order book was last accessed
    LastAccessed time.Time
    
    // Mutex for thread safety
    mu sync.RWMutex
    
    // Logger
    logger *zap.Logger
}

// NewOrderBook creates a new order book
func NewOrderBook(symbol string, logger *zap.Logger) *OrderBook {
    bids := &OrderHeap{
        Orders: make([]*Order, 0),
        Side:   OrderSideBuy,
    }
    asks := &OrderHeap{
        Orders: make([]*Order, 0),
        Side:   OrderSideSell,
    }
    stopBids := &OrderHeap{
        Orders: make([]*Order, 0),
        Side:   OrderSideBuy,
    }
    stopAsks := &OrderHeap{
        Orders: make([]*Order, 0),
        Side:   OrderSideSell,
    }
    heap.Init(bids)
    heap.Init(asks)
    heap.Init(stopBids)
    heap.Init(stopAsks)
    
    return &OrderBook{
        Symbol:       symbol,
        Bids:         bids,
        Asks:         asks,
        Orders:       make(map[string]*Order),
        StopBids:     stopBids,
        StopAsks:     stopAsks,
        LastPrice:    0,
        LastAccessed: time.Now(),
        logger:       logger,
    }
}

// Update all methods to update LastAccessed
func (ob *OrderBook) AddOrder(order *Order) ([]*Trade, error) {
    ob.mu.Lock()
    defer ob.mu.Unlock()
    
    // Update last accessed time
    ob.LastAccessed = time.Now()
    
    // Rest of the method remains the same
    // ...
}
```

#### 3.3 Create a Cleanup Service

```go
// internal/trading/order_matching/cleanup_service.go
package order_matching

import (
    "context"
    "time"
    
    "go.uber.org/zap"
)

// CleanupService periodically cleans up unused order books
type CleanupService struct {
    engine *Engine
    logger *zap.Logger
    
    // Cleanup interval
    interval time.Duration
    
    // Maximum age of unused order books
    maxAge time.Duration
    
    // Context for cancellation
    ctx context.Context
    cancel context.CancelFunc
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(engine *Engine, logger *zap.Logger) *CleanupService {
    ctx, cancel := context.WithCancel(context.Background())
    
    return &CleanupService{
        engine:   engine,
        logger:   logger,
        interval: 1 * time.Hour,
        maxAge:   24 * time.Hour,
        ctx:      ctx,
        cancel:   cancel,
    }
}

// Start starts the cleanup service
func (s *CleanupService) Start() {
    s.logger.Info("Starting order book cleanup service",
        zap.Duration("interval", s.interval),
        zap.Duration("max_age", s.maxAge))
    
    go func() {
        ticker := time.NewTicker(s.interval)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                s.engine.CleanupUnusedOrderBooks(s.maxAge)
            case <-s.ctx.Done():
                return
            }
        }
    }()
}

// Stop stops the cleanup service
func (s *CleanupService) Stop() {
    s.logger.Info("Stopping order book cleanup service")
    s.cancel()
}
```

#### 3.4 Register in Application Module

```go
// internal/app/app.go (modified)
var Module = fx.Options(
    // Provide order matching engine
    fx.Provide(
        func(logger *zap.Logger) *order_matching.Engine {
            return order_matching.NewEngine(logger)
        },
    ),
    
    // Provide cleanup service
    fx.Provide(
        func(engine *order_matching.Engine, logger *zap.Logger) *order_matching.CleanupService {
            return order_matching.NewCleanupService(engine, logger)
        },
    ),
    
    // Register lifecycle hooks
    fx.Invoke(
        func(lc fx.Lifecycle, cleanupService *order_matching.CleanupService) {
            lc.Append(fx.Hook{
                OnStart: func(ctx context.Context) error {
                    cleanupService.Start()
                    return nil
                },
                OnStop: func(ctx context.Context) error {
                    cleanupService.Stop()
                    return nil
                },
            })
        },
    ),
    
    // ... other modules
)
```

## 4. Configuration-Driven Loading

### 4.1 Create Configuration Options

```go
// internal/config/lazy_loading_config.go
package config

// LazyLoadingConfig contains configuration for lazy loading
type LazyLoadingConfig struct {
    // EnableLazyLoading enables lazy loading
    EnableLazyLoading bool `json:"enable_lazy_loading" yaml:"enable_lazy_loading"`
    
    // Components to lazy load
    Components LazyLoadingComponentsConfig `json:"components" yaml:"components"`
    
    // Metrics configuration
    Metrics LazyLoadingMetricsConfig `json:"metrics" yaml:"metrics"`
}

// LazyLoadingComponentsConfig contains configuration for lazy loading components
type LazyLoadingComponentsConfig struct {
    // RiskReporter enables lazy loading for risk reporter
    RiskReporter bool `json:"risk_reporter" yaml:"risk_reporter"`
    
    // StrategyOptimizer enables lazy loading for strategy optimizer
    StrategyOptimizer bool `json:"strategy_optimizer" yaml:"strategy_optimizer"`
    
    // OrderMatchingEngine enables lazy loading for order matching engine
    OrderMatchingEngine bool `json:"order_matching_engine" yaml:"order_matching_engine"`
    
    // RiskValidator enables lazy loading for risk validator
    RiskValidator bool `json:"risk_validator" yaml:"risk_validator"`
    
    // StrategyFactory enables lazy loading for strategy factory
    StrategyFactory bool `json:"strategy_factory" yaml:"strategy_factory"`
}

// LazyLoadingMetricsConfig contains configuration for lazy loading metrics
type LazyLoadingMetricsConfig struct {
    // EnableMetrics enables metrics collection
    EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`
    
    // ReportInterval is the interval for reporting metrics
    ReportInterval string `json:"report_interval" yaml:"report_interval"`
}
```

### 4.2 Modify Application Module to Use Configuration

```go
// internal/app/app.go (modified)
var Module = fx.Options(
    // Provide lazy loading metrics
    fx.Provide(
        func(config *config.Config, logger *zap.Logger) *lazy.LazyLoadingMetrics {
            if !config.LazyLoading.EnableLazyLoading || !config.LazyLoading.Metrics.EnableMetrics {
                return nil
            }
            
            return lazy.NewLazyLoadingMetrics()
        },
    ),
    
    // Register lazy modules conditionally
    fx.Invoke(
        func(config *config.Config) fx.Option {
            if !config.LazyLoading.EnableLazyLoading {
                return fx.Options()
            }
            
            var options []fx.Option
            
            if config.LazyLoading.Components.RiskReporter {
                options = append(options, riskfx.LazyRiskReporterModule)
            }
            
            if config.LazyLoading.Components.StrategyOptimizer {
                options = append(options, optimizationfx.LazyStrategyOptimizerModule)
            }
            
            // Add other components
            
            return fx.Options(options...)
        },
    ),
    
    // ... other modules
)
```

## 5. Monitoring and Metrics

### 5.1 Enhance LazyLoadingMetrics

```go
// internal/architecture/fx/lazy/metrics.go (modified)
package lazy

import (
    "sync"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
)

// LazyLoadingMetrics collects metrics for lazy loading
type LazyLoadingMetrics struct {
    mu                  sync.RWMutex
    initializations     map[string]int64
    initializationErr   map[string]int64
    initializationTimes map[string][]time.Duration
    
    // Prometheus metrics
    initializationCount    *prometheus.CounterVec
    initializationErrorCount *prometheus.CounterVec
    initializationDuration *prometheus.HistogramVec
    memoryUsage           *prometheus.GaugeVec
}

// NewLazyLoadingMetrics creates a new LazyLoadingMetrics
func NewLazyLoadingMetrics() *LazyLoadingMetrics {
    metrics := &LazyLoadingMetrics{
        initializations:     make(map[string]int64),
        initializationErr:   make(map[string]int64),
        initializationTimes: make(map[string][]time.Duration),
        
        initializationCount: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "lazy_loading_initialization_count",
                Help: "Number of lazy loading initializations",
            },
            []string{"component"},
        ),
        
        initializationErrorCount: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "lazy_loading_initialization_error_count",
                Help: "Number of lazy loading initialization errors",
            },
            []string{"component"},
        ),
        
        initializationDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "lazy_loading_initialization_duration_seconds",
                Help: "Duration of lazy loading initializations",
                Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
            },
            []string{"component"},
        ),
        
        memoryUsage: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "lazy_loading_memory_usage_bytes",
                Help: "Memory usage of lazily loaded components",
            },
            []string{"component"},
        ),
    }
    
    // Register metrics with Prometheus
    prometheus.MustRegister(
        metrics.initializationCount,
        metrics.initializationErrorCount,
        metrics.initializationDuration,
        metrics.memoryUsage,
    )
    
    return metrics
}

// RecordInitialization records a successful initialization
func (m *LazyLoadingMetrics) RecordInitialization(component string, duration time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.initializations[component]++
    m.initializationTimes[component] = append(m.initializationTimes[component], duration)
    
    // Update Prometheus metrics
    m.initializationCount.WithLabelValues(component).Inc()
    m.initializationDuration.WithLabelValues(component).Observe(duration.Seconds())
}

// RecordInitializationError records an initialization error
func (m *LazyLoadingMetrics) RecordInitializationError(component string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.initializationErr[component]++
    
    // Update Prometheus metrics
    m.initializationErrorCount.WithLabelValues(component).Inc()
}

// RecordMemoryUsage records memory usage for a component
func (m *LazyLoadingMetrics) RecordMemoryUsage(component string, bytes int64) {
    // Update Prometheus metrics
    m.memoryUsage.WithLabelValues(component).Set(float64(bytes))
}

// GetInitializationCount gets the initialization count for a component
func (m *LazyLoadingMetrics) GetInitializationCount(component string) int64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    return m.initializations[component]
}

// GetInitializationErrorCount gets the initialization error count for a component
func (m *LazyLoadingMetrics) GetInitializationErrorCount(component string) int64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    return m.initializationErr[component]
}

// GetAverageInitializationTime gets the average initialization time for a component
func (m *LazyLoadingMetrics) GetAverageInitializationTime(component string) time.Duration {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    times := m.initializationTimes[component]
    if len(times) == 0 {
        return 0
    }
    
    var total time.Duration
    for _, t := range times {
        total += t
    }
    
    return total / time.Duration(len(times))
}
```

## 6. Implementation Timeline

### Phase 1: Core Framework Enhancement (Week 1-2)
- Enhance the lazy loading framework
- Add metrics and monitoring
- Implement configuration-driven loading

### Phase 2: High-Priority Components (Week 3-4)
- Implement lazy loading for Risk Reporter
- Implement lazy loading for Strategy Optimizer
- Implement lazy loading for Order Matching Engine

### Phase 3: Medium-Priority Components (Week 5-6)
- Implement lazy loading for Risk Validator
- Implement lazy loading for Strategy Factory

### Phase 4: Testing and Optimization (Week 7-8)
- Comprehensive testing of lazy loading
- Performance benchmarking
- Memory usage optimization
- Documentation and examples

