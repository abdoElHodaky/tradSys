# Strategy API Documentation

## Overview

The Strategy API provides interfaces and components for creating, managing, and optimizing trading strategies. It leverages the fx dependency injection framework for modularity and testability.

## Core Interfaces

### Strategy

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

### StrategyFactory

The `StrategyFactory` interface defines the contract for creating strategies:

```go
type StrategyFactory interface {
    // CreateStrategy creates a strategy from a configuration
    CreateStrategy(config StrategyConfig) (Strategy, error)

    // GetAvailableStrategyTypes returns the available strategy types
    GetAvailableStrategyTypes() []string
}
```

### StrategyConfig

The `StrategyConfig` struct defines the configuration for a strategy:

```go
type StrategyConfig struct {
    // Name is the name of the strategy
    Name string `json:"name"`

    // Type is the type of the strategy
    Type string `json:"type"`

    // Symbols are the trading symbols for the strategy
    Symbols []string `json:"symbols"`

    // Parameters are the strategy parameters
    Parameters map[string]interface{} `json:"parameters"`

    // RiskLimits are the risk limits for the strategy
    RiskLimits map[string]interface{} `json:"risk_limits"`
}
```

## Strategy Components

### StrategyFactory

The `StrategyFactory` is responsible for creating strategies based on their type and configuration. It maintains a registry of strategy creators and provides methods for registering new strategy types.

Example usage:

```go
// Create a strategy factory
factory := fx.NewStrategyFactory(params)

// Register a custom strategy type
factory.RegisterStrategyType("custom_strategy", func(config strategy.StrategyConfig) (strategy.Strategy, error) {
    return NewCustomStrategy(config), nil
})

// Create a strategy
config := strategy.StrategyConfig{
    Name:    "my_strategy",
    Type:    "mean_reversion",
    Symbols: []string{"BTC-USD"},
    Parameters: map[string]interface{}{
        "lookback_period": 20,
        "entry_threshold": 2.0,
        "exit_threshold":  0.5,
    },
}
strategyInstance, err := factory.CreateStrategy(config)
```

### StrategyRegistry

The `StrategyRegistry` manages registered strategies and provides methods for retrieving them by name.

Example usage:

```go
// Get a strategy by name
strategy, exists := registry.GetStrategy("my_strategy")
if exists {
    // Use the strategy
}

// Get all registered strategies
allStrategies := registry.GetAllStrategies()
```

### StrategyManager

The `StrategyManager` manages strategy lifecycle and provides methods for starting and stopping strategies.

Example usage:

```go
// Start a strategy
err := manager.StartStrategy(ctx, "my_strategy")

// Stop a strategy
err := manager.StopStrategy(ctx, "my_strategy")
```

### StrategyMetricsCollector

The `StrategyMetricsCollector` collects performance metrics from strategies and provides methods for retrieving them.

Example usage:

```go
// Collect metrics from all strategies
metrics := collector.CollectMetrics()
```

## Strategy Optimization

### StrategyOptimizer

The `StrategyOptimizer` optimizes strategy parameters using various optimization methods:

- Grid search
- Random search
- Genetic algorithm

Example usage:

```go
// Create an optimization configuration
config := optimization.OptimizationConfig{
    StrategyType: "mean_reversion",
    StrategyName: "my_strategy",
    Symbols:      []string{"BTC-USD"},
    Parameters: map[string]optimization.ParameterRange{
        "lookback_period": {
            Min:       10,
            Max:       50,
            Step:      5,
            IsInteger: true,
        },
        "entry_threshold": {
            Min:  1.0,
            Max:  3.0,
            Step: 0.1,
        },
        "exit_threshold": {
            Min:  0.1,
            Max:  1.0,
            Step: 0.1,
        },
    },
    Method:     optimization.OptimizationMethodGrid,
    Iterations: 100,
    Metric:     "sharpe_ratio",
    Maximize:   true,
}

// Run optimization
result, err := optimizer.Optimize(ctx, config)

// Use the best parameters
bestParams := result.BestParameters
```

### StrategyEvaluator

The `StrategyEvaluator` evaluates strategies against historical data and calculates performance metrics.

Example usage:

```go
// Evaluate a strategy
metrics, err := evaluator.Evaluate(ctx, config)
```

### Backtester

The `Backtester` performs backtesting of strategies against historical data.

Example usage:

```go
// Run a backtest
result, err := backtester.Backtest(ctx, strategy)

// Access backtest results
totalReturn := result.TotalPnL
sharpeRatio := result.RiskMetrics["sharpe_ratio"]
```

## Mean Reversion Strategy

The `MeanReversionStrategy` is a built-in strategy that trades based on mean reversion principles using Bollinger Bands and z-scores.

Configuration parameters:

- `lookback_period`: The number of periods to look back for calculating the mean
- `update_interval`: The interval in seconds between strategy updates
- `std_dev_period`: The number of periods for calculating standard deviation
- `entry_threshold`: The z-score threshold for entering positions
- `exit_threshold`: The z-score threshold for exiting positions

Example configuration:

```json
{
    "name": "btc_mean_reversion",
    "type": "mean_reversion",
    "symbols": ["BTC-USD"],
    "parameters": {
        "lookback_period": 20,
        "update_interval": 5,
        "std_dev_period": 20,
        "entry_threshold": 2.0,
        "exit_threshold": 0.5
    }
}
```

## Timeframe Analysis

The timeframe analysis components provide functionality for analyzing market data across different timeframes.

### TimeframeAggregator

The `TimeframeAggregator` aggregates trades into OHLCV candles for different timeframes:

- 1 minute (1m)
- 5 minutes (5m)
- 15 minutes (15m)
- 30 minutes (30m)
- 1 hour (1h)
- 4 hours (4h)
- 1 day (1d)

Example usage:

```go
// Process a trade
aggregator.ProcessTrade(&timeframe.Trade{
    Symbol:    "BTC-USD",
    Price:     50000.0,
    Volume:    1.0,
    Timestamp: time.Now(),
})

// Get current candle
candle := aggregator.GetCurrentCandle("BTC-USD", timeframe.Interval1h)

// Get historical candles
candles := aggregator.GetHistoricalCandles("BTC-USD", timeframe.Interval1h, 100)
```

### IndicatorCalculator

The `IndicatorCalculator` calculates technical indicators for different timeframes:

- Simple Moving Average (SMA)
- Exponential Moving Average (EMA)
- Relative Strength Index (RSI)
- Moving Average Convergence Divergence (MACD)
- Bollinger Bands
- Average True Range (ATR)
- Z-Score

Example usage:

```go
// Calculate SMA
sma, err := calculator.CalculateSMA("BTC-USD", timeframe.Interval1h, 20)

// Calculate RSI
rsi, err := calculator.CalculateRSI("BTC-USD", timeframe.Interval1h, 14)

// Calculate Bollinger Bands
bb, err := calculator.CalculateBollingerBands("BTC-USD", timeframe.Interval1h, 20, 2.0, 2.0)

// Calculate indicator across multiple timeframes
intervals := []timeframe.TimeframeInterval{
    timeframe.Interval1h,
    timeframe.Interval4h,
    timeframe.Interval1d,
}
results, err := calculator.CalculateMultipleTimeframeIndicator(
    "BTC-USD",
    intervals,
    timeframe.IndicatorRSI,
    map[string]interface{}{
        "period": 14,
    },
)
```

## fx Integration

The strategy components are integrated with the fx dependency injection framework through the following modules:

- `strategy/fx/module.go`: Provides the strategy components
- `strategy/optimization/fx/module.go`: Provides the strategy optimization components
- `trading/market_data/timeframe/fx/module.go`: Provides the timeframe analysis components

Example usage:

```go
// Create an fx application with strategy components
app := fx.New(
    strategy.fx.Module,
    strategy.optimization.fx.Module,
    trading.market_data.timeframe.fx.Module,
    // Other modules...
)

// Start the application
app.Start(ctx)
```

## Error Handling

The strategy components use the following error handling patterns:

- Returning errors from methods
- Logging errors with zap
- Using circuit breakers for resilience
- Providing detailed error messages

## Concurrency

The strategy components use the following concurrency patterns:

- Using worker pools for concurrent task execution
- Using mutexes for thread safety
- Using channels for communication
- Using context for cancellation

## Performance Considerations

- Use worker pools for concurrent operations
- Use circuit breakers for resilience
- Use timeouts for long-running operations
- Use context for cancellation
- Use mutexes for thread safety

