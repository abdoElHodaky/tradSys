package coordination

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MemoryPressureLevel represents the level of memory pressure
type MemoryPressureLevel int

const (
	// MemoryPressureLow indicates low memory pressure
	MemoryPressureLow MemoryPressureLevel = iota
	// MemoryPressureMedium indicates medium memory pressure
	MemoryPressureMedium
	// MemoryPressureHigh indicates high memory pressure
	MemoryPressureHigh
	// MemoryPressureCritical indicates critical memory pressure
	MemoryPressureCritical
)

// LockManagerConfig contains configuration for the lock manager
type LockManagerConfig struct {
	// DeadlockDetectionEnabled enables deadlock detection
	DeadlockDetectionEnabled bool
	// LockTimeout is the default timeout for acquiring locks
	LockTimeout time.Duration
}

// LockInfo contains information about a lock
type LockInfo struct {
	// Lock identity
	Name string
	
	// Lock object
	Lock sync.Locker
	
	// Current state
	CurrentHolder   string
	AcquisitionTime time.Time
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
}

// NewLockManager creates a new lock manager
func NewLockManager(config LockManagerConfig, logger *zap.Logger) *LockManager {
	return &LockManager{
		config:      config,
		locks:       make(map[string]*LockInfo),
		lockOrder:   make([]string, 0),
		lockHolders: make(map[string]string),
		holderLocks: make(map[string][]string),
		logger:      logger,
	}
}

// RegisterLock registers a lock with the manager
func (l *LockManager) RegisterLock(name string, lock sync.Locker) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if the lock is already registered
	if _, exists := l.locks[name]; exists {
		return
	}
	
	// Register the lock
	l.locks[name] = &LockInfo{
		Name:  name,
		Lock:  lock,
	}
	
	// Add to the lock order
	l.lockOrder = append(l.lockOrder, name)
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
		if l.wouldCreateDeadlock(lockName, holderName) {
			l.mu.Unlock()
			return fmt.Errorf("acquiring lock %s by %s would create a deadlock", lockName, holderName)
		}
	}
	
	l.mu.Unlock()
	
	// Acquire the lock
	lockInfo.Lock.Lock()
	
	// Update lock info
	l.mu.Lock()
	lockInfo.CurrentHolder = holderName
	lockInfo.AcquisitionTime = time.Now()
	
	// Update deadlock detection info
	l.lockHolders[lockName] = holderName
	if _, exists := l.holderLocks[holderName]; !exists {
		l.holderLocks[holderName] = make([]string, 0)
	}
	l.holderLocks[holderName] = append(l.holderLocks[holderName], lockName)
	l.mu.Unlock()
	
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
	if lockInfo.CurrentHolder != holderName {
		l.mu.Unlock()
		return fmt.Errorf("lock %s not held by %s", lockName, holderName)
	}
	
	// Update deadlock detection info
	delete(l.lockHolders, lockName)
	if locks, exists := l.holderLocks[holderName]; exists {
		for i, lock := range locks {
			if lock == lockName {
				l.holderLocks[holderName] = append(locks[:i], locks[i+1:]...)
				break
			}
		}
		if len(l.holderLocks[holderName]) == 0 {
			delete(l.holderLocks, holderName)
		}
	}
	
	// Update lock info
	lockInfo.CurrentHolder = ""
	
	l.mu.Unlock()
	
	// Release the lock
	lockInfo.Lock.Unlock()
	
	return nil
}

// wouldCreateDeadlock checks if acquiring a lock would create a deadlock
func (l *LockManager) wouldCreateDeadlock(lockName, holderName string) bool {
	// Check if the holder already holds locks
	if locks, exists := l.holderLocks[holderName]; exists && len(locks) > 0 {
		// Get the index of the lock in the lock order
		lockIndex := -1
		for i, name := range l.lockOrder {
			if name == lockName {
				lockIndex = i
				break
			}
		}
		
		// Check if the lock is already held by another holder
		if currentHolder, exists := l.lockHolders[lockName]; exists && currentHolder != holderName {
			// Check if the current holder is waiting for any locks held by the new holder
			for _, heldLock := range locks {
				if l.isWaitingFor(currentHolder, heldLock) {
					return true
				}
			}
		}
		
		// Check if the lock order would be violated
		for _, heldLock := range locks {
			heldLockIndex := -1
			for i, name := range l.lockOrder {
				if name == heldLock {
					heldLockIndex = i
					break
				}
			}
			
			if heldLockIndex > lockIndex {
				return true
			}
		}
	}
	
	return false
}

// isWaitingFor checks if a holder is waiting for a lock
func (l *LockManager) isWaitingFor(holderName, lockName string) bool {
	// Check if the lock is held by another holder
	if currentHolder, exists := l.lockHolders[lockName]; exists && currentHolder != holderName {
		// Check if the current holder is waiting for any locks held by the holder
		if locks, exists := l.holderLocks[currentHolder]; exists {
			for _, heldLock := range locks {
				if heldLock == lockName {
					return false // The holder already has the lock
				}
				
				// Recursively check if the current holder is waiting for any locks held by the holder
				if l.isWaitingFor(currentHolder, heldLock) {
					return true
				}
			}
		}
	}
	
	return false
}

// GetLockInfo returns information about a lock
func (l *LockManager) GetLockInfo(lockName string) (*LockInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if the lock exists
	lockInfo, exists := l.locks[lockName]
	if !exists {
		return nil, fmt.Errorf("lock %s not registered", lockName)
	}
	
	return lockInfo, nil
}

// GetAllLockInfo returns information about all locks
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

