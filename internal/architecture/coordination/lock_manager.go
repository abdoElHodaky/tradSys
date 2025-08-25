package coordination

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LockManager provides a unified approach to lock management
// to prevent deadlocks and ensure consistent lock acquisition order.
type LockManager struct {
	// Lock registry
	locks map[string]*LockInfo
	
	// Global lock for protecting the registry
	mu sync.Mutex
	
	// Logger
	logger *zap.Logger
	
	// Lock acquisition order
	lockOrder []string
	
	// Deadlock detection
	deadlockDetection bool
	lockHolders       map[string]string // lock -> holder
	holderLocks       map[string][]string // holder -> locks
	
	// Lock timeouts
	lockTimeouts map[string]time.Duration
	defaultTimeout time.Duration
}

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
	IsHeld       bool
	CurrentHolder string
	AcquiredAt    time.Time
}

// NewLockManager creates a new lock manager
func NewLockManager(logger *zap.Logger) *LockManager {
	return &LockManager{
		locks:            make(map[string]*LockInfo),
		logger:           logger,
		lockOrder:        make([]string, 0),
		deadlockDetection: true,
		lockHolders:      make(map[string]string),
		holderLocks:      make(map[string][]string),
		lockTimeouts:     make(map[string]time.Duration),
		defaultTimeout:   5 * time.Second,
	}
}

// RegisterLock registers a lock with the manager
func (m *LockManager) RegisterLock(name string, lock sync.Locker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.locks[name]; exists {
		m.logger.Warn("Lock already registered", zap.String("lock", name))
		return
	}
	
	m.locks[name] = &LockInfo{
		Name:            name,
		Lock:            lock,
		AcquisitionCount: 0,
		ContentionCount:  0,
		TotalHeldTime:    0,
		IsHeld:           false,
	}
	
	// Add to lock order
	m.lockOrder = append(m.lockOrder, name)
	
	// Sort lock order by name to ensure consistent ordering
	sort.Strings(m.lockOrder)
}

// SetLockTimeout sets the timeout for a lock
func (m *LockManager) SetLockTimeout(name string, timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.lockTimeouts[name] = timeout
}

// SetDefaultTimeout sets the default timeout for locks
func (m *LockManager) SetDefaultTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.defaultTimeout = timeout
}

// EnableDeadlockDetection enables or disables deadlock detection
func (m *LockManager) EnableDeadlockDetection(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.deadlockDetection = enabled
}

// AcquireLock acquires a lock
func (m *LockManager) AcquireLock(name string, holder string) error {
	// Check if lock exists
	m.mu.Lock()
	lockInfo, exists := m.locks[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("lock %s not registered", name)
	}
	
	// Check for potential deadlock
	if m.deadlockDetection {
		if err := m.checkForDeadlock(name, holder); err != nil {
			m.mu.Unlock()
			return err
		}
	}
	
	// Get timeout
	timeout, exists := m.lockTimeouts[name]
	if !exists {
		timeout = m.defaultTimeout
	}
	m.mu.Unlock()
	
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
		return fmt.Errorf("timeout acquiring lock %s", name)
	}
	
	// Update lock info
	m.mu.Lock()
	defer m.mu.Unlock()
	
	lockInfo.AcquisitionCount++
	lockInfo.IsHeld = true
	lockInfo.CurrentHolder = holder
	lockInfo.AcquiredAt = time.Now()
	lockInfo.LastAcquired = time.Now()
	
	// Update deadlock detection info
	if m.deadlockDetection {
		m.lockHolders[name] = holder
		
		if _, exists := m.holderLocks[holder]; !exists {
			m.holderLocks[holder] = make([]string, 0)
		}
		
		m.holderLocks[holder] = append(m.holderLocks[holder], name)
	}
	
	return nil
}

// ReleaseLock releases a lock
func (m *LockManager) ReleaseLock(name string, holder string) error {
	// Check if lock exists
	m.mu.Lock()
	lockInfo, exists := m.locks[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("lock %s not registered", name)
	}
	
	// Check if lock is held by the specified holder
	if lockInfo.IsHeld && lockInfo.CurrentHolder != holder {
		m.mu.Unlock()
		return fmt.Errorf("lock %s is held by %s, not %s", name, lockInfo.CurrentHolder, holder)
	}
	
	// Update lock info before releasing
	if lockInfo.IsHeld {
		heldTime := time.Since(lockInfo.AcquiredAt)
		lockInfo.TotalHeldTime += heldTime
		lockInfo.IsHeld = false
		lockInfo.LastReleased = time.Now()
		
		// Update deadlock detection info
		if m.deadlockDetection {
			delete(m.lockHolders, name)
			
			if locks, exists := m.holderLocks[holder]; exists {
				newLocks := make([]string, 0, len(locks)-1)
				for _, lock := range locks {
					if lock != name {
						newLocks = append(newLocks, lock)
					}
				}
				
				if len(newLocks) == 0 {
					delete(m.holderLocks, holder)
				} else {
					m.holderLocks[holder] = newLocks
				}
			}
		}
	}
	
	m.mu.Unlock()
	
	// Release the lock
	lockInfo.Lock.Unlock()
	
	return nil
}

// checkForDeadlock checks if acquiring a lock would cause a deadlock
func (m *LockManager) checkForDeadlock(lockName string, holder string) error {
	// If the holder doesn't hold any locks, there can't be a deadlock
	holderLocks, exists := m.holderLocks[holder]
	if !exists || len(holderLocks) == 0 {
		return nil
	}
	
	// Check if the lock is already held by someone else
	currentHolder, lockHeld := m.lockHolders[lockName]
	if !lockHeld {
		return nil
	}
	
	if currentHolder == holder {
		return fmt.Errorf("lock %s is already held by %s", lockName, holder)
	}
	
	// Check if the current holder is waiting for any locks held by this holder
	visited := make(map[string]bool)
	return m.detectCycle(currentHolder, holder, visited)
}

// detectCycle detects cycles in the lock dependency graph
func (m *LockManager) detectCycle(current string, target string, visited map[string]bool) error {
	if current == target {
		return fmt.Errorf("deadlock detected: circular lock dependency")
	}
	
	if visited[current] {
		return nil
	}
	
	visited[current] = true
	
	// Check locks held by the current holder
	locks, exists := m.holderLocks[current]
	if !exists {
		return nil
	}
	
	for _, lockName := range locks {
		// Check who's waiting for this lock
		for waitingHolder := range m.holderLocks {
			if waitingHolder == current {
				continue
			}
			
			// Check if this holder is waiting for the lock
			for _, waitingLock := range m.holderLocks[waitingHolder] {
				if waitingLock == lockName {
					if err := m.detectCycle(waitingHolder, target, visited); err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}

// GetLockStats gets statistics for a lock
func (m *LockManager) GetLockStats(name string) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	lockInfo, exists := m.locks[name]
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
func (m *LockManager) GetAllLockStats() map[string]map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stats := make(map[string]map[string]interface{})
	
	for name, lockInfo := range m.locks {
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

// GetLockOrder gets the lock acquisition order
func (m *LockManager) GetLockOrder() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Return a copy to avoid concurrent modification
	order := make([]string, len(m.lockOrder))
	copy(order, m.lockOrder)
	
	return order
}

