package utils

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// PerformanceMonitor provides comprehensive performance monitoring capabilities
type PerformanceMonitor struct {
	metrics interfaces.MetricsCollector
	logger  interfaces.Logger

	// Performance counters
	requestCount uint64
	errorCount   uint64
	totalLatency uint64
	maxLatency   uint64
	minLatency   uint64

	// Memory tracking
	allocations   uint64
	deallocations uint64
	gcCount       uint64

	// Concurrency tracking
	activeGoroutines int64
	maxGoroutines    int64

	// Circuit breaker state
	circuitOpen     bool
	failureCount    uint64
	lastFailureTime time.Time

	mu sync.RWMutex
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(metrics interfaces.MetricsCollector, logger interfaces.Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:    metrics,
		logger:     logger,
		minLatency: ^uint64(0), // Max uint64 value
	}
}

// TrackRequest tracks a request's performance metrics
func (pm *PerformanceMonitor) TrackRequest(ctx context.Context, operation string, fn func() error) error {
	start := time.Now()

	// Increment active goroutines
	active := atomic.AddInt64(&pm.activeGoroutines, 1)
	if active > atomic.LoadInt64(&pm.maxGoroutines) {
		atomic.StoreInt64(&pm.maxGoroutines, active)
	}

	defer func() {
		atomic.AddInt64(&pm.activeGoroutines, -1)

		// Track latency
		latency := time.Since(start)
		pm.recordLatency(latency)

		// Record metrics
		if pm.metrics != nil {
			pm.metrics.RecordTimer(operation+".duration", latency, map[string]string{
				"operation": operation,
			})
		}
	}()

	// Check circuit breaker
	if pm.isCircuitOpen() {
		atomic.AddUint64(&pm.errorCount, 1)
		if pm.metrics != nil {
			pm.metrics.IncrementCounter(operation+".circuit_breaker_open", map[string]string{
				"operation": operation,
			})
		}
		return NewCircuitBreakerError("circuit breaker is open")
	}

	// Execute the function
	err := fn()

	// Track results
	atomic.AddUint64(&pm.requestCount, 1)
	if err != nil {
		atomic.AddUint64(&pm.errorCount, 1)
		pm.recordFailure()

		if pm.metrics != nil {
			pm.metrics.IncrementCounter(operation+".errors", map[string]string{
				"operation": operation,
			})
		}
	} else {
		pm.recordSuccess()

		if pm.metrics != nil {
			pm.metrics.IncrementCounter(operation+".success", map[string]string{
				"operation": operation,
			})
		}
	}

	return err
}

// recordLatency records latency statistics
func (pm *PerformanceMonitor) recordLatency(latency time.Duration) {
	latencyNs := uint64(latency.Nanoseconds())

	atomic.AddUint64(&pm.totalLatency, latencyNs)

	// Update max latency
	for {
		current := atomic.LoadUint64(&pm.maxLatency)
		if latencyNs <= current {
			break
		}
		if atomic.CompareAndSwapUint64(&pm.maxLatency, current, latencyNs) {
			break
		}
	}

	// Update min latency
	for {
		current := atomic.LoadUint64(&pm.minLatency)
		if latencyNs >= current {
			break
		}
		if atomic.CompareAndSwapUint64(&pm.minLatency, current, latencyNs) {
			break
		}
	}
}

// recordFailure records a failure for circuit breaker logic
func (pm *PerformanceMonitor) recordFailure() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.failureCount++
	pm.lastFailureTime = time.Now()

	// Open circuit breaker if failure rate is too high
	if pm.failureCount >= 5 && time.Since(pm.lastFailureTime) < time.Minute {
		pm.circuitOpen = true
		if pm.logger != nil {
			pm.logger.Warn("Circuit breaker opened due to high failure rate")
		}
	}
}

// recordSuccess records a success for circuit breaker logic
func (pm *PerformanceMonitor) recordSuccess() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Reset failure count on success
	pm.failureCount = 0

	// Close circuit breaker if it was open
	if pm.circuitOpen {
		pm.circuitOpen = false
		if pm.logger != nil {
			pm.logger.Info("Circuit breaker closed after successful request")
		}
	}
}

// isCircuitOpen checks if the circuit breaker is open
func (pm *PerformanceMonitor) isCircuitOpen() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Auto-close circuit breaker after timeout
	if pm.circuitOpen && time.Since(pm.lastFailureTime) > 30*time.Second {
		pm.circuitOpen = false
		if pm.logger != nil {
			pm.logger.Info("Circuit breaker auto-closed after timeout")
		}
	}

	return pm.circuitOpen
}

// GetStatistics returns current performance statistics
func (pm *PerformanceMonitor) GetStatistics() *PerformanceStatistics {
	requestCount := atomic.LoadUint64(&pm.requestCount)
	errorCount := atomic.LoadUint64(&pm.errorCount)
	totalLatency := atomic.LoadUint64(&pm.totalLatency)
	maxLatency := atomic.LoadUint64(&pm.maxLatency)
	minLatency := atomic.LoadUint64(&pm.minLatency)
	activeGoroutines := atomic.LoadInt64(&pm.activeGoroutines)
	maxGoroutines := atomic.LoadInt64(&pm.maxGoroutines)

	stats := &PerformanceStatistics{
		RequestCount:     requestCount,
		ErrorCount:       errorCount,
		SuccessCount:     requestCount - errorCount,
		ActiveGoroutines: activeGoroutines,
		MaxGoroutines:    maxGoroutines,
		CircuitOpen:      pm.isCircuitOpen(),
	}

	if requestCount > 0 {
		stats.ErrorRate = float64(errorCount) / float64(requestCount)
		stats.AverageLatency = time.Duration(totalLatency / requestCount)
	}

	if maxLatency > 0 {
		stats.MaxLatency = time.Duration(maxLatency)
	}

	if minLatency != ^uint64(0) {
		stats.MinLatency = time.Duration(minLatency)
	}

	return stats
}

// TrackMemoryUsage tracks memory usage statistics
func (pm *PerformanceMonitor) TrackMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if pm.metrics != nil {
		pm.metrics.RecordGauge("memory.alloc", float64(m.Alloc), nil)
		pm.metrics.RecordGauge("memory.total_alloc", float64(m.TotalAlloc), nil)
		pm.metrics.RecordGauge("memory.sys", float64(m.Sys), nil)
		pm.metrics.RecordGauge("memory.num_gc", float64(m.NumGC), nil)
		pm.metrics.RecordGauge("memory.gc_cpu_fraction", m.GCCPUFraction, nil)
	}

	// Track allocations
	atomic.StoreUint64(&pm.allocations, m.Mallocs)
	atomic.StoreUint64(&pm.deallocations, m.Frees)
	atomic.StoreUint64(&pm.gcCount, uint64(m.NumGC))
}

// PerformanceStatistics contains performance statistics
type PerformanceStatistics struct {
	RequestCount     uint64        `json:"request_count"`
	ErrorCount       uint64        `json:"error_count"`
	SuccessCount     uint64        `json:"success_count"`
	ErrorRate        float64       `json:"error_rate"`
	AverageLatency   time.Duration `json:"average_latency"`
	MinLatency       time.Duration `json:"min_latency"`
	MaxLatency       time.Duration `json:"max_latency"`
	ActiveGoroutines int64         `json:"active_goroutines"`
	MaxGoroutines    int64         `json:"max_goroutines"`
	CircuitOpen      bool          `json:"circuit_open"`
}

// CircuitBreakerError represents a circuit breaker error
type CircuitBreakerError struct {
	message string
}

// NewCircuitBreakerError creates a new circuit breaker error
func NewCircuitBreakerError(message string) *CircuitBreakerError {
	return &CircuitBreakerError{message: message}
}

// Error implements the error interface
func (e *CircuitBreakerError) Error() string {
	return e.message
}

// ObjectPool provides a generic object pool for performance optimization
type ObjectPool[T any] struct {
	pool sync.Pool
	new  func() T
}

// NewObjectPool creates a new object pool
func NewObjectPool[T any](newFunc func() T) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
		new: newFunc,
	}
}

// Get gets an object from the pool
func (p *ObjectPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns an object to the pool
func (p *ObjectPool[T]) Put(obj T) {
	p.pool.Put(obj)
}

// BatchProcessor provides efficient batch processing capabilities
type BatchProcessor[T any] struct {
	batchSize    int
	flushTimeout time.Duration
	processor    func([]T) error

	buffer []T
	mu     sync.Mutex
	timer  *time.Timer
	stopCh chan struct{}
	doneCh chan struct{}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor[T any](batchSize int, flushTimeout time.Duration, processor func([]T) error) *BatchProcessor[T] {
	bp := &BatchProcessor[T]{
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		processor:    processor,
		buffer:       make([]T, 0, batchSize),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	go bp.flushLoop()
	return bp
}

// Add adds an item to the batch
func (bp *BatchProcessor[T]) Add(item T) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.buffer = append(bp.buffer, item)

	// Reset timer
	if bp.timer != nil {
		bp.timer.Stop()
	}
	bp.timer = time.AfterFunc(bp.flushTimeout, bp.flush)

	// Flush if batch is full
	if len(bp.buffer) >= bp.batchSize {
		return bp.flushLocked()
	}

	return nil
}

// Flush flushes the current batch
func (bp *BatchProcessor[T]) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.flushLocked()
}

// flushLocked flushes the current batch (must be called with lock held)
func (bp *BatchProcessor[T]) flushLocked() error {
	if len(bp.buffer) == 0 {
		return nil
	}

	// Stop timer
	if bp.timer != nil {
		bp.timer.Stop()
		bp.timer = nil
	}

	// Process batch
	batch := make([]T, len(bp.buffer))
	copy(batch, bp.buffer)
	bp.buffer = bp.buffer[:0] // Reset buffer

	return bp.processor(batch)
}

// flush is called by the timer
func (bp *BatchProcessor[T]) flush() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.flushLocked()
}

// flushLoop runs the periodic flush loop
func (bp *BatchProcessor[T]) flushLoop() {
	defer close(bp.doneCh)

	ticker := time.NewTicker(bp.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-bp.stopCh:
			bp.Flush() // Final flush
			return
		case <-ticker.C:
			bp.Flush()
		}
	}
}

// Stop stops the batch processor
func (bp *BatchProcessor[T]) Stop() {
	close(bp.stopCh)
	<-bp.doneCh
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	rate       int
	capacity   int
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, capacity int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Refill tokens
	tokensToAdd := int(elapsed.Seconds()) * rl.rate
	if tokensToAdd > 0 {
		rl.tokens = min(rl.capacity, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}

	// Check if request is allowed
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
