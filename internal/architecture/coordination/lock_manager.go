package coordination

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// LockInfo contains information about a lock
type LockInfo struct {
	// Lock
	mu sync.Mutex

	// Statistics
	AcquisitionCount int64
	TotalHeldTime    int64 // nanoseconds
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

	// DeadlockDetectionInterval is the interval at which to check for deadlocks
	DeadlockDetectionInterval time.Duration

	// MaxLockHoldTime is the maximum time a lock can be held before it's considered a potential deadlock
	MaxLockHoldTime time.Duration

	// EnableMetrics enables metrics collection
	EnableMetrics bool
}

// DefaultLockManagerConfig returns the default lock manager configuration
func DefaultLockManagerConfig() LockManagerConfig {
	return LockManagerConfig{
		DeadlockDetectionEnabled:   true,
		LockTimeout:                5 * time.Second,
		DeadlockDetectionInterval:  1 * time.Second,
		MaxLockHoldTime:            30 * time.Second,
		EnableMetrics:              true,
	}
}

// LockManager manages locks and prevents deadlocks
type LockManager struct {
	// Configuration
	config LockManagerConfig
	
	// Locks
	locks      map[string]*LockInfo
	locksMutex sync.RWMutex
	
	// Lock acquisition graph for deadlock detection
	lockGraph      map[string]map[string]bool
	lockGraphMutex sync.RWMutex
	
	// Statistics
	totalLocks        int64
	totalAcquisitions int64
	totalTimeouts     int64
	totalDeadlocks    int64
	
	// Context for cancellation
	ctx        context.Context
	cancelFunc context.CancelFunc
	
	// Wait group for goroutines
	wg sync.WaitGroup
	
	// Logger
	logger *zap.Logger
}

// NewLockManager creates a new lock manager
func NewLockManager(config LockManagerConfig, logger *zap.Logger) *LockManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	lm := &LockManager{
		config:     config,
		locks:      make(map[string]*LockInfo),
		lockGraph:  make(map[string]map[string]bool),
		ctx:        ctx,
		cancelFunc: cancel,
		logger:     logger,
	}
	
	// Start deadlock detection if enabled
	if config.DeadlockDetectionEnabled {
		lm.wg.Add(1)
		go lm.deadlockDetectionLoop()
	}
	
	return lm
}

// RegisterLock registers a lock with the lock manager
func (lm *LockManager) RegisterLock(lockID string) error {
	lm.locksMutex.Lock()
	defer lm.locksMutex.Unlock()
	
	// Check if the lock already exists
	if _, exists := lm.locks[lockID]; exists {
		return fmt.Errorf("lock %s already exists", lockID)
	}
	
	// Create the lock
	lm.locks[lockID] = &LockInfo{
		LastAcquired: time.Time{},
		LastReleased: time.Time{},
	}
	
	// Update statistics
	atomic.AddInt64(&lm.totalLocks, 1)
	
	lm.logger.Debug("Registered lock",
		zap.String("lockID", lockID),
	)
	
	return nil
}

// AcquireLock acquires a lock
func (lm *LockManager) AcquireLock(lockID, holderID string) error {
	return lm.AcquireLockWithTimeout(lockID, holderID, lm.config.LockTimeout)
}

// AcquireLockWithTimeout acquires a lock with a timeout
func (lm *LockManager) AcquireLockWithTimeout(lockID, holderID string, timeout time.Duration) error {
	// Get the lock info
	lockInfo, err := lm.getLockInfo(lockID)
	if err != nil {
		return err
	}
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Try to acquire the lock
	acquired := make(chan struct{})
	go func() {
		// Lock the mutex
		lockInfo.mu.Lock()
		
		// Check if the lock is already held
		if lockInfo.IsHeld {
			// Record the lock acquisition attempt in the lock graph
			lm.recordLockAcquisitionAttempt(holderID, lockInfo.CurrentHolder)
			
			// Wait for the lock to be released
			for lockInfo.IsHeld {
				// Unlock the mutex while waiting
				lockInfo.mu.Unlock()
				
				// Sleep for a short time
				time.Sleep(10 * time.Millisecond)
				
				// Lock the mutex again
				lockInfo.mu.Lock()
			}
		}
		
		// Mark the lock as held
		lockInfo.IsHeld = true
		lockInfo.CurrentHolder = holderID
		lockInfo.AcquiredAt = time.Now()
		lockInfo.LastAcquired = lockInfo.AcquiredAt
		
		// Update statistics
		atomic.AddInt64(&lockInfo.AcquisitionCount, 1)
		atomic.AddInt64(&lm.totalAcquisitions, 1)
		
		// Unlock the mutex
		lockInfo.mu.Unlock()
		
		// Signal that the lock has been acquired
		close(acquired)
	}()
	
	// Wait for the lock to be acquired or the timeout to expire
	select {
	case <-acquired:
		// Lock acquired
		lm.logger.Debug("Acquired lock",
			zap.String("lockID", lockID),
			zap.String("holderID", holderID),
		)
		return nil
	case <-ctx.Done():
		// Timeout
		atomic.AddInt64(&lm.totalTimeouts, 1)
		return fmt.Errorf("timeout acquiring lock %s: %w", lockID, ctx.Err())
	}
}

// ReleaseLock releases a lock
func (lm *LockManager) ReleaseLock(lockID, holderID string) error {
	// Get the lock info
	lockInfo, err := lm.getLockInfo(lockID)
	if err != nil {
		return err
	}
	
	// Lock the mutex
	lockInfo.mu.Lock()
	defer lockInfo.mu.Unlock()
	
	// Check if the lock is held
	if !lockInfo.IsHeld {
		return fmt.Errorf("lock %s is not held", lockID)
	}
	
	// Check if the holder is correct
	if lockInfo.CurrentHolder != holderID {
		return fmt.Errorf("lock %s is held by %s, not %s", lockID, lockInfo.CurrentHolder, holderID)
	}
	
	// Calculate the held time
	heldTime := time.Since(lockInfo.AcquiredAt)
	
	// Update statistics
	atomic.AddInt64(&lockInfo.TotalHeldTime, int64(heldTime))
	
	// Mark the lock as released
	lockInfo.IsHeld = false
	lockInfo.CurrentHolder = ""
	lockInfo.LastReleased = time.Now()
	
	// Remove the holder from the lock graph
	lm.removeLockHolder(holderID)
	
	lm.logger.Debug("Released lock",
		zap.String("lockID", lockID),
		zap.String("holderID", holderID),
		zap.Duration("heldTime", heldTime),
	)
	
	return nil
}

// IsLockHeld checks if a lock is held
func (lm *LockManager) IsLockHeld(lockID string) (bool, error) {
	// Get the lock info
	lockInfo, err := lm.getLockInfo(lockID)
	if err != nil {
		return false, err
	}
	
	// Lock the mutex
	lockInfo.mu.Lock()
	defer lockInfo.mu.Unlock()
	
	return lockInfo.IsHeld, nil
}

// GetLockHolder gets the holder of a lock
func (lm *LockManager) GetLockHolder(lockID string) (string, error) {
	// Get the lock info
	lockInfo, err := lm.getLockInfo(lockID)
	if err != nil {
		return "", err
	}
	
	// Lock the mutex
	lockInfo.mu.Lock()
	defer lockInfo.mu.Unlock()
	
	if !lockInfo.IsHeld {
		return "", fmt.Errorf("lock %s is not held", lockID)
	}
	
	return lockInfo.CurrentHolder, nil
}

// GetLockInfo gets information about a lock
func (lm *LockManager) GetLockInfo(lockID string) (map[string]interface{}, error) {
	// Get the lock info
	lockInfo, err := lm.getLockInfo(lockID)
	if err != nil {
		return nil, err
	}
	
	// Lock the mutex
	lockInfo.mu.Lock()
	defer lockInfo.mu.Unlock()
	
	// Create the result
	result := make(map[string]interface{})
	result["acquisitionCount"] = atomic.LoadInt64(&lockInfo.AcquisitionCount)
	result["totalHeldTime"] = time.Duration(atomic.LoadInt64(&lockInfo.TotalHeldTime))
	result["lastAcquired"] = lockInfo.LastAcquired
	result["lastReleased"] = lockInfo.LastReleased
	result["isHeld"] = lockInfo.IsHeld
	result["currentHolder"] = lockInfo.CurrentHolder
	
	if lockInfo.IsHeld {
		result["acquiredAt"] = lockInfo.AcquiredAt
		result["heldTime"] = time.Since(lockInfo.AcquiredAt)
	}
	
	return result, nil
}

// GetStats gets statistics about the lock manager
func (lm *LockManager) GetStats() map[string]interface{} {
	lm.locksMutex.RLock()
	defer lm.locksMutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["totalLocks"] = atomic.LoadInt64(&lm.totalLocks)
	stats["totalAcquisitions"] = atomic.LoadInt64(&lm.totalAcquisitions)
	stats["totalTimeouts"] = atomic.LoadInt64(&lm.totalTimeouts)
	stats["totalDeadlocks"] = atomic.LoadInt64(&lm.totalDeadlocks)
	
	// Get lock counts
	heldLocks := 0
	for _, lockInfo := range lm.locks {
		lockInfo.mu.Lock()
		if lockInfo.IsHeld {
			heldLocks++
		}
		lockInfo.mu.Unlock()
	}
	stats["heldLocks"] = heldLocks
	
	return stats
}

// Shutdown shuts down the lock manager
func (lm *LockManager) Shutdown() {
	// Cancel the context
	lm.cancelFunc()
	
	// Wait for goroutines to finish
	lm.wg.Wait()
	
	lm.logger.Info("Lock manager shutdown complete")
}

// getLockInfo gets the lock info for a lock
func (lm *LockManager) getLockInfo(lockID string) (*LockInfo, error) {
	lm.locksMutex.RLock()
	lockInfo, exists := lm.locks[lockID]
	lm.locksMutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("lock %s does not exist", lockID)
	}
	
	return lockInfo, nil
}

// recordLockAcquisitionAttempt records a lock acquisition attempt in the lock graph
func (lm *LockManager) recordLockAcquisitionAttempt(waiter, holder string) {
	lm.lockGraphMutex.Lock()
	defer lm.lockGraphMutex.Unlock()
	
	// Initialize the waiter's entry if it doesn't exist
	if _, exists := lm.lockGraph[waiter]; !exists {
		lm.lockGraph[waiter] = make(map[string]bool)
	}
	
	// Record that waiter is waiting for holder
	lm.lockGraph[waiter][holder] = true
}

// removeLockHolder removes a holder from the lock graph
func (lm *LockManager) removeLockHolder(holder string) {
	lm.lockGraphMutex.Lock()
	defer lm.lockGraphMutex.Unlock()
	
	// Remove the holder from the lock graph
	delete(lm.lockGraph, holder)
	
	// Remove the holder from all waiters
	for waiter, holders := range lm.lockGraph {
		delete(holders, holder)
		
		// Remove the waiter if it's not waiting for any holders
		if len(holders) == 0 {
			delete(lm.lockGraph, waiter)
		}
	}
}

// detectDeadlocks detects deadlocks in the lock graph
func (lm *LockManager) detectDeadlocks() [][]string {
	lm.lockGraphMutex.RLock()
	defer lm.lockGraphMutex.RUnlock()
	
	deadlocks := make([][]string, 0)
	
	// Check for cycles in the lock graph
	for waiter := range lm.lockGraph {
		visited := make(map[string]bool)
		path := make([]string, 0)
		
		if lm.detectCycle(waiter, visited, path, &deadlocks) {
			// Deadlock detected
			atomic.AddInt64(&lm.totalDeadlocks, 1)
		}
	}
	
	return deadlocks
}

// detectCycle detects a cycle in the lock graph starting from the given node
func (lm *LockManager) detectCycle(node string, visited map[string]bool, path []string, deadlocks *[][]string) bool {
	// Mark the node as visited
	visited[node] = true
	
	// Add the node to the path
	path = append(path, node)
	
	// Check all nodes that this node is waiting for
	for holder := range lm.lockGraph[node] {
		// If the holder is already in the path, we have a cycle
		for i, pathNode := range path {
			if pathNode == holder {
				// Extract the cycle
				cycle := append([]string{}, path[i:]...)
				cycle = append(cycle, holder)
				
				// Add the cycle to the deadlocks
				*deadlocks = append(*deadlocks, cycle)
				
				return true
			}
		}
		
		// If the holder hasn't been visited, check it recursively
		if !visited[holder] {
			if lm.detectCycle(holder, visited, path, deadlocks) {
				return true
			}
		}
	}
	
	return false
}

// deadlockDetectionLoop runs the deadlock detection loop
func (lm *LockManager) deadlockDetectionLoop() {
	defer lm.wg.Done()
	
	ticker := time.NewTicker(lm.config.DeadlockDetectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Detect deadlocks
			deadlocks := lm.detectDeadlocks()
			
			// Log deadlocks
			for _, cycle := range deadlocks {
				lm.logger.Warn("Deadlock detected",
					zap.Strings("cycle", cycle),
				)
			}
			
			// Check for long-held locks
			lm.checkLongHeldLocks()
		case <-lm.ctx.Done():
			return
		}
	}
}

// checkLongHeldLocks checks for locks that have been held for a long time
func (lm *LockManager) checkLongHeldLocks() {
	lm.locksMutex.RLock()
	defer lm.locksMutex.RUnlock()
	
	now := time.Now()
	
	for lockID, lockInfo := range lm.locks {
		lockInfo.mu.Lock()
		
		if lockInfo.IsHeld && now.Sub(lockInfo.AcquiredAt) > lm.config.MaxLockHoldTime {
			lm.logger.Warn("Lock held for a long time",
				zap.String("lockID", lockID),
				zap.String("holder", lockInfo.CurrentHolder),
				zap.Duration("heldTime", now.Sub(lockInfo.AcquiredAt)),
			)
		}
		
		lockInfo.mu.Unlock()
	}
}

// ForceReleaseLock forcibly releases a lock (use with caution)
func (lm *LockManager) ForceReleaseLock(lockID string) error {
	// Get the lock info
	lockInfo, err := lm.getLockInfo(lockID)
	if err != nil {
		return err
	}
	
	// Lock the mutex
	lockInfo.mu.Lock()
	defer lockInfo.mu.Unlock()
	
	// Check if the lock is held
	if !lockInfo.IsHeld {
		return fmt.Errorf("lock %s is not held", lockID)
	}
	
	// Get the current holder
	holder := lockInfo.CurrentHolder
	
	// Calculate the held time
	heldTime := time.Since(lockInfo.AcquiredAt)
	
	// Update statistics
	atomic.AddInt64(&lockInfo.TotalHeldTime, int64(heldTime))
	
	// Mark the lock as released
	lockInfo.IsHeld = false
	lockInfo.CurrentHolder = ""
	lockInfo.LastReleased = time.Now()
	
	// Remove the holder from the lock graph
	lm.removeLockHolder(holder)
	
	lm.logger.Warn("Forcibly released lock",
		zap.String("lockID", lockID),
		zap.String("holder", holder),
		zap.Duration("heldTime", heldTime),
	)
	
	return nil
}

// AcquireMultipleLocks acquires multiple locks atomically
func (lm *LockManager) AcquireMultipleLocks(lockIDs []string, holderID string) error {
	return lm.AcquireMultipleLocksWithTimeout(lockIDs, holderID, lm.config.LockTimeout)
}

// AcquireMultipleLocksWithTimeout acquires multiple locks atomically with a timeout
func (lm *LockManager) AcquireMultipleLocksWithTimeout(lockIDs []string, holderID string, timeout time.Duration) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Sort the lock IDs to prevent deadlocks
	sortedLockIDs := make([]string, len(lockIDs))
	copy(sortedLockIDs, lockIDs)
	sort.Strings(sortedLockIDs)
	
	// Acquire the locks in order
	acquired := make([]string, 0, len(sortedLockIDs))
	
	// Function to release acquired locks in case of failure
	releaseAcquired := func() {
		for _, lockID := range acquired {
			lm.ReleaseLock(lockID, holderID)
		}
	}
	
	// Try to acquire all locks
	for _, lockID := range sortedLockIDs {
		// Check if the context is cancelled
		if ctx.Err() != nil {
			releaseAcquired()
			return fmt.Errorf("timeout acquiring locks: %w", ctx.Err())
		}
		
		// Acquire the lock
		if err := lm.AcquireLockWithTimeout(lockID, holderID, timeout); err != nil {
			releaseAcquired()
			return fmt.Errorf("failed to acquire lock %s: %w", lockID, err)
		}
		
		// Add the lock to the acquired list
		acquired = append(acquired, lockID)
	}
	
	return nil
}

// ReleaseMultipleLocks releases multiple locks
func (lm *LockManager) ReleaseMultipleLocks(lockIDs []string, holderID string) error {
	// Release the locks in reverse order
	for i := len(lockIDs) - 1; i >= 0; i-- {
		if err := lm.ReleaseLock(lockIDs[i], holderID); err != nil {
			return fmt.Errorf("failed to release lock %s: %w", lockIDs[i], err)
		}
	}
	
	return nil
}

