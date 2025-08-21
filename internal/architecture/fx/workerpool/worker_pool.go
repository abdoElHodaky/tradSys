package workerpool

import (
	"context"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WorkerPoolModule provides the optimized worker pool components
// following Fx's modular design pattern
var WorkerPoolModule = fx.Options(
	// Provide the worker pool factory
	fx.Provide(NewWorkerPoolFactory),
	
	// Provide the default worker pool configuration
	fx.Provide(NewDefaultWorkerPoolConfig),
	
	// Provide the worker pool metrics collector
	fx.Provide(NewWorkerPoolMetrics),
	
	// Register lifecycle hooks
	fx.Invoke(registerWorkerPoolHooks),
)

// WorkerPoolConfig contains configuration for worker pools
type WorkerPoolConfig struct {
	// DefaultSize is the default number of workers in the pool
	DefaultSize int
	
	// DefaultQueueSize is the default size of the task queue
	DefaultQueueSize int
	
	// ExpiryDuration is the duration after which idle workers are cleaned up
	ExpiryDuration time.Duration
	
	// PreAlloc determines whether to pre-allocate memory for workers
	PreAlloc bool
	
	// MaxBlockingTasks is the maximum number of tasks that can be blocked on submission
	// 0 means no limit
	MaxBlockingTasks int
	
	// Nonblocking determines whether to return immediately when submitting a task
	// to a full pool
	Nonblocking bool
	
	// PanicHandler is called when a panic occurs in a worker
	PanicHandler func(interface{})
}

// NewDefaultWorkerPoolConfig returns the default worker pool configuration
// This follows Fx's pattern of providing default configurations as injectable dependencies
func NewDefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		DefaultSize:      10,
		DefaultQueueSize: 100,
		ExpiryDuration:   time.Minute,
		PreAlloc:         false,
		MaxBlockingTasks: 0,
		Nonblocking:      false,
		PanicHandler: func(i interface{}) {
			// Default panic handler just recovers silently
		},
	}
}

// WorkerPoolMetrics collects metrics for worker pools
// Following Fx's pattern of separating concerns
type WorkerPoolMetrics struct {
	logger *zap.Logger
	mu     sync.RWMutex
	stats  map[string]*PoolStats
}

// PoolStats contains statistics for a worker pool
type PoolStats struct {
	Name          string
	RunningWorkers int
	FreeWorkers    int
	Capacity       int
	TasksSubmitted int64
	TasksCompleted int64
	TasksFailed    int64
}

// NewWorkerPoolMetrics creates a new worker pool metrics collector
func NewWorkerPoolMetrics(logger *zap.Logger) *WorkerPoolMetrics {
	return &WorkerPoolMetrics{
		logger: logger,
		stats:  make(map[string]*PoolStats),
	}
}

// RecordTaskSubmitted records a task submission for a worker pool
func (m *WorkerPoolMetrics) RecordTaskSubmitted(name string) {
	m.mu.RLock()
	stats, ok := m.stats[name]
	m.mu.RUnlock()
	
	if !ok {
		m.mu.Lock()
		stats, ok = m.stats[name]
		if !ok {
			stats = &PoolStats{Name: name}
			m.stats[name] = stats
		}
		m.mu.Unlock()
	}
	
	stats.TasksSubmitted++
}

// RecordTaskCompleted records a task completion for a worker pool
func (m *WorkerPoolMetrics) RecordTaskCompleted(name string) {
	m.mu.RLock()
	stats, ok := m.stats[name]
	m.mu.RUnlock()
	
	if !ok {
		return
	}
	
	stats.TasksCompleted++
}

// RecordTaskFailed records a task failure for a worker pool
func (m *WorkerPoolMetrics) RecordTaskFailed(name string, err error) {
	m.mu.RLock()
	stats, ok := m.stats[name]
	m.mu.RUnlock()
	
	if !ok {
		return
	}
	
	stats.TasksFailed++
	
	m.logger.Debug("Worker pool task failed",
		zap.String("pool", name),
		zap.Error(err))
}

// UpdatePoolStats updates the statistics for a worker pool
func (m *WorkerPoolMetrics) UpdatePoolStats(name string, pool *ants.Pool) {
	m.mu.RLock()
	stats, ok := m.stats[name]
	m.mu.RUnlock()
	
	if !ok {
		m.mu.Lock()
		stats, ok = m.stats[name]
		if !ok {
			stats = &PoolStats{Name: name}
			m.stats[name] = stats
		}
		m.mu.Unlock()
	}
	
	stats.RunningWorkers = pool.Running()
	stats.FreeWorkers = pool.Free()
	stats.Capacity = pool.Cap()
}

// GetAllStats returns statistics for all worker pools
func (m *WorkerPoolMetrics) GetAllStats() map[string]*PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	result := make(map[string]*PoolStats, len(m.stats))
	for name, stats := range m.stats {
		statsCopy := *stats
		result[name] = &statsCopy
	}
	
	return result
}

// WorkerPoolFactory creates and manages worker pools
// This is the main service that will be injected into other components
type WorkerPoolFactory struct {
	logger   *zap.Logger
	metrics  *WorkerPoolMetrics
	config   *WorkerPoolConfig
	pools    map[string]*ants.Pool
	poolsMu  sync.RWMutex
}

// NewWorkerPoolFactory creates a new worker pool factory
// Following Fx's dependency injection pattern
func NewWorkerPoolFactory(
	logger *zap.Logger,
	metrics *WorkerPoolMetrics,
	config *WorkerPoolConfig,
) *WorkerPoolFactory {
	return &WorkerPoolFactory{
		logger:  logger,
		metrics: metrics,
		config:  config,
		pools:   make(map[string]*ants.Pool),
	}
}

// CreateWorkerPool creates a new worker pool with the given name and default configuration
func (f *WorkerPoolFactory) CreateWorkerPool(name string) (*ants.Pool, error) {
	// Check if pool already exists
	f.poolsMu.RLock()
	pool, exists := f.pools[name]
	f.poolsMu.RUnlock()
	
	if exists {
		return pool, nil
	}
	
	// Create options from default config
	options := ants.Options{
		ExpiryDuration: f.config.ExpiryDuration,
		PreAlloc:       f.config.PreAlloc,
		MaxBlockingTasks: f.config.MaxBlockingTasks,
		Nonblocking:    f.config.Nonblocking,
		PanicHandler:   f.config.PanicHandler,
		Logger:         f.logger,
	}
	
	// Create the pool
	pool, err := ants.NewPool(f.config.DefaultSize, ants.WithOptions(options))
	if err != nil {
		f.logger.Error("Failed to create worker pool",
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}
	
	// Store the pool
	f.poolsMu.Lock()
	f.pools[name] = pool
	f.poolsMu.Unlock()
	
	f.logger.Info("Created worker pool",
		zap.String("name", name),
		zap.Int("size", f.config.DefaultSize))
	
	return pool, nil
}

// CreateCustomWorkerPool creates a new worker pool with custom options
func (f *WorkerPoolFactory) CreateCustomWorkerPool(name string, size int, options ants.Options) (*ants.Pool, error) {
	// Check if pool already exists
	f.poolsMu.RLock()
	pool, exists := f.pools[name]
	f.poolsMu.RUnlock()
	
	if exists {
		return pool, nil
	}
	
	// Create the pool
	pool, err := ants.NewPool(size, ants.WithOptions(options))
	if err != nil {
		f.logger.Error("Failed to create custom worker pool",
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}
	
	// Store the pool
	f.poolsMu.Lock()
	f.pools[name] = pool
	f.poolsMu.Unlock()
	
	f.logger.Info("Created custom worker pool",
		zap.String("name", name),
		zap.Int("size", size))
	
	return pool, nil
}

// GetWorkerPool gets a worker pool by name
func (f *WorkerPoolFactory) GetWorkerPool(name string) (*ants.Pool, bool) {
	f.poolsMu.RLock()
	defer f.poolsMu.RUnlock()
	
	pool, ok := f.pools[name]
	return pool, ok
}

// GetOrCreateWorkerPool gets a worker pool by name or creates it if it doesn't exist
func (f *WorkerPoolFactory) GetOrCreateWorkerPool(name string) (*ants.Pool, error) {
	// Check if pool already exists
	f.poolsMu.RLock()
	pool, exists := f.pools[name]
	f.poolsMu.RUnlock()
	
	if exists {
		return pool, nil
	}
	
	// Create a new pool
	return f.CreateWorkerPool(name)
}

// Submit submits a task to the specified worker pool
func (f *WorkerPoolFactory) Submit(poolName string, task func()) error {
	// Get or create the pool
	pool, err := f.GetOrCreateWorkerPool(poolName)
	if err != nil {
		return err
	}
	
	// Record metrics
	f.metrics.RecordTaskSubmitted(poolName)
	
	// Submit the task
	return pool.Submit(func() {
		defer func() {
			if r := recover(); r != nil {
				if f.config.PanicHandler != nil {
					f.config.PanicHandler(r)
				}
				f.metrics.RecordTaskFailed(poolName, nil)
			} else {
				f.metrics.RecordTaskCompleted(poolName)
			}
		}()
		
		task()
	})
}

// SubmitTask submits a task that returns an error to the specified worker pool
func (f *WorkerPoolFactory) SubmitTask(poolName string, task func() error) error {
	// Get or create the pool
	pool, err := f.GetOrCreateWorkerPool(poolName)
	if err != nil {
		return err
	}
	
	// Record metrics
	f.metrics.RecordTaskSubmitted(poolName)
	
	// Submit the task
	return pool.Submit(func() {
		defer func() {
			if r := recover(); r != nil {
				if f.config.PanicHandler != nil {
					f.config.PanicHandler(r)
				}
				f.metrics.RecordTaskFailed(poolName, nil)
			}
		}()
		
		err := task()
		if err != nil {
			f.metrics.RecordTaskFailed(poolName, err)
		} else {
			f.metrics.RecordTaskCompleted(poolName)
		}
	})
}

// Release releases a worker pool by name
func (f *WorkerPoolFactory) Release(name string) {
	f.poolsMu.Lock()
	defer f.poolsMu.Unlock()
	
	pool, ok := f.pools[name]
	if !ok {
		return
	}
	
	pool.Release()
	delete(f.pools, name)
	
	f.logger.Info("Released worker pool", zap.String("name", name))
}

// ReleaseAll releases all worker pools
func (f *WorkerPoolFactory) ReleaseAll() {
	f.poolsMu.Lock()
	defer f.poolsMu.Unlock()
	
	for name, pool := range f.pools {
		pool.Release()
		f.logger.Info("Released worker pool", zap.String("name", name))
	}
	
	f.pools = make(map[string]*ants.Pool)
}

// GetStats returns statistics for all worker pools
func (f *WorkerPoolFactory) GetStats() map[string]*PoolStats {
	f.poolsMu.RLock()
	defer f.poolsMu.RUnlock()
	
	// Update stats for all pools
	for name, pool := range f.pools {
		f.metrics.UpdatePoolStats(name, pool)
	}
	
	return f.metrics.GetAllStats()
}

// registerWorkerPoolHooks registers lifecycle hooks for the worker pool components
// This follows Fx's lifecycle management pattern
func registerWorkerPoolHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	factory *WorkerPoolFactory,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting worker pool components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping worker pool components")
			
			// Log statistics for all worker pools
			stats := factory.GetStats()
			for name, stat := range stats {
				logger.Info("Worker pool statistics",
					zap.String("name", name),
					zap.Int("running_workers", stat.RunningWorkers),
					zap.Int("free_workers", stat.FreeWorkers),
					zap.Int64("tasks_submitted", stat.TasksSubmitted),
					zap.Int64("tasks_completed", stat.TasksCompleted),
					zap.Int64("tasks_failed", stat.TasksFailed))
			}
			
			// Release all worker pools
			factory.ReleaseAll()
			
			return nil
		},
	})
}

