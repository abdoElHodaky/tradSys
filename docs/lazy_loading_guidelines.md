# Lazy Loading Guidelines

This document provides guidelines for implementing and using lazy loading in the tradSys platform.

## What is Lazy Loading?

Lazy loading is a design pattern that defers the initialization of an object until it is needed. This can improve performance by:

1. Reducing startup time
2. Conserving memory
3. Avoiding unnecessary initialization of components that may not be used

## Lazy Loading Architecture

The tradSys platform implements lazy loading using the `LazyProvider` pattern in the `internal/architecture/fx/lazy` package. This pattern integrates with the dependency injection framework (Uber FX) to provide lazy initialization of components.

### Key Components

1. **LazyProvider**: Wraps a component constructor for lazy initialization
2. **LazyLoadingMetrics**: Collects metrics for lazy loading
3. **Lazy Modules**: FX modules that provide lazily loaded components

## When to Use Lazy Loading

Lazy loading is appropriate for components that:

1. Are resource-intensive to initialize
2. Are not always needed during normal operation
3. Have a significant impact on startup time
4. Consume substantial memory

Examples of good candidates for lazy loading:

- Risk management systems
- Historical data analysis components
- Advanced order matching engines
- Reporting services

## Implementing Lazy Loading

### Step 1: Create a Lazy Module

Create a new file in the component's package under an `fx` subdirectory:

```go
// internal/component/fx/lazy_module.go
package fx

import (
    "github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
    "github.com/abdoElHodaky/tradSys/internal/component"
    "go.uber.org/fx"
    "go.uber.org/zap"
)

// LazyComponentModule provides lazily loaded components
var LazyComponentModule = fx.Options(
    // Provide lazily loaded components
    provideLazyComponent,
    
    // Register lifecycle hooks
    fx.Invoke(registerLazyComponentHooks),
)

// provideLazyComponent provides a lazily loaded component
func provideLazyComponent(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
    return lazy.NewLazyProvider(
        "component-name",
        func(config *component.Config, logger *zap.Logger) (*component.Component, error) {
            logger.Info("Lazily initializing component")
            return component.NewComponent(config, logger)
        },
        logger,
        metrics,
    )
}

// registerLazyComponentHooks registers lifecycle hooks for the lazy component
func registerLazyComponentHooks(
    lc fx.Lifecycle,
    logger *zap.Logger,
    componentProvider *lazy.LazyProvider,
) {
    logger.Info("Registering lazy component hooks")
    
    // Register shutdown hook to clean up resources
    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            // Only clean up if the component was initialized
            if !componentProvider.IsInitialized() {
                return nil
            }
            
            // Get the component
            instance, err := componentProvider.Get()
            if err != nil {
                logger.Error("Failed to get component during shutdown", zap.Error(err))
                return err
            }
            
            // Clean up resources
            component := instance.(*component.Component)
            // Perform cleanup...
            
            logger.Info("Component resources cleaned up")
            return nil
        },
    })
}

// GetComponent gets the component, initializing it if necessary
func GetComponent(provider *lazy.LazyProvider) (*component.Component, error) {
    instance, err := provider.Get()
    if err != nil {
        return nil, err
    }
    return instance.(*component.Component), nil
}
```

### Step 2: Register the Lazy Module

Register the lazy module in your application's FX module:

```go
// internal/app/app.go
package app

import (
    "github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
    componentfx "github.com/abdoElHodaky/tradSys/internal/component/fx"
    "go.uber.org/fx"
)

var Module = fx.Options(
    // Provide lazy loading metrics
    fx.Provide(lazy.NewLazyLoadingMetrics),
    
    // Register lazy modules
    componentfx.LazyComponentModule,
    
    // ... other modules
)
```

### Step 3: Use the Lazy Component

Use the lazy component in your application:

```go
// internal/app/service.go
package app

import (
    "github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
    componentfx "github.com/abdoElHodaky/tradSys/internal/component/fx"
    "go.uber.org/zap"
)

type Service struct {
    logger            *zap.Logger
    componentProvider *lazy.LazyProvider
}

func NewService(logger *zap.Logger, componentProvider *lazy.LazyProvider) *Service {
    return &Service{
        logger:            logger,
        componentProvider: componentProvider,
    }
}

func (s *Service) DoSomething() error {
    // Get the component, which will be lazily initialized if needed
    component, err := componentfx.GetComponent(s.componentProvider)
    if err != nil {
        return err
    }
    
    // Use the component
    return component.DoSomething()
}
```

## Best Practices

### 1. Proper Resource Management

Always clean up resources when a lazily loaded component is no longer needed:

```go
// Register shutdown hook
lc.Append(fx.Hook{
    OnStop: func(ctx context.Context) error {
        if !provider.IsInitialized() {
            return nil
        }
        
        instance, err := provider.Get()
        if err != nil {
            return err
        }
        
        component := instance.(*Component)
        return component.Cleanup()
    },
})
```

### 2. Error Handling

Handle initialization errors properly:

```go
component, err := GetComponent(provider)
if err != nil {
    // Handle error
    logger.Error("Failed to initialize component", zap.Error(err))
    return err
}
```

### 3. Thread Safety

Ensure thread safety when accessing lazily loaded components:

```go
// The LazyProvider is thread-safe, but your component might not be
component, err := GetComponent(provider)
if err != nil {
    return err
}

// If your component is not thread-safe, use appropriate synchronization
component.mu.Lock()
defer component.mu.Unlock()
component.DoSomething()
```

### 4. Metrics and Monitoring

Use the provided metrics to monitor lazy loading:

```go
// Get initialization count
count := metrics.GetInitializationCount("component-name")

// Get initialization error count
errorCount := metrics.GetInitializationErrorCount("component-name")

// Get average initialization time
avgTime := metrics.GetAverageInitializationTime("component-name")
```

### 5. Avoid Circular Dependencies

Be careful with circular dependencies when using lazy loading:

```go
// AVOID THIS:
// Component A depends on Component B
// Component B depends on Component A
```

### 6. Documentation

Document which components are lazily loaded and why:

```go
// LazyRiskModule provides lazily loaded risk management components.
// These components are lazily loaded because:
// 1. They are resource-intensive to initialize
// 2. They are only needed when executing trades
// 3. They have a significant impact on startup time
var LazyRiskModule = fx.Options(...)
```

## Troubleshooting

### Common Issues

1. **Initialization Failures**: Check the error returned by `Get()` and the logs for initialization errors
2. **Performance Issues**: Monitor initialization times using the metrics
3. **Memory Leaks**: Ensure resources are properly cleaned up when components are no longer needed
4. **Deadlocks**: Be careful with locks and avoid circular dependencies

### Debugging

1. **Enable Debug Logging**: Set the log level to debug to see more detailed logs
2. **Check Metrics**: Use the metrics to monitor initialization times and error counts
3. **Check Initialization Status**: Use `IsInitialized()` to check if a component has been initialized

## Examples

See the following examples of lazy loading in the tradSys platform:

1. **Risk Management System**: `internal/risk/fx/lazy_module.go`
2. **Historical Data Analysis**: `internal/trading/market_data/historical/fx/lazy_module.go`

