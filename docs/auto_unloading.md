# Automatic Component Unloading Based on Memory Pressure

This document describes the automatic component unloading system implemented to manage memory pressure in the TradSys codebase.

## Overview

The automatic component unloading system proactively manages memory resources by unloading less critical components when memory pressure increases. This helps prevent out-of-memory conditions and ensures that critical components have the resources they need.

## Key Features

1. **Memory Pressure Detection**: The system continuously monitors memory usage and categorizes pressure into four levels:
   - **Low**: Normal operation, no action needed
   - **Medium**: Beginning to experience pressure, consider unloading idle components
   - **High**: Significant pressure, aggressively unload idle components
   - **Critical**: Severe pressure, unload all non-essential components

2. **Priority-Based Unloading**: Components are unloaded based on:
   - **Priority**: Lower priority components are unloaded first
   - **Idle Time**: Components that haven't been accessed recently are unloaded first
   - **Memory Usage**: Larger components may be prioritized for unloading

3. **Adaptive Behavior**: The system adapts its behavior based on memory pressure:
   - **Idle Time Threshold**: Decreases as pressure increases
   - **Number of Components**: More components are unloaded as pressure increases
   - **Unloading Aggressiveness**: Becomes more aggressive at higher pressure levels

4. **Component Protection**: Components can be protected from unloading:
   - **In-Use Protection**: Components currently in use are never unloaded
   - **Critical Components**: High-priority components are unloaded only under extreme pressure
   - **Recent Access**: Recently accessed components are less likely to be unloaded

## Implementation Details

### Memory Manager

The `MemoryManager` is responsible for tracking memory usage and managing component unloading:

```go
type MemoryManager struct {
    // Configuration
    config MemoryManagerConfig
    
    // Total memory usage
    totalUsage int64
    
    // Component memory usage
    componentUsage map[string]*ComponentMemoryInfo
    
    // Mutex for thread safety
    mu sync.RWMutex
    
    // Logger
    logger *zap.Logger
    
    // Unload callback
    unloadCallback func(ctx context.Context, componentName string) error
    
    // Stop channel for the background checker
    stopCh chan struct{}
}
```

### Memory Pressure Levels

Memory pressure is calculated as a percentage of the total memory limit:

```go
func (m *MemoryManager) GetMemoryPressureLevel() MemoryPressureLevel {
    usagePercentage := float64(m.totalUsage) / float64(m.config.TotalLimit)
    
    if usagePercentage >= m.config.CriticalThreshold {
        return MemoryPressureCritical
    } else if usagePercentage >= m.config.HighThreshold {
        return MemoryPressureHigh
    } else if usagePercentage >= m.config.MediumThreshold {
        return MemoryPressureMedium
    } else {
        return MemoryPressureLow
    }
}
```

### Automatic Unloading Process

The automatic unloading process runs in the background and periodically checks memory pressure:

```go
func (m *MemoryManager) runAutoUnloader(ctx context.Context) {
    ticker := time.NewTicker(time.Duration(m.config.CheckInterval) * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            m.checkAndUnloadComponents(ctx)
        case <-m.stopCh:
            return
        case <-ctx.Done():
            return
        }
    }
}
```

When memory pressure is detected, the system selects components for unloading:

```go
func (m *MemoryManager) checkAndUnloadComponents(ctx context.Context) {
    // Get the current memory pressure level
    pressureLevel := m.GetMemoryPressureLevel()
    
    // If memory pressure is low, do nothing
    if pressureLevel == MemoryPressureLow {
        return
    }
    
    // Get the minimum idle time based on pressure level
    minIdleTime := m.getMinIdleTimeForPressureLevel(pressureLevel)
    
    // Get the maximum number of components to unload based on pressure level
    maxUnload := m.getMaxUnloadForPressureLevel(pressureLevel)
    
    // Get the candidates for unloading
    candidates := m.getUnloadCandidates(minIdleTime, maxUnload)
    
    // Unload the candidates
    for _, candidate := range candidates {
        if m.unloadCallback != nil {
            err := m.unloadCallback(ctx, candidate.Name)
            if err != nil {
                m.logger.Error("Failed to unload component",
                    zap.String("component", candidate.Name),
                    zap.Error(err),
                )
                continue
            }
            
            m.logger.Info("Automatically unloaded component due to memory pressure",
                zap.String("component", candidate.Name),
                zap.String("type", candidate.Type),
                zap.Int64("memory_freed", candidate.MemoryUsage),
                zap.String("pressure_level", m.pressureLevelToString(pressureLevel)),
            )
        }
    }
}
```

### Component Selection for Unloading

Components are selected for unloading based on priority and idle time:

```go
func (m *MemoryManager) getUnloadCandidates(minIdleTime time.Duration, maxCount int) []*ComponentMemoryInfo {
    now := time.Now()
    candidates := make([]*ComponentMemoryInfo, 0)
    
    // Collect candidates
    for _, info := range m.componentUsage {
        // Skip components that are in use
        if info.InUse {
            continue
        }
        
        // Check if the component has been idle for long enough
        idleTime := now.Sub(info.LastAccess)
        if idleTime < minIdleTime {
            continue
        }
        
        candidates = append(candidates, info)
    }
    
    // Sort candidates by priority (higher priority last) and then by idle time (most idle first)
    sort.Slice(candidates, func(i, j int) bool {
        if candidates[i].Priority != candidates[j].Priority {
            return candidates[i].Priority > candidates[j].Priority
        }
        return candidates[i].LastAccess.Before(candidates[j].LastAccess)
    })
    
    // Limit the number of candidates
    if len(candidates) > maxCount {
        candidates = candidates[:maxCount]
    }
    
    return candidates
}
```

## Integration with Component Coordinator

The automatic unloading system is integrated with the `ComponentCoordinator`:

```go
func NewComponentCoordinator(config CoordinatorConfig, logger *zap.Logger) *ComponentCoordinator {
    // Create the memory manager
    memoryManager := NewMemoryManager(config.MemoryConfig, logger)
    
    coordinator := &ComponentCoordinator{
        components:     make(map[string]*ComponentInfo),
        memoryManager:  memoryManager,
        timeoutManager: timeoutManager,
        config:         config,
        logger:         logger,
    }
    
    // Set the unload callback
    memoryManager.SetUnloadCallback(coordinator.unloadComponent)
    
    // Start the automatic unloader if enabled
    if config.AutoUnloadEnabled {
        memoryManager.StartAutoUnloader(context.Background())
    }
    
    return coordinator
}
```

The `unloadComponent` method is used as a callback to unload components:

```go
func (c *ComponentCoordinator) unloadComponent(ctx context.Context, name string) error {
    return c.ShutdownComponent(ctx, name)
}
```

## Configuration

The automatic unloading system can be configured through the `MemoryManagerConfig`:

```go
type MemoryManagerConfig struct {
    // Total memory limit
    TotalLimit int64
    
    // Low memory threshold (percentage of total limit)
    LowThreshold float64
    
    // Medium memory threshold (percentage of total limit)
    MediumThreshold float64
    
    // High memory threshold (percentage of total limit)
    HighThreshold float64
    
    // Critical memory threshold (percentage of total limit)
    CriticalThreshold float64
    
    // Automatic unloading enabled
    AutoUnloadEnabled bool
    
    // Minimum idle time before a component can be unloaded (seconds)
    MinIdleTime int
    
    // Check interval for automatic unloading (seconds)
    CheckInterval int
}
```

Default configuration values are provided:

```go
func DefaultMemoryManagerConfig() MemoryManagerConfig {
    return MemoryManagerConfig{
        TotalLimit:        1024 * 1024 * 1024, // 1GB
        LowThreshold:      0.6,                // 60%
        MediumThreshold:   0.75,               // 75%
        HighThreshold:     0.85,               // 85%
        CriticalThreshold: 0.95,               // 95%
        AutoUnloadEnabled: true,
        MinIdleTime:       300,                // 5 minutes
        CheckInterval:     60,                 // 1 minute
    }
}
```

## Usage Example

Here's an example of how to configure and use the automatic unloading system:

```go
// Create a custom memory manager configuration
memoryConfig := coordination.MemoryManagerConfig{
    TotalLimit:        2 * 1024 * 1024 * 1024, // 2GB
    LowThreshold:      0.7,                    // 70%
    MediumThreshold:   0.8,                    // 80%
    HighThreshold:     0.9,                    // 90%
    CriticalThreshold: 0.95,                   // 95%
    AutoUnloadEnabled: true,
    MinIdleTime:       600,                    // 10 minutes
    CheckInterval:     120,                    // 2 minutes
}

// Create a coordinator configuration
coordinatorConfig := coordination.CoordinatorConfig{
    MemoryConfig:      memoryConfig,
    TimeoutConfig:     coordination.DefaultTimeoutManagerConfig(),
    AutoUnloadEnabled: true,
}

// Create the component coordinator
coordinator := coordination.NewComponentCoordinator(coordinatorConfig, logger)

// Register components
// ...

// The automatic unloading system will now monitor memory usage and unload components as needed
```

## Benefits

1. **Reduced Memory Usage**: The system automatically frees memory by unloading idle components.
2. **Improved Stability**: By preventing out-of-memory conditions, the system improves overall stability.
3. **Prioritized Resource Allocation**: Critical components are given priority for memory resources.
4. **Adaptive Behavior**: The system adapts its behavior based on the current memory pressure.
5. **Configurable**: The system can be configured to match specific requirements and resource constraints.

## Conclusion

The automatic component unloading system provides a proactive approach to memory management, ensuring that the system remains stable and responsive even under memory pressure. By unloading idle and less critical components, it frees up resources for more important operations, improving overall system performance and reliability.

