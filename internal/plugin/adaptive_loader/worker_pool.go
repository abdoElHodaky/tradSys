package adaptive_loader

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Task represents a unit of work to be executed by the worker pool
type Task struct {
	// The function to execute
	Execute func() error
	
	// Task metadata
	Name        string
	Priority    int
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	
	// Error handling
	Error       error
	RetryCount  int
	MaxRetries  int
	RetryDelay  time.Duration
	
	// Context for cancellation
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTask creates a new task with the given name and execution function
func NewTask(name string, execute func() error) *Task {
	ctx, cancel := context.WithCancel(context.Background())
	return &Task{
		Name:       name,
		Execute:    execute,
		Priority:   0, // Default priority
		CreatedAt:  time.Now(),
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// WithPriority sets the priority of the task
func (t *Task) WithPriority(priority int) *Task {
	t.Priority = priority
	return t
}

// WithMaxRetries sets the maximum number of retries for the task
func (t *Task) WithMaxRetries(maxRetries int) *Task {
	t.MaxRetries = maxRetries
	return t
}

// WithRetryDelay sets the delay between retries for the task
func (t *Task) WithRetryDelay(delay time.Duration) *Task {
	t.RetryDelay = delay
	return t
}

// WithContext sets the context for the task
func (t *Task) WithContext(ctx context.Context) *Task {
	if t.cancel != nil {
		t.cancel()
	}
	t.ctx, t.cancel = context.WithCancel(ctx)
	return t
}

// Cancel cancels the task
func (t *Task) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// IsCancelled checks if the task has been cancelled
func (t *Task) IsCancelled() bool {
	select {
	case <-t.ctx.Done():
		return true
	default:
		return false
	}
}

// WorkerPool manages a pool of workers for executing tasks
type WorkerPool struct {
	// Configuration
	numWorkers      int
	queueSize       int
	
	// Task queue
	taskQueue       chan *Task
	priorityQueue   []*Task
	queueMu         sync.Mutex
	
	// Worker management
	workers         []*worker
	workerWg        sync.WaitGroup
	
	// Pool state
	running         int32
	stopCh          chan struct{}
	
	// Metrics
	completedTasks  int64
	failedTasks     int64
	
	// Logging
	logger          *zap.Logger
}

// worker represents a worker in the pool
type worker struct {
	id        int
	pool      *WorkerPool
	taskCh    chan *Task
	stopCh    chan struct{}
	logger    *zap.Logger
}

// NewWorkerPool creates a new worker pool with the given number of workers
func NewWorkerPool(numWorkers, queueSize int, logger *zap.Logger) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	
	if queueSize <= 0 {
		queueSize = numWorkers * 10
	}
	
	return &WorkerPool{
		numWorkers:     numWorkers,
		queueSize:      queueSize,
		taskQueue:      make(chan *Task, queueSize),
		priorityQueue:  make([]*Task, 0),
		workers:        make([]*worker, 0, numWorkers),
		stopCh:         make(chan struct{}),
		logger:         logger,
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		// Already running
		return
	}
	
	// Start workers
	for i := 0; i < p.numWorkers; i++ {
		w := &worker{
			id:     i,
			pool:   p,
			taskCh: make(chan *Task),
			stopCh: make(chan struct{}),
			logger: p.logger.With(zap.Int("worker_id", i)),
		}
		
		p.workers = append(p.workers, w)
		p.workerWg.Add(1)
		
		go w.start()
	}
	
	// Start task dispatcher
	go p.dispatch()
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		// Not running
		return
	}
	
	// Signal all workers to stop
	close(p.stopCh)
	
	// Wait for all workers to finish
	p.workerWg.Wait()
}

// Submit submits a task to the worker pool
func (p *WorkerPool) Submit(task *Task) error {
	if atomic.LoadInt32(&p.running) == 0 {
		return fmt.Errorf("worker pool is not running")
	}
	
	// Check if the task has a high priority
	if task.Priority > 0 {
		// Add to priority queue
		p.queueMu.Lock()
		p.priorityQueue = append(p.priorityQueue, task)
		// Sort by priority (higher first)
		p.sortPriorityQueue()
		p.queueMu.Unlock()
		return nil
	}
	
	// Try to add to regular queue
	select {
	case p.taskQueue <- task:
		return nil
	default:
		// Queue is full
		return fmt.Errorf("task queue is full")
	}
}

// sortPriorityQueue sorts the priority queue by priority (higher first)
func (p *WorkerPool) sortPriorityQueue() {
	// Simple insertion sort for small queues
	for i := 1; i < len(p.priorityQueue); i++ {
		task := p.priorityQueue[i]
		j := i - 1
		for j >= 0 && p.priorityQueue[j].Priority < task.Priority {
			p.priorityQueue[j+1] = p.priorityQueue[j]
			j--
		}
		p.priorityQueue[j+1] = task
	}
}

// dispatch dispatches tasks to workers
func (p *WorkerPool) dispatch() {
	for {
		var task *Task
		
		// First check priority queue
		p.queueMu.Lock()
		if len(p.priorityQueue) > 0 {
			task = p.priorityQueue[0]
			p.priorityQueue = p.priorityQueue[1:]
			p.queueMu.Unlock()
		} else {
			p.queueMu.Unlock()
			
			// Then check regular queue
			select {
			case task = <-p.taskQueue:
				// Got a task
			case <-p.stopCh:
				// Pool is stopping
				return
			}
		}
		
		// Find an available worker
		dispatched := false
		for _, w := range p.workers {
			select {
			case w.taskCh <- task:
				dispatched = true
				break
			default:
				// Worker is busy, try next one
			}
		}
		
		// If no worker is available, wait for one
		if !dispatched {
			// Pick a random worker and wait for it
			w := p.workers[0]
			select {
			case w.taskCh <- task:
				// Task dispatched
			case <-p.stopCh:
				// Pool is stopping
				return
			}
		}
	}
}

// GetStats returns statistics about the worker pool
func (p *WorkerPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"num_workers":     p.numWorkers,
		"queue_size":      p.queueSize,
		"queue_length":    len(p.taskQueue),
		"priority_queue":  len(p.priorityQueue),
		"completed_tasks": atomic.LoadInt64(&p.completedTasks),
		"failed_tasks":    atomic.LoadInt64(&p.failedTasks),
		"running":         atomic.LoadInt32(&p.running) == 1,
	}
}

// start starts the worker
func (w *worker) start() {
	defer w.pool.workerWg.Done()
	
	w.logger.Debug("Worker started")
	
	for {
		select {
		case task := <-w.taskCh:
			// Execute the task
			w.executeTask(task)
		case <-w.stopCh:
			// Worker is stopping
			w.logger.Debug("Worker stopped")
			return
		case <-w.pool.stopCh:
			// Pool is stopping
			w.logger.Debug("Worker stopped (pool stopping)")
			return
		}
	}
}

// executeTask executes a task with retry logic
func (w *worker) executeTask(task *Task) {
	// Check if the task has been cancelled
	if task.IsCancelled() {
		w.logger.Debug("Task cancelled before execution",
			zap.String("task", task.Name))
		return
	}
	
	// Set start time
	task.StartedAt = time.Now()
	
	// Execute the task with retries
	var err error
	for attempt := 0; attempt <= task.MaxRetries; attempt++ {
		if attempt > 0 {
			// Log retry attempt
			w.logger.Debug("Retrying task",
				zap.String("task", task.Name),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", task.MaxRetries))
			
			// Wait before retrying
			select {
			case <-task.ctx.Done():
				// Task cancelled
				task.Error = fmt.Errorf("task cancelled during retry wait: %w", task.ctx.Err())
				return
			case <-time.After(task.RetryDelay * time.Duration(attempt)):
				// Continue with retry
			}
		}
		
		// Execute the task
		err = task.Execute()
		if err == nil {
			// Task completed successfully
			break
		}
		
		// Task failed
		task.Error = err
		
		// Check if we should retry
		if attempt >= task.MaxRetries {
			// No more retries
			w.logger.Warn("Task failed after max retries",
				zap.String("task", task.Name),
				zap.Error(err),
				zap.Int("max_retries", task.MaxRetries))
			
			// Update metrics
			atomic.AddInt64(&w.pool.failedTasks, 1)
			return
		}
	}
	
	// Set completion time
	task.CompletedAt = time.Now()
	
	// Update metrics
	if err == nil {
		atomic.AddInt64(&w.pool.completedTasks, 1)
	} else {
		atomic.AddInt64(&w.pool.failedTasks, 1)
	}
}
