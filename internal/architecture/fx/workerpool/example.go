package workerpool

import (
	"errors"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

// ExampleUsage demonstrates how to use the worker pool
func ExampleUsage(logger *zap.Logger) {
	// Create a worker pool factory
	factory := NewWorkerPoolFactory(WorkerPoolParams{
		Logger: logger,
	})
	
	// Example 1: Basic usage
	err := factory.Submit("example", func() {
		// Simulate work
		time.Sleep(100 * time.Millisecond)
		logger.Info("Task completed")
	})
	
	if err != nil {
		logger.Error("Failed to submit task", zap.Error(err))
	}
	
	// Example 2: With error handling
	err = factory.SubmitTask("example-with-error", func() error {
		// Simulate work
		time.Sleep(100 * time.Millisecond)
		
		// Simulate an error
		return errors.New("task failed")
	})
	
	if err != nil {
		logger.Error("Failed to submit task", zap.Error(err))
	}
	
	// Example 3: With timeout
	err = factory.SubmitWithTimeout("example-with-timeout", func() {
		// Simulate work that takes longer than the timeout
		time.Sleep(200 * time.Millisecond)
		logger.Info("Task completed (but may have timed out)")
	}, 100*time.Millisecond)
	
	if err != nil {
		logger.Error("Task timed out", zap.Error(err))
	}
	
	// Example 4: With custom options
	options := ants.Options{
		ExpiryDuration: 5 * time.Minute,
		PreAlloc:       true,
		MaxBlockingTasks: 100,
		Nonblocking:    true,
	}
	
	pool, err := factory.GetWorkerPoolWithOptions("custom-example", 10, &options)
	if err != nil {
		logger.Error("Failed to create worker pool", zap.Error(err))
	} else {
		// Use the pool directly
		err = pool.Submit(func() {
			// Simulate work
			time.Sleep(100 * time.Millisecond)
			logger.Info("Custom pool task completed")
		})
		
		if err != nil {
			logger.Error("Failed to submit task to custom pool", zap.Error(err))
		}
	}
	
	// Example 5: Get metrics
	metrics := factory.GetMetrics()
	
	logger.Info("Worker pool metrics",
		zap.Int64("executions", metrics.GetExecutionCount("example")),
		zap.Int64("successes", metrics.GetSuccessCount("example")),
		zap.Int64("failures", metrics.GetFailureCount("example")),
		zap.Float64("success_rate", metrics.GetSuccessRate("example")),
		zap.Duration("avg_execution_time", metrics.GetAverageExecutionTime("example")))
	
	// Example 6: Get pool stats
	running, capacity, ok := factory.GetPoolStats("example")
	if ok {
		logger.Info("Worker pool stats",
			zap.String("name", "example"),
			zap.Int("running", running),
			zap.Int("capacity", capacity))
	}
	
	// Wait for tasks to complete
	time.Sleep(300 * time.Millisecond)
	
	// Release a specific pool
	factory.ReleasePool("example")
	
	// Release all pools
	factory.Release()
}

