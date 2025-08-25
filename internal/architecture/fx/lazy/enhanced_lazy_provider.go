package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ProviderOption is an option for the EnhancedLazyProvider
type ProviderOption func(*EnhancedLazyProvider)

// WithMemoryEstimate sets the memory estimate for the provider
func WithMemoryEstimate(memoryEstimate int64) ProviderOption {
	return func(p *EnhancedLazyProvider) {
		p.memoryEstimate = memoryEstimate
	}
}

// WithTimeout sets the timeout for the provider
func WithTimeout(timeout time.Duration) ProviderOption {
	return func(p *EnhancedLazyProvider) {
		p.timeout = timeout
	}
}

// WithPriority sets the priority for the provider
func WithPriority(priority int) ProviderOption {
	return func(p *EnhancedLazyProvider) {
		p.priority = priority
	}
}

// EnhancedLazyProvider is an enhanced version of the LazyProvider with additional features
type EnhancedLazyProvider struct {
	// Component name
	name string
	
	// Provider function
	providerFn func(*zap.Logger) (interface{}, error)
	
	// Logger
	logger *zap.Logger
	
	// Metrics
	metrics interface{}
	
	// Memory estimate
	memoryEstimate int64
	
	// Timeout
	timeout time.Duration
	
	// Priority (lower is higher priority)
	priority int
	
	// Component instance
	instance interface{}
	
	// Initialization state
	initialized bool
	
	// Mutex for thread safety
	mu sync.RWMutex
}

// NewEnhancedLazyProvider creates a new enhanced lazy provider
func NewEnhancedLazyProvider(
	name string,
	providerFn func(*zap.Logger) (interface{}, error),
	logger *zap.Logger,
	metrics interface{},
	options ...ProviderOption,
) *EnhancedLazyProvider {
	provider := &EnhancedLazyProvider{
		name:          name,
		providerFn:    providerFn,
		logger:        logger,
		metrics:       metrics,
		memoryEstimate: 0,
		timeout:       30 * time.Second,
		priority:      50, // Default priority (middle)
		initialized:   false,
	}
	
	// Apply options
	for _, option := range options {
		option(provider)
	}
	
	return provider
}

// Initialize initializes the component
func (p *EnhancedLazyProvider) Initialize(ctx context.Context) (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check if already initialized
	if p.initialized {
		return p.instance, nil
	}
	
	// Initialize the component
	start := time.Now()
	instance, err := p.providerFn(p.logger)
	if err != nil {
		p.logger.Error("Failed to initialize component",
			zap.String("component", p.name),
			zap.Error(err),
		)
		return nil, err
	}
	
	// Log initialization time
	p.logger.Info("Component initialized",
		zap.String("component", p.name),
		zap.Duration("duration", time.Since(start)),
	)
	
	// Store the instance
	p.instance = instance
	p.initialized = true
	
	return instance, nil
}

// Get gets the component instance
func (p *EnhancedLazyProvider) Get() (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Check if initialized
	if !p.initialized {
		return nil, fmt.Errorf("component %s not initialized", p.name)
	}
	
	return p.instance, nil
}

// Shutdown shuts down the component
func (p *EnhancedLazyProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check if initialized
	if !p.initialized {
		return nil
	}
	
	// Check if the instance implements Shutdowner
	if shutdowner, ok := p.instance.(Shutdowner); ok {
		err := shutdowner.Shutdown(ctx)
		if err != nil {
			p.logger.Error("Failed to shutdown component",
				zap.String("component", p.name),
				zap.Error(err),
			)
			return err
		}
	}
	
	// Reset the instance
	p.instance = nil
	p.initialized = false
	
	p.logger.Info("Component shut down",
		zap.String("component", p.name),
	)
	
	return nil
}

// GetMemoryEstimate gets the memory estimate for the component
func (p *EnhancedLazyProvider) GetMemoryEstimate() int64 {
	return p.memoryEstimate
}

// GetTimeout gets the timeout for the component
func (p *EnhancedLazyProvider) GetTimeout() time.Duration {
	return p.timeout
}

// GetPriority gets the priority for the component
func (p *EnhancedLazyProvider) GetPriority() int {
	return p.priority
}

// IsInitialized checks if the component is initialized
func (p *EnhancedLazyProvider) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.initialized
}

// Shutdowner is an interface for components that can be shut down
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

