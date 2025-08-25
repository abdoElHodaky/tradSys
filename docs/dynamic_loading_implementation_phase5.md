# Dynamic Loading Implementation Plan - Phase 5: CQRS Integration

This document outlines the implementation details for Phase 5 of the dynamic loading improvements, focusing on CQRS (Command Query Responsibility Segregation) integration.

## 1. CQRS Architecture Overview

### 1.1 Core Principles

The CQRS pattern separates operations that read data (queries) from operations that update data (commands):

- **Commands**: Represent intentions to change the system state
- **Queries**: Represent requests for information without changing state
- **Events**: Represent facts about changes that have occurred

This separation allows for:
- Independent scaling of read and write workloads
- Specialized optimization of read and write paths
- Better handling of complex domain logic

### 1.2 Integration with Dynamic Loading

For the dynamic loading system, CQRS will be implemented with these components:

```
./internal/
  ├── plugin/
  │   ├── adaptive_loader/
  │   │   ├── commands/       # Command handlers for plugin operations
  │   │   ├── queries/        # Query handlers for plugin information
  │   │   ├── events/         # Event definitions and handlers
  │   │   └── models/         # Domain models
  │   └── cqrs/
  │       ├── command_bus.go  # Command dispatch infrastructure
  │       ├── query_bus.go    # Query dispatch infrastructure
  │       ├── event_bus.go    # Event publication infrastructure
  │       └── middleware.go   # Cross-cutting concerns (logging, validation)
  └── strategy/
      └── plugin/
          ├── commands/       # Strategy-specific commands
          ├── queries/        # Strategy-specific queries
          └── events/         # Strategy-specific events
```

## 2. Command Implementation

### 2.1 Command Structure

Commands are implemented as immutable data structures:

```go
// LoadPluginCommand represents a request to load a plugin
type LoadPluginCommand struct {
    FilePath     string
    Timeout      time.Duration
    Priority     int
    ValidateOnly bool
}

// UnloadPluginCommand represents a request to unload a plugin
type UnloadPluginCommand struct {
    PluginID string
    Force    bool
}
```

### 2.2 Command Handlers

Command handlers contain the business logic for processing commands:

```go
// LoadPluginCommandHandler handles the LoadPluginCommand
type LoadPluginCommandHandler struct {
    loader *AdaptivePluginLoader
    logger *zap.Logger
}

// Handle processes the LoadPluginCommand
func (h *LoadPluginCommandHandler) Handle(ctx context.Context, cmd *LoadPluginCommand) error {
    // Create a task for loading the plugin
    task := NewTask(fmt.Sprintf("load_plugin_%s", filepath.Base(cmd.FilePath)), func() error {
        return h.loader.LoadPlugin(ctx, cmd.FilePath)
    }).
        WithPriority(cmd.Priority).
        WithTimeout(cmd.Timeout)
    
    // Submit the task to the worker pool
    return h.loader.workerPool.Submit(task)
}
```

## 3. Query Implementation

### 3.1 Query Structure

Queries are also implemented as immutable data structures:

```go
// GetPluginInfoQuery represents a request for plugin information
type GetPluginInfoQuery struct {
    PluginID string
}

// ListLoadedPluginsQuery represents a request for all loaded plugins
type ListLoadedPluginsQuery struct {
    TypeFilter string
    Limit      int
    Offset     int
}
```

### 3.2 Query Handlers

Query handlers retrieve and return data without modifying state:

```go
// GetPluginInfoQueryHandler handles the GetPluginInfoQuery
type GetPluginInfoQueryHandler struct {
    loader *AdaptivePluginLoader
    logger *zap.Logger
}

// Handle processes the GetPluginInfoQuery
func (h *GetPluginInfoQueryHandler) Handle(ctx context.Context, query *GetPluginInfoQuery) (*PluginInfo, error) {
    // Use read lock for queries
    h.loader.pluginMu.RLock()
    defer h.loader.pluginMu.RUnlock()
    
    plugin, exists := h.loader.loadedPlugins[query.PluginID]
    if !exists {
        return nil, fmt.Errorf("plugin not found: %s", query.PluginID)
    }
    
    return plugin.Info, nil
}
```

## 4. Event System

### 4.1 Event Structure

Events represent facts about changes that have occurred:

```go
// PluginLoadedEvent represents the fact that a plugin was loaded
type PluginLoadedEvent struct {
    PluginID   string
    FilePath   string
    LoadTime   time.Time
    MemoryUsed int64
}

// PluginUnloadedEvent represents the fact that a plugin was unloaded
type PluginUnloadedEvent struct {
    PluginID     string
    UnloadTime   time.Time
    UnloadReason string
}
```

### 4.2 Event Handlers

Event handlers react to events without returning values:

```go
// PluginLoadedEventHandler handles the PluginLoadedEvent
type PluginLoadedEventHandler struct {
    metrics *MetricsCollector
    logger  *zap.Logger
}

// Handle processes the PluginLoadedEvent
func (h *PluginLoadedEventHandler) Handle(ctx context.Context, event *PluginLoadedEvent) {
    // Update metrics
    h.metrics.IncPluginLoadCount()
    h.metrics.ObservePluginLoadTime(event.LoadTime)
    h.metrics.ObservePluginMemoryUsage(event.MemoryUsed)
    
    // Log the event
    h.logger.Info("Plugin loaded",
        zap.String("plugin_id", event.PluginID),
        zap.String("file_path", event.FilePath),
        zap.Time("load_time", event.LoadTime),
        zap.Int64("memory_used", event.MemoryUsed))
}
```

## 5. Command and Query Buses

### 5.1 Command Bus

The command bus dispatches commands to their handlers:

```go
// CommandBus dispatches commands to their handlers
type CommandBus struct {
    handlers map[reflect.Type]CommandHandler
    middleware []CommandMiddleware
    logger   *zap.Logger
}

// Dispatch sends a command to its handler
func (b *CommandBus) Dispatch(ctx context.Context, command interface{}) error {
    commandType := reflect.TypeOf(command)
    handler, exists := b.handlers[commandType]
    if !exists {
        return fmt.Errorf("no handler registered for command type %v", commandType)
    }
    
    // Apply middleware
    next := handler.Handle
    for i := len(b.middleware) - 1; i >= 0; i-- {
        next = b.middleware[i](next)
    }
    
    return next(ctx, command)
}
```

### 5.2 Query Bus

The query bus dispatches queries to their handlers:

```go
// QueryBus dispatches queries to their handlers
type QueryBus struct {
    handlers map[reflect.Type]QueryHandler
    middleware []QueryMiddleware
    logger   *zap.Logger
}

// Dispatch sends a query to its handler
func (b *QueryBus) Dispatch(ctx context.Context, query interface{}) (interface{}, error) {
    queryType := reflect.TypeOf(query)
    handler, exists := b.handlers[queryType]
    if !exists {
        return nil, fmt.Errorf("no handler registered for query type %v", queryType)
    }
    
    // Apply middleware
    next := handler.Handle
    for i := len(b.middleware) - 1; i >= 0; i-- {
        next = b.middleware[i](next)
    }
    
    return next(ctx, query)
}
```

## 6. Integration with Existing System

### 6.1 Refactoring Existing Operations

Current direct method calls will be refactored to use the command and query buses:

**Before:**
```go
// Direct method call
err := loader.LoadPlugin(ctx, filePath)
```

**After:**
```go
// Using command bus
cmd := &LoadPluginCommand{FilePath: filePath}
err := commandBus.Dispatch(ctx, cmd)
```

### 6.2 Middleware Implementation

Middleware provides cross-cutting concerns:

```go
// LoggingMiddleware logs commands before and after execution
func LoggingMiddleware(logger *zap.Logger) CommandMiddleware {
    return func(next CommandHandlerFunc) CommandHandlerFunc {
        return func(ctx context.Context, command interface{}) error {
            start := time.Now()
            logger.Debug("Executing command", 
                zap.String("command_type", reflect.TypeOf(command).String()))
            
            err := next(ctx, command)
            
            logger.Debug("Command executed",
                zap.String("command_type", reflect.TypeOf(command).String()),
                zap.Duration("duration", time.Since(start)),
                zap.Error(err))
            
            return err
        }
    }
}
```

## 7. Benefits

- **Improved Separation of Concerns**: Clear distinction between read and write operations
- **Enhanced Scalability**: Independent scaling of read and write workloads
- **Better Performance**: Optimized query paths for frequently accessed data
- **Increased Maintainability**: Simpler, more focused components
- **Improved Testability**: Easier to test commands and queries in isolation
- **Enhanced Monitoring**: Better visibility into system operations through events

## 8. Future Improvements

- Implement event sourcing for complete audit trail
- Add distributed command and query handling
- Implement read models for complex query scenarios
- Add event versioning for backward compatibility
- Implement saga pattern for complex transactions
