package coordination

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LockInfo contains information about a lock
type LockInfo struct {
	// Lock name
	Name string
	
	// Lock object
	Lock sync.Locker
	
	// Current holder
	CurrentHolder string
	
	// Acquisition time
	AcquisitionTime time.Time
}

// LockManagerConfig contains configuration for the lock manager
type LockManagerConfig struct {
	// Deadlock detection enabled
	DeadlockDetectionEnabled bool
	
	// Lock timeout
	LockTimeout time.Duration
}

// DefaultLockManagerConfig returns the default lock manager configuration
func DefaultLockManagerConfig() LockManagerConfig {
	return LockManagerConfig{
		DeadlockDetectionEnabled: true,
		LockTimeout:              5 * time.Second,
	}
}

// LockManager manages locks to prevent deadlocks
type LockManager struct {
	// Configuration
	config LockManagerConfig
	
	// Lock registry
	locks map[string]*LockInfo
	
	// Lock acquisition order
	lockOrder []string
	
	// Deadlock detection
	lockHolders map[string]string
	holderLocks map[string][]string
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
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

// AcquireLock acquires a lock
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

// ReleaseLock releases a lock
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
		return fmt.Errorf("lock %s is not held by %s", lockName, holderName)
	}
	
	// Update lock info
	lockInfo.CurrentHolder = ""
	
	// Update deadlock detection info
	delete(l.lockHolders, lockName)
	if locks, exists := l.holderLocks[holderName]; exists {
		newLocks := make([]string, 0)
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
	
	l.mu.Unlock()
	
	// Release the lock
	lockInfo.Lock.Unlock()
	
	return nil
}

// wouldCreateDeadlock checks if acquiring a lock would create a deadlock
func (l *LockManager) wouldCreateDeadlock(lockName, holderName string) bool {
	// If the holder doesn't hold any locks, there's no deadlock
	holderLocks, exists := l.holderLocks[holderName]
	if !exists || len(holderLocks) == 0 {
		return false
	}
	
	// Check if the lock is already held by someone else
	currentHolder, exists := l.lockHolders[lockName]
	if !exists || currentHolder == "" || currentHolder == holderName {
		return false
	}
	
	// Check if the current holder is waiting for a lock held by the new holder
	return l.isWaitingFor(currentHolder, holderName, make(map[string]bool))
}

// isWaitingFor checks if holder1 is waiting for a lock held by holder2
func (l *LockManager) isWaitingFor(holder1, holder2 string, visited map[string]bool) bool {
	// If we've already visited this holder, there's a cycle
	if visited[holder1] {
		return false
	}
	
	// Mark the holder as visited
	visited[holder1] = true
	
	// Get the locks held by holder1
	locks, exists := l.holderLocks[holder1]
	if !exists || len(locks) == 0 {
		return false
	}
	
	// Check if holder1 is waiting for a lock held by holder2
	for _, lockName := range locks {
		currentHolder, exists := l.lockHolders[lockName]
		if !exists || currentHolder == "" {
			continue
		}
		
		// If holder1 is waiting for a lock held by holder2, there's a deadlock
		if currentHolder == holder2 {
			return true
		}
		
		// Check if the current holder is waiting for a lock held by holder2
		if l.isWaitingFor(currentHolder, holder2, visited) {
			return true
		}
	}
	
	return false
}

// GetLockInfo gets information about a lock
func (l *LockManager) GetLockInfo(lockName string) (*LockInfo, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	lockInfo, exists := l.locks[lockName]
	if !exists {
		return nil, fmt.Errorf("lock %s not registered", lockName)
	}
	
	return lockInfo, nil
}

// GetAllLockInfo gets information about all locks
func (l *LockManager) GetAllLockInfo() []*LockInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	result := make([]*LockInfo, 0, len(l.locks))
	for _, lockInfo := range l.locks {
		result = append(result, lockInfo)
	}
	
	return result
}

// GetLockOrder gets the lock acquisition order
func (l *LockManager) GetLockOrder() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	result := make([]string, len(l.lockOrder))
	copy(result, l.lockOrder)
	
	return result
}

