package adaptive_loader

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// WorkerPoolConfig contains configuration for the worker pool
type WorkerPoolConfig struct {
	// MaxWorkers is the maximum number of workers
	MaxWorkers int

	// MaxQueuedTasks is the maximum number of queued tasks
	MaxQueuedTasks int

	// WorkerIdleTimeout is the timeout for idle workers
	WorkerIdleTimeout time.Duration

	// TaskTimeout is the timeout for tasks
	TaskTimeout time.Duration
}

// DefaultWorkerPoolConfig returns the default worker pool configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		MaxWorkers:        10,
		MaxQueuedTasks:    100,
		WorkerIdleTimeout: 30 * time.Second,
		TaskTimeout:       5 * time.Minute,
	}
}

// Task represents a task to be executed by the worker pool
type Task struct {
	// Function to execute
	Func func() error

	// Context for cancellation
	Ctx context.Context

	// Timeout for the task
	Timeout time.Duration

	// Result channel
	Result chan<- error
}

// WorkerPool is a pool of workers for executing tasks
type WorkerPool struct {
	// Configuration
	config WorkerPoolConfig

	// Task queue
	tasks chan Task

	// Worker management
	workerCount int64
	running     int64
	stopCh      chan struct{}
	wg          sync.WaitGroup

	// Statistics
	completedTasks uint64
	failedTasks    uint64
	timeoutTasks   uint64
	totalTaskTime  int64

	// Logger
	logger *zap.Logger
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config WorkerPoolConfig, logger *zap.Logger) *WorkerPool {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &WorkerPool{
		config:        config,
		tasks:         make(chan Task, config.MaxQueuedTasks),
		stopCh:        make(chan struct{}),
		logger:        logger,
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() error {
	// Check if already running
	if !atomic.CompareAndSwapInt64(&p.running, 0, 1) {
		return fmt.Errorf("worker pool already running")
	}

	p.logger.Info("Starting worker pool",
		zap.Int("maxWorkers", p.config.MaxWorkers),
		zap.Int("maxQueuedTasks", p.config.MaxQueuedTasks),
	)

	// Start initial workers
	for i := 0; i < p.config.MaxWorkers; i++ {
		p.startWorker()
	}

	return nil
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	// Check if already stopped
	if !atomic.CompareAndSwapInt64(&p.running, 1, 0) {
		return
	}

	p.logger.Info("Stopping worker pool")

	// Signal all workers to stop
	close(p.stopCh)

	// Wait for all workers to finish
	p.wg.Wait()

	p.logger.Info("Worker pool stopped")
}

// Submit submits a task to the worker pool
func (p *WorkerPool) Submit(ctx context.Context, task func() error, timeout time.Duration) error {
	// Check if running
	if atomic.LoadInt64(&p.running) == 0 {
		return fmt.Errorf("worker pool not running")
	}

	// Create result channel
	resultCh := make(chan error, 1)

	// Create task
	t := Task{
		Func:    task,
		Ctx:     ctx,
		Timeout: timeout,
		Result:  resultCh,
	}

	// Submit task
	select {
	case p.tasks <- t:
		// Task submitted
	default:
		// Queue full
		return fmt.Errorf("task queue full")
	}

	// Wait for result
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SubmitAsync submits a task to the worker pool without waiting for the result
func (p *WorkerPool) SubmitAsync(ctx context.Context, task func() error, timeout time.Duration) error {
	// Check if running
	if atomic.LoadInt64(&p.running) == 0 {
		return fmt.Errorf("worker pool not running")
	}

	// Create task
	t := Task{
		Func:    task,
		Ctx:     ctx,
		Timeout: timeout,
		Result:  nil,
	}

	// Submit task
	select {
	case p.tasks <- t:
		// Task submitted
		return nil
	default:
		// Queue full
		return fmt.Errorf("task queue full")
	}
}

// startWorker starts a new worker
func (p *WorkerPool) startWorker() {
	p.wg.Add(1)
	atomic.AddInt64(&p.workerCount, 1)

	go func() {
		defer func() {
			atomic.AddInt64(&p.workerCount, -1)
			p.wg.Done()

			// Handle panics
			if r := recover(); r != nil {
				p.logger.Error("Worker panic",
					zap.Any("panic", r),
				)
			}
		}()

		p.logger.Debug("Worker started")

		// Worker loop
		for {
			select {
			case <-p.stopCh:
				// Worker pool stopped
				p.logger.Debug("Worker stopped")
				return
			case task := <-p.tasks:
				// Execute task
				p.executeTask(task)
			case <-time.After(p.config.WorkerIdleTimeout):
				// Worker idle timeout
				workerCount := atomic.LoadInt64(&p.workerCount)
				if workerCount > 1 {
					// Only exit if there are other workers
					p.logger.Debug("Worker idle timeout",
						zap.Int64("workerCount", workerCount),
					)
					return
				}
			}
		}
	}()
}

// executeTask executes a task
func (p *WorkerPool) executeTask(task Task) {
	// Create context with timeout
	var ctx context.Context
	var cancel context.CancelFunc

	if task.Timeout > 0 {
		ctx, cancel = context.WithTimeout(task.Ctx, task.Timeout)
	} else {
		ctx, cancel = context.WithTimeout(task.Ctx, p.config.TaskTimeout)
	}
	defer cancel()

	// Execute task in a goroutine
	resultCh := make(chan error, 1)
	startTime := time.Now()

	go func() {
		defer func() {
			// Handle panics
			if r := recover(); r != nil {
				err := fmt.Errorf("task panic: %v", r)
				p.logger.Error("Task panic",
					zap.Any("panic", r),
				)
				resultCh <- err
			}
		}()

		// Execute task
		resultCh <- task.Func()
	}()

	// Wait for result or timeout
	var err error
	select {
	case err = <-resultCh:
		// Task completed
		if err != nil {
			atomic.AddUint64(&p.failedTasks, 1)
			p.logger.Debug("Task failed",
				zap.Error(err),
				zap.Duration("duration", time.Since(startTime)),
			)
		} else {
			atomic.AddUint64(&p.completedTasks, 1)
			p.logger.Debug("Task completed",
				zap.Duration("duration", time.Since(startTime)),
			)
		}
	case <-ctx.Done():
		// Task timeout or context cancelled
		err = ctx.Err()
		atomic.AddUint64(&p.timeoutTasks, 1)
		p.logger.Debug("Task timeout",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)),
		)
	}

	// Update statistics
	atomic.AddInt64(&p.totalTaskTime, time.Since(startTime).Nanoseconds())

	// Send result if result channel is provided
	if task.Result != nil {
		task.Result <- err
	}

	// Start a new worker if needed
	workerCount := atomic.LoadInt64(&p.workerCount)
	if workerCount < int64(p.config.MaxWorkers) {
		p.startWorker()
	}
}

// GetStats gets the worker pool statistics
func (p *WorkerPool) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["running"] = atomic.LoadInt64(&p.running) == 1
	stats["workerCount"] = atomic.LoadInt64(&p.workerCount)
	stats["completedTasks"] = atomic.LoadUint64(&p.completedTasks)
	stats["failedTasks"] = atomic.LoadUint64(&p.failedTasks)
	stats["timeoutTasks"] = atomic.LoadUint64(&p.timeoutTasks)
	stats["totalTaskTime"] = time.Duration(atomic.LoadInt64(&p.totalTaskTime))
	stats["averageTaskTime"] = time.Duration(0)

	// Calculate average task time
	completedTasks := atomic.LoadUint64(&p.completedTasks)
	if completedTasks > 0 {
		stats["averageTaskTime"] = time.Duration(atomic.LoadInt64(&p.totalTaskTime)) / time.Duration(completedTasks)
	}

	return stats
}

// IsRunning returns whether the worker pool is running
func (p *WorkerPool) IsRunning() bool {
	return atomic.LoadInt64(&p.running) == 1
}

// GetWorkerCount returns the number of workers
func (p *WorkerPool) GetWorkerCount() int {
	return int(atomic.LoadInt64(&p.workerCount))
}

// GetCompletedTasks returns the number of completed tasks
func (p *WorkerPool) GetCompletedTasks() uint64 {
	return atomic.LoadUint64(&p.completedTasks)
}

// GetFailedTasks returns the number of failed tasks
func (p *WorkerPool) GetFailedTasks() uint64 {
	return atomic.LoadUint64(&p.failedTasks)
}

// GetTimeoutTasks returns the number of timeout tasks
func (p *WorkerPool) GetTimeoutTasks() uint64 {
	return atomic.LoadUint64(&p.timeoutTasks)
}

// GetTotalTaskTime returns the total task time
func (p *WorkerPool) GetTotalTaskTime() time.Duration {
	return time.Duration(atomic.LoadInt64(&p.totalTaskTime))
}

// GetAverageTaskTime returns the average task time
func (p *WorkerPool) GetAverageTaskTime() time.Duration {
	completedTasks := atomic.LoadUint64(&p.completedTasks)
	if completedTasks == 0 {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&p.totalTaskTime)) / time.Duration(completedTasks)
}
