# Troubleshooting Guide

This guide provides solutions for common issues encountered when working with the tradSys platform, particularly focusing on lazy loading and plugin systems.

## Lazy Loading Issues

### Component Not Initializing

**Symptoms:**
- Error: "failed to get component: component not initialized"
- Component functions return errors

**Possible Causes:**
1. Constructor function is failing
2. Dependencies are not available
3. Circular dependencies

**Solutions:**
1. Check the logs for initialization errors
2. Ensure all dependencies are properly provided
3. Break circular dependencies by using lazy loading for one of the components
4. Check the component constructor for errors

```go
// Example of checking initialization status
if !provider.IsInitialized() {
    logger.Error("Component not initialized")
    // Handle the error
}
```

### Memory Leaks

**Symptoms:**
- Increasing memory usage over time
- Performance degradation

**Possible Causes:**
1. Resources not being cleaned up
2. Cached data not being evicted

**Solutions:**
1. Implement proper cleanup in shutdown hooks
2. Add cache eviction policies
3. Monitor memory usage with metrics

```go
// Example of cleanup in shutdown hook
lc.Append(fx.Hook{
    OnStop: func(ctx context.Context) error {
        if !provider.IsInitialized() {
            return nil
        }
        
        component, err := GetComponent(provider)
        if err != nil {
            return err
        }
        
        return component.Cleanup()
    },
})
```

### Performance Issues

**Symptoms:**
- Slow initialization
- Delays when accessing components

**Possible Causes:**
1. Heavy initialization logic
2. Resource contention
3. Network or I/O operations during initialization

**Solutions:**
1. Optimize initialization code
2. Use caching where appropriate
3. Move network or I/O operations to background goroutines
4. Monitor initialization times with metrics

```go
// Example of monitoring initialization times
avgTime := metrics.GetAverageInitializationTime("component-name")
logger.Info("Component initialization time", zap.Duration("avg_time", avgTime))
```

## Plugin System Issues

### Plugin Not Loading

**Symptoms:**
- Error: "failed to open plugin: plugin.Open("path/to/plugin.so"): no such file or directory"
- Plugin not appearing in available plugins list

**Possible Causes:**
1. Plugin file not found
2. Plugin file not accessible
3. Plugin compiled with wrong Go version
4. Plugin compiled for wrong architecture

**Solutions:**
1. Check if the plugin file exists in the correct directory
2. Check file permissions
3. Recompile the plugin with the correct Go version
4. Recompile the plugin for the correct architecture

```bash
# Example of compiling a plugin with the correct Go version and architecture
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o plugin.so plugin.go
```

### Symbol Not Found

**Symptoms:**
- Error: "plugin does not export PluginInfo: plugin has no symbol named PluginInfo"
- Error: "plugin does not export CreateConnector: plugin has no symbol named CreateConnector"

**Possible Causes:**
1. Plugin does not export the required symbols
2. Symbol names are incorrect
3. Plugin compiled with different package structure

**Solutions:**
1. Ensure the plugin exports all required symbols
2. Check symbol names for typos
3. Ensure the plugin is compiled with the correct package structure

```go
// Example of exporting required symbols
package main

import (
    "github.com/abdoElHodaky/tradSys/internal/exchange/connectors/plugin"
)

// PluginInfo must be exported
var PluginInfo = &plugin.PluginInfo{
    // ...
}

// CreateConnector must be exported
func CreateConnector(config connectors.ExchangeConfig, logger *zap.Logger) (connectors.ExchangeConnector, error) {
    // ...
}
```

### Version Mismatch

**Symptoms:**
- Error: "plugin API version 1.1.0 is not compatible with core version 1.0.0"
- Error: "plugin requires core version at least 1.2.0, but current version is 1.0.0"

**Possible Causes:**
1. Plugin API version is not compatible with the core version
2. Plugin requires a newer core version

**Solutions:**
1. Update the plugin to be compatible with the current core version
2. Update the core to a version compatible with the plugin
3. Use a different plugin that is compatible with the current core version

```go
// Example of specifying version compatibility
var PluginInfo = &plugin.PluginInfo{
    // ...
    APIVersion:     "1.0.0",
    MinCoreVersion: "1.0.0",
    MaxCoreVersion: "2.0.0",
}
```

### Plugin Crashes

**Symptoms:**
- Error: "panic in plugin: runtime error: invalid memory address or nil pointer dereference"
- Application crashes when using a plugin

**Possible Causes:**
1. Plugin has a bug
2. Plugin is accessing nil pointers
3. Plugin is not properly initialized

**Solutions:**
1. Fix the bug in the plugin
2. Add nil checks in the plugin
3. Ensure the plugin is properly initialized
4. Use panic recovery in the plugin loader

```go
// Example of panic recovery in plugin creation
func CreateConnector(config connectors.ExchangeConfig, logger *zap.Logger) (connectors.ExchangeConnector, error) {
    var connector connectors.ExchangeConnector
    var err error
    
    func() {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("panic in plugin: %v", r)
                logger.Error("Panic in plugin", zap.Any("panic", r))
            }
        }()
        
        connector = &ExampleConnector{
            config: config,
            logger: logger,
        }
    }()
    
    return connector, err
}
```

## Resource Management Issues

### Resource Leaks

**Symptoms:**
- Increasing memory usage
- Open file descriptors not being closed
- Network connections not being closed

**Possible Causes:**
1. Resources not being cleaned up
2. Deferred cleanup not being executed
3. Errors during cleanup being ignored

**Solutions:**
1. Implement proper cleanup in shutdown hooks
2. Use defer for cleanup operations
3. Handle errors during cleanup
4. Implement resource tracking

```go
// Example of proper resource cleanup
func (c *Component) Cleanup() error {
    var errs []error
    
    // Close file
    if c.file != nil {
        if err := c.file.Close(); err != nil {
            errs = append(errs, fmt.Errorf("failed to close file: %w", err))
        }
        c.file = nil
    }
    
    // Close connection
    if c.conn != nil {
        if err := c.conn.Close(); err != nil {
            errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
        }
        c.conn = nil
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("errors during cleanup: %v", errs)
    }
    
    return nil
}
```

### Cache Overflow

**Symptoms:**
- Increasing memory usage
- Performance degradation
- Out of memory errors

**Possible Causes:**
1. Cache not being evicted
2. Cache size not being limited
3. Cache entries not expiring

**Solutions:**
1. Implement cache eviction policies
2. Add cache size limits
3. Add cache entry expiration
4. Monitor cache size with metrics

```go
// Example of cache eviction
func (c *Cache) Add(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Check if we need to evict entries
    if len(c.entries) >= c.maxSize {
        c.evictOldest()
    }
    
    // Add the new entry
    c.entries[key] = &Entry{
        Value:     value,
        Timestamp: time.Now(),
    }
}

func (c *Cache) evictOldest() {
    var oldestKey string
    var oldestTime time.Time
    
    for key, entry := range c.entries {
        if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
            oldestKey = key
            oldestTime = entry.Timestamp
        }
    }
    
    if oldestKey != "" {
        delete(c.entries, oldestKey)
    }
}
```

## Concurrency Issues

### Race Conditions

**Symptoms:**
- Inconsistent behavior
- Unexpected values
- Panics with "concurrent map read/write"

**Possible Causes:**
1. Shared data being accessed without synchronization
2. Improper use of mutexes
3. Goroutines accessing shared data

**Solutions:**
1. Use proper synchronization (mutexes, channels)
2. Use sync.Map for concurrent map access
3. Use atomic operations for simple counters
4. Run tests with the race detector

```go
// Example of proper mutex usage
func (c *Component) GetValue(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    value, ok := c.data[key]
    return value, ok
}

func (c *Component) SetValue(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.data[key] = value
}
```

### Deadlocks

**Symptoms:**
- Application hangs
- Goroutines blocked indefinitely

**Possible Causes:**
1. Circular lock dependencies
2. Not releasing locks
3. Waiting for a channel that never receives a value

**Solutions:**
1. Avoid circular lock dependencies
2. Always release locks (use defer)
3. Use context with timeout for operations that might block
4. Use channels with select and timeout

```go
// Example of using context with timeout
func (c *Component) DoOperation(ctx context.Context) error {
    select {
    case c.semaphore <- struct{}{}:
        defer func() { <-c.semaphore }()
        // Do the operation
        return nil
    case <-ctx.Done():
        return fmt.Errorf("operation timed out: %w", ctx.Err())
    }
}
```

## Monitoring and Debugging

### Enabling Debug Logging

To enable debug logging, set the log level to debug:

```go
logger, _ := zap.NewDevelopment()
```

### Using Metrics

Use the provided metrics to monitor the system:

```go
// Get initialization count
count := metrics.GetInitializationCount("component-name")

// Get initialization error count
errorCount := metrics.GetInitializationErrorCount("component-name")

// Get average initialization time
avgTime := metrics.GetAverageInitializationTime("component-name")
```

### Checking Component Health

Use the health check functions to monitor component health:

```go
// Check if a connector is healthy
healthy := registry.GetConnectorHealth("exchange-name")

// Set connector health status
registry.SetConnectorHealth("exchange-name", false)
```

### Debugging Plugins

To debug plugins, use the following techniques:

1. **Enable Debug Logging**: Set the log level to debug
2. **Check Plugin Info**: Print the plugin info to verify it's loaded correctly
3. **Check Plugin Directory**: Ensure the plugin is in the correct directory
4. **Check Plugin Permissions**: Ensure the plugin file has the correct permissions
5. **Use Go's Plugin Debugging Tools**: Use `go tool trace` and `go tool pprof`

```go
// Example of debugging plugin loading
logger.Debug("Loading plugin",
    zap.String("path", path),
    zap.String("plugin_dir", l.pluginDir))

// After loading, print plugin info
logger.Debug("Plugin info",
    zap.String("name", info.Name),
    zap.String("version", info.Version),
    zap.String("api_version", info.APIVersion),
    zap.String("min_core_version", info.MinCoreVersion),
    zap.String("max_core_version", info.MaxCoreVersion))
```

