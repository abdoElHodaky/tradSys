package architecture

import (
	"context"
	"sync"
	"sync/atomic"
)

// Task represents a task to be executed by the worker pool
type Task func() error

// WorkerPool implements a pool of workers for parallel task execution
type WorkerPool struct {
	name         string
	size         int
	taskQueue    chan Task
	wg           sync.WaitGroup
	activeWorkers int32 // atomic
	ctx          context.Context
	cancel       context.CancelFunc
	started      bool
	mu           sync.Mutex
}

// WorkerPoolOptions contains options for creating a worker pool
type WorkerPoolOptions struct {
	Name      string
	Size      int
	QueueSize int
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(options WorkerPoolOptions) *WorkerPool {
	if options.Size <= 0 {
		options.Size = 10
	}
	if options.QueueSize <= 0 {
		options.QueueSize = 100
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		name:      options.Name,
		size:      options.Size,
		taskQueue: make(chan Task, options.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if wp.started {
		return
	}
	
	wp.started = true
	
	// Start workers
	for i := 0; i < wp.size; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// Stop stops the worker pool and waits for all workers to finish
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if !wp.started {
		return
	}
	
	wp.cancel()
	close(wp.taskQueue)
	wp.wg.Wait()
	wp.started = false
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task Task) bool {
	select {
	case wp.taskQueue <- task:
		return true
	default:
		// Queue is full
		return false
	}
}

// SubmitWait submits a task to the worker pool and waits until it's accepted
func (wp *WorkerPool) SubmitWait(ctx context.Context, task Task) bool {
	select {
	case wp.taskQueue <- task:
		return true
	case <-ctx.Done():
		return false
	}
}

// worker is the main worker loop
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	
	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				// Channel closed, exit worker
				return
			}
			
			// Increment active workers count
			atomic.AddInt32(&wp.activeWorkers, 1)
			
			// Execute task
			_ = task()
			
			// Decrement active workers count
			atomic.AddInt32(&wp.activeWorkers, -1)
			
		case <-wp.ctx.Done():
			// Context cancelled, exit worker
			return
		}
	}
}

// ActiveWorkers returns the number of currently active workers
func (wp *WorkerPool) ActiveWorkers() int {
	return int(atomic.LoadInt32(&wp.activeWorkers))
}

// QueueSize returns the current size of the task queue
func (wp *WorkerPool) QueueSize() int {
	return len(wp.taskQueue)
}

// QueueCapacity returns the capacity of the task queue
func (wp *WorkerPool) QueueCapacity() int {
	return cap(wp.taskQueue)
}

// Size returns the size of the worker pool
func (wp *WorkerPool) Size() int {
	return wp.size
}

