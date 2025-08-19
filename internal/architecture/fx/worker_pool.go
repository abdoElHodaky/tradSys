package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WorkerPoolModule provides the worker pool components
var WorkerPoolModule = fx.Options(
	// Provide the worker pool factory
	fx.Provide(NewWorkerPoolFactory),
	
	// Register lifecycle hooks
	fx.Invoke(registerWorkerPoolHooks),
)

// WorkerPoolConfig contains configuration for worker pools
type WorkerPoolConfig struct {
	// DefaultSize is the default number of workers in the pool
	DefaultSize int
	
	// DefaultQueueSize is the default size of the task queue
	DefaultQueueSize int
}

// DefaultWorkerPoolConfig returns the default worker pool configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		DefaultSize:      10,
		DefaultQueueSize: 100,
	}
}

// WorkerPoolFactory creates worker pools
type WorkerPoolFactory struct {
	logger *zap.Logger
	config WorkerPoolConfig
	pools  map[string]*architecture.WorkerPool
}

// NewWorkerPoolFactory creates a new worker pool factory
func NewWorkerPoolFactory(logger *zap.Logger) *WorkerPoolFactory {
	return &WorkerPoolFactory{
		logger: logger,
		config: DefaultWorkerPoolConfig(),
		pools:  make(map[string]*architecture.WorkerPool),
	}
}

// CreateWorkerPool creates a new worker pool with the given name
func (f *WorkerPoolFactory) CreateWorkerPool(name string) *architecture.WorkerPool {
	pool := architecture.NewWorkerPool(architecture.WorkerPoolOptions{
		Name:      name,
		Size:      f.config.DefaultSize,
		QueueSize: f.config.DefaultQueueSize,
	})
	
	f.pools[name] = pool
	f.logger.Info("Created worker pool", zap.String("name", name))
	
	return pool
}

// CreateCustomWorkerPool creates a new worker pool with custom options
func (f *WorkerPoolFactory) CreateCustomWorkerPool(options architecture.WorkerPoolOptions) *architecture.WorkerPool {
	pool := architecture.NewWorkerPool(options)
	
	f.pools[options.Name] = pool
	f.logger.Info("Created custom worker pool", zap.String("name", options.Name))
	
	return pool
}

// GetWorkerPool gets a worker pool by name
func (f *WorkerPoolFactory) GetWorkerPool(name string) *architecture.WorkerPool {
	return f.pools[name]
}

// registerWorkerPoolHooks registers lifecycle hooks for the worker pool components
func registerWorkerPoolHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	factory *WorkerPoolFactory,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting worker pool components")
			
			// Start all worker pools
			for name, pool := range factory.pools {
				logger.Info("Starting worker pool", zap.String("name", name))
				pool.Start()
			}
			
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping worker pool components")
			
			// Stop all worker pools
			for name, pool := range factory.pools {
				logger.Info("Stopping worker pool", zap.String("name", name))
				pool.Stop()
			}
			
			return nil
		},
	})
}

