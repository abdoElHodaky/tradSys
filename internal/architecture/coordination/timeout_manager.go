package coordination

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

//<<<<<<< codegen-bot/integrate-coordination-system
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
//=======
// TimeoutManager provides a unified approach to timeout management
// to resolve conflicts between different timeout mechanisms.
type TimeoutManager struct {
	// Default timeout
	defaultTimeout time.Duration
//>>>>>>> main
	
	// Component timeouts
	timeouts map[string]time.Duration
	
//<<<<<<< codegen-bot/integrate-coordination-system
	// Mutex for thread safety
//=======
	// Operation timeouts
	operationTimeouts map[string]time.Duration
	
	// Mutex for protecting timeouts
//>>>>>>> main
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
//<<<<<<< codegen-bot/integrate-coordination-system
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
//=======
	
	// Active timeouts
	activeTimeouts     map[string]context.CancelFunc
	activeTimeoutsMu   sync.Mutex
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(logger *zap.Logger) *TimeoutManager {
	return &TimeoutManager{
		defaultTimeout:    30 * time.Second,
		timeouts:          make(map[string]time.Duration),
		operationTimeouts: make(map[string]time.Duration),
		activeTimeouts:    make(map[string]context.CancelFunc),
		logger:            logger,
	}
}

// SetDefaultTimeout sets the default timeout
func (t *TimeoutManager) SetDefaultTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.defaultTimeout = timeout
}

// SetComponentTimeout sets the timeout for a component
func (t *TimeoutManager) SetComponentTimeout(component string, timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.timeouts[component] = timeout
}

// SetOperationTimeout sets the timeout for an operation
func (t *TimeoutManager) SetOperationTimeout(operation string, timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.operationTimeouts[operation] = timeout
}

// GetTimeout gets the timeout for a component and operation
func (t *TimeoutManager) GetTimeout(component string, operation string) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// Check for operation-specific timeout
	if operation != "" {
		if timeout, exists := t.operationTimeouts[operation]; exists {
			return timeout
		}
		
		// Check for component-operation timeout
		if timeout, exists := t.operationTimeouts[component+"."+operation]; exists {
			return timeout
		}
	}
	
	// Check for component-specific timeout
	if timeout, exists := t.timeouts[component]; exists {
		return timeout
	}
	
	// Use default timeout
	return t.defaultTimeout
}

// WithTimeout creates a context with a timeout for a component and operation
func (t *TimeoutManager) WithTimeout(ctx context.Context, component string, operation string) (context.Context, context.CancelFunc) {
	timeout := t.GetTimeout(component, operation)
	return context.WithTimeout(ctx, timeout)
}

// WithTimeoutID creates a context with a timeout and registers it with an ID
func (t *TimeoutManager) WithTimeoutID(ctx context.Context, id string, component string, operation string) (context.Context, context.CancelFunc) {
	timeout := t.GetTimeout(component, operation)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	
	// Register the timeout
	t.activeTimeoutsMu.Lock()
	t.activeTimeouts[id] = cancel
	t.activeTimeoutsMu.Unlock()
	
	// Create a wrapper cancel function that also removes the timeout
	wrappedCancel := func() {
		cancel()
		
		t.activeTimeoutsMu.Lock()
		delete(t.activeTimeouts, id)
		t.activeTimeoutsMu.Unlock()
	}
	
	return timeoutCtx, wrappedCancel
}

// CancelTimeout cancels a timeout by ID
func (t *TimeoutManager) CancelTimeout(id string) bool {
	t.activeTimeoutsMu.Lock()
	defer t.activeTimeoutsMu.Unlock()
	
	if cancel, exists := t.activeTimeouts[id]; exists {
		cancel()
		delete(t.activeTimeouts, id)
		return true
	}
	
	return false
}

// GetTimeoutStats gets timeout statistics
func (t *TimeoutManager) GetTimeoutStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	t.activeTimeoutsMu.Lock()
	defer t.activeTimeoutsMu.Unlock()
	
	stats := map[string]interface{}{
		"default_timeout":     t.defaultTimeout.String(),
		"component_timeouts":  make(map[string]string),
		"operation_timeouts":  make(map[string]string),
		"active_timeouts":     len(t.activeTimeouts),
	}
	
	componentTimeouts := stats["component_timeouts"].(map[string]string)
	for component, timeout := range t.timeouts {
		componentTimeouts[component] = timeout.String()
	}
	
	operationTimeouts := stats["operation_timeouts"].(map[string]string)
	for operation, timeout := range t.operationTimeouts {
		operationTimeouts[operation] = timeout.String()
	}
	
	return stats
//>>>>>>> main
}

