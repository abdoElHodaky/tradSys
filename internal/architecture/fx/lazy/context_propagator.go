package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ContextPropagator ensures that context (timeouts, cancellation) is properly
// propagated through lazy-loaded components.
type ContextPropagator struct {
	logger *zap.Logger
	
	// Active contexts
	contexts     map[string]context.Context
	cancels      map[string]context.CancelFunc
	contextsMu   sync.RWMutex
	
	// Default timeout
	defaultTimeout time.Duration
}

// NewContextPropagator creates a new context propagator
func NewContextPropagator(logger *zap.Logger) *ContextPropagator {
	return &ContextPropagator{
		logger:         logger,
		contexts:       make(map[string]context.Context),
		cancels:        make(map[string]context.CancelFunc),
		defaultTimeout: 30 * time.Second,
	}
}

// CreateContext creates a new context with the given ID
func (p *ContextPropagator) CreateContext(id string) context.Context {
	return p.CreateContextWithTimeout(id, p.defaultTimeout)
}

// CreateContextWithTimeout creates a new context with the given ID and timeout
func (p *ContextPropagator) CreateContextWithTimeout(id string, timeout time.Duration) context.Context {
	p.contextsMu.Lock()
	defer p.contextsMu.Unlock()
	
	// Cancel existing context if any
	if cancel, ok := p.cancels[id]; ok {
		cancel()
	}
	
	// Create new context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	
	// Store context and cancel function
	p.contexts[id] = ctx
	p.cancels[id] = cancel
	
	return ctx
}

// CreateContextWithParent creates a new context with the given ID and parent context
func (p *ContextPropagator) CreateContextWithParent(id string, parent context.Context) context.Context {
	p.contextsMu.Lock()
	defer p.contextsMu.Unlock()
	
	// Cancel existing context if any
	if cancel, ok := p.cancels[id]; ok {
		cancel()
	}
	
	// Create new context
	ctx, cancel := context.WithCancel(parent)
	
	// Store context and cancel function
	p.contexts[id] = ctx
	p.cancels[id] = cancel
	
	return ctx
}

// GetContext gets the context with the given ID
func (p *ContextPropagator) GetContext(id string) (context.Context, error) {
	p.contextsMu.RLock()
	defer p.contextsMu.RUnlock()
	
	ctx, ok := p.contexts[id]
	if !ok {
		return nil, fmt.Errorf("context not found: %s", id)
	}
	
	return ctx, nil
}

// CancelContext cancels the context with the given ID
func (p *ContextPropagator) CancelContext(id string) error {
	p.contextsMu.Lock()
	defer p.contextsMu.Unlock()
	
	cancel, ok := p.cancels[id]
	if !ok {
		return fmt.Errorf("context not found: %s", id)
	}
	
	// Cancel context
	cancel()
	
	// Remove from maps
	delete(p.contexts, id)
	delete(p.cancels, id)
	
	return nil
}

// WithValue creates a new context with the given key-value pair
func (p *ContextPropagator) WithValue(id string, key, value interface{}) (context.Context, error) {
	p.contextsMu.Lock()
	defer p.contextsMu.Unlock()
	
	ctx, ok := p.contexts[id]
	if !ok {
		return nil, fmt.Errorf("context not found: %s", id)
	}
	
	// Create new context with value
	newCtx := context.WithValue(ctx, key, value)
	
	// Update context
	p.contexts[id] = newCtx
	
	return newCtx, nil
}

// SetDefaultTimeout sets the default timeout for new contexts
func (p *ContextPropagator) SetDefaultTimeout(timeout time.Duration) {
	p.contextsMu.Lock()
	defer p.contextsMu.Unlock()
	
	p.defaultTimeout = timeout
}

// GetDefaultTimeout gets the default timeout for new contexts
func (p *ContextPropagator) GetDefaultTimeout() time.Duration {
	p.contextsMu.RLock()
	defer p.contextsMu.RUnlock()
	
	return p.defaultTimeout
}

// Close cancels all active contexts
func (p *ContextPropagator) Close() {
	p.contextsMu.Lock()
	defer p.contextsMu.Unlock()
	
	// Cancel all contexts
	for id, cancel := range p.cancels {
		cancel()
		delete(p.contexts, id)
		delete(p.cancels, id)
	}
}

// ContextAwareProvider is an interface for providers that are context-aware
type ContextAwareProvider interface {
	// GetWithContext returns the instance with context
	GetWithContext(ctx context.Context) (interface{}, error)
}

// ContextMiddleware is middleware for propagating context
type ContextMiddleware struct {
	propagator *ContextPropagator
	logger     *zap.Logger
}

// NewContextMiddleware creates a new context middleware
func NewContextMiddleware(propagator *ContextPropagator, logger *zap.Logger) *ContextMiddleware {
	return &ContextMiddleware{
		propagator: propagator,
		logger:     logger,
	}
}

// WithContext wraps a function to propagate context
func (m *ContextMiddleware) WithContext(id string, fn func(context.Context) error) error {
	// Get context
	ctx, err := m.propagator.GetContext(id)
	if err != nil {
		// Create a new context if not found
		ctx = m.propagator.CreateContext(id)
	}
	
	// Call function with context
	return fn(ctx)
}

// WithContextAndTimeout wraps a function to propagate context with timeout
func (m *ContextMiddleware) WithContextAndTimeout(id string, timeout time.Duration, fn func(context.Context) error) error {
	// Create context with timeout
	ctx := m.propagator.CreateContextWithTimeout(id, timeout)
	
	// Call function with context
	return fn(ctx)
}

// WithContextAndDeadline wraps a function to propagate context with deadline
func (m *ContextMiddleware) WithContextAndDeadline(id string, deadline time.Time, fn func(context.Context) error) error {
	// Get context
	parentCtx, err := m.propagator.GetContext(id)
	if err != nil {
		// Create a new context if not found
		parentCtx = context.Background()
	}
	
	// Create context with deadline
	ctx, cancel := context.WithDeadline(parentCtx, deadline)
	defer cancel()
	
	// Call function with context
	return fn(ctx)
}

