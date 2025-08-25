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

// ComponentInfo contains information about a component
type ComponentInfo struct {
	// Component name
	Name string

	// Component type
	Type string

	// Component provider
	Provider *lazy.EnhancedLazyProvider

	// Dependencies
	Dependencies []string

	// Whether the component is initialized
	Initialized bool

	// Memory usage
	MemoryUsage int64

	// Priority (lower is higher priority)
	Priority int

	// Last access time
	LastAccess time.Time
}

// ComponentCoordinator provides a unified coordination layer for component initialization,
// resource management, and lifecycle control to resolve conflicts between different
// mitigation systems.
type ComponentCoordinator struct {
	// Component registry
	components   map[string]*ComponentInfo
	componentsMu sync.RWMutex

	// Resource management
	memoryManager *MemoryManager

	// Initialization coordination
	initManager *lazy.InitializationManager

	// Timeout management
	timeoutManager *TimeoutManager

	// Metrics collection
	metricsCollector *metrics.Collector

	// Logging
	logger *zap.Logger

	// Configuration
	config CoordinatorConfig
}

// ComponentInfo contains information about a registered component
type ComponentInfo struct {
	// Component identity
	Name string
	Type string

	// Component state
	Provider      *lazy.EnhancedLazyProvider
	IsInitialized bool
	LastAccess    time.Time

	// Resource usage
	MemoryUsage int64
	CPUUsage    float64

	// Dependencies
	Dependencies []string

	// Configuration
	Priority int
	Timeout  time.Duration

	// Metrics
	InitTime    time.Duration
	AccessCount int64
}

// CoordinatorConfig contains configuration for the component coordinator
type CoordinatorConfig struct {
	// Memory manager configuration
	MemoryConfig MemoryManagerConfig

	// Timeout manager configuration
	TimeoutConfig TimeoutManagerConfig

	// Automatic unloading enabled
	AutoUnloadEnabled bool

	// Minimum idle time before a component can be unloaded (seconds)
	MinIdleTime int

	// Check interval for automatic unloading (seconds)
	CheckInterval int
}

// DefaultCoordinatorConfig returns the default coordinator configuration
func DefaultCoordinatorConfig() CoordinatorConfig {
	return CoordinatorConfig{
		MemoryConfig:      DefaultMemoryManagerConfig(),
		TimeoutConfig:     DefaultTimeoutManagerConfig(),
		AutoUnloadEnabled: true,
		MinIdleTime:       300, // 5 minutes
		CheckInterval:     60,  // 1 minute
	}
}

// ComponentCoordinator coordinates component initialization and resource management
type ComponentCoordinator struct {
	// Component registry
	components map[string]*ComponentInfo

	// Resource management
	memoryManager *MemoryManager

	// Timeout management
	timeoutManager *TimeoutManager

	// Configuration
	config CoordinatorConfig

	// Mutex for thread safety
	mu sync.RWMutex

	// Logger
	logger *zap.Logger
}

// NewComponentCoordinator creates a new component coordinator
func NewComponentCoordinator(config CoordinatorConfig, logger *zap.Logger) *ComponentCoordinator {
	// Create the memory manager
	memoryManager := NewMemoryManager(config.MemoryConfig, logger)

	// Create the timeout manager
	timeoutManager := NewTimeoutManager(config.TimeoutConfig, logger)

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

	// Check if the component is already registered

	if _, exists := c.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}

	// Create component info
	info := &ComponentInfo{
		Name:         name,
		Type:         componentType,
		Provider:     provider,
		Dependencies: dependencies,
		Initialized:  false,
		MemoryUsage:  provider.GetMemoryEstimate(),
		Priority:     provider.GetPriority(),
		LastAccess:   time.Now(),
	}

	// Register the component
	c.components[name] = info

	// Register with the memory manager
	c.memoryManager.RegisterComponent(name, componentType, info.MemoryUsage, info.Priority)

	// Register with the timeout manager
	c.timeoutManager.RegisterComponent(name, provider.GetTimeout())
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
	// Get the component info
	info, err := c.GetComponentInfo(name)
	if err != nil {
		return nil, err
	}

	// Check if the component is already initialized
	if info.Initialized {
		// Mark the component as accessed
		c.memoryManager.MarkComponentAccessed(name)
		c.memoryManager.MarkComponentInUse(name, true)

		// Get the component
		component, err := info.Provider.Get()
		if err != nil {
			return nil, err
		}

		// Mark the component as not in use
		c.memoryManager.MarkComponentInUse(name, false)

		return component, nil
	}

	// Initialize the component
	return c.initializeComponent(ctx, info)
}

// initializeComponent initializes a component
func (c *ComponentCoordinator) initializeComponent(ctx context.Context, info *ComponentInfo) (interface{}, error) {
	// Create a timeout context
	timeoutCtx, cancel := c.timeoutManager.GetTimeoutContext(ctx, info.Name)
	defer cancel()

	// Check if memory can be allocated
	if !c.memoryManager.CanAllocate(info.Name, info.MemoryUsage) {
		// Try to free memory
		freed, err := c.memoryManager.FreeMemory(ctx, info.MemoryUsage)
		if err != nil || !freed {
			return nil, fmt.Errorf("insufficient memory to initialize component %s", info.Name)
		}
	}

	// Allocate memory
	err := c.memoryManager.AllocateMemory(info.Name, info.MemoryUsage)
	if err != nil {
		return nil, err
	}

	// Mark the component as in use
	c.memoryManager.MarkComponentInUse(info.Name, true)

	// Initialize the component
	component, err := info.Provider.Initialize(timeoutCtx)
	if err != nil {
		// Mark the component as not in use
		c.memoryManager.MarkComponentInUse(info.Name, false)
		return nil, err
	}

	// Mark the component as initialized
	c.mu.Lock()
	info.Initialized = true
	c.mu.Unlock()

	// Mark the component as not in use
	c.memoryManager.MarkComponentInUse(info.Name, false)

	return component, nil
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
	// Get the component info
	info, err := c.GetComponentInfo(name)
	if err != nil {
		return err
	}

	// Check if the component is initialized
	if !info.Initialized {
		return nil
	}

	// Create a timeout context
	timeoutCtx, cancel := c.timeoutManager.GetTimeoutContext(ctx, info.Name)
	defer cancel()

	// Shutdown the component
	err = info.Provider.Shutdown(timeoutCtx)
	if err != nil {
		return err
	}

	// Mark the component as not initialized
	c.mu.Lock()
	info.Initialized = false
	c.mu.Unlock()

	// Unregister from the memory manager
	c.memoryManager.UnregisterComponent(name)

	// Re-register with the memory manager with zero usage
	c.memoryManager.RegisterComponent(name, info.Type, 0, info.Priority)
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

// unloadComponent unloads a component (used as a callback for the memory manager)
func (c *ComponentCoordinator) unloadComponent(ctx context.Context, name string) error {
	return c.ShutdownComponent(ctx, name)
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
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*ComponentInfo, 0, len(c.components))
	for _, info := range c.components {
		result = append(result, info)
	}

	return result
}

// GetMemoryManager gets the memory manager
func (c *ComponentCoordinator) GetMemoryManager() *MemoryManager {
	return c.memoryManager
}

// GetTimeoutManager gets the timeout manager
func (c *ComponentCoordinator) GetTimeoutManager() *TimeoutManager {
	return c.timeoutManager
}

// Shutdown shuts down the coordinator and all components
func (c *ComponentCoordinator) Shutdown(ctx context.Context) error {
	// Stop the automatic unloader
	c.memoryManager.StopAutoUnloader()

	// Get all components
	components := c.ListComponents()

	// Shutdown all components
	var lastErr error
	for _, info := range components {
		err := c.ShutdownComponent(ctx, info.Name)
		if err != nil {
			lastErr = err
			c.logger.Error("Failed to shutdown component",
				zap.String("component", info.Name),
				zap.Error(err),
			)
		}
	}

	return lastErr
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
