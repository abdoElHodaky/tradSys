package adaptive_loader

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// DeadlockPreventionConfig contains configuration for deadlock prevention
type DeadlockPreventionConfig struct {
	// Enabled enables deadlock prevention
	Enabled bool

	// DefaultTimeout is the default timeout for acquiring locks
	DefaultTimeout time.Duration

	// LockTimeouts contains timeouts for specific locks
	LockTimeouts map[string]time.Duration
}

// DefaultDeadlockPreventionConfig returns the default deadlock prevention configuration
func DefaultDeadlockPreventionConfig() DeadlockPreventionConfig {
	return DeadlockPreventionConfig{
		Enabled:       true,
		DefaultTimeout: 5 * time.Second,
		LockTimeouts:  make(map[string]time.Duration),
	}
}

// DeadlockPrevention provides deadlock prevention utilities
type DeadlockPrevention struct {
	// Configuration
	config DeadlockPreventionConfig

	// Logger
	logger *zap.Logger

	// Lock timeouts
	lockTimeouts map[string]time.Duration
}

// NewDeadlockPrevention creates a new deadlock prevention utility
func NewDeadlockPrevention(config DeadlockPreventionConfig, logger *zap.Logger) *DeadlockPrevention {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &DeadlockPrevention{
		config:      config,
		logger:      logger,
		lockTimeouts: config.LockTimeouts,
	}
}

// AcquireLock attempts to acquire a mutex with a timeout to prevent deadlocks
func (d *DeadlockPrevention) AcquireLock(mu *sync.Mutex, name string) bool {
	if !d.config.Enabled {
		mu.Lock()
		return true
	}

	// Get timeout for this lock
	timeout, exists := d.lockTimeouts[name]
	if !exists {
		timeout = d.config.DefaultTimeout
	}

	// Create a channel to signal when the lock is acquired
	done := make(chan struct{}, 1)

	// Try to acquire the lock in a goroutine
	go func() {
		mu.Lock()
		done <- struct{}{}
	}()

	// Wait for the lock with timeout
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		d.logger.Warn("Potential deadlock detected",
			zap.String("lock", name),
			zap.Duration("timeout", timeout))
		return false
	}
}

// AcquireLockRO attempts to acquire a read lock with a timeout to prevent deadlocks
func (d *DeadlockPrevention) AcquireLockRO(mu *sync.RWMutex, name string) bool {
	if !d.config.Enabled {
		mu.RLock()
		return true
	}

	// Get timeout for this lock
	timeout, exists := d.lockTimeouts[name]
	if !exists {
		timeout = d.config.DefaultTimeout
	}

	// Create a channel to signal when the lock is acquired
	done := make(chan struct{}, 1)

	// Try to acquire the lock in a goroutine
	go func() {
		mu.RLock()
		done <- struct{}{}
	}()

	// Wait for the lock with timeout
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		d.logger.Warn("Potential deadlock detected",
			zap.String("lock", name),
			zap.String("mode", "read"),
			zap.Duration("timeout", timeout))
		return false
	}
}

// AcquireLockWO attempts to acquire a write lock with a timeout to prevent deadlocks
func (d *DeadlockPrevention) AcquireLockWO(mu *sync.RWMutex, name string) bool {
	if !d.config.Enabled {
		mu.Lock()
		return true
	}

	// Get timeout for this lock
	timeout, exists := d.lockTimeouts[name]
	if !exists {
		timeout = d.config.DefaultTimeout
	}

	// Create a channel to signal when the lock is acquired
	done := make(chan struct{}, 1)

	// Try to acquire the lock in a goroutine
	go func() {
		mu.Lock()
		done <- struct{}{}
	}()

	// Wait for the lock with timeout
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		d.logger.Warn("Potential deadlock detected",
			zap.String("lock", name),
			zap.String("mode", "write"),
			zap.Duration("timeout", timeout))
		return false
	}
}

// SetLockTimeout sets the timeout for a specific lock
func (d *DeadlockPrevention) SetLockTimeout(lockName string, timeout time.Duration) {
	d.lockTimeouts[lockName] = timeout
}

// SetDefaultTimeout sets the default timeout for all locks
func (d *DeadlockPrevention) SetDefaultTimeout(timeout time.Duration) {
	d.config.DefaultTimeout = timeout
}

// Enable enables or disables deadlock prevention
func (d *DeadlockPrevention) Enable(enabled bool) {
	d.config.Enabled = enabled
}

// IsEnabled returns whether deadlock prevention is enabled
func (d *DeadlockPrevention) IsEnabled() bool {
	return d.config.Enabled
}

// GetLockTimeout gets the timeout for a specific lock
func (d *DeadlockPrevention) GetLockTimeout(lockName string) time.Duration {
	timeout, exists := d.lockTimeouts[lockName]
	if !exists {
		return d.config.DefaultTimeout
	}
	return timeout
}

// GetDefaultTimeout gets the default timeout for all locks
func (d *DeadlockPrevention) GetDefaultTimeout() time.Duration {
	return d.config.DefaultTimeout
}

// GetLockTimeouts gets all lock timeouts
func (d *DeadlockPrevention) GetLockTimeouts() map[string]time.Duration {
	// Create a copy to avoid race conditions
	timeoutsCopy := make(map[string]time.Duration)
	for name, timeout := range d.lockTimeouts {
		timeoutsCopy[name] = timeout
	}
	return timeoutsCopy
}

