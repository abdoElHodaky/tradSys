package coordination

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TimeoutManagerConfig contains configuration for the timeout manager
type TimeoutManagerConfig struct {
	// Default timeout
	DefaultTimeout time.Duration
}

// DefaultTimeoutManagerConfig returns the default timeout manager configuration
func DefaultTimeoutManagerConfig() TimeoutManagerConfig {
	return TimeoutManagerConfig{
		DefaultTimeout: 30 * time.Second,
	}
}

// TimeoutManager manages timeouts for component operations
type TimeoutManager struct {
	// Configuration
	config TimeoutManagerConfig
	
	// Component timeouts
	timeouts map[string]time.Duration
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(config TimeoutManagerConfig, logger *zap.Logger) *TimeoutManager {
	return &TimeoutManager{
		config:   config,
		timeouts: make(map[string]time.Duration),
		logger:   logger,
	}
}

// RegisterComponent registers a component with the timeout manager
func (t *TimeoutManager) RegisterComponent(name string, timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// If timeout is zero, use the default
	if timeout == 0 {
		timeout = t.config.DefaultTimeout
	}
	
	t.timeouts[name] = timeout
}

// GetTimeout gets the timeout for a component
func (t *TimeoutManager) GetTimeout(name string) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	timeout, exists := t.timeouts[name]
	if !exists {
		return t.config.DefaultTimeout
	}
	
	return timeout
}

// GetTimeoutContext creates a context with a timeout for a component
func (t *TimeoutManager) GetTimeoutContext(ctx context.Context, name string) (context.Context, context.CancelFunc) {
	timeout := t.GetTimeout(name)
	return context.WithTimeout(ctx, timeout)
}

// UpdateTimeout updates the timeout for a component
func (t *TimeoutManager) UpdateTimeout(name string, timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.timeouts[name] = timeout
}

// UnregisterComponent unregisters a component from the timeout manager
func (t *TimeoutManager) UnregisterComponent(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	delete(t.timeouts, name)
}

