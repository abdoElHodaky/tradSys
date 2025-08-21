package workerpool

import (
	"context"
	"errors"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

// This file provides example usage of the worker pool components
// It is not meant to be used in production, but rather to demonstrate
// how to use the worker pool components in a way that follows Fx benefits

// ExampleService demonstrates how to use the worker pool in a service
type ExampleService struct {
	logger      *zap.Logger
	workerPool  *WorkerPoolFactory
}

// NewExampleService creates a new example service
// This follows Fx's dependency injection pattern
func NewExampleService(
	logger *zap.Logger,
	workerPool *WorkerPoolFactory,
) *ExampleService {
	return &ExampleService{
		logger:     logger,
		workerPool: workerPool,
	}
}

// ProcessItems demonstrates how to use the worker pool to process items in parallel
func (s *ExampleService) ProcessItems(ctx context.Context, items []string) error {
	// Create a worker pool for this specific operation
	poolName := "process-items"
	
	// Create a custom worker pool with specific options if needed
	options := ants.Options{
		ExpiryDuration: time.Minute,
		PreAlloc:       true,
		PanicHandler: func(i interface{}) {
			s.logger.Error("Panic in worker",
				zap.Any("panic", i))
		},
	}
	
	pool, err := s.workerPool.CreateCustomWorkerPool(poolName, 20, options)
	if err != nil {
		return err
	}
	
	// Create a wait group to wait for all tasks to complete
	var errCount int32
	var completed = make(chan struct{})
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Process items in parallel
	go func() {
		for i, item := range items {
			// Capture loop variables
			i := i
			item := item
			
			// Submit task to worker pool
			err := s.workerPool.SubmitTask(poolName, func() error {
				// Check if context is cancelled
				if ctx.Err() != nil {
					return ctx.Err()
				}
				
				s.logger.Debug("Processing item",
					zap.Int("index", i),
					zap.String("item", item))
				
				// Simulate processing
				time.Sleep(100 * time.Millisecond)
				
				// Return success or error
				if i%10 == 0 {
					return errors.New("simulated error")
				}
				
				return nil
			})
			
			if err != nil {
				s.logger.Error("Failed to submit task",
					zap.Error(err))
			}
		}
		
		// Signal that all tasks have been submitted
		close(completed)
	}()
	
	// Wait for all tasks to complete or context to be cancelled
	select {
	case <-completed:
		// All tasks have been submitted
		s.logger.Info("All tasks submitted")
	case <-ctx.Done():
		// Context cancelled
		return ctx.Err()
	}
	
	// Get statistics
	stats := s.workerPool.GetStats()[poolName]
	if stats != nil {
		s.logger.Info("Processing completed",
			zap.Int64("submitted", stats.TasksSubmitted),
			zap.Int64("completed", stats.TasksCompleted),
			zap.Int64("failed", stats.TasksFailed))
	}
	
	return nil
}

// BatchProcessData demonstrates how to use the worker pool for batch processing
func (s *ExampleService) BatchProcessData(data [][]byte) error {
	// Use the default worker pool
	poolName := "batch-processor"
	
	// Process data in batches
	for i, batch := range data {
		// Capture loop variables
		i := i
		batch := batch
		
		// Submit task to worker pool
		err := s.workerPool.Submit(poolName, func() {
			s.logger.Debug("Processing batch",
				zap.Int("batch", i),
				zap.Int("size", len(batch)))
			
			// Simulate processing
			time.Sleep(50 * time.Millisecond)
		})
		
		if err != nil {
			s.logger.Error("Failed to submit batch",
				zap.Int("batch", i),
				zap.Error(err))
			return err
		}
	}
	
	return nil
}

// ScheduleTask demonstrates how to schedule a task with the worker pool
func (s *ExampleService) ScheduleTask(task func() error) error {
	// Use a dedicated worker pool for scheduled tasks
	poolName := "scheduler"
	
	// Submit the task
	return s.workerPool.SubmitTask(poolName, task)
}

