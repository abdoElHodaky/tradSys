package coordination

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
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

// RegisterComponent registers a component with the coordinator
func (c *ComponentCoordinator) RegisterComponent(
	name string,
	componentType string,
	provider *lazy.EnhancedLazyProvider,
	dependencies []string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if the component is already registered
	if _, exists := c.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}
	
	// Create the component info
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
	
	return nil
}

// unloadComponent unloads a component (used as a callback for the memory manager)
func (c *ComponentCoordinator) unloadComponent(ctx context.Context, name string) error {
	return c.ShutdownComponent(ctx, name)
}

// GetComponentInfo gets information about a component
func (c *ComponentCoordinator) GetComponentInfo(name string) (*ComponentInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
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
}

