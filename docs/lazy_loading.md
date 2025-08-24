# Lazy Loading and Dynamic Loading

This document describes the lazy loading and dynamic loading features in TradSys.

## Lazy Loading

Lazy loading defers the initialization of components until they are actually needed. This can significantly reduce startup time and memory usage for components that are not used in every session.

### Architecture

The lazy loading architecture consists of the following components:

- `LazyProvider`: Wraps a component constructor for lazy initialization
- `ProxyModule`: Creates a proxy for a module that defers initialization
- `LazyLifecycle`: Manages the lifecycle of lazily loaded components
- `LazyLoadingMetrics`: Collects metrics for lazy loading

### Usage

#### Creating a Lazy Provider

```go
// Create a lazy provider
provider := lazy.NewLazyProvider(
    "component-name",
    func() (*MyComponent, error) {
        return NewMyComponent(), nil
    },
    logger,
    metrics,
)

// Get the component (initializes it if needed)
component, err := provider.Get()
if err != nil {
    // Handle error
}

// Use the component
myComponent := component.(*MyComponent)
```

#### Creating a Proxy Module

```go
// Create a proxy module
module := lazy.NewProxyModule(
    "module-name",
    func() (*MyModule, error) {
        return NewMyModule(), nil
    },
    logger,
    metrics,
)

// Register the module with fx
app := fx.New(
    module.AsOption(),
    // Other options...
)
```

#### Using Lazy Loading with fx

```go
// Create an fx application with lazy loading
app := fx.New(
    lazy.Module,
    lazy.ProvideLazy("my-component", func() (*MyComponent, error) {
        return NewMyComponent(), nil
    }),
    // Other options...
)
```

### Benefits

- Reduced startup time
- Lower memory usage
- Improved resource utilization
- Better scalability

## Dynamic Loading (Plugins)

Dynamic loading allows loading components at runtime from external plugin files. This enables extending the system without recompiling the main application.

### Architecture

The plugin architecture consists of the following components:

- `PluginLoader`: Loads strategy plugins from .so files
- `PluginRegistry`: Manages strategy plugins and registers them with the strategy factory
- `StrategyPlugin`: Interface that must be implemented by plugins

### Creating a Plugin

To create a plugin, you need to:

1. Implement the `StrategyPlugin` interface
2. Export the required symbols
3. Compile the plugin as a shared object (.so) file

Example plugin:

```go
package main

import (
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/abdoElHodaky/tradSys/internal/strategy/plugin"
	"go.uber.org/zap"
)

// PluginInfo is the exported plugin information
var PluginInfo = &plugin.PluginInfo{
	Name:        "MyStrategy",
	Version:     "1.0.0",
	Author:      "Your Name",
	Description: "My custom strategy",
	StrategyType: "my-strategy",
}

// CreateStrategy is the exported function to create a strategy
func CreateStrategy(config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error) {
	return NewMyStrategy(config, logger)
}

// MyStrategy implements the Strategy interface
type MyStrategy struct {
	// Implementation...
}

// Implement the Strategy interface methods...
```

### Compiling a Plugin

```bash
go build -buildmode=plugin -o my_strategy.so my_strategy.go
```

### Loading Plugins

Plugins are automatically loaded from the plugin directory when the application starts. The default plugin directory is `/etc/tradsys/plugins`.

### Benefits

- Extensibility without recompilation
- Third-party strategy integration
- Isolation of custom code
- Versioning and updates without downtime

## Integration with Strategy System

Both lazy loading and dynamic loading are integrated with the strategy system:

- The strategy manager and metrics collector are lazily loaded
- Strategy plugins can be dynamically loaded and registered with the strategy factory

Example:

```go
// Create an fx application with lazy loading and plugin support
app := fx.New(
    lazy.Module,
    strategy.fx.LazyModule,
    strategy.plugin.Module,
    // Other options...
)
```

## Performance Considerations

- Lazy loading may introduce a slight delay when a component is first accessed
- Plugin loading adds some overhead at startup
- Plugins may have slightly lower performance than built-in components due to the dynamic loading mechanism

## Metrics and Monitoring

The lazy loading system collects the following metrics:

- Number of initializations per component
- Number of initialization errors per component
- Initialization times per component

These metrics can be used to monitor the performance and health of the lazy loading system.

