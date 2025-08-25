package coordination

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LockInfo contains information about a lock
type LockInfo struct {
	// Lock identity
	Name string
	
	// Lock object
	Lock sync.Locker
	
	// Lock statistics
	AcquisitionCount int64
	ContentionCount  int64
	TotalHeldTime    time.Duration
	LastAcquired     time.Time
	LastReleased     time.Time
	
	// Current state
	IsHeld        bool
	CurrentHolder string
	AcquiredAt    time.Time
}

// LockManagerConfig contains configuration for the lock manager
type LockManagerConfig struct {
	// DeadlockDetectionEnabled enables deadlock detection
	DeadlockDetectionEnabled bool
	
	// LockTimeout is the default timeout for acquiring locks
	LockTimeout time.Duration
}

// DefaultLockManagerConfig returns the default lock manager configuration
func DefaultLockManagerConfig() LockManagerConfig {
	return LockManagerConfig{
		DeadlockDetectionEnabled: true,
		LockTimeout:              5 * time.Second,
	}
}

// LockManager manages locks and prevents deadlocks
type LockManager struct {
	// Configuration
	config LockManagerConfig
	
	// Locks
	locks map[string]*LockInfo
	mu    sync.Mutex
	
	// Logger
	logger *zap.Logger
	
	// Deadlock detection
	lockOrder   []string
	lockHolders map[string]string    // lock -> holder
	holderLocks map[string][]string  // holder -> locks
	
	// Lock timeouts
	lockTimeouts map[string]time.Duration
}

// NewLockManager creates a new lock manager
func NewLockManager(config LockManagerConfig, logger *zap.Logger) *LockManager {
	return &LockManager{
		config:       config,
		locks:        make(map[string]*LockInfo),
		lockOrder:    make([]string, 0),
		lockHolders:  make(map[string]string),
		holderLocks:  make(map[string][]string),
		lockTimeouts: make(map[string]time.Duration),
		logger:       logger,
	}
}

// RegisterLock registers a lock with the manager
func (l *LockManager) RegisterLock(name string, lock sync.Locker) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if the lock is already registered
	if _, exists := l.locks[name]; exists {
		l.logger.Warn("Lock already registered", zap.String("lock", name))
		return
	}
	
	// Register the lock
	l.locks[name] = &LockInfo{
		Name:             name,
		Lock:             lock,
		AcquisitionCount: 0,
		ContentionCount:  0,
		TotalHeldTime:    0,
		IsHeld:           false,
	}
	
	// Add to the lock order
	l.lockOrder = append(l.lockOrder, name)
	
	// Sort lock order by name to ensure consistent ordering
	sort.Strings(l.lockOrder)
}

// SetLockTimeout sets the timeout for a specific lock
func (l *LockManager) SetLockTimeout(name string, timeout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.lockTimeouts[name] = timeout
}

// SetDefaultTimeout sets the default timeout for all locks
func (l *LockManager) SetDefaultTimeout(timeout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.config.LockTimeout = timeout
}

// EnableDeadlockDetection enables or disables deadlock detection
func (l *LockManager) EnableDeadlockDetection(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.config.DeadlockDetectionEnabled = enabled
}

// AcquireLock acquires a lock with the given name
func (l *LockManager) AcquireLock(lockName, holderName string) error {
	l.mu.Lock()
	
	// Check if the lock exists
	lockInfo, exists := l.locks[lockName]
	if !exists {
		l.mu.Unlock()
		return fmt.Errorf("lock %s not registered", lockName)
	}
	
	// Check for deadlocks if enabled
	if l.config.DeadlockDetectionEnabled {
		// Check if acquiring this lock would create a deadlock
		if err := l.checkForDeadlock(lockName, holderName); err != nil {
			l.mu.Unlock()
			return err
		}
	}
	
	// Get timeout for this lock
	timeout, exists := l.lockTimeouts[lockName]
	if !exists {
		timeout = l.config.LockTimeout
	}
	
	l.mu.Unlock()
	
	// Create a channel for timeout
	timeoutCh := time.After(timeout)
	
	// Try to acquire the lock with timeout
	lockCh := make(chan struct{})
	
	go func() {
		lockInfo.Lock.Lock()
		close(lockCh)
	}()
	
	// Wait for lock acquisition or timeout
	select {
	case <-lockCh:
		// Lock acquired
		break
	case <-timeoutCh:
		// Timeout
		return fmt.Errorf("timeout acquiring lock %s", lockName)
	}
	
	// Update lock info
	l.mu.Lock()
	defer l.mu.Unlock()
	
	lockInfo.AcquisitionCount++
	lockInfo.IsHeld = true
	lockInfo.CurrentHolder = holderName
	lockInfo.AcquiredAt = time.Now()
	lockInfo.LastAcquired = time.Now()
	
	// Update deadlock detection info
	if l.config.DeadlockDetectionEnabled {
		l.lockHolders[lockName] = holderName
		
		if _, exists := l.holderLocks[holderName]; !exists {
			l.holderLocks[holderName] = make([]string, 0)
		}
		
		l.holderLocks[holderName] = append(l.holderLocks[holderName], lockName)
	}
	
	return nil
}

// ReleaseLock releases a lock with the given name
func (l *LockManager) ReleaseLock(lockName, holderName string) error {
	l.mu.Lock()
	
	// Check if the lock exists
	lockInfo, exists := l.locks[lockName]
	if !exists {
		l.mu.Unlock()
		return fmt.Errorf("lock %s not registered", lockName)
	}
	
	// Check if the holder is the current holder
	if lockInfo.IsHeld && lockInfo.CurrentHolder != holderName {
		l.mu.Unlock()
		return fmt.Errorf("lock %s is held by %s, not %s", lockName, lockInfo.CurrentHolder, holderName)
	}
	
	// Update lock info before releasing
	if lockInfo.IsHeld {
		heldTime := time.Since(lockInfo.AcquiredAt)
		lockInfo.TotalHeldTime += heldTime
		lockInfo.IsHeld = false
		lockInfo.LastReleased = time.Now()
		
		// Update deadlock detection info
		if l.config.DeadlockDetectionEnabled {
			delete(l.lockHolders, lockName)
			
			if locks, exists := l.holderLocks[holderName]; exists {
				newLocks := make([]string, 0, len(locks)-1)
				for _, lock := range locks {
					if lock != lockName {
						newLocks = append(newLocks, lock)
					}
				}
				
				if len(newLocks) == 0 {
					delete(l.holderLocks, holderName)
				} else {
					l.holderLocks[holderName] = newLocks
				}
			}
		}
	}
	
	l.mu.Unlock()
	
	// Release the lock
	lockInfo.Lock.Unlock()
	
	return nil
}

// checkForDeadlock checks if acquiring a lock would cause a deadlock
func (l *LockManager) checkForDeadlock(lockName string, holderName string) error {
	// If the holder doesn't hold any locks, there can't be a deadlock
	holderLocks, exists := l.holderLocks[holderName]
	if !exists || len(holderLocks) == 0 {
		return nil
	}
	
	// Check if the lock is already held by someone else
	currentHolder, lockHeld := l.lockHolders[lockName]
	if !lockHeld {
		return nil
	}
	
	if currentHolder == holderName {
		return fmt.Errorf("lock %s is already held by %s", lockName, holderName)
	}
	
	// Check if the current holder is waiting for any locks held by this holder
	visited := make(map[string]bool)
	return l.detectCycle(currentHolder, holderName, visited)
}

// detectCycle detects cycles in the lock dependency graph
func (l *LockManager) detectCycle(current string, target string, visited map[string]bool) error {
	if current == target {
		return fmt.Errorf("deadlock detected: circular lock dependency")
	}
	
	if visited[current] {
		return nil
	}
	
	visited[current] = true
	
	// Check locks held by the current holder
	locks, exists := l.holderLocks[current]
	if !exists {
		return nil
	}
	
	for _, lockName := range locks {
		// Check who's waiting for this lock
		for waitingHolder := range l.holderLocks {
			if waitingHolder == current {
				continue
			}
			
			// Check if this holder is waiting for the lock
			for _, waitingLock := range l.holderLocks[waitingHolder] {
				if waitingLock == lockName {
					if err := l.detectCycle(waitingHolder, target, visited); err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}

// GetLockInfo gets information about a lock
func (l *LockManager) GetLockInfo(lockName string) (*LockInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	lockInfo, exists := l.locks[lockName]
	if !exists {
		return nil, fmt.Errorf("lock %s not registered", lockName)
	}
	
	return lockInfo, nil
}

// GetAllLockInfo gets information about all locks
func (l *LockManager) GetAllLockInfo() map[string]*LockInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Create a copy of the locks map
	locks := make(map[string]*LockInfo, len(l.locks))
	for name, lockInfo := range l.locks {
		locks[name] = lockInfo
	}
	
	return locks
}

// GetLockStats gets statistics for a lock
func (l *LockManager) GetLockStats(name string) (map[string]interface{}, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	lockInfo, exists := l.locks[name]
	if !exists {
		return nil, fmt.Errorf("lock %s not registered", name)
	}
	
	stats := map[string]interface{}{
		"name":              lockInfo.Name,
		"acquisition_count": lockInfo.AcquisitionCount,
		"contention_count":  lockInfo.ContentionCount,
		"total_held_time":   lockInfo.TotalHeldTime.String(),
		"is_held":           lockInfo.IsHeld,
	}
	
	if lockInfo.IsHeld {
		stats["current_holder"] = lockInfo.CurrentHolder
		stats["acquired_at"] = lockInfo.AcquiredAt
		stats["held_time"] = time.Since(lockInfo.AcquiredAt).String()
	}
	
	if !lockInfo.LastAcquired.IsZero() {
		stats["last_acquired"] = lockInfo.LastAcquired
	}
	
	if !lockInfo.LastReleased.IsZero() {
		stats["last_released"] = lockInfo.LastReleased
	}
	
	return stats, nil
}

// GetAllLockStats gets statistics for all locks
func (l *LockManager) GetAllLockStats() map[string]map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	stats := make(map[string]map[string]interface{})
	
	for name, lockInfo := range l.locks {
		lockStats := map[string]interface{}{
			"name":              lockInfo.Name,
			"acquisition_count": lockInfo.AcquisitionCount,
			"contention_count":  lockInfo.ContentionCount,
			"total_held_time":   lockInfo.TotalHeldTime.String(),
			"is_held":           lockInfo.IsHeld,
		}
		
		if lockInfo.IsHeld {
			lockStats["current_holder"] = lockInfo.CurrentHolder
			lockStats["acquired_at"] = lockInfo.AcquiredAt
			lockStats["held_time"] = time.Since(lockInfo.AcquiredAt).String()
		}
		
		if !lockInfo.LastAcquired.IsZero() {
			lockStats["last_acquired"] = lockInfo.LastAcquired
		}
		
		if !lockInfo.LastReleased.IsZero() {
			lockStats["last_released"] = lockInfo.LastReleased
		}
		
		stats[name] = lockStats
	}
	
	return stats
}

// GetLockOrder returns the lock order
func (l *LockManager) GetLockOrder() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Create a copy of the lock order
	lockOrder := make([]string, len(l.lockOrder))
	copy(lockOrder, l.lockOrder)
	
	return lockOrder
}

// SetLockOrder sets the lock order
func (l *LockManager) SetLockOrder(lockOrder []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if all locks in the lock order are registered
	for _, name := range lockOrder {
		if _, exists := l.locks[name]; !exists {
			return fmt.Errorf("lock %s not registered", name)
		}
	}
	
	// Check if all registered locks are in the lock order
	for name := range l.locks {
		found := false
		for _, orderName := range lockOrder {
			if orderName == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("lock %s not in lock order", name)
		}
	}
	
	// Set the lock order
	l.lockOrder = make([]string, len(lockOrder))
	copy(l.lockOrder, lockOrder)
	
	return nil
}

// SortLockOrder sorts the lock order
func (l *LockManager) SortLockOrder() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Sort the lock order
	sort.Strings(l.lockOrder)
}

