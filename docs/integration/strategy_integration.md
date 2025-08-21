# Strategy Integration Guide

This guide explains how to integrate trading strategies with the TradSys platform using the fx dependency injection framework.

## Overview

TradSys provides a flexible framework for implementing and integrating trading strategies. Strategies can be implemented by:

1. Implementing the `Strategy` interface
2. Registering the strategy with the `StrategyFactory`
3. Configuring the strategy through the `StrategyConfig` struct

## Implementing a Strategy

### Step 1: Implement the Strategy Interface

The `Strategy` interface defines the contract for all trading strategies:

```go
type Strategy interface {
    // Name returns the name of the strategy
    Name() string

    // Initialize initializes the strategy
    Initialize(ctx context.Context) error

    // Start starts the strategy
    Start(ctx context.Context) error

    // Stop stops the strategy
    Stop(ctx context.Context) error

    // IsRunning returns whether the strategy is running
    IsRunning() bool

    // OnMarketData processes market data updates
    OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error

    // OnOrderUpdate processes order updates
    OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error

    // GetPerformanceMetrics returns performance metrics for the strategy
    GetPerformanceMetrics() map[string]interface{}
}
```

You can implement this interface directly or extend the `BaseStrategy` class which provides common functionality:

```go
// Example strategy implementation
type MyStrategy struct {
    *strategy.BaseStrategy

    // Strategy-specific fields
    symbols        []string
    lookbackPeriod int
    entryThreshold float64
    exitThreshold  float64
    
    // Strategy state
    prices         map[string][]float64
    positions      map[string]float64
    
    // Dependencies
    workerPool     *workerpool.WorkerPoolFactory
    circuitBreaker *resilience.CircuitBreakerFactory
    orderExecution *order_execution.OrderExecutionService
    
    // Mutex for thread safety
    mu             sync.RWMutex
}

// Initialize initializes the strategy
func (s *MyStrategy) Initialize(ctx context.Context) error {
    if err := s.BaseStrategy.Initialize(ctx); err != nil {
        return err
    }
    
    // Initialize strategy-specific fields
    s.prices = make(map[string][]float64)
    s.positions = make(map[string]float64)
    
    s.logger.Info("My strategy initialized",
        zap.Strings("symbols", s.symbols),
        zap.Int("lookback_period", s.lookbackPeriod),
        zap.Float64("entry_threshold", s.entryThreshold),
        zap.Float64("exit_threshold", s.exitThreshold))
    
    return nil
}

// OnMarketData processes market data updates
func (s *MyStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
    if !s.IsRunning() {
        return nil
    }
    
    // Check if this data is for one of our symbols
    symbolFound := false
    for _, symbol := range s.symbols {
        if data.Symbol == symbol {
            symbolFound = true
            break
        }
    }
    
    if !symbolFound {
        return nil
    }
    
    // Process market data in a worker pool
    err := s.workerPool.SubmitTask("my-strategy-"+s.name, func() error {
        return s.processMarketData(ctx, data)
    })
    
    if err != nil {
        s.logger.Error("Failed to submit market data processing task",
            zap.Error(err),
            zap.String("symbol", data.Symbol))
        return err
    }
    
    return nil
}

// processMarketData processes market data and generates trading signals
func (s *MyStrategy) processMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Update price series
    if _, ok := s.prices[data.Symbol]; !ok {
        s.prices[data.Symbol] = make([]float64, 0, s.lookbackPeriod+100)
    }
    
    s.prices[data.Symbol] = append(s.prices[data.Symbol], data.Price)
    
    // Trim price series if it exceeds lookback period
    if len(s.prices[data.Symbol]) > s.lookbackPeriod {
        s.prices[data.Symbol] = s.prices[data.Symbol][len(s.prices[data.Symbol])-s.lookbackPeriod:]
    }
    
    // Generate trading signals
    if len(s.prices[data.Symbol]) >= s.lookbackPeriod {
        // Calculate signal
        signal := s.calculateSignal(data.Symbol)
        
        // Execute trading logic based on signal
        if signal > s.entryThreshold {
            s.enterLongPosition(ctx, data.Symbol, data.Price)
        } else if signal < -s.entryThreshold {
            s.enterShortPosition(ctx, data.Symbol, data.Price)
        } else if (signal < s.exitThreshold && s.positions[data.Symbol] > 0) ||
                  (signal > -s.exitThreshold && s.positions[data.Symbol] < 0) {
            s.exitPosition(ctx, data.Symbol, data.Price)
        }
    }
    
    return nil
}

// OnOrderUpdate processes order updates
func (s *MyStrategy) OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error {
    // Process order updates
    return nil
}

// GetPerformanceMetrics returns performance metrics for the strategy
func (s *MyStrategy) GetPerformanceMetrics() map[string]interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Return performance metrics
    return map[string]interface{}{
        "positions": s.positions,
    }
}

// Helper methods
func (s *MyStrategy) calculateSignal(symbol string) float64 {
    // Calculate signal
    return 0.0
}

func (s *MyStrategy) enterLongPosition(ctx context.Context, symbol string, price float64) error {
    // Enter long position
    return nil
}

func (s *MyStrategy) enterShortPosition(ctx context.Context, symbol string, price float64) error {
    // Enter short position
    return nil
}

func (s *MyStrategy) exitPosition(ctx context.Context, symbol string, price float64) error {
    // Exit position
    return nil
}
```

### Step 2: Create a Constructor Function

Create a constructor function for your strategy:

```go
// NewMyStrategy creates a new MyStrategy
func NewMyStrategy(
    name string,
    symbols []string,
    lookbackPeriod int,
    entryThreshold float64,
    exitThreshold float64,
    workerPool *workerpool.WorkerPoolFactory,
    circuitBreaker *resilience.CircuitBreakerFactory,
    orderExecution *order_execution.OrderExecutionService,
    logger *zap.Logger,
) *MyStrategy {
    baseStrategy := &strategy.BaseStrategy{
        name:   name,
        logger: logger,
    }
    
    return &MyStrategy{
        BaseStrategy:    baseStrategy,
        symbols:         symbols,
        lookbackPeriod:  lookbackPeriod,
        entryThreshold:  entryThreshold,
        exitThreshold:   exitThreshold,
        workerPool:      workerPool,
        circuitBreaker:  circuitBreaker,
        orderExecution:  orderExecution,
        prices:          make(map[string][]float64),
        positions:       make(map[string]float64),
    }
}
```

### Step 3: Register the Strategy with the StrategyFactory

Register your strategy with the `StrategyFactory`:

```go
// Register the strategy with the factory
factory.RegisterStrategyType("my_strategy", func(config strategy.StrategyConfig) (strategy.Strategy, error) {
    // Extract parameters
    params := config.Parameters
    
    // Get required parameters with defaults
    lookbackPeriod := getIntParam(params, "lookback_period", 20)
    entryThreshold := getFloatParam(params, "entry_threshold", 2.0)
    exitThreshold := getFloatParam(params, "exit_threshold", 0.5)
    
    // Create the strategy
    s := NewMyStrategy(
        config.Name,
        config.Symbols,
        lookbackPeriod,
        entryThreshold,
        exitThreshold,
        workerPool,
        circuitBreaker,
        orderExecution,
        logger,
    )
    
    return s, nil
})
```

## Integrating with fx

### Step 1: Create an fx Module

Create an fx module for your strategy:

```go
// Module provides the my strategy components
var Module = fx.Options(
    // Provide the strategy creator
    fx.Provide(NewMyStrategyCreator),
    
    // Register the strategy type
    fx.Invoke(registerMyStrategy),
)

// MyStrategyCreatorParams contains parameters for creating a MyStrategyCreator
type MyStrategyCreatorParams struct {
    fx.In
    
    Logger                *zap.Logger
    WorkerPoolFactory     *workerpool.WorkerPoolFactory
    CircuitBreakerFactory *resilience.CircuitBreakerFactory
    OrderExecutionService *order_execution.OrderExecutionService
    StrategyFactory       *strategy.fx.StrategyFactory
}

// MyStrategyCreator creates MyStrategy instances
type MyStrategyCreator struct {
    logger                *zap.Logger
    workerPoolFactory     *workerpool.WorkerPoolFactory
    circuitBreakerFactory *resilience.CircuitBreakerFactory
    orderExecutionService *order_execution.OrderExecutionService
}

// NewMyStrategyCreator creates a new MyStrategyCreator
func NewMyStrategyCreator(params MyStrategyCreatorParams) *MyStrategyCreator {
    return &MyStrategyCreator{
        logger:                params.Logger,
        workerPoolFactory:     params.WorkerPoolFactory,
        circuitBreakerFactory: params.CircuitBreakerFactory,
        orderExecutionService: params.OrderExecutionService,
    }
}

// CreateStrategy creates a MyStrategy
func (c *MyStrategyCreator) CreateStrategy(config strategy.StrategyConfig) (strategy.Strategy, error) {
    // Extract parameters
    params := config.Parameters
    
    // Get required parameters with defaults
    lookbackPeriod := getIntParam(params, "lookback_period", 20)
    entryThreshold := getFloatParam(params, "entry_threshold", 2.0)
    exitThreshold := getFloatParam(params, "exit_threshold", 0.5)
    
    // Create the strategy
    s := NewMyStrategy(
        config.Name,
        config.Symbols,
        lookbackPeriod,
        entryThreshold,
        exitThreshold,
        c.workerPoolFactory,
        c.circuitBreakerFactory,
        c.orderExecutionService,
        c.logger,
    )
    
    return s, nil
}

// registerMyStrategy registers the MyStrategy type with the StrategyFactory
func registerMyStrategy(
    factory *strategy.fx.StrategyFactory,
    creator *MyStrategyCreator,
) {
    factory.RegisterStrategyType("my_strategy", creator.CreateStrategy)
}
```

### Step 2: Include the Module in Your Application

Include your strategy module in your fx application:

```go
// Create an fx application with your strategy module
app := fx.New(
    // Core modules
    architecture.fx.Module,
    trading.fx.Module,
    strategy.fx.Module,
    
    // Your strategy module
    my_strategy.Module,
    
    // Other modules...
)

// Start the application
app.Start(ctx)
```

## Configuring a Strategy

Configure your strategy through the `StrategyConfig` struct:

```go
// Create a strategy configuration
config := strategy.StrategyConfig{
    Name:    "my_strategy_instance",
    Type:    "my_strategy",
    Symbols: []string{"BTC-USD", "ETH-USD"},
    Parameters: map[string]interface{}{
        "lookback_period": 30,
        "entry_threshold": 2.5,
        "exit_threshold":  0.3,
    },
    RiskLimits: map[string]interface{}{
        "max_position_size": 10.0,
        "max_drawdown":      0.1,
    },
}

// Create the strategy
strategyInstance, err := factory.CreateStrategy(config)
if err != nil {
    // Handle error
}

// Initialize the strategy
err = strategyInstance.Initialize(ctx)
if err != nil {
    // Handle error
}

// Start the strategy
err = strategyInstance.Start(ctx)
if err != nil {
    // Handle error
}
```

## Using the Strategy Manager

The `StrategyManager` provides methods for managing strategy lifecycle:

```go
// Start a strategy
err := manager.StartStrategy(ctx, "my_strategy_instance")
if err != nil {
    // Handle error
}

// Stop a strategy
err := manager.StopStrategy(ctx, "my_strategy_instance")
if err != nil {
    // Handle error
}
```

## Using the Strategy Registry

The `StrategyRegistry` provides methods for retrieving strategies:

```go
// Get a strategy by name
strategy, exists := registry.GetStrategy("my_strategy_instance")
if exists {
    // Use the strategy
    metrics := strategy.GetPerformanceMetrics()
}

// Get all registered strategies
allStrategies := registry.GetAllStrategies()
for name, strategy := range allStrategies {
    // Use the strategies
}
```

## Using the Strategy Metrics Collector

The `StrategyMetricsCollector` provides methods for collecting performance metrics:

```go
// Collect metrics from all strategies
metrics := collector.CollectMetrics()
for strategyName, strategyMetrics := range metrics {
    // Use the metrics
}
```

## Best Practices

1. **Use the BaseStrategy**: Extend the `BaseStrategy` class to get common functionality for free.

2. **Use Worker Pools**: Use worker pools for concurrent operations to avoid blocking the main thread.

3. **Use Circuit Breakers**: Use circuit breakers for resilience to avoid cascading failures.

4. **Use Mutexes**: Use mutexes for thread safety to avoid race conditions.

5. **Use Logging**: Use the logger for debugging and monitoring.

6. **Use Context**: Use context for cancellation to avoid resource leaks.

7. **Use Dependency Injection**: Use dependency injection for testability and modularity.

8. **Use Configuration**: Use configuration for flexibility and reusability.

9. **Use Error Handling**: Use proper error handling for robustness.

10. **Use Performance Metrics**: Collect and expose performance metrics for monitoring and optimization.

## Troubleshooting

### Strategy Not Starting

If your strategy is not starting, check the following:

1. Make sure the strategy is registered with the `StrategyFactory`.
2. Make sure the strategy is initialized before starting.
3. Check the logs for error messages.

### Strategy Not Receiving Market Data

If your strategy is not receiving market data, check the following:

1. Make sure the strategy is running.
2. Make sure the strategy is subscribed to the correct symbols.
3. Check the logs for error messages.

### Strategy Not Executing Orders

If your strategy is not executing orders, check the following:

1. Make sure the order execution service is properly configured.
2. Make sure the strategy has the necessary permissions.
3. Check the logs for error messages.

## Example: Mean Reversion Strategy

Here's a complete example of a mean reversion strategy:

```go
// MeanReversionStrategy implements a mean reversion trading strategy
type MeanReversionStrategy struct {
    *strategy.BaseStrategy
    
    // Strategy parameters
    symbols        []string
    lookbackPeriod int
    updateInterval int
    stdDevPeriod   int
    entryThreshold float64
    exitThreshold  float64
    
    // Strategy state
    prices         map[string][]float64
    zScores        map[string]float64
    positions      map[string]float64
    lastUpdate     map[string]time.Time
    
    // Concurrency control
    mu             sync.RWMutex
    
    // Dependencies
    workerPool     *workerpool.WorkerPoolFactory
    circuitBreaker *resilience.CircuitBreakerFactory
    orderExecution *order_execution.OrderExecutionService
    
    // Performance metrics
    processedUpdates int64
    executedTrades   int64
    pnl              float64
}

// NewMeanReversionStrategy creates a new MeanReversionStrategy
func NewMeanReversionStrategy(
    name string,
    symbols []string,
    lookbackPeriod int,
    updateInterval int,
    stdDevPeriod int,
    entryThreshold float64,
    exitThreshold float64,
    workerPool *workerpool.WorkerPoolFactory,
    circuitBreaker *resilience.CircuitBreakerFactory,
    orderExecution *order_execution.OrderExecutionService,
    logger *zap.Logger,
) *MeanReversionStrategy {
    baseStrategy := &strategy.BaseStrategy{
        name:   name,
        logger: logger,
    }
    
    return &MeanReversionStrategy{
        BaseStrategy:    baseStrategy,
        symbols:         symbols,
        lookbackPeriod:  lookbackPeriod,
        updateInterval:  updateInterval,
        stdDevPeriod:    stdDevPeriod,
        entryThreshold:  entryThreshold,
        exitThreshold:   exitThreshold,
        workerPool:      workerPool,
        circuitBreaker:  circuitBreaker,
        orderExecution:  orderExecution,
        prices:          make(map[string][]float64),
        zScores:         make(map[string]float64),
        positions:       make(map[string]float64),
        lastUpdate:      make(map[string]time.Time),
    }
}

// Initialize initializes the strategy
func (s *MeanReversionStrategy) Initialize(ctx context.Context) error {
    if err := s.BaseStrategy.Initialize(ctx); err != nil {
        return err
    }
    
    s.logger.Info("Mean reversion strategy initialized",
        zap.Strings("symbols", s.symbols),
        zap.Int("lookback_period", s.lookbackPeriod),
        zap.Int("update_interval", s.updateInterval),
        zap.Int("std_dev_period", s.stdDevPeriod),
        zap.Float64("entry_threshold", s.entryThreshold),
        zap.Float64("exit_threshold", s.exitThreshold))
    
    return nil
}

// OnMarketData processes market data updates
func (s *MeanReversionStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
    if !s.IsRunning() {
        return nil
    }
    
    // Check if this data is for one of our symbols
    symbolFound := false
    for _, symbol := range s.symbols {
        if data.Symbol == symbol {
            symbolFound = true
            break
        }
    }
    
    if !symbolFound {
        return nil
    }
    
    // Process market data in a worker pool
    err := s.workerPool.SubmitTask("mean-reversion-"+s.name, func() error {
        return s.processMarketData(ctx, data)
    })
    
    if err != nil {
        s.logger.Error("Failed to submit market data processing task",
            zap.Error(err),
            zap.String("symbol", data.Symbol))
        return err
    }
    
    return nil
}

// processMarketData processes market data and generates trading signals
func (s *MeanReversionStrategy) processMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Increment processed updates counter
    s.processedUpdates++
    
    // Update price series
    if _, ok := s.prices[data.Symbol]; !ok {
        s.prices[data.Symbol] = make([]float64, 0, s.lookbackPeriod+100)
    }
    
    s.prices[data.Symbol] = append(s.prices[data.Symbol], data.Price)
    
    // Trim price series if it exceeds lookback period
    if len(s.prices[data.Symbol]) > s.lookbackPeriod {
        s.prices[data.Symbol] = s.prices[data.Symbol][len(s.prices[data.Symbol])-s.lookbackPeriod:]
    }
    
    // Check if it's time to update signals
    lastUpdate, ok := s.lastUpdate[data.Symbol]
    if !ok || time.Since(lastUpdate) >= time.Duration(s.updateInterval)*time.Second {
        // Calculate z-score
        if len(s.prices[data.Symbol]) >= s.stdDevPeriod {
            // Calculate mean
            var sum float64
            for i := len(s.prices[data.Symbol]) - s.stdDevPeriod; i < len(s.prices[data.Symbol]); i++ {
                sum += s.prices[data.Symbol][i]
            }
            mean := sum / float64(s.stdDevPeriod)
            
            // Calculate standard deviation
            var variance float64
            for i := len(s.prices[data.Symbol]) - s.stdDevPeriod; i < len(s.prices[data.Symbol]); i++ {
                variance += math.Pow(s.prices[data.Symbol][i] - mean, 2)
            }
            stdDev := math.Sqrt(variance / float64(s.stdDevPeriod))
            
            // Calculate z-score
            latestPrice := s.prices[data.Symbol][len(s.prices[data.Symbol])-1]
            zScore := (latestPrice - mean) / stdDev
            
            // Store the z-score
            s.zScores[data.Symbol] = zScore
            
            // Generate trading signals
            currentPosition, hasPosition := s.positions[data.Symbol]
            if !hasPosition {
                currentPosition = 0
            }
            
            // Short signal: Price is significantly above mean (high z-score)
            if zScore > s.entryThreshold && currentPosition >= 0 {
                if err := s.enterShortPosition(ctx, data.Symbol, data.Price); err != nil {
                    s.logger.Error("Failed to enter short position",
                        zap.Error(err),
                        zap.String("symbol", data.Symbol),
                        zap.Float64("price", data.Price),
                        zap.Float64("z_score", zScore))
                }
            }
            // Long signal: Price is significantly below mean (low z-score)
            else if zScore < -s.entryThreshold && currentPosition <= 0 {
                if err := s.enterLongPosition(ctx, data.Symbol, data.Price); err != nil {
                    s.logger.Error("Failed to enter long position",
                        zap.Error(err),
                        zap.String("symbol", data.Symbol),
                        zap.Float64("price", data.Price),
                        zap.Float64("z_score", zScore))
                }
            }
            // Exit signal: Price reverts back toward mean
            else if (zScore < s.exitThreshold && currentPosition < 0) ||
                    (zScore > -s.exitThreshold && currentPosition > 0) {
                if err := s.exitPosition(ctx, data.Symbol, data.Price); err != nil {
                    s.logger.Error("Failed to exit position",
                        zap.Error(err),
                        zap.String("symbol", data.Symbol),
                        zap.Float64("price", data.Price),
                        zap.Float64("z_score", zScore))
                }
            }
        }
        
        s.lastUpdate[data.Symbol] = time.Now()
    }
    
    return nil
}

// enterLongPosition enters a long position
func (s *MeanReversionStrategy) enterLongPosition(ctx context.Context, symbol string, price float64) error {
    // Create order request
    request := &orders.OrderRequest{
        Symbol:    symbol,
        Side:      orders.OrderSide_BUY,
        Type:      orders.OrderType_MARKET,
        Quantity:  1.0, // Fixed position size for simplicity
        Price:     price,
        AccountId: "account1",
    }
    
    // Use circuit breaker for resilience
    result := s.circuitBreaker.ExecuteWithContext(ctx, "order_execution", func(ctx context.Context) (interface{}, error) {
        return s.orderExecution.ExecuteOrder(ctx, request)
    })
    
    if result.Error != nil {
        return result.Error
    }
    
    // Update position
    s.positions[symbol] = 1.0
    
    // Increment executed trades counter
    s.executedTrades++
    
    s.logger.Info("Entered long position",
        zap.String("symbol", symbol),
        zap.Float64("price", price),
        zap.Float64("z_score", s.zScores[symbol]))
    
    return nil
}

// enterShortPosition enters a short position
func (s *MeanReversionStrategy) enterShortPosition(ctx context.Context, symbol string, price float64) error {
    // Create order request
    request := &orders.OrderRequest{
        Symbol:    symbol,
        Side:      orders.OrderSide_SELL,
        Type:      orders.OrderType_MARKET,
        Quantity:  1.0, // Fixed position size for simplicity
        Price:     price,
        AccountId: "account1",
    }
    
    // Use circuit breaker for resilience
    result := s.circuitBreaker.ExecuteWithContext(ctx, "order_execution", func(ctx context.Context) (interface{}, error) {
        return s.orderExecution.ExecuteOrder(ctx, request)
    })
    
    if result.Error != nil {
        return result.Error
    }
    
    // Update position
    s.positions[symbol] = -1.0
    
    // Increment executed trades counter
    s.executedTrades++
    
    s.logger.Info("Entered short position",
        zap.String("symbol", symbol),
        zap.Float64("price", price),
        zap.Float64("z_score", s.zScores[symbol]))
    
    return nil
}

// exitPosition exits a position
func (s *MeanReversionStrategy) exitPosition(ctx context.Context, symbol string, price float64) error {
    currentPosition := s.positions[symbol]
    if currentPosition == 0 {
        return nil
    }
    
    // Create order request
    request := &orders.OrderRequest{
        Symbol:    symbol,
        Side:      orders.OrderSide_BUY,
        Type:      orders.OrderType_MARKET,
        Quantity:  math.Abs(currentPosition),
        Price:     price,
        AccountId: "account1",
    }
    
    if currentPosition > 0 {
        request.Side = orders.OrderSide_SELL
    }
    
    // Use circuit breaker for resilience
    result := s.circuitBreaker.ExecuteWithContext(ctx, "order_execution", func(ctx context.Context) (interface{}, error) {
        return s.orderExecution.ExecuteOrder(ctx, request)
    })
    
    if result.Error != nil {
        return result.Error
    }
    
    // Calculate P&L
    entryPrice := s.prices[symbol][len(s.prices[symbol])-2] // Simplified, should use actual entry price
    pnl := 0.0
    if currentPosition > 0 {
        pnl = (price - entryPrice) * currentPosition
    } else {
        pnl = (entryPrice - price) * -currentPosition
    }
    
    // Update P&L
    s.pnl += pnl
    
    // Reset position
    s.positions[symbol] = 0.0
    
    // Increment executed trades counter
    s.executedTrades++
    
    s.logger.Info("Exited position",
        zap.String("symbol", symbol),
        zap.Float64("price", price),
        zap.Float64("z_score", s.zScores[symbol]),
        zap.Float64("pnl", pnl),
        zap.Float64("total_pnl", s.pnl))
    
    return nil
}

// OnOrderUpdate processes order updates
func (s *MeanReversionStrategy) OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error {
    // Process order updates
    return nil
}

// GetPerformanceMetrics returns performance metrics for the strategy
func (s *MeanReversionStrategy) GetPerformanceMetrics() map[string]interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    return map[string]interface{}{
        "processed_updates": s.processedUpdates,
        "executed_trades":   s.executedTrades,
        "pnl":              s.pnl,
        "positions":        s.positions,
        "z_scores":         s.zScores,
    }
}
```

## Conclusion

This guide has shown you how to implement and integrate trading strategies with the TradSys platform using the fx dependency injection framework. By following these guidelines, you can create modular, testable, and maintainable strategies that leverage the full power of the TradSys platform.

