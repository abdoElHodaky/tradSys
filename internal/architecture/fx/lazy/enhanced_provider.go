package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EnhancedLazyProvider provides enhanced lazy loading with additional features
// to address conflicts and bottlenecks identified in the analysis.
type EnhancedLazyProvider struct {
	name           string
	initFunc       func(*zap.Logger) (interface{}, error)
	logger         *zap.Logger
	metrics        *LazyLoadingMetrics
	instance       interface{}
	initialized    bool
	initError      error
	mu             sync.RWMutex
	initInProgress bool
	waitCh         chan struct{}
	priority       int
	timeout        time.Duration
	memoryEstimate int64
}

// NewEnhancedLazyProvider creates a new enhanced lazy provider
func NewEnhancedLazyProvider(
	name string,
	initFunc func(*zap.Logger) (interface{}, error),
	logger *zap.Logger,
	metrics *LazyLoadingMetrics,
	options ...EnhancedProviderOption,
) *EnhancedLazyProvider {
	provider := &EnhancedLazyProvider{
		name:           name,
		initFunc:       initFunc,
		logger:         logger,
		metrics:        metrics,
		waitCh:         make(chan struct{}),
		priority:       50, // Default priority (0-100, lower is higher priority)
		timeout:        30 * time.Second,
		memoryEstimate: 0, // Unknown by default
	}

	// Apply options
	for _, option := range options {
		option(provider)
	}

	return provider
}

// EnhancedProviderOption configures an enhanced lazy provider
type EnhancedProviderOption func(*EnhancedLazyProvider)

// WithPriority sets the initialization priority
func WithPriority(priority int) EnhancedProviderOption {
	return func(p *EnhancedLazyProvider) {
		p.priority = priority
	}
}

// WithTimeout sets the initialization timeout
func WithTimeout(timeout time.Duration) EnhancedProviderOption {
	return func(p *EnhancedLazyProvider) {
		p.timeout = timeout
	}
}

// WithMemoryEstimate sets the estimated memory usage
func WithMemoryEstimate(bytes int64) EnhancedProviderOption {
	return func(p *EnhancedLazyProvider) {
		p.memoryEstimate = bytes
	}
}

// Get returns the instance, initializing it if necessary
func (p *EnhancedLazyProvider) Get() (interface{}, error) {
	// Fast path: check if already initialized
	p.mu.RLock()
	if p.initialized {
		instance := p.instance
		p.mu.RUnlock()
		return instance, nil
	}
	if p.initError != nil {
		err := p.initError
		p.mu.RUnlock()
		return nil, err
	}
	p.mu.RUnlock()

	// Slow path: initialize
	return p.initialize()
}

// GetWithContext returns the instance with context timeout
func (p *EnhancedLazyProvider) GetWithContext(ctx context.Context) (interface{}, error) {
	// Fast path: check if already initialized
	p.mu.RLock()
	if p.initialized {
		instance := p.instance
		p.mu.RUnlock()
		return instance, nil
	}
	if p.initError != nil {
		err := p.initError
		p.mu.RUnlock()
		return nil, err
	}
	p.mu.RUnlock()

	// Create a channel for the result
	resultCh := make(chan struct {
		instance interface{}
		err      error
	})

	// Initialize in a goroutine
	go func() {
		instance, err := p.initialize()
		resultCh <- struct {
			instance interface{}
			err      error
		}{instance, err}
	}()

	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.instance, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("initialization canceled: %w", ctx.Err())
	}
}

// initialize initializes the instance
func (p *EnhancedLazyProvider) initialize() (interface{}, error) {
	p.mu.Lock()

	// Check if another goroutine is already initializing
	if p.initInProgress {
		// Wait for initialization to complete
		waitCh := p.waitCh
		p.mu.Unlock()
		<-waitCh

		// Check the result
		p.mu.RLock()
		defer p.mu.RUnlock()
		if p.initialized {
			return p.instance, nil
		}
		return nil, p.initError
	}

	// Start initialization
	p.initInProgress = true
	p.mu.Unlock()

	// Create a new wait channel for this initialization
	waitCh := make(chan struct{})
	p.mu.Lock()
	p.waitCh = waitCh
	p.mu.Unlock()

	// Initialize with timeout
	initCtx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	// Create a channel for the initialization result
	resultCh := make(chan struct {
		instance interface{}
		err      error
	})

	// Initialize in a goroutine
	go func() {
		startTime := time.Now()
		p.logger.Debug("Initializing component", zap.String("component", p.name))

		instance, err := p.initFunc(p.logger)

		resultCh <- struct {
			instance interface{}
			err      error
		}{instance, err}

		duration := time.Since(startTime)
		if err != nil {
			p.logger.Error("Failed to initialize component",
				zap.String("component", p.name),
				zap.Error(err),
				zap.Duration("duration", duration))
			p.metrics.RecordInitializationError(p.name, duration)
		} else {
			p.logger.Info("Component initialized",
				zap.String("component", p.name),
				zap.Duration("duration", duration))
			p.metrics.RecordInitialization(p.name, duration)
		}
	}()

	// Wait for initialization or timeout
	var instance interface{}
	var err error
	select {
	case result := <-resultCh:
		instance = result.instance
		err = result.err
	case <-initCtx.Done():
		err = fmt.Errorf("initialization timed out after %v", p.timeout)
		p.logger.Error("Component initialization timed out",
			zap.String("component", p.name),
			zap.Duration("timeout", p.timeout))
	}

	// Update state
	p.mu.Lock()
	p.initialized = err == nil
	p.initError = err
	p.instance = instance
	p.initInProgress = false
	close(waitCh)
	p.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return instance, nil
}

// IsInitialized returns whether the instance has been initialized
func (p *EnhancedLazyProvider) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initialized
}

// GetPriority returns the initialization priority
func (p *EnhancedLazyProvider) GetPriority() int {
	return p.priority
}

// GetMemoryEstimate returns the estimated memory usage
func (p *EnhancedLazyProvider) GetMemoryEstimate() int64 {
	return p.memoryEstimate
}

// Reset resets the provider, forcing reinitialization on next Get
func (p *EnhancedLazyProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Only reset if not in progress
	if !p.initInProgress {
		p.initialized = false
		p.initError = nil
		p.instance = nil
		p.waitCh = make(chan struct{})
	}
}

// GetName returns the name of the provider
func (p *EnhancedLazyProvider) GetName() string {
	return p.name
}

