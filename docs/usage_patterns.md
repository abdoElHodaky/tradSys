# Common Usage Patterns for Lazy Loading

This document describes common usage patterns for the lazy loading system in the TradSys codebase.

## Basic Usage

The most basic usage pattern is to create a lazy-loaded component and use it:

```go
// Create a component coordinator
coordinator := coordination.NewComponentCoordinator(
    coordination.DefaultCoordinatorConfig(),
    logger,
)

// Create a lazy-loaded component
lazyService, err := historical_lazy.NewLazyHistoricalDataService(
    coordinator,
    historical.Config{
        MaxCacheSize: 100 * 1024 * 1024, // 100MB
        CacheTTL:     time.Hour,
    },
    logger,
)
if err != nil {
    // Handle error
}

// Use the service - it will be initialized on first use
data, err := lazyService.GetHistoricalData(
    ctx,
    "BTC-USD",
    time.Now().Add(-24*time.Hour),
    time.Now(),
    "1h",
)
if err != nil {
    // Handle error
}

// Shutdown the service when done
err = lazyService.Shutdown(ctx)
if err != nil {
    // Handle error
}

// Shutdown the coordinator
err = coordinator.Shutdown(ctx)
if err != nil {
    // Handle error
}
```

## Multiple Components

When using multiple lazy-loaded components, they can all share the same coordinator:

```go
// Create a component coordinator
coordinator := coordination.NewComponentCoordinator(
    coordination.DefaultCoordinatorConfig(),
    logger,
)

// Create multiple lazy-loaded services
historicalService := createHistoricalService(coordinator, logger)
orderService := createOrderService(coordinator, logger)
connectionPool := createConnectionPool(coordinator, logger)

// Use the services as needed
// ...

// Shutdown all services
historicalService.Shutdown(ctx)
orderService.Shutdown(ctx)
connectionPool.Shutdown(ctx)

// Shutdown the coordinator
coordinator.Shutdown(ctx)
```

## Dynamic Loading

For components that need to be loaded dynamically, such as plugins or connectors, use the appropriate manager:

```go
// Create a component coordinator
coordinator := coordination.NewComponentCoordinator(
    coordination.DefaultCoordinatorConfig(),
    logger,
)

// Create a lock manager
lockManager := coordination.NewLockManager(
    coordination.DefaultLockManagerConfig(),
    logger,
)

// Create a connector factory
factory := &connectors.DefaultConnectorFactory{}

// Create a lazy connector manager
connectorManager, err := exchange_lazy.NewLazyConnectorManager(
    coordinator,
    lockManager,
    factory,
    connectors.ConnectorConfig{},
    logger,
)
if err != nil {
    // Handle error
}

// Dynamically load and use connectors
connector, err := connectorManager.GetConnector(ctx, "binance")
if err != nil {
    // Handle error
}

// Use the connector
data, err := connector.GetMarketData(ctx, "BTC-USD")
if err != nil {
    // Handle error
}

// Release the connector when done
err = connectorManager.ReleaseConnector(ctx, "binance")
if err != nil {
    // Handle error
}

// Shutdown the connector manager
connectorManager.ShutdownAll(ctx)

// Shutdown the coordinator
coordinator.Shutdown(ctx)
```

## Memory Pressure Handling

To handle memory pressure, configure the memory manager with appropriate thresholds:

```go
// Create a custom memory manager configuration
memoryConfig := coordination.MemoryManagerConfig{
    TotalLimit:        1024 * 1024 * 1024, // 1GB
    LowThreshold:      0.6,                // 60%
    MediumThreshold:   0.75,               // 75%
    HighThreshold:     0.85,               // 85%
    CriticalThreshold: 0.95,               // 95%
    AutoUnloadEnabled: true,
    MinIdleTime:       300,                // 5 minutes
    CheckInterval:     60,                 // 1 minute
}

// Create a coordinator configuration
coordinatorConfig := coordination.CoordinatorConfig{
    MemoryConfig:      memoryConfig,
    TimeoutConfig:     coordination.DefaultTimeoutManagerConfig(),
    AutoUnloadEnabled: true,
}

// Create a component coordinator
coordinator := coordination.NewComponentCoordinator(
    coordinatorConfig,
    logger,
)

// Create and use components as needed
// ...

// Check memory pressure
memoryManager := coordinator.GetMemoryManager()
pressureLevel := memoryManager.GetMemoryPressureLevel()

if pressureLevel >= coordination.MemoryPressureHigh {
    // Take action to reduce memory usage
    logger.Warn("High memory pressure detected",
        zap.String("pressure_level", pressureLevel.String()),
        zap.Int64("memory_usage", memoryManager.GetMemoryUsage()),
        zap.Int64("memory_limit", memoryManager.GetMemoryLimit()),
    )
    
    // Manually free memory if needed
    freed, err := memoryManager.FreeMemory(ctx, 100*1024*1024) // Try to free 100MB
    if err != nil {
        // Handle error
    }
    
    if freed {
        logger.Info("Successfully freed memory")
    } else {
        logger.Warn("Failed to free enough memory")
    }
}

// Shutdown the coordinator
coordinator.Shutdown(ctx)
```

## Dependency Management

For components with dependencies, specify them when registering the component:

```go
// Create a component coordinator
coordinator := coordination.NewComponentCoordinator(
    coordination.DefaultCoordinatorConfig(),
    logger,
)

// Create a provider function
providerFn := func(log *zap.Logger) (interface{}, error) {
    return myservice.NewService(config, log)
}

// Create a lazy provider
provider := lazy.NewEnhancedLazyProvider(
    "my-service",
    providerFn,
    logger,
    nil, // Metrics will be handled by the coordinator
    lazy.WithMemoryEstimate(100*1024*1024), // 100MB estimate
    lazy.WithTimeout(30*time.Second),
    lazy.WithPriority(20), // High priority
)

// Register with the coordinator, specifying dependencies
err := coordinator.RegisterComponent(
    "my-service",
    "service",
    provider,
    []string{"database-service", "cache-service"}, // Dependencies
)
if err != nil {
    // Handle error
}

// The coordinator will ensure that dependencies are initialized first
service, err := coordinator.GetComponent(ctx, "my-service")
if err != nil {
    // Handle error
}

// Use the service
// ...

// Shutdown the coordinator
coordinator.Shutdown(ctx)
```

## Monitoring

To monitor lazy-loaded components, use the dashboard:

```go
// Create a component coordinator
coordinator := coordination.NewComponentCoordinator(
    coordination.DefaultCoordinatorConfig(),
    logger,
)

// Create and register components
// ...

// Create the dashboard
dashboard := monitoring.NewLazyComponentDashboard(coordinator, logger)

// Start the dashboard
go func() {
    err := dashboard.Start(":8080")
    if err != nil {
        logger.Error("Failed to start dashboard", zap.Error(err))
    }
}()

// Use components as needed
// ...

// Stop the dashboard
dashboard.Stop(ctx)

// Shutdown the coordinator
coordinator.Shutdown(ctx)
```

The dashboard provides a web interface at http://localhost:8080 with real-time metrics and component information.

## Integration with Application

To integrate the lazy loading system with your application, use the provided module:

```go
// In your application's main package
import (
    "github.com/abdoElHodaky/tradSys/internal/app"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        // Include the lazy integration module
        app.LazyIntegrationModule,
        
        // Include your application modules
        myapp.Module,
        
        // Provide any additional dependencies
        fx.Provide(
            // ...
        ),
        
        // Register lifecycle hooks
        fx.Invoke(func(
            lc fx.Lifecycle,
            coordinator *coordination.ComponentCoordinator,
            logger *zap.Logger,
        ) {
            // Register shutdown hook
            lc.Append(fx.Hook{
                OnStop: func(ctx context.Context) error {
                    return coordinator.Shutdown(ctx)
                },
            })
        }),
    ).Run()
}
```

## Best Practices

1. **Share the Coordinator**: Use a single `ComponentCoordinator` for all lazy-loaded components to ensure proper resource management.

2. **Set Appropriate Memory Estimates**: Provide accurate memory estimates when creating lazy providers to help the memory manager make informed decisions.

3. **Use Priorities**: Set appropriate priorities for components based on their importance. Critical components should have higher priorities (lower numbers).

4. **Handle Timeouts**: Set appropriate timeouts for component initialization to prevent hanging.

5. **Shutdown Components**: Always shutdown components when they are no longer needed to free resources.

6. **Monitor Memory Usage**: Use the dashboard or memory manager to monitor memory usage and pressure levels.

7. **Use Dependency Management**: Specify component dependencies when registering components to ensure proper initialization order.

8. **Handle Errors**: Always check for errors when creating, using, or shutting down components.

9. **Use the Appropriate Manager**: Use the appropriate manager for dynamic loading (e.g., `LazyConnectorManager` for exchange connectors).

10. **Configure Memory Pressure Thresholds**: Configure memory pressure thresholds based on your application's requirements and available resources.

## Conclusion

The lazy loading system provides a flexible and efficient way to manage components in the TradSys codebase. By following these common usage patterns and best practices, you can ensure that your application makes the most of the system's capabilities while maintaining resource efficiency and reliability.

