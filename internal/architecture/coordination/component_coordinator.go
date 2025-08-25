package coordination

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"go.uber.org/zap"
)

// ComponentCoordinator provides a unified coordination layer for component initialization,
// resource management, and lifecycle control to resolve conflicts between different
// mitigation systems.
type ComponentCoordinator struct {
	// Component registry
	components     map[string]*ComponentInfo
	componentsMu   sync.RWMutex
	
	// Resource management
	memoryManager  *MemoryManager
	
	// Initialization coordination
	initManager    *lazy.InitializationManager
	
	// Timeout management
	timeoutManager *TimeoutManager
	
	// Metrics collection
	metricsCollector *metrics.Collector
	
	// Logging
	logger         *zap.Logger
	
	// Configuration
	config         CoordinatorConfig
}

// ComponentInfo contains information about a registered component
type ComponentInfo struct {
	// Component identity
	Name           string
	Type           string
	
	// Component state
	Provider       *lazy.EnhancedLazyProvider
	IsInitialized  bool
	LastAccess     time.Time
	
	// Resource usage
	MemoryUsage    int64
	CPUUsage       float64
	
	// Dependencies
	Dependencies   []string
	
	// Configuration
	Priority       int
	Timeout        time.Duration
	
	// Metrics
	InitTime       time.Duration
	AccessCount    int64
}

// CoordinatorConfig contains configuration for the component coordinator
type CoordinatorConfig struct {
	// Memory management
	TotalMemoryLimit    int64
	ComponentMemoryLimit int64
	
	// Initialization
	DefaultTimeout      time.Duration
	DefaultPriority     int
	
	// Resource management
	EnableMemoryTracking bool
	EnableCPUTracking    bool
	
	// Metrics
	MetricsEnabled      bool
	MetricsSampleRate   float64
}

// DefaultCoordinatorConfig returns the default configuration
func DefaultCoordinatorConfig() CoordinatorConfig {
	return CoordinatorConfig{
		TotalMemoryLimit:     1024 * 1024 * 1024 * 4, // 4GB
		ComponentMemoryLimit: 1024 * 1024 * 512,      // 512MB
		DefaultTimeout:       30 * time.Second,
		DefaultPriority:      50,
		EnableMemoryTracking: true,
		EnableCPUTracking:    true,
		MetricsEnabled:       true,
		MetricsSampleRate:    0.1, // 10% sampling
	}
}

// NewComponentCoordinator creates a new component coordinator
func NewComponentCoordinator(config CoordinatorConfig, logger *zap.Logger) *ComponentCoordinator {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	
	metricsCollector := metrics.NewCollector("component_coordinator", metrics.WithSampleRate(config.MetricsSampleRate))
	
	return &ComponentCoordinator{
		components:       make(map[string]*ComponentInfo),
		memoryManager:    NewMemoryManager(config.TotalMemoryLimit, logger),
		initManager:      lazy.NewInitializationManager(logger),
		timeoutManager:   NewTimeoutManager(logger),
		metricsCollector: metricsCollector,
		logger:           logger,
		config:           config,
	}
}

// RegisterComponent registers a component with the coordinator
func (c *ComponentCoordinator) RegisterComponent(
	name string,
	componentType string,
	provider *lazy.EnhancedLazyProvider,
	dependencies []string,
) error {
	c.componentsMu.Lock()
	defer c.componentsMu.Unlock()
	
	if _, exists := c.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}
	
	// Create component info
	info := &ComponentInfo{
		Name:         name,
		Type:         componentType,
		Provider:     provider,
		IsInitialized: false,
		LastAccess:   time.Now(),
		MemoryUsage:  provider.GetMemoryEstimate(),
		Dependencies: dependencies,
		Priority:     provider.GetPriority(),
		Timeout:      c.config.DefaultTimeout,
		AccessCount:  0,
	}
	
	// Register with memory manager
	if c.config.EnableMemoryTracking {
		c.memoryManager.RegisterComponent(name, provider.GetMemoryEstimate())
	}
	
	// Register with initialization manager
	c.initManager.RegisterComponent(name, provider, dependencies)
	
	// Store component info
	c.components[name] = info
	
	c.logger.Info("Component registered", 
		zap.String("component", name),
		zap.String("type", componentType),
		zap.Int("priority", provider.GetPriority()),
		zap.Int64("memory_estimate", provider.GetMemoryEstimate()),
	)
	
	return nil
}

// GetComponent gets a component, initializing it if necessary
func (c *ComponentCoordinator) GetComponent(ctx context.Context, name string) (interface{}, error) {
	// Check if component exists
	c.componentsMu.RLock()
	info, exists := c.components[name]
	c.componentsMu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("component %s not registered", name)
	}
	
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, info.Timeout)
	defer cancel()
	
	// Check if memory is available
	if c.config.EnableMemoryTracking {
		if !c.memoryManager.CanAllocate(name, info.MemoryUsage) {
			// Try to free memory
			freed, err := c.memoryManager.FreeMemory(info.MemoryUsage)
			if err != nil || !freed {
				return nil, fmt.Errorf("insufficient memory to initialize component %s: %w", name, err)
			}
		}
	}
	
	// Get the component through the initialization manager
	startTime := time.Now()
	instance, err := c.initManager.GetComponent(timeoutCtx, name)
	initTime := time.Since(startTime)
	
	if err != nil {
		c.logger.Error("Failed to initialize component",
			zap.String("component", name),
			zap.Error(err),
			zap.Duration("init_time", initTime),
		)
		
		if c.config.MetricsEnabled {
			c.metricsCollector.RecordError("component_init_error", map[string]string{
				"component": name,
				"error":     err.Error(),
			})
		}
		
		return nil, err
	}
	
	// Update component info
	c.componentsMu.Lock()
	info.IsInitialized = true
	info.LastAccess = time.Now()
	info.InitTime = initTime
	info.AccessCount++
	c.componentsMu.Unlock()
	
	// Record metrics
	if c.config.MetricsEnabled {
		c.metricsCollector.RecordLatency("component_init_time", initTime, map[string]string{
			"component": name,
		})
		c.metricsCollector.Increment("component_access_count", map[string]string{
			"component": name,
		})
	}
	
	c.logger.Debug("Component initialized",
		zap.String("component", name),
		zap.Duration("init_time", initTime),
	)
	
	return instance, nil
}

// InitializeComponents initializes components in dependency order
func (c *ComponentCoordinator) InitializeComponents(ctx context.Context, componentNames []string) error {
	return c.initManager.InitializeComponents(ctx, componentNames)
}

// ShutdownComponent shuts down a component
func (c *ComponentCoordinator) ShutdownComponent(ctx context.Context, name string) error {
	c.componentsMu.Lock()
	defer c.componentsMu.Unlock()
	
	info, exists := c.components[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
	if !info.IsInitialized {
		return nil // Already shut down
	}
	
	// Reset the provider
	info.Provider.Reset()
	
	// Update component info
	info.IsInitialized = false
	
	// Release memory
	if c.config.EnableMemoryTracking {
		c.memoryManager.ReleaseMemory(name)
	}
	
	c.logger.Info("Component shut down", zap.String("component", name))
	
	return nil
}

// GetComponentInfo gets information about a component
func (c *ComponentCoordinator) GetComponentInfo(name string) (*ComponentInfo, error) {
	c.componentsMu.RLock()
	defer c.componentsMu.RUnlock()
	
	info, exists := c.components[name]
	if !exists {
		return nil, fmt.Errorf("component %s not registered", name)
	}
	
	return info, nil
}

// ListComponents lists all registered components
func (c *ComponentCoordinator) ListComponents() []*ComponentInfo {
	c.componentsMu.RLock()
	defer c.componentsMu.RUnlock()
	
	components := make([]*ComponentInfo, 0, len(c.components))
	for _, info := range c.components {
		components = append(components, info)
	}
	
	return components
}

// GetMemoryUsage gets the current memory usage
func (c *ComponentCoordinator) GetMemoryUsage() int64 {
	if c.config.EnableMemoryTracking {
		return c.memoryManager.GetTotalUsage()
	}
	return 0
}

// GetMemoryLimit gets the memory limit
func (c *ComponentCoordinator) GetMemoryLimit() int64 {
	if c.config.EnableMemoryTracking {
		return c.memoryManager.GetTotalLimit()
	}
	return 0
}

