# Plugin Development Guide

This guide provides instructions for developing plugins for the tradSys platform. The platform supports two main types of plugins:

1. **Exchange Connector Plugins**: For connecting to trading exchanges
2. **Technical Indicator Plugins**: For implementing custom technical indicators

## Plugin Architecture

Plugins in tradSys are implemented using Go's plugin system, which allows for dynamic loading of code at runtime. Each plugin is compiled as a shared object (`.so` file) that exports specific symbols that the platform can use to interact with the plugin.

### Plugin Lifecycle

1. **Loading**: The platform scans the plugin directory and loads all `.so` files
2. **Initialization**: The platform initializes the plugin by calling its initialization function
3. **Usage**: The platform creates instances of the plugin's components as needed
4. **Shutdown**: When the platform shuts down, it calls the plugin's shutdown function

## Exchange Connector Plugins

Exchange connector plugins allow the platform to connect to different trading exchanges.

### Required Exports

Each exchange connector plugin must export the following symbols:

1. **PluginInfo**: A struct containing information about the plugin
2. **CreateConnector**: A function that creates an exchange connector
3. **InitializePlugin** (optional): A function that initializes the plugin
4. **ShutdownPlugin** (optional): A function that shuts down the plugin

### Example Implementation

```go
package main

import (
    "github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
    "github.com/abdoElHodaky/tradSys/internal/exchange/connectors/plugin"
    "go.uber.org/zap"
)

// PluginInfo contains information about the plugin
var PluginInfo = &plugin.PluginInfo{
    Name:           "Example Exchange Connector",
    Version:        "1.0.0",
    Author:         "Your Name",
    Description:    "An example exchange connector plugin",
    ExchangeName:   "example-exchange",
    APIVersion:     "1.0.0",
    MinCoreVersion: "1.0.0",
    MaxCoreVersion: "",
    Dependencies:   []string{},
}

// CreateConnector creates an exchange connector
func CreateConnector(config connectors.ExchangeConfig, logger *zap.Logger) (connectors.ExchangeConnector, error) {
    return NewExampleConnector(config, logger)
}

// InitializePlugin initializes the plugin
func InitializePlugin() error {
    // Perform any initialization here
    return nil
}

// ShutdownPlugin shuts down the plugin
func ShutdownPlugin() error {
    // Perform any cleanup here
    return nil
}

// ExampleConnector implements the ExchangeConnector interface
type ExampleConnector struct {
    config connectors.ExchangeConfig
    logger *zap.Logger
}

// NewExampleConnector creates a new example connector
func NewExampleConnector(config connectors.ExchangeConfig, logger *zap.Logger) (*ExampleConnector, error) {
    return &ExampleConnector{
        config: config,
        logger: logger,
    }, nil
}

// Implement the ExchangeConnector interface methods...
```

## Technical Indicator Plugins

Technical indicator plugins allow the platform to use custom technical indicators for market analysis.

### Required Exports

Each technical indicator plugin must export the following symbols:

1. **PluginInfo**: A struct containing information about the plugin
2. **CreateIndicator**: A function that creates a technical indicator
3. **InitializePlugin** (optional): A function that initializes the plugin
4. **ShutdownPlugin** (optional): A function that shuts down the plugin

### Example Implementation

```go
package main

import (
    "github.com/abdoElHodaky/tradSys/internal/trading/market_data/indicators"
    "github.com/abdoElHodaky/tradSys/internal/trading/market_data/indicators/plugin"
    "github.com/abdoElHodaky/tradSys/proto/marketdata"
    "go.uber.org/zap"
)

// PluginInfo contains information about the plugin
var PluginInfo = &plugin.PluginInfo{
    Name:           "Example Technical Indicator",
    Version:        "1.0.0",
    Author:         "Your Name",
    Description:    "An example technical indicator plugin",
    IndicatorName:  "example-indicator",
    DefaultParams: map[string]interface{}{
        "period": 14,
    },
    APIVersion:     "1.0.0",
    MinCoreVersion: "1.0.0",
    MaxCoreVersion: "",
    Dependencies:   []string{},
}

// CreateIndicator creates a technical indicator
func CreateIndicator(params indicators.IndicatorParams, logger *zap.Logger) (indicators.Indicator, error) {
    return NewExampleIndicator(params, logger)
}

// InitializePlugin initializes the plugin
func InitializePlugin() error {
    // Perform any initialization here
    return nil
}

// ShutdownPlugin shuts down the plugin
func ShutdownPlugin() error {
    // Perform any cleanup here
    return nil
}

// ExampleIndicator implements the Indicator interface
type ExampleIndicator struct {
    params indicators.IndicatorParams
    logger *zap.Logger
}

// NewExampleIndicator creates a new example indicator
func NewExampleIndicator(params indicators.IndicatorParams, logger *zap.Logger) (*ExampleIndicator, error) {
    return &ExampleIndicator{
        params: params,
        logger: logger,
    }, nil
}

// Implement the Indicator interface methods...
```

## Building Plugins

To build a plugin, use the `-buildmode=plugin` flag with the Go compiler:

```bash
go build -buildmode=plugin -o example_exchange.so example_exchange.go
```

## Plugin Versioning

Plugins must specify their API version and the core version range they are compatible with:

- **APIVersion**: The version of the plugin API
- **MinCoreVersion**: The minimum core version the plugin is compatible with
- **MaxCoreVersion**: The maximum core version the plugin is compatible with (empty string means compatible with any future version)

The platform will check these versions when loading plugins to ensure compatibility.

## Best Practices

1. **Error Handling**: Always handle errors properly and provide meaningful error messages
2. **Resource Management**: Clean up resources when the plugin is shut down
3. **Thread Safety**: Ensure your plugin is thread-safe, as it may be used by multiple goroutines
4. **Logging**: Use the provided logger for all logging
5. **Configuration**: Use the provided configuration for all configurable parameters
6. **Versioning**: Keep your plugin's API version and core version range up to date

## Troubleshooting

### Common Issues

1. **Plugin Not Loading**: Ensure the plugin is compiled with the correct Go version and architecture
2. **Symbol Not Found**: Ensure the plugin exports all required symbols
3. **Version Mismatch**: Ensure the plugin's API version and core version range are compatible with the platform
4. **Panic During Initialization**: Check for nil pointers or other initialization issues

### Debugging

1. **Enable Debug Logging**: Set the log level to debug to see more detailed logs
2. **Check Plugin Directory**: Ensure the plugin is in the correct directory
3. **Check Plugin Permissions**: Ensure the plugin file has the correct permissions
4. **Check Dependencies**: Ensure all dependencies are available

## Example Plugins

See the `examples/plugins` directory for example plugins:

- `examples/plugins/exchange/binance.go`: An example Binance exchange connector plugin
- `examples/plugins/indicators/rsi.go`: An example RSI technical indicator plugin

