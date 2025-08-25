package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"go.uber.org/zap"
)

// DummyComponent is a dummy component for testing memory pressure
type DummyComponent struct {
	name       string
	memorySize int64
	data       []byte
	logger     *zap.Logger
}

// NewDummyComponent creates a new dummy component
func NewDummyComponent(name string, memorySize int64, logger *zap.Logger) *DummyComponent {
	return &DummyComponent{
		name:       name,
		memorySize: memorySize,
		logger:     logger,
	}
}

// Initialize initializes the component
func (c *DummyComponent) Initialize() error {
	c.logger.Info("Initializing component", zap.String("name", c.name), zap.Int64("memory", c.memorySize))
	
	// Allocate memory
	c.data = make([]byte, c.memorySize)
	
	// Fill with some data to prevent optimization
	for i := range c.data {
		c.data[i] = byte(i % 256)
	}
	
	return nil
}

// Shutdown shuts down the component
func (c *DummyComponent) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down component", zap.String("name", c.name))
	
	// Release memory
	c.data = nil
	
	return nil
}

// DummyComponentProvider is a provider for dummy components
type DummyComponentProvider struct {
	name       string
	memorySize int64
	component  *DummyComponent
	logger     *zap.Logger
	mu         sync.Mutex
}

// NewDummyComponentProvider creates a new dummy component provider
func NewDummyComponentProvider(name string, memorySize int64, logger *zap.Logger) *DummyComponentProvider {
	return &DummyComponentProvider{
		name:       name,
		memorySize: memorySize,
		logger:     logger,
	}
}

// Get gets the component
func (p *DummyComponentProvider) Get() (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.component == nil {
		return nil, fmt.Errorf("component not initialized")
	}
	
	return p.component, nil
}

// Initialize initializes the component
func (p *DummyComponentProvider) Initialize(ctx context.Context) (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.component != nil {
		return p.component, nil
	}
	
	p.component = NewDummyComponent(p.name, p.memorySize, p.logger)
	err := p.component.Initialize()
	if err != nil {
		return nil, err
	}
	
	return p.component, nil
}

// Shutdown shuts down the component
func (p *DummyComponentProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.component == nil {
		return nil
	}
	
	err := p.component.Shutdown(ctx)
	if err != nil {
		return err
	}
	
	p.component = nil
	return nil
}

// GetMemoryEstimate gets the memory estimate for the component
func (p *DummyComponentProvider) GetMemoryEstimate() int64 {
	return p.memorySize
}

// GetTimeout gets the timeout for the component
func (p *DummyComponentProvider) GetTimeout() time.Duration {
	return 5 * time.Second
}

// GetPriority gets the priority for the component
func (p *DummyComponentProvider) GetPriority() int {
	// Lower number means higher priority
	if p.name == "high-priority" {
		return 10
	} else if p.name == "medium-priority" {
		return 50
	} else {
		return 90
	}
}

// IsInitialized checks if the component is initialized
func (p *DummyComponentProvider) IsInitialized() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	return p.component != nil
}

func main() {
	// Create a logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a custom memory manager configuration with a small limit to demonstrate memory pressure
	memoryConfig := coordination.MemoryManagerConfig{
		TotalLimit:        200 * 1024 * 1024, // 200MB
		LowThreshold:      0.6,               // 60%
		MediumThreshold:   0.75,              // 75%
		HighThreshold:     0.85,              // 85%
		CriticalThreshold: 0.95,              // 95%
		AutoUnloadEnabled: true,
		MinIdleTime:       5,                 // 5 seconds for demo purposes
		CheckInterval:     2,                 // 2 seconds for demo purposes
	}

	// Create a coordinator configuration
	coordinatorConfig := coordination.CoordinatorConfig{
		MemoryConfig:      memoryConfig,
		TimeoutConfig:     coordination.DefaultTimeoutManagerConfig(),
		AutoUnloadEnabled: true,
		MinIdleTime:       5,  // 5 seconds for demo purposes
		CheckInterval:     2,  // 2 seconds for demo purposes
	}

	// Create a component coordinator
	coordinator := coordination.NewComponentCoordinator(
		coordinatorConfig,
		logger,
	)

	// Create a context
	ctx := context.Background()

	// Register components with different priorities and memory sizes
	components := []struct {
		name       string
		memorySize int64
		priority   string
	}{
		{"component-1", 50 * 1024 * 1024, "high-priority"},   // 50MB, high priority
		{"component-2", 50 * 1024 * 1024, "medium-priority"}, // 50MB, medium priority
		{"component-3", 50 * 1024 * 1024, "low-priority"},    // 50MB, low priority
		{"component-4", 50 * 1024 * 1024, "low-priority"},    // 50MB, low priority
		{"component-5", 50 * 1024 * 1024, "low-priority"},    // 50MB, low priority
	}

	for _, c := range components {
		provider := NewDummyComponentProvider(c.priority, c.memorySize, logger)
		err := coordinator.RegisterComponent(
			c.name,
			"dummy",
			provider,
			[]string{},
		)
		if err != nil {
			logger.Error("Failed to register component", zap.String("name", c.name), zap.Error(err))
		}
	}

	// Initialize all components to create memory pressure
	fmt.Println("Initializing all components...")
	for _, c := range components {
		_, err := coordinator.GetComponent(ctx, c.name)
		if err != nil {
			logger.Error("Failed to initialize component", zap.String("name", c.name), zap.Error(err))
		}
	}

	// Print memory usage
	memoryManager := coordinator.GetMemoryManager()
	fmt.Printf("Total memory usage: %d bytes\n", memoryManager.GetMemoryUsage())
	fmt.Printf("Memory limit: %d bytes\n", memoryManager.GetMemoryLimit())
	fmt.Printf("Memory pressure level: %v\n", memoryManager.GetMemoryPressureLevel())

	// Wait for automatic unloading to occur
	fmt.Println("Waiting for automatic unloading...")
	time.Sleep(10 * time.Second)

	// Print memory usage again
	fmt.Printf("Total memory usage after waiting: %d bytes\n", memoryManager.GetMemoryUsage())
	fmt.Printf("Memory pressure level after waiting: %v\n", memoryManager.GetMemoryPressureLevel())

	// Try to initialize a new component
	fmt.Println("Initializing a new component...")
	provider := NewDummyComponentProvider("high-priority", 50*1024*1024, logger)
	err := coordinator.RegisterComponent(
		"new-component",
		"dummy",
		provider,
		[]string{},
	)
	if err != nil {
		logger.Error("Failed to register new component", zap.Error(err))
	}

	_, err = coordinator.GetComponent(ctx, "new-component")
	if err != nil {
		logger.Error("Failed to initialize new component", zap.Error(err))
	} else {
		fmt.Println("Successfully initialized new component")
	}

	// Print memory usage again
	fmt.Printf("Total memory usage after new component: %d bytes\n", memoryManager.GetMemoryUsage())
	fmt.Printf("Memory pressure level after new component: %v\n", memoryManager.GetMemoryPressureLevel())

	// Shutdown the coordinator
	coordinator.Shutdown(ctx)
}

