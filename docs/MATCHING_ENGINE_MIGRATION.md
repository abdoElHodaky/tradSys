# Matching Engine Migration Guide

This document provides guidance for migrating from the duplicate matching engine implementations to the new unified matching engine.

## Background

The TradSys codebase previously had significant code duplication between:
- `internal/core/matching/` 
- `internal/orders/matching/`

This resulted in ~2,500 lines of duplicated code with only minor differences in field access patterns. The new unified implementation eliminates this duplication while maintaining full API compatibility.

## Migration Overview

### What Changed

1. **Consolidated Implementation**: All matching engine types now use a single, unified implementation
2. **Eliminated Duplication**: Removed ~2,500 lines of duplicate code
3. **Improved Architecture**: Uses proper interfaces and dependency injection
4. **Enhanced Performance**: Object pooling and optimized data structures
5. **Better Error Handling**: Standardized error types and handling
6. **Comprehensive Metrics**: Built-in performance monitoring

### What Stayed the Same

1. **API Compatibility**: All existing APIs remain unchanged
2. **Configuration**: Existing configuration files work without modification
3. **Engine Types**: All engine types (HFT, Standard, Optimized) are still supported
4. **Performance**: Performance characteristics are maintained or improved

## Migration Steps

### Step 1: Update Imports

**Before:**
```go
import (
    "github.com/abdoElHodaky/tradSys/internal/core/matching"
    "github.com/abdoElHodaky/tradSys/internal/orders/matching"
)
```

**After:**
```go
import (
    "github.com/abdoElHodaky/tradSys/internal/matching"
    "github.com/abdoElHodaky/tradSys/pkg/interfaces"
    "github.com/abdoElHodaky/tradSys/pkg/types"
)
```

### Step 2: Use the Factory Pattern

**Before:**
```go
// Direct instantiation (deprecated)
engine := &matching.HFTEngine{
    Config: config,
}
```

**After:**
```go
// Factory-based creation (recommended)
factory := matching.NewFactory(logger, publisher)
engine, err := factory.CreateEngine(config)
if err != nil {
    return err
}
```

### Step 3: Update Type References

**Before:**
```go
// Old types from duplicate implementations
order := &matching.Order{
    Type: matching.OrderTypeLimit,
    Side: matching.OrderSideBuy,
}
```

**After:**
```go
// New standardized types
order := &types.Order{
    Type: types.OrderTypeLimit,
    Side: types.OrderSideBuy,
}
```

### Step 4: Interface-Based Programming

**Before:**
```go
// Concrete type dependency
func ProcessOrders(engine *matching.HFTEngine) {
    // ...
}
```

**After:**
```go
// Interface-based dependency
func ProcessOrders(engine interfaces.MatchingEngine) {
    // ...
}
```

## Configuration Migration

### Existing Configuration

Your existing configuration files will continue to work without changes:

```yaml
matching:
  engine_type: "hft"
  max_orders_per_symbol: 10000
  tick_size: 0.01
  processing_timeout: 100ms
  enable_metrics: true
```

### Enhanced Configuration Options

New optional configuration parameters are available:

```yaml
matching:
  engine_type: "unified"  # New unified type
  max_orders_per_symbol: 10000
  tick_size: 0.01
  processing_timeout: 100ms
  enable_metrics: true
  pool_size: 100          # New: Object pool size
  buffer_size: 1000       # New: Buffer size
  worker_count: 4         # New: Worker threads
  max_latency: 1ms        # New: Maximum acceptable latency
  enable_order_book: true # New: Order book features
  order_book_depth: 20    # New: Order book depth
```

## Code Examples

### Basic Engine Creation

```go
package main

import (
    "context"
    "log"
    
    "github.com/abdoElHodaky/tradSys/internal/matching"
    "github.com/abdoElHodaky/tradSys/pkg/config"
    "github.com/abdoElHodaky/tradSys/pkg/types"
)

func main() {
    // Create factory
    factory := matching.NewFactory(logger, publisher)
    
    // Create engine with default configuration
    engine, err := factory.CreateEngineWithDefaults(matching.EngineTypeHFT)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start the engine
    ctx := context.Background()
    if err := engine.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer engine.Stop(ctx)
    
    // Process an order
    order := &types.Order{
        ID:       "order-123",
        UserID:   "user-456",
        Symbol:   "BTCUSD",
        Side:     types.OrderSideBuy,
        Type:     types.OrderTypeLimit,
        Price:    50000.0,
        Quantity: 1.0,
    }
    
    trades, err := engine.ProcessOrder(ctx, order)
    if err != nil {
        log.Printf("Error processing order: %v", err)
        return
    }
    
    log.Printf("Order processed, generated %d trades", len(trades))
}
```

### Advanced Configuration

```go
func createCustomEngine() (interfaces.MatchingEngine, error) {
    config := &config.MatchingConfig{
        EngineType:         "unified",
        MaxOrdersPerSymbol: 50000,
        TickSize:           0.001,
        ProcessingTimeout:  50 * time.Millisecond,
        EnableMetrics:      true,
        PoolSize:           200,
        BufferSize:         2000,
        WorkerCount:        8,
        MaxLatency:         500 * time.Microsecond,
        EnableOrderBook:    true,
        OrderBookDepth:     50,
    }
    
    factory := matching.NewFactory(logger, publisher)
    return factory.CreateEngine(config)
}
```

### Migration from Legacy Engines

```go
func migrateFromLegacy() (interfaces.MatchingEngine, error) {
    factory := matching.NewFactory(logger, publisher)
    
    // Automatically migrate from legacy engine types
    engine, err := factory.MigrateFromLegacyEngine("high_frequency")
    if err != nil {
        return nil, err
    }
    
    return engine, nil
}
```

## Performance Considerations

### Memory Usage

The new unified engine includes several memory optimizations:

1. **Object Pooling**: Reuses order and trade objects to reduce GC pressure
2. **Efficient Data Structures**: Optimized order book implementation
3. **Configurable Buffers**: Tunable buffer sizes for different workloads

### Latency Improvements

1. **Reduced Code Paths**: Elimination of duplicate code reduces instruction cache misses
2. **Optimized Algorithms**: Improved matching algorithms
3. **Concurrent Processing**: Configurable worker threads for parallel processing

### Throughput Enhancements

1. **Batch Processing**: Support for batch order processing
2. **Lock-Free Operations**: Atomic operations where possible
3. **Efficient Synchronization**: Reduced lock contention

## Testing Migration

### Unit Tests

Update your unit tests to use the new interfaces:

```go
func TestOrderProcessing(t *testing.T) {
    factory := matching.NewFactory(mockLogger, mockPublisher)
    engine, err := factory.CreateEngineWithDefaults(matching.EngineTypeUnified)
    require.NoError(t, err)
    
    ctx := context.Background()
    err = engine.Start(ctx)
    require.NoError(t, err)
    defer engine.Stop(ctx)
    
    order := &types.Order{
        ID:       "test-order",
        UserID:   "test-user",
        Symbol:   "TESTSYM",
        Side:     types.OrderSideBuy,
        Type:     types.OrderTypeLimit,
        Price:    100.0,
        Quantity: 10.0,
    }
    
    trades, err := engine.ProcessOrder(ctx, order)
    require.NoError(t, err)
    assert.NotNil(t, trades)
}
```

### Integration Tests

Test the migration with your existing integration test suite:

```go
func TestMigrationCompatibility(t *testing.T) {
    // Test that new engine produces same results as old engine
    factory := matching.NewFactory(logger, publisher)
    
    // Test all engine types
    engineTypes := []matching.EngineType{
        matching.EngineTypeUnified,
        matching.EngineTypeHFT,
        matching.EngineTypeStandard,
        matching.EngineTypeOptimized,
    }
    
    for _, engineType := range engineTypes {
        t.Run(string(engineType), func(t *testing.T) {
            engine, err := factory.CreateEngineWithDefaults(engineType)
            require.NoError(t, err)
            
            // Run your existing test scenarios
            testOrderProcessing(t, engine)
            testOrderCancellation(t, engine)
            testOrderBookManagement(t, engine)
        })
    }
}
```

## Monitoring and Metrics

### Built-in Metrics

The new engine provides comprehensive metrics:

```go
func monitorEngine(engine interfaces.MatchingEngine) {
    metrics := engine.GetMetrics()
    
    log.Printf("Orders Processed: %d", metrics.OrdersProcessed)
    log.Printf("Trades Executed: %d", metrics.TradesExecuted)
    log.Printf("Average Latency: %v", metrics.AverageLatency)
    log.Printf("Throughput: %.2f orders/sec", metrics.ThroughputPerSec)
    log.Printf("Active Orders: %d", metrics.ActiveOrders)
}
```

### Custom Metrics

You can also implement custom metrics collection:

```go
func setupCustomMetrics(engine interfaces.MatchingEngine) {
    // Subscribe to order book updates
    engine.SubscribeOrderBook("BTCUSD", func(orderBook *types.OrderBook) {
        // Custom order book metrics
        spread := orderBook.GetSpread()
        midPrice := orderBook.GetMidPrice()
        
        // Send to your metrics system
        metricsCollector.RecordGauge("orderbook.spread", spread, map[string]string{
            "symbol": orderBook.Symbol,
        })
        metricsCollector.RecordGauge("orderbook.mid_price", midPrice, map[string]string{
            "symbol": orderBook.Symbol,
        })
    })
    
    // Subscribe to trade updates
    engine.SubscribeTrades("BTCUSD", func(trade *types.Trade) {
        // Custom trade metrics
        metricsCollector.IncrementCounter("trades.executed", map[string]string{
            "symbol": trade.Symbol,
            "side":   string(trade.TakerSide),
        })
    })
}
```

## Troubleshooting

### Common Issues

1. **Import Errors**: Update import paths to use the new unified package
2. **Type Mismatches**: Use types from `pkg/types` instead of internal packages
3. **Configuration Issues**: Validate configuration using the factory's validation methods

### Debugging

Enable debug logging to troubleshoot issues:

```go
// Enable debug logging
logger.SetLevel("debug")

// Create engine with debug logging
factory := matching.NewFactory(logger, publisher)
engine, err := factory.CreateEngine(config)
```

### Performance Issues

If you experience performance issues after migration:

1. **Check Configuration**: Ensure worker count and buffer sizes are appropriate
2. **Monitor Metrics**: Use built-in metrics to identify bottlenecks
3. **Profile Memory**: Use Go's built-in profiling tools
4. **Adjust Pool Sizes**: Tune object pool sizes based on workload

## Rollback Plan

If you need to rollback to the old implementation:

1. **Revert Imports**: Change imports back to the old packages
2. **Restore Configuration**: Use old configuration format
3. **Update Code**: Revert to direct instantiation instead of factory pattern

However, we recommend addressing any issues with the new implementation rather than rolling back, as the old duplicate code will be removed in future versions.

## Support

For migration support:

1. **Documentation**: Refer to this guide and the API documentation
2. **Examples**: Check the examples in the `examples/` directory
3. **Tests**: Review the test files for usage patterns
4. **Issues**: Report any issues through the project's issue tracker

## Timeline

- **Phase 1** (Current): New unified engine available alongside old engines
- **Phase 2** (Next release): Old engines marked as deprecated
- **Phase 3** (Future release): Old engines removed completely

We recommend migrating as soon as possible to take advantage of the improvements and avoid future compatibility issues.
