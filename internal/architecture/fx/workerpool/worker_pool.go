package workerpool

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrPoolClosed      = errors.New("worker pool is closed")
	ErrPoolOverloaded  = errors.New("worker pool is overloaded")
	ErrTaskPanicked    = errors.New("task panicked during execution")
	ErrTaskTimeout     = errors.New("task timed out")
	ErrInvalidPoolSize = errors.New("invalid pool size")
)

// WorkerPoolFactory creates and manages worker pools
type WorkerPoolFactory struct {
	logger      *zap.Logger
	pools       map[string]*ants.Pool
	poolOptions map[string]*ants.Options
	metrics     *WorkerPoolMetrics
	mu          sync.RWMutex
}

// WorkerPoolParams contains parameters for creating a WorkerPoolFactory
type WorkerPoolParams struct {
	fx.In

	Logger *zap.Logger
}

// NewWorkerPoolFactory creates a new WorkerPoolFactory
func NewWorkerPoolFactory(params WorkerPoolParams) *WorkerPoolFactory {
	metrics := NewWorkerPoolMetrics()
	
	return &WorkerPoolFactory{
		logger:      params.Logger,
		pools:       make(map[string]*ants.Pool),
		poolOptions: make(map[string]*ants.Options),
		metrics:     metrics,
	}
}

// DefaultOptions returns the default worker pool options
func DefaultOptions() *ants.Options {
	return &ants.Options{
		ExpiryDuration: 10 * time.Minute,
		PreAlloc:       true,
		MaxBlockingTasks: 1000,
		Nonblocking:    false,
		PanicHandler: func(i interface{}) {
			// Panic handler will be set in GetWorkerPool
		},
	}
}

// GetWorkerPool gets or creates a worker pool with the given name
func (f *WorkerPoolFactory) GetWorkerPool(name string, size int) (*ants.Pool, error) {
	if size <= 0 {
		return nil, ErrInvalidPoolSize
	}
	
	f.mu.RLock()
	pool, exists := f.pools[name]
	f.mu.RUnlock()
	
	if exists {
		return pool, nil
	}
	
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	if pool, exists = f.pools[name]; exists {
		return pool, nil
	}
	
	// Create a new worker pool with default options
	options := DefaultOptions()
	
	// Set panic handler to record metrics
	options.PanicHandler = func(i interface{}) {
		f.logger.Error("Worker pool task panicked",
			zap.String("pool", name),
			zap.Any("panic", i))
		
		f.metrics.RecordPanic(name)
	}
	
	pool, err := ants.NewPool(size, ants.WithOptions(*options))
	if err != nil {
		return nil, err
	}
	
	f.pools[name] = pool
	f.poolOptions[name] = options
	
	f.logger.Info("Created worker pool",
		zap.String("name", name),
		zap.Int("size", size))
	
	return pool, nil
}

// GetWorkerPoolWithOptions gets or creates a worker pool with custom options
func (f *WorkerPoolFactory) GetWorkerPoolWithOptions(name string, size int, options *ants.Options) (*ants.Pool, error) {
	if size <= 0 {
		return nil, ErrInvalidPoolSize
	}
	
	f.mu.RLock()
	pool, exists := f.pools[name]
	f.mu.RUnlock()
	
	if exists {
		// Check if options have changed
		f.mu.RLock()
		currentOptions := f.poolOptions[name]
		f.mu.RUnlock()
		
		// If options are the same, return the existing pool
		if currentOptions.ExpiryDuration == options.ExpiryDuration &&
			currentOptions.PreAlloc == options.PreAlloc &&
			currentOptions.MaxBlockingTasks == options.MaxBlockingTasks &&
			currentOptions.Nonblocking == options.Nonblocking {
			return pool, nil
		}
		
		// Options have changed, release the old pool and create a new one
		f.mu.Lock()
		defer f.mu.Unlock()
		
		// Check again in case another goroutine released it while we were waiting for the lock
		if pool, exists = f.pools[name]; exists {
			pool.Release()
			delete(f.pools, name)
		}
	} else {
		f.mu.Lock()
		defer f.mu.Unlock()
	}
	
	// Set panic handler to record metrics
	if options.PanicHandler == nil {
		options.PanicHandler = func(i interface{}) {
			f.logger.Error("Worker pool task panicked",
				zap.String("pool", name),
				zap.Any("panic", i))
			
			f.metrics.RecordPanic(name)
		}
	}
	
	// Create a new worker pool with custom options
	pool, err := ants.NewPool(size, ants.WithOptions(*options))
	if err != nil {
		return nil, err
	}
	
	f.pools[name] = pool
	f.poolOptions[name] = options
	
	f.logger.Info("Created worker pool with custom options",
		zap.String("name", name),
		zap.Int("size", size))
	
	return pool, nil
}

// Submit submits a task to a worker pool
func (f *WorkerPoolFactory) Submit(poolName string, task func()) error {
	// Get or create a worker pool with default size (number of CPUs)
	pool, err := f.GetWorkerPool(poolName, ants.DefaultAntsPoolSize)
	if err != nil {
		return err
	}
	
	startTime := time.Now()
	
	// Submit the task to the pool
	err = pool.Submit(func() {
		defer func() {
			if rec := recover(); rec != nil {
				f.logger.Error("Task panicked",
					zap.String("pool", poolName),
					zap.Any("panic", rec))
				
				f.metrics.RecordPanic(poolName)
			}
			
			f.metrics.RecordExecution(poolName, err == nil, time.Since(startTime))
		}()
		
		task()
	})
	
	if err != nil {
		if errors.Is(err, ants.ErrPoolClosed) {
			return ErrPoolClosed
		}
		if errors.Is(err, ants.ErrPoolOverload) {
			f.metrics.RecordRejection(poolName)
			return ErrPoolOverloaded
		}
		return err
	}
	
	return nil
}

// SubmitTask submits a task that returns an error to a worker pool
func (f *WorkerPoolFactory) SubmitTask(poolName string, task func() error) error {
	// Get or create a worker pool with default size (number of CPUs)
	pool, err := f.GetWorkerPool(poolName, ants.DefaultAntsPoolSize)
	if err != nil {
		return err
	}
	
	startTime := time.Now()
	
	// Submit the task to the pool
	err = pool.Submit(func() {
		defer func() {
			if rec := recover(); rec != nil {
				f.logger.Error("Task panicked",
					zap.String("pool", poolName),
					zap.Any("panic", rec))
				
				f.metrics.RecordPanic(poolName)
			}
			
			f.metrics.RecordExecution(poolName, err == nil, time.Since(startTime))
		}()
		
		taskErr := task()
		if taskErr != nil {
			f.logger.Error("Task failed",
				zap.String("pool", poolName),
				zap.Error(taskErr))
			
			f.metrics.RecordFailure(poolName)
		}
	})
	
	if err != nil {
		if errors.Is(err, ants.ErrPoolClosed) {
			return ErrPoolClosed
		}
		if errors.Is(err, ants.ErrPoolOverload) {
			f.metrics.RecordRejection(poolName)
			return ErrPoolOverloaded
		}
		return err
	}
	
	return nil
}

// SubmitWithTimeout submits a task with a timeout to a worker pool
func (f *WorkerPoolFactory) SubmitWithTimeout(poolName string, task func(), timeout time.Duration) error {
	// Get or create a worker pool with default size (number of CPUs)
	pool, err := f.GetWorkerPool(poolName, ants.DefaultAntsPoolSize)
	if err != nil {
		return err
	}
	
	startTime := time.Now()
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Create a channel to signal task completion
	done := make(chan struct{})
	
	// Submit the task to the pool
	err = pool.Submit(func() {
		defer func() {
			if rec := recover(); rec != nil {
				f.logger.Error("Task panicked",
					zap.String("pool", poolName),
					zap.Any("panic", rec))
				
				f.metrics.RecordPanic(poolName)
			}
			
			close(done)
		}()
		
		task()
	})
	
	if err != nil {
		if errors.Is(err, ants.ErrPoolClosed) {
			return ErrPoolClosed
		}
		if errors.Is(err, ants.ErrPoolOverload) {
			f.metrics.RecordRejection(poolName)
			return ErrPoolOverloaded
		}
		return err
	}
	
	// Wait for task completion or timeout
	select {
	case <-done:
		f.metrics.RecordExecution(poolName, true, time.Since(startTime))
		return nil
	case <-ctx.Done():
		f.metrics.RecordTimeout(poolName)
		return ErrTaskTimeout
	}
}

// ReleasePool releases a worker pool
func (f *WorkerPoolFactory) ReleasePool(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if pool, exists := f.pools[name]; exists {
		pool.Release()
		delete(f.pools, name)
		delete(f.poolOptions, name)
		
		f.logger.Info("Released worker pool", zap.String("name", name))
	}
}

// GetPoolStats returns statistics for a worker pool
func (f *WorkerPoolFactory) GetPoolStats(name string) (running int, capacity int, ok bool) {
	f.mu.RLock()
	pool, exists := f.pools[name]
	f.mu.RUnlock()
	
	if !exists {
		return 0, 0, false
	}
	
	return pool.Running(), pool.Cap(), true
}

// GetMetrics returns the worker pool metrics
func (f *WorkerPoolFactory) GetMetrics() *WorkerPoolMetrics {
	return f.metrics
}

// Release releases all worker pools
func (f *WorkerPoolFactory) Release() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	for name, pool := range f.pools {
		pool.Release()
		f.logger.Info("Released worker pool", zap.String("name", name))
	}
	
	f.pools = make(map[string]*ants.Pool)
	f.poolOptions = make(map[string]*ants.Options)
}

// WorkerPoolMetrics collects metrics for worker pools
type WorkerPoolMetrics struct {
	mu sync.RWMutex
	
	// Execution metrics
	executions map[string]int64
	successes  map[string]int64
	failures   map[string]int64
	rejections map[string]int64
	timeouts   map[string]int64
	panics     map[string]int64
	
	// Latency metrics
	executionTimes map[string][]time.Duration
}

// NewWorkerPoolMetrics creates a new WorkerPoolMetrics
func NewWorkerPoolMetrics() *WorkerPoolMetrics {
	return &WorkerPoolMetrics{
		executions:     make(map[string]int64),
		successes:      make(map[string]int64),
		failures:       make(map[string]int64),
		rejections:     make(map[string]int64),
		timeouts:       make(map[string]int64),
		panics:         make(map[string]int64),
		executionTimes: make(map[string][]time.Duration),
	}
}

// RecordExecution records an execution of a worker pool task
func (m *WorkerPoolMetrics) RecordExecution(poolName string, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executions[poolName]++
	if success {
		m.successes[poolName]++
	} else {
		m.failures[poolName]++
	}
	
	if _, ok := m.executionTimes[poolName]; !ok {
		m.executionTimes[poolName] = make([]time.Duration, 0, 100)
	}
	
	m.executionTimes[poolName] = append(m.executionTimes[poolName], duration)
	
	// Keep only the last 100 execution times
	if len(m.executionTimes[poolName]) > 100 {
		m.executionTimes[poolName] = m.executionTimes[poolName][1:]
	}
}

// RecordFailure records a failure of a worker pool task
func (m *WorkerPoolMetrics) RecordFailure(poolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.failures[poolName]++
}

// RecordRejection records a rejection of a worker pool task
func (m *WorkerPoolMetrics) RecordRejection(poolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.rejections[poolName]++
}

// RecordTimeout records a timeout of a worker pool task
func (m *WorkerPoolMetrics) RecordTimeout(poolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.timeouts[poolName]++
}

// RecordPanic records a panic of a worker pool task
func (m *WorkerPoolMetrics) RecordPanic(poolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.panics[poolName]++
}

// GetExecutionCount returns the number of executions for a worker pool
func (m *WorkerPoolMetrics) GetExecutionCount(poolName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.executions[poolName]
}

// GetSuccessCount returns the number of successful executions for a worker pool
func (m *WorkerPoolMetrics) GetSuccessCount(poolName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.successes[poolName]
}

// GetFailureCount returns the number of failed executions for a worker pool
func (m *WorkerPoolMetrics) GetFailureCount(poolName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.failures[poolName]
}

// GetRejectionCount returns the number of rejected tasks for a worker pool
func (m *WorkerPoolMetrics) GetRejectionCount(poolName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.rejections[poolName]
}

// GetTimeoutCount returns the number of timed out tasks for a worker pool
func (m *WorkerPoolMetrics) GetTimeoutCount(poolName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.timeouts[poolName]
}

// GetPanicCount returns the number of panicked tasks for a worker pool
func (m *WorkerPoolMetrics) GetPanicCount(poolName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.panics[poolName]
}

// GetSuccessRate returns the success rate for a worker pool
func (m *WorkerPoolMetrics) GetSuccessRate(poolName string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	executions := m.executions[poolName]
	if executions == 0 {
		return 0
	}
	
	return float64(m.successes[poolName]) / float64(executions)
}

// GetAverageExecutionTime returns the average execution time for a worker pool
func (m *WorkerPoolMetrics) GetAverageExecutionTime(poolName string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	times, ok := m.executionTimes[poolName]
	if !ok || len(times) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	
	return sum / time.Duration(len(times))
}

// Reset resets all metrics
func (m *WorkerPoolMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executions = make(map[string]int64)
	m.successes = make(map[string]int64)
	m.failures = make(map[string]int64)
	m.rejections = make(map[string]int64)
	m.timeouts = make(map[string]int64)
	m.panics = make(map[string]int64)
	m.executionTimes = make(map[string][]time.Duration)
}
