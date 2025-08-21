# Worker Pool Package

This package provides an optimized worker pool implementation for the TradSys platform, using the high-performance `github.com/panjf2000/ants/v2` package while ensuring it follows Fx benefits and best practices.

## Key Features

- **Fx Integration**: Fully integrated with Uber's Fx dependency injection framework
- **Lifecycle Management**: Proper resource initialization and cleanup
- **Metrics Collection**: Built-in metrics collection for worker pool events
- **High Performance**: Uses the battle-tested `ants` library for optimal performance
- **Customizable**: Highly configurable worker pool behavior
- **Panic Recovery**: Built-in panic recovery for worker tasks
- **Statistics**: Comprehensive statistics for monitoring and debugging

## Usage

### Basic Usage

```go
// In your Fx application
app := fx.New(
    // Include the worker pool module
    workerpool.Module,
    
    // Provide your services
    fx.Provide(
        NewMyService,
    ),
)

// In your service
type MyService struct {
    workerPool *workerpool.WorkerPoolFactory
}

func NewMyService(workerPool *workerpool.WorkerPoolFactory) *MyService {
    return &MyService{
        workerPool: workerPool,
    }
}

func (s *MyService) ProcessItems(items []string) error {
    for _, item := range items {
        item := item // Capture loop variable
        err := s.workerPool.Submit("my-pool", func() {
            // Process the item
            processItem(item)
        })
        if err != nil {
            return err
        }
    }
    return nil
}
```

### With Error Handling

```go
func (s *MyService) ProcessItemsWithErrorHandling(items []string) error {
    for _, item := range items {
        item := item // Capture loop variable
        err := s.workerPool.SubmitTask("my-pool", func() error {
            // Process the item
            return processItem(item)
        })
        if err != nil {
            return err
        }
    }
    return nil
}
```

### Custom Pool Configuration

```go
func (s *MyService) SetupCustomWorkerPool() (*ants.Pool, error) {
    options := ants.Options{
        ExpiryDuration: time.Minute,
        PreAlloc:       true,
        Nonblocking:    true,
        PanicHandler: func(i interface{}) {
            // Custom panic handler
            log.Printf("Worker panic: %v", i)
        },
    }
    
    return s.workerPool.CreateCustomWorkerPool("custom-pool", 50, options)
}
```

### Getting Pool Statistics

```go
func (s *MyService) LogPoolStatistics() {
    stats := s.workerPool.GetStats()
    for name, stat := range stats {
        log.Printf("Pool %s: Running=%d, Free=%d, Submitted=%d, Completed=%d, Failed=%d",
            name, stat.RunningWorkers, stat.FreeWorkers, 
            stat.TasksSubmitted, stat.TasksCompleted, stat.TasksFailed)
    }
}
```

## Benefits Over Previous Implementation

1. **Higher Performance**: Uses the highly optimized `ants` library which outperforms manual worker pool implementations
2. **Better Dependency Injection**: Follows Fx's dependency injection pattern more closely
3. **Separation of Concerns**: Separates metrics collection from worker pool logic
4. **Better Error Handling**: Improved error handling and panic recovery
5. **More Configurable**: More configuration options for worker pools
6. **Better Metrics**: More detailed metrics collection
7. **Better Documentation**: More comprehensive documentation and examples
8. **Memory Efficiency**: More efficient memory usage with worker reuse and cleanup

## Performance Comparison

The `ants` library used in this implementation has been benchmarked against other worker pool implementations and shows significant performance advantages:

- Lower memory usage due to worker reuse
- Higher throughput for task processing
- Better scalability under high load
- Lower latency for task execution

## Future Improvements

- Add support for prioritized task queues
- Add support for task cancellation
- Add support for task timeouts
- Add support for more sophisticated scheduling strategies
- Add support for Prometheus metrics integration

