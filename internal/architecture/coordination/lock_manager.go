package coordination

import (
	"fmt"
//<<<<<<< codegen-bot/integrate-coordination-system
//=======
	"sort"
//>>>>>>> main
	"sync"
	"time"

	"go.uber.org/zap"
)

//<<<<<<< codegen-bot/integrate-coordination-system
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
	
//=======
// LockManager provides a unified approach to lock management
// to prevent deadlocks and ensure consistent lock acquisition order.
type LockManager struct {
	// Lock registry
	locks map[string]*LockInfo
	
	// Global lock for protecting the registry
	mu sync.Mutex
	
	// Logger
	logger *zap.Logger
	
//>>>>>>> main
	// Lock acquisition order
	lockOrder []string
	
	// Deadlock detection
//<<<<<<< codegen-bot/integrate-coordination-system
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
//=======
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
//>>>>>>> main
	}
}

// RegisterLock registers a lock with the manager
//<<<<<<< codegen-bot/integrate-coordination-system
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
//=======
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
//>>>>>>> main
	
	return nil
}

// ReleaseLock releases a lock
//<<<<<<< codegen-bot/integrate-coordination-system
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
//=======
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
//>>>>>>> main
	
	// Release the lock
	lockInfo.Lock.Unlock()
	
	return nil
}

//<<<<<<< codegen-bot/integrate-coordination-system
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
//=======
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
//>>>>>>> main
}

