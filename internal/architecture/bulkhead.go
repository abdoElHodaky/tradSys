package architecture

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// Bulkhead implements the bulkhead pattern to isolate failures and
// prevent cascading failures in distributed systems
type Bulkhead struct {
	name            string
	maxConcurrency  int64
	maxWaitingQueue int64
	activeCalls     int64 // atomic
	waitingCalls    int64 // atomic
	semaphore       chan struct{}
	waitingQueue    chan struct{}
	mu              sync.RWMutex
}

// BulkheadOptions contains options for creating a bulkhead
type BulkheadOptions struct {
	Name            string
	MaxConcurrency  int64
	MaxWaitingQueue int64
}

// NewBulkhead creates a new bulkhead
func NewBulkhead(options BulkheadOptions) *Bulkhead {
	if options.MaxConcurrency <= 0 {
		options.MaxConcurrency = 10
	}
	if options.MaxWaitingQueue <= 0 {
		options.MaxWaitingQueue = 100
	}

	return &Bulkhead{
		name:            options.Name,
		maxConcurrency:  options.MaxConcurrency,
		maxWaitingQueue: options.MaxWaitingQueue,
		semaphore:       make(chan struct{}, options.MaxConcurrency),
		waitingQueue:    make(chan struct{}, options.MaxWaitingQueue),
	}
}

// Execute executes the given function within the bulkhead
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	// Try to enter the waiting queue
	if atomic.LoadInt64(&b.waitingCalls) >= b.maxWaitingQueue {
		return errors.New("bulkhead waiting queue is full")
	}

	// Increment waiting calls counter
	atomic.AddInt64(&b.waitingCalls, 1)

	// Add to waiting queue
	select {
	case b.waitingQueue <- struct{}{}:
		// Successfully entered waiting queue
	default:
		// Waiting queue is full (this should not happen due to the check above)
		atomic.AddInt64(&b.waitingCalls, -1)
		return errors.New("bulkhead waiting queue is full")
	}

	// Decrement waiting calls counter when we leave the waiting queue
	defer func() {
		<-b.waitingQueue
		atomic.AddInt64(&b.waitingCalls, -1)
	}()

	// Try to acquire a semaphore
	select {
	case b.semaphore <- struct{}{}:
		// Successfully acquired semaphore
	case <-ctx.Done():
		return errors.New("context cancelled while waiting for bulkhead")
	}

	// Increment active calls counter
	atomic.AddInt64(&b.activeCalls, 1)

	// Release semaphore when done
	defer func() {
		<-b.semaphore
		atomic.AddInt64(&b.activeCalls, -1)
	}()

	// Execute the function
	return fn()
}

// ActiveCalls returns the current number of active calls
func (b *Bulkhead) ActiveCalls() int64 {
	return atomic.LoadInt64(&b.activeCalls)
}

// WaitingCalls returns the current number of waiting calls
func (b *Bulkhead) WaitingCalls() int64 {
	return atomic.LoadInt64(&b.waitingCalls)
}

// MaxConcurrency returns the maximum number of concurrent calls
func (b *Bulkhead) MaxConcurrency() int64 {
	return b.maxConcurrency
}

// MaxWaitingQueue returns the maximum size of the waiting queue
func (b *Bulkhead) MaxWaitingQueue() int64 {
	return b.maxWaitingQueue
}
