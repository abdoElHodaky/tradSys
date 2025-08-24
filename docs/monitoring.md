# Monitoring and Metrics

This document describes the monitoring and metrics capabilities of the tradSys platform, with a focus on lazy loading and plugin systems.

## Metrics Overview

The tradSys platform provides comprehensive metrics for monitoring the health, performance, and resource usage of the system. These metrics are particularly important for lazy loading and plugin systems, which can have complex initialization patterns and resource usage.

## Lazy Loading Metrics

The `LazyLoadingMetrics` type in the `internal/architecture/fx/lazy` package provides metrics for lazy loading:

```go
// LazyLoadingMetrics collects metrics for lazy loading
type LazyLoadingMetrics struct {
    mu                  sync.RWMutex
    initializations     map[string]int64
    initializationErr   map[string]int64
    initializationTimes map[string][]time.Duration
}
```

### Available Metrics

1. **Initialization Count**: The number of times a component has been initialized
2. **Initialization Error Count**: The number of times initialization has failed
3. **Average Initialization Time**: The average time it takes to initialize a component

### Accessing Metrics

```go
// Get initialization count
count := metrics.GetInitializationCount("component-name")

// Get initialization error count
errorCount := metrics.GetInitializationErrorCount("component-name")

// Get average initialization time
avgTime := metrics.GetAverageInitializationTime("component-name")
```

## Plugin System Metrics

The plugin system provides metrics for monitoring plugin loading, usage, and health:

### Plugin Loading Metrics

1. **Plugin Load Count**: The number of plugins loaded
2. **Plugin Load Error Count**: The number of plugin loading errors
3. **Plugin Load Time**: The time it takes to load plugins

### Plugin Usage Metrics

1. **Plugin Usage Count**: The number of times a plugin has been used
2. **Plugin Creation Time**: The time it takes to create a plugin instance
3. **Plugin Error Count**: The number of errors encountered when using a plugin

### Plugin Health Metrics

1. **Plugin Health Status**: The health status of a plugin
2. **Plugin Resource Usage**: The resources used by a plugin
3. **Plugin Error Rate**: The rate of errors encountered when using a plugin

## Health Monitoring

The tradSys platform provides health monitoring for components and plugins:

### Component Health

```go
// Check if a component is healthy
if !provider.IsInitialized() {
    logger.Warn("Component not initialized")
}

// Get initialization error
_, err := provider.Get()
if err != nil {
    logger.Error("Component initialization failed", zap.Error(err))
}
```

### Plugin Health

```go
// Check if a connector is healthy
healthy := registry.GetConnectorHealth("exchange-name")

// Set connector health status
registry.SetConnectorHealth("exchange-name", false)
```

## Resource Monitoring

The tradSys platform provides resource monitoring for components and plugins:

### Memory Usage

```go
// Get memory usage of historical data service
memoryUsage := historicalDataService.GetMemoryUsage()

// Get cache statistics
cacheStats := historicalDataService.GetCacheStats()
```

### Connection Monitoring

```go
// Get active connections
activeConnections := registry.GetActiveConnections()

// Get connection statistics
connectionStats := registry.GetConnectionStats()
```

## Logging

The tradSys platform uses structured logging with `zap` to provide detailed information about the system:

```go
// Log component initialization
logger.Info("Lazily initializing component",
    zap.String("component", "risk-manager"),
    zap.Duration("duration", time.Since(startTime)))

// Log plugin loading
logger.Info("Loaded exchange connector plugin",
    zap.String("name", info.Name),
    zap.String("version", info.Version),
    zap.String("author", info.Author),
    zap.String("exchange", info.ExchangeName))

// Log errors
logger.Error("Failed to initialize component",
    zap.String("component", "risk-manager"),
    zap.Error(err))
```

## Alerting

The tradSys platform can be configured to send alerts when certain conditions are met:

### Health Alerts

```go
// Send alert when component health changes
if !provider.IsInitialized() {
    alerts.Send("Component not initialized", "component-name")
}

// Send alert when plugin health changes
if !registry.GetConnectorHealth("exchange-name") {
    alerts.Send("Connector unhealthy", "exchange-name")
}
```

### Resource Alerts

```go
// Send alert when memory usage exceeds threshold
if historicalDataService.GetMemoryUsage() > threshold {
    alerts.Send("Memory usage exceeded threshold", "historical-data-service")
}

// Send alert when cache size exceeds threshold
cacheStats := historicalDataService.GetCacheStats()
if cacheStats["entries"].(int) > threshold {
    alerts.Send("Cache size exceeded threshold", "historical-data-service")
}
```

## Dashboard Integration

The tradSys platform can be integrated with monitoring dashboards:

### Prometheus Integration

```go
// Register metrics with Prometheus
prometheus.Register(metrics)

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
```

### Grafana Dashboard

The tradSys platform provides a Grafana dashboard for monitoring the system:

1. **System Overview**: Overall system health and performance
2. **Lazy Loading**: Initialization times and error rates
3. **Plugin System**: Plugin loading, usage, and health
4. **Resource Usage**: Memory, CPU, and connection usage
5. **Error Rates**: Error rates for components and plugins

## Best Practices

### 1. Monitor Initialization Times

```go
// Record initialization time
startTime := time.Now()
component, err := provider.Get()
logger.Info("Component initialization",
    zap.String("component", "risk-manager"),
    zap.Duration("duration", time.Since(startTime)))
```

### 2. Track Resource Usage

```go
// Track memory usage
memoryUsage := historicalDataService.GetMemoryUsage()
logger.Info("Memory usage",
    zap.String("component", "historical-data-service"),
    zap.Int64("memory_usage", memoryUsage))
```

### 3. Monitor Health Status

```go
// Monitor component health
if !provider.IsInitialized() {
    logger.Warn("Component not initialized",
        zap.String("component", "risk-manager"))
}

// Monitor plugin health
if !registry.GetConnectorHealth("exchange-name") {
    logger.Warn("Connector unhealthy",
        zap.String("exchange", "exchange-name"))
}
```

### 4. Set Up Alerts

```go
// Set up alerts for component health
alerts.SetupHealthAlert("component-name", func() bool {
    return provider.IsInitialized()
})

// Set up alerts for resource usage
alerts.SetupResourceAlert("historical-data-service", func() bool {
    return historicalDataService.GetMemoryUsage() < threshold
})
```

### 5. Use Structured Logging

```go
// Use structured logging
logger.Info("Component initialized",
    zap.String("component", "risk-manager"),
    zap.Duration("duration", time.Since(startTime)),
    zap.Int("cache_size", cacheSize))
```

## Conclusion

Proper monitoring and metrics are essential for maintaining a healthy and performant system. The tradSys platform provides comprehensive monitoring capabilities for lazy loading and plugin systems, allowing you to track initialization times, resource usage, and health status.

