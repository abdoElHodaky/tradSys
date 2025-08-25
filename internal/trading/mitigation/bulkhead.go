package mitigation

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BulkheadConfig represents the configuration for a bulkhead
type BulkheadConfig struct {
	// MaxConcurrentCalls is the maximum number of concurrent calls allowed
	MaxConcurrentCalls int
	// MaxQueueSize is the maximum number of calls that can be queued
	MaxQueueSize int
	// QueueTimeout is the maximum time a call can wait in the queue
	QueueTimeout time.Duration
}

// DefaultBulkheadConfig returns a default configuration for a bulkhead
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrentCalls: 10,
		MaxQueueSize:       50,
		QueueTimeout:       5 * time.Second,
	}
}

// BulkheadMetrics tracks metrics for the bulkhead
type BulkheadMetrics struct {
	// Executed is the number of calls that were executed
	Executed int64
	// Rejected is the number of calls that were rejected
	Rejected int64
	// QueueTimeouts is the number of calls that timed out in the queue
	QueueTimeouts int64
	// CurrentConcurrentCalls is the current number of concurrent calls
	CurrentConcurrentCalls int
	// CurrentQueueSize is the current size of the queue
	CurrentQueueSize int
	// MaxConcurrencyReached is the number of times max concurrency was reached
	MaxConcurrencyReached int64
	// MaxQueueSizeReached is the number of times max queue size was reached
	MaxQueueSizeReached int64
}

// Bulkhead implements the bulkhead pattern to limit concurrent calls
type Bulkhead struct {
	name      string
	config    BulkheadConfig
	semaphore chan struct{}
	queue     chan struct{}
	metrics   *BulkheadMetrics
	mutex     sync.RWMutex
	logger    *zap.Logger
}

// NewBulkhead creates a new bulkhead with the given name and configuration
func NewBulkhead(name string, config BulkheadConfig, logger *zap.Logger) *Bulkhead {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &Bulkhead{
		name:      name,
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrentCalls),
		queue:     make(chan struct{}, config.MaxQueueSize),
		metrics: &BulkheadMetrics{
			Executed:               0,
			Rejected:               0,
			QueueTimeouts:          0,
			CurrentConcurrentCalls: 0,
			CurrentQueueSize:       0,
			MaxConcurrencyReached:  0,
			MaxQueueSizeReached:    0,
		},
		logger: logger.With(zap.String("component", "bulkhead"), zap.String("name", name)),
	}
}

// Execute executes the given function with bulkhead protection
func (b *Bulkhead) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Try to enter the queue
	select {
	case b.queue <- struct{}{}:
		// Successfully entered the queue
		b.mutex.Lock()
		b.metrics.CurrentQueueSize++
		b.mutex.Unlock()
		
		b.logger.Debug("Request queued",
			zap.String("name", b.name),
			zap.Int("queue_size", b.metrics.CurrentQueueSize))
	default:
		// Queue is full
		b.mutex.Lock()
		b.metrics.Rejected++
		b.metrics.MaxQueueSizeReached++
		b.mutex.Unlock()
		
		b.logger.Debug("Request rejected, queue full",
			zap.String("name", b.name),
			zap.Int("max_queue_size", b.config.MaxQueueSize))
		return ErrBulkheadQueueFull
	}

	// Dequeue when done
	defer func() {
		<-b.queue
		b.mutex.Lock()
		b.metrics.CurrentQueueSize--
		b.mutex.Unlock()
	}()

	// Wait for execution slot with timeout
	queueTimer := time.NewTimer(b.config.QueueTimeout)
	defer queueTimer.Stop()

	select {
	case b.semaphore <- struct{}{}:
		// Got execution slot
		b.mutex.Lock()
		b.metrics.CurrentConcurrentCalls++
		if b.metrics.CurrentConcurrentCalls == b.config.MaxConcurrentCalls {
			b.metrics.MaxConcurrencyReached++
		}
		b.mutex.Unlock()
		
		b.logger.Debug("Request executing",
			zap.String("name", b.name),
			zap.Int("concurrent_calls", b.metrics.CurrentConcurrentCalls))
	case <-queueTimer.C:
		// Timeout waiting for execution slot
		b.mutex.Lock()
		b.metrics.QueueTimeouts++
		b.mutex.Unlock()
		
		b.logger.Debug("Request timed out in queue",
			zap.String("name", b.name),
			zap.Duration("queue_timeout", b.config.QueueTimeout))
		return ErrBulkheadQueueTimeout
	case <-ctx.Done():
		// Context cancelled while waiting
		return ctx.Err()
	}

	// Release execution slot when done
	defer func() {
		<-b.semaphore
		b.mutex.Lock()
		b.metrics.CurrentConcurrentCalls--
		b.metrics.Executed++
		b.mutex.Unlock()
	}()

	// Execute the function
	return fn(ctx)
}

// GetMetrics returns a copy of the current metrics
func (b *Bulkhead) GetMetrics() BulkheadMetrics {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	return BulkheadMetrics{
		Executed:               b.metrics.Executed,
		Rejected:               b.metrics.Rejected,
		QueueTimeouts:          b.metrics.QueueTimeouts,
		CurrentConcurrentCalls: b.metrics.CurrentConcurrentCalls,
		CurrentQueueSize:       b.metrics.CurrentQueueSize,
		MaxConcurrencyReached:  b.metrics.MaxConcurrencyReached,
		MaxQueueSizeReached:    b.metrics.MaxQueueSizeReached,
	}
}

// UpdateConfig updates the bulkhead configuration
func (b *Bulkhead) UpdateConfig(config BulkheadConfig) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	// Create new channels with updated capacities
	newSemaphore := make(chan struct{}, config.MaxConcurrentCalls)
	newQueue := make(chan struct{}, config.MaxQueueSize)
	
	// Transfer existing permits to new channels
	for i := 0; i < b.metrics.CurrentConcurrentCalls; i++ {
		newSemaphore <- struct{}{}
	}
	
	for i := 0; i < b.metrics.CurrentQueueSize; i++ {
		newQueue <- struct{}{}
	}
	
	// Update channels and config
	b.semaphore = newSemaphore
	b.queue = newQueue
	b.config = config
	
	b.logger.Info("Bulkhead configuration updated",
		zap.String("name", b.name),
		zap.Int("max_concurrent_calls", config.MaxConcurrentCalls),
		zap.Int("max_queue_size", config.MaxQueueSize),
		zap.Duration("queue_timeout", config.QueueTimeout))
}

// Reset resets the bulkhead metrics
func (b *Bulkhead) Reset() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	b.metrics.Executed = 0
	b.metrics.Rejected = 0
	b.metrics.QueueTimeouts = 0
	b.metrics.MaxConcurrencyReached = 0
	b.metrics.MaxQueueSizeReached = 0
	
	b.logger.Info("Bulkhead metrics reset", zap.String("name", b.name))
}

// ErrBulkheadQueueFull is returned when the bulkhead queue is full
var ErrBulkheadQueueFull = BulkheadError{message: "bulkhead queue is full"}

// ErrBulkheadQueueTimeout is returned when a request times out in the queue
var ErrBulkheadQueueTimeout = BulkheadError{message: "request timed out waiting in bulkhead queue"}

// BulkheadError represents a bulkhead error
type BulkheadError struct {
	message string
}

// Error returns the error message
func (e BulkheadError) Error() string {
	return e.message
}

