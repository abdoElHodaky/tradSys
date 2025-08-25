# Lazy Loading Integration with Coordination System

This document describes how the coordination system has been integrated with the existing lazy loading components in the TradSys codebase.

## Overview

The integration provides a unified approach to lazy loading and dynamic loading of components, addressing the following key aspects:

1. **Resource Management**: Centralized memory management and allocation
2. **Lifecycle Control**: Unified component initialization and shutdown
3. **Dependency Management**: Proper handling of component dependencies
4. **Metrics Collection**: Consolidated metrics for all lazy-loaded components
5. **Timeout Handling**: Consistent timeout management across components

## Integrated Components

### 1. Historical Data Service

The `LazyHistoricalDataService` provides lazy loading for the historical market data service:

```go
type LazyHistoricalDataService struct {
    coordinator   *coordination.ComponentCoordinator
    componentName string
    config        historical.Config
    logger        *zap.Logger
}
```

**Key Features**:
- Memory-aware initialization based on cache size
- Timeout handling for data requests
- Resource tracking for historical data

### 2. Order Management Service

The `LazyOrderService` provides lazy loading for the order management service:

```go
type LazyOrderService struct {
    coordinator   *coordination.ComponentCoordinator
    componentName string
    config        order_management.OrderServiceConfig
    logger        *zap.Logger
}
```

**Key Features**:
- High-priority initialization for trading operations
- Consistent timeout handling for order operations
- Resource tracking for order management

### 3. Matching Algorithm Plugins

The `LazyPluginLoader` provides dynamic loading for matching algorithm plugins:

```go
type LazyPluginLoader struct {
    coordinator         *coordination.ComponentCoordinator
    componentNamePrefix string
    config              plugin.LoaderConfig
    logger              *zap.Logger
    lockManager         *coordination.LockManager
}
```

**Key Features**:
- Thread-safe plugin loading with lock management
- Memory-aware plugin initialization
- Dynamic loading and unloading of algorithms

### 4. WebSocket Server

The `LazyOptimizedWebSocketServer` provides lazy loading for the WebSocket server:

```go
type LazyOptimizedWebSocketServer struct {
    coordinator   *coordination.ComponentCoordinator
    componentName string
    config        ws.WebSocketConfig
    logger        *zap.Logger
}
```

**Key Features**:
- Very high-priority initialization for client connections
- Memory-aware buffer and worker pool management
- Resource tracking for WebSocket connections

### 5. Exchange Connectors

The `LazyConnectorManager` provides dynamic loading for exchange connectors:

```go
type LazyConnectorManager struct {
    coordinator         *coordination.ComponentCoordinator
    componentNamePrefix string
    config              connectors.ConnectorConfig
    logger              *zap.Logger
    lockManager         *coordination.LockManager
    factory             connectors.ConnectorFactory
    activeConnectors    map[string]bool
    activeConnectorsMu  sync.RWMutex
}
```

**Key Features**:
- Dynamic loading of exchange connectors
- Thread-safe connector management
- Resource tracking for exchange connections

### 6. Risk Validator Plugins

The `LazyValidatorRegistry` provides dynamic loading for risk validator plugins:

```go
type LazyValidatorRegistry struct {
    coordinator         *coordination.ComponentCoordinator
    componentNamePrefix string
    config              plugin.RegistryConfig
    logger              *zap.Logger
    lockManager         *coordination.LockManager
    loader              *plugin.Loader
    activeValidators    map[string]bool
    activeValidatorsMu  sync.RWMutex
}
```

**Key Features**:
- Dynamic loading of risk validators
- Thread-safe validator management
- Resource tracking for validator plugins

### 7. Strategy Components

The `LazyStrategyManager` provides dynamic loading for trading strategies:

```go
type LazyStrategyManager struct {
    coordinator         *coordination.ComponentCoordinator
    componentNamePrefix string
    logger              *zap.Logger
    lockManager         *coordination.LockManager
    factory             strategy.StrategyFactory
    activeStrategies    map[string]bool
    activeStrategiesMu  sync.RWMutex
}
```

**Key Features**:
- Dynamic loading of trading strategies
- Memory estimation based on strategy type
- Resource tracking for strategy components

### 8. Connection Pool

The `LazyConnectionPool` provides lazy loading for the connection pool:

```go
type LazyConnectionPool struct {
    coordinator   *coordination.ComponentCoordinator
    componentName string
    config        performance.PoolConfig
    logger        *zap.Logger
}
```

**Key Features**:
- Memory-aware pool initialization
- High-priority initialization for connections
- Resource tracking for connection pool

## Integration Architecture

The integration follows a consistent pattern across all components:

1. **Component Registration**:
   - Each component is registered with the `ComponentCoordinator`
   - Memory estimates are provided for resource management
   - Priorities are set based on component importance
   - Timeouts are configured for initialization

2. **Lazy Initialization**:
   - Components are initialized only when first accessed
   - The coordinator manages memory allocation before initialization
   - Initialization follows dependency order when applicable

3. **Method Delegation**:
   - Each lazy wrapper delegates method calls to the underlying component
   - The component is retrieved from the coordinator on each call
   - Type checking ensures correct component types

4. **Resource Management**:
   - Memory usage is tracked by the coordinator
   - Components can be shut down to free resources
   - Priority-based memory allocation ensures critical components get resources

5. **Thread Safety**:
   - Lock management ensures thread-safe component access
   - Deadlock detection prevents lock acquisition issues
   - Consistent lock acquisition order is enforced

## Usage Example

Here's an example of how to use the lazy-loaded historical data service:

```go
// Create the coordinator and other required components
coordinator := coordination.NewComponentCoordinator(
    coordination.DefaultCoordinatorConfig(),
    logger,
)

// Create the lazy historical data service
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

// The service can be shut down when no longer needed
err = lazyService.Shutdown(ctx)
if err != nil {
    // Handle error
}
```

## Application Integration

The integration is provided through the `LazyIntegrationModule` in the `internal/app` package:

```go
var LazyIntegrationModule = fx.Options(
    coordination.Module,
    fx.Provide(
        NewLazyHistoricalDataService,
        NewLazyOrderService,
        NewLazyMatchingAlgorithmLoader,
        NewLazyWebSocketServer,
        NewLazyConnectorManager,
        NewLazyValidatorRegistry,
        NewLazyStrategyManager,
        NewLazyConnectionPool,
    ),
)
```

This module can be included in the application to enable lazy loading for all supported components.

## Benefits of Integration

1. **Reduced Memory Usage**:
   - Components are only initialized when needed
   - Memory is allocated based on priority and availability
   - Unused components can be unloaded to free memory

2. **Improved Startup Time**:
   - Components are initialized lazily, reducing startup time
   - Initialization follows dependency order, preventing deadlocks
   - Critical components can be prioritized

3. **Better Resource Management**:
   - Centralized memory management prevents memory spikes
   - Consistent timeout handling prevents resource leaks
   - Unified metrics collection provides visibility into resource usage

4. **Enhanced Reliability**:
   - Deadlock detection prevents lock-related issues
   - Consistent error handling improves reliability
   - Resource limits prevent out-of-memory conditions

5. **Simplified Development**:
   - Consistent pattern across all components
   - Clear separation of concerns
   - Unified configuration system

## Conclusion

The integration of the coordination system with the existing lazy loading components provides a comprehensive solution for resource management, lifecycle control, and dependency management. By centralizing these concerns, the system ensures consistent behavior across all components while maintaining the benefits of lazy loading and dynamic loading.

The system is designed to be flexible and configurable, allowing for fine-tuning based on specific requirements and resource constraints. It integrates seamlessly with the existing codebase, enhancing its capabilities without requiring significant changes to the component interfaces.

