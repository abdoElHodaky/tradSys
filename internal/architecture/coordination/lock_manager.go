package coordination

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LockInfo contains information about a lock
type LockInfo struct {
	// Lock
	Lock sync.Mutex

	// Statistics
	AcquisitionCount int
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
	lockHolders map[string]string   // lockName -> holderName
	holderLocks map[string][]string // holderName -> []lockName
	
	// Lock timeouts
	lockTimeouts map[string]time.Duration // lockName -> timeout
}

// NewLockManager creates a new lock manager
func NewLockManager(config LockManagerConfig, logger *zap.Logger) *LockManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &LockManager{
		config:       config,
		locks:        make(map[string]*LockInfo),
		lockHolders:  make(map[string]string),
		holderLocks:  make(map[string][]string),
		lockTimeouts: make(map[string]time.Duration),
		logger:       logger,
	}
}

// RegisterLock registers a lock with the manager
func (l *LockManager) RegisterLock(lockName string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if _, exists := l.locks[lockName]; exists {
		return fmt.Errorf("lock %s already registered", lockName)
	}
	
	l.locks[lockName] = &LockInfo{
		AcquisitionCount: 0,
		TotalHeldTime:    0,
		LastAcquired:     time.Time{},
		LastReleased:     time.Time{},
		IsHeld:           false,
		CurrentHolder:    "",
		AcquiredAt:       time.Time{},
	}
	
	l.logger.Debug("Registered lock", zap.String("lock", lockName))
	
	return nil
}

// UnregisterLock unregisters a lock from the manager
func (l *LockManager) UnregisterLock(lockName string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	lockInfo, exists := l.locks[lockName]
	if !exists {
		return fmt.Errorf("lock %s not registered", lockName)
	}
	
	if lockInfo.IsHeld {
		return fmt.Errorf("lock %s is currently held by %s", lockName, lockInfo.CurrentHolder)
	}
	
	delete(l.locks, lockName)
	delete(l.lockTimeouts, lockName)
	delete(l.lockHolders, lockName)
	
	l.logger.Debug("Unregistered lock", zap.String("lock", lockName))
	
	return nil
}

// SetLockTimeout sets the timeout for a specific lock
func (l *LockManager) SetLockTimeout(lockName string, timeout time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if _, exists := l.locks[lockName]; !exists {
		return fmt.Errorf("lock %s not registered", lockName)
	}
	
	l.lockTimeouts[lockName] = timeout
	
	return nil
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
func (l *LockManager) AcquireLock(ctx context.Context, lockName, holderName string) error {
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
	
	// Create a channel to signal when the lock is acquired
	lockAcquired := make(chan struct{}, 1)
	
	// Try to acquire the lock in a goroutine
	go func() {
		lockInfo.Lock.Lock()
		lockAcquired <- struct{}{}
	}()
	
	// Release the manager lock while waiting for the lock
	l.mu.Unlock()
	
	// Wait for the lock with timeout
	select {
	case <-lockAcquired:
		// Lock acquired
	case <-time.After(timeout):
		// Timeout
		return fmt.Errorf("timeout acquiring lock %s after %s", lockName, timeout)
	case <-ctx.Done():
		// Context cancelled
		return fmt.Errorf("context cancelled while acquiring lock %s: %w", lockName, ctx.Err())
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
	
	l.logger.Debug("Acquired lock",
		zap.String("lock", lockName),
		zap.String("holder", holderName),
	)
	
	return nil
}

// TryAcquireLock tries to acquire a lock without blocking
func (l *LockManager) TryAcquireLock(lockName, holderName string) (bool, error) {
	l.mu.Lock()
	
	// Check if the lock exists
	lockInfo, exists := l.locks[lockName]
	if !exists {
		l.mu.Unlock()
		return false, fmt.Errorf("lock %s not registered", lockName)
	}
	
	// Check for deadlocks if enabled
	if l.config.DeadlockDetectionEnabled {
		// Check if acquiring this lock would create a deadlock
		if err := l.checkForDeadlock(lockName, holderName); err != nil {
			l.mu.Unlock()
			return false, err
		}
	}
	
	// Try to acquire the lock
	acquired := make(chan bool, 1)
	go func() {
		acquired <- lockInfo.Lock.TryLock()
	}()
	
	// Release the manager lock while waiting for the result
	l.mu.Unlock()
	
	// Get the result
	if !<-acquired {
		return false, nil
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
	
	l.logger.Debug("Acquired lock (try)",
		zap.String("lock", lockName),
		zap.String("holder", holderName),
	)
	
	return true, nil
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
	
	// Check if the lock is held by the specified holder
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
		
		l.logger.Debug("Released lock",
			zap.String("lock", lockName),
			zap.String("holder", holderName),
			zap.Duration("heldTime", heldTime),
		)
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
	
	// If the lock is already held by this holder, there's no deadlock
	if currentHolder == holderName {
		return nil
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
	
	// Check if any of the locks held by the current holder are waiting for locks held by other holders
	for _, lockName := range locks {
		nextHolder, exists := l.lockHolders[lockName]
		if !exists {
			continue
		}
		
		if err := l.detectCycle(nextHolder, target, visited); err != nil {
			return err
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
	
	// Create a copy to avoid race conditions
	infoCopy := &LockInfo{
		AcquisitionCount: lockInfo.AcquisitionCount,
		TotalHeldTime:    lockInfo.TotalHeldTime,
		LastAcquired:     lockInfo.LastAcquired,
		LastReleased:     lockInfo.LastReleased,
		IsHeld:           lockInfo.IsHeld,
		CurrentHolder:    lockInfo.CurrentHolder,
		AcquiredAt:       lockInfo.AcquiredAt,
	}
	
	return infoCopy, nil
}

// GetAllLockInfo gets information about all locks
func (l *LockManager) GetAllLockInfo() map[string]*LockInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Create a copy to avoid race conditions
	infoCopy := make(map[string]*LockInfo)
	for name, lockInfo := range l.locks {
		infoCopy[name] = &LockInfo{
			AcquisitionCount: lockInfo.AcquisitionCount,
			TotalHeldTime:    lockInfo.TotalHeldTime,
			LastAcquired:     lockInfo.LastAcquired,
			LastReleased:     lockInfo.LastReleased,
			IsHeld:           lockInfo.IsHeld,
			CurrentHolder:    lockInfo.CurrentHolder,
			AcquiredAt:       lockInfo.AcquiredAt,
		}
	}
	
	return infoCopy
}

// GetLockHolders gets the current lock holders
func (l *LockManager) GetLockHolders() map[string]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Create a copy to avoid race conditions
	holdersCopy := make(map[string]string)
	for name, holder := range l.lockHolders {
		holdersCopy[name] = holder
	}
	
	return holdersCopy
}

// GetHolderLocks gets the locks held by a holder
func (l *LockManager) GetHolderLocks(holderName string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	locks, exists := l.holderLocks[holderName]
	if !exists {
		return nil
	}
	
	// Create a copy to avoid race conditions
	locksCopy := make([]string, len(locks))
	copy(locksCopy, locks)
	
	return locksCopy
}

// IsLockHeld checks if a lock is held
func (l *LockManager) IsLockHeld(lockName string) (bool, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	lockInfo, exists := l.locks[lockName]
	if !exists {
		return false, "", fmt.Errorf("lock %s not registered", lockName)
	}
	
	return lockInfo.IsHeld, lockInfo.CurrentHolder, nil
}

// ResetStatistics resets the lock statistics
func (l *LockManager) ResetStatistics() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for _, lockInfo := range l.locks {
		lockInfo.AcquisitionCount = 0
		lockInfo.TotalHeldTime = 0
		lockInfo.LastAcquired = time.Time{}
		lockInfo.LastReleased = time.Time{}
	}
}

// GetStatistics gets the lock statistics
func (l *LockManager) GetStatistics() map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	stats := make(map[string]interface{})
	stats["lockCount"] = len(l.locks)
	stats["deadlockDetectionEnabled"] = l.config.DeadlockDetectionEnabled
	stats["defaultTimeout"] = l.config.LockTimeout
	
	lockStats := make(map[string]interface{})
	for name, lockInfo := range l.locks {
		lockStats[name] = map[string]interface{}{
			"acquisitionCount": lockInfo.AcquisitionCount,
			"totalHeldTime":    lockInfo.TotalHeldTime,
			"lastAcquired":     lockInfo.LastAcquired,
			"lastReleased":     lockInfo.LastReleased,
			"isHeld":           lockInfo.IsHeld,
			"currentHolder":    lockInfo.CurrentHolder,
			"acquiredAt":       lockInfo.AcquiredAt,
		}
	}
	stats["locks"] = lockStats
	
	return stats
}

