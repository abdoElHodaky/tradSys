package adaptive_loader

import (
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// BackpressureManager manages backpressure for the plugin loader
type BackpressureManager struct {
	// Configuration
	enabled           bool
	maxLoad           int64
	cooldownPeriod    time.Duration
	
	// State
	currentLoad       int64
	rejectionCount    int64
	lastRejectionTime time.Time
	
	// Logging
	logger            *zap.Logger
}

// NewBackpressureManager creates a new backpressure manager
func NewBackpressureManager(maxLoad int64, logger *zap.Logger) *BackpressureManager {
	return &BackpressureManager{
		enabled:        true,
		maxLoad:        maxLoad,
		cooldownPeriod: 5 * time.Second,
		logger:         logger,
	}
}

// SetEnabled enables or disables backpressure
func (b *BackpressureManager) SetEnabled(enabled bool) {
	b.enabled = enabled
}

// SetMaxLoad sets the maximum load
func (b *BackpressureManager) SetMaxLoad(maxLoad int64) {
	b.maxLoad = maxLoad
}

// SetCooldownPeriod sets the cooldown period
func (b *BackpressureManager) SetCooldownPeriod(period time.Duration) {
	b.cooldownPeriod = period
}

// IncreaseLoad increases the current load by the given amount
func (b *BackpressureManager) IncreaseLoad(amount int64) {
	atomic.AddInt64(&b.currentLoad, amount)
}

// DecreaseLoad decreases the current load by the given amount
func (b *BackpressureManager) DecreaseLoad(amount int64) {
	atomic.AddInt64(&b.currentLoad, -amount)
}

// GetCurrentLoad returns the current load
func (b *BackpressureManager) GetCurrentLoad() int64 {
	return atomic.LoadInt64(&b.currentLoad)
}

// GetRejectionCount returns the number of rejected operations
func (b *BackpressureManager) GetRejectionCount() int64 {
	return atomic.LoadInt64(&b.rejectionCount)
}

// ShouldRejectOperation checks if an operation should be rejected due to backpressure
func (b *BackpressureManager) ShouldRejectOperation(operationLoad int64) bool {
	if !b.enabled {
		return false
	}
	
	currentLoad := atomic.LoadInt64(&b.currentLoad)
	
	// Check if adding this operation would exceed the maximum load
	if currentLoad+operationLoad > b.maxLoad {
		// Increment rejection count
		atomic.AddInt64(&b.rejectionCount, 1)
		b.lastRejectionTime = time.Now()
		
		b.logger.Warn("Operation rejected due to backpressure",
			zap.Int64("current_load", currentLoad),
			zap.Int64("operation_load", operationLoad),
			zap.Int64("max_load", b.maxLoad),
			zap.Int64("rejection_count", atomic.LoadInt64(&b.rejectionCount)))
		
		return true
	}
	
	return false
}

// ExecuteWithBackpressure executes an operation with backpressure control
func (b *BackpressureManager) ExecuteWithBackpressure(
	operationLoad int64,
	operation func() error,
) error {
	// Check if the operation should be rejected
	if b.ShouldRejectOperation(operationLoad) {
		return fmt.Errorf("operation rejected due to backpressure")
	}
	
	// Increase load
	b.IncreaseLoad(operationLoad)
	
	// Execute operation
	err := operation()
	
	// Decrease load
	b.DecreaseLoad(operationLoad)
	
	return err
}

// GetStats returns statistics about the backpressure manager
func (b *BackpressureManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":            b.enabled,
		"max_load":           b.maxLoad,
		"current_load":       atomic.LoadInt64(&b.currentLoad),
		"rejection_count":    atomic.LoadInt64(&b.rejectionCount),
		"last_rejection":     b.lastRejectionTime,
		"cooldown_period":    b.cooldownPeriod,
		"load_percentage":    float64(atomic.LoadInt64(&b.currentLoad)) / float64(b.maxLoad) * 100,
	}
}
