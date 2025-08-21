# Worker Pool for Go with Fx Integration

This package provides a high-performance worker pool implementation for Go applications using Uber's Fx dependency injection framework. It is built on top of the [panjf2000/ants](https://github.com/panjf2000/ants) package and adds the following features:

- Integration with Uber's Fx dependency injection framework
- Comprehensive metrics collection
- Timeout support
- Error handling
- Panic recovery
- Lifecycle management

## Usage

### Basic Usage

```go
// Create a worker pool factory
factory := workerpool.NewWorkerPoolFactory(workerpool.WorkerPoolParams{
    Logger: logger,
})

// Submit a task to a worker pool
err := factory.Submit("example", func() {
    // Your code here
    time.Sleep(100 * time.Millisecond)
    logger.Info("Task completed")
})

if err != nil {
    logger.Error("Failed to submit task", zap.Error(err))
}
```

### With Error Handling

```go
err := factory.SubmitTask("example-with-error", func() error {
    // Your code here
    time.Sleep(100 * time.Millisecond)
    
    // Return an error if the task fails
    return errors.New("task failed")
})

if err != nil {
    logger.Error("Failed to submit task", zap.Error(err))
}
```

### With Timeout

```go
err := factory.SubmitWithTimeout("example-with-timeout", func() {
    // Your code here
    time.Sleep(200 * time.Millisecond)
    logger.Info("Task completed (but may have timed out)")
}, 100*time.Millisecond)

if err != nil {
    logger.Error("Task timed out", zap.Error(err))
}
```

### With Custom Options

```go
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
        // Your code here
        time.Sleep(100 * time.Millisecond)
        logger.Info("Custom pool task completed")
    })
    
    if err != nil {
        logger.Error("Failed to submit task to custom pool", zap.Error(err))
    }
}
```

### Getting Metrics

```go
metrics := factory.GetMetrics()

logger.Info("Worker pool metrics",
    zap.Int64("executions", metrics.GetExecutionCount("example")),
    zap.Int64("successes", metrics.GetSuccessCount("example")),
    zap.Int64("failures", metrics.GetFailureCount("example")),
    zap.Float64("success_rate", metrics.GetSuccessRate("example")),
    zap.Duration("avg_execution_time", metrics.GetAverageExecutionTime("example")))
```

### Getting Pool Stats

```go
running, capacity, ok := factory.GetPoolStats("example")
if ok {
    logger.Info("Worker pool stats",
        zap.String("name", "example"),
        zap.Int("running", running),
        zap.Int("capacity", capacity))
}
```

### Releasing Pools

```go
// Release a specific pool
factory.ReleasePool("example")

// Release all pools
factory.Release()
```

## Fx Integration

To use the worker pool with Uber's Fx, you can use the provided module:

```go
app := fx.New(
    fx.Provide(
        // Provide a logger
        func() *zap.Logger {
            logger, _ := zap.NewDevelopment()
            return logger
        },
    ),
    
    // Include the worker pool module
    workerpool.Module,
    
    // Use the worker pool in your components
    fx.Invoke(func(wp *workerpool.WorkerPoolFactory) {
        // Use the worker pool
    }),
)
```

## Features

### Worker Pool Options

The worker pool supports the following options:

- **ExpiryDuration**: The duration after which idle workers are cleaned up.
- **PreAlloc**: Whether to pre-allocate memory for workers.
- **MaxBlockingTasks**: The maximum number of tasks that can be blocked waiting for a worker.
- **Nonblocking**: Whether to return an error immediately if the pool is full.
- **PanicHandler**: A function to handle panics in worker tasks.

### Metrics

The worker pool collects the following metrics:

- **Executions**: The number of executions for a worker pool.
- **Successes**: The number of successful executions for a worker pool.
- **Failures**: The number of failed executions for a worker pool.
- **Rejections**: The number of rejected tasks for a worker pool.
- **Timeouts**: The number of timed out tasks for a worker pool.
- **Panics**: The number of panicked tasks for a worker pool.
- **Success Rate**: The success rate for a worker pool.
- **Average Execution Time**: The average execution time for a worker pool.

### Lifecycle Management

The worker pool factory is integrated with Fx's lifecycle management. When the application stops, it logs the worker pool metrics for all worker pools and releases them.

## Dependencies

- [github.com/panjf2000/ants/v2](https://github.com/panjf2000/ants): The underlying worker pool implementation.
- [go.uber.org/fx](https://github.com/uber-go/fx): Dependency injection framework.
- [go.uber.org/zap](https://github.com/uber-go/zap): Logging framework.

