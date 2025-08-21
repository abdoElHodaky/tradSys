package workerpool

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the worker pool components
var Module = fx.Options(
	// Provide the worker pool factory
	fx.Provide(NewWorkerPoolFactory),
	
	// Register lifecycle hooks
	fx.Invoke(registerHooks),
)

// registerHooks registers lifecycle hooks for the worker pool components
func registerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	workerPool *WorkerPoolFactory,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx fx.Context) error {
			logger.Info("Starting worker pool components")
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping worker pool components")
			
			// Log worker pool metrics
			for _, name := range []string{"market-data-processor", "order-processor"} {
				running, capacity, ok := workerPool.GetPoolStats(name)
				if ok {
					logger.Info("Worker pool stats",
						zap.String("name", name),
						zap.Int("running", running),
						zap.Int("capacity", capacity),
						zap.Int64("executions", workerPool.GetMetrics().GetExecutionCount(name)),
						zap.Int64("successes", workerPool.GetMetrics().GetSuccessCount(name)),
						zap.Int64("failures", workerPool.GetMetrics().GetFailureCount(name)),
						zap.Int64("rejections", workerPool.GetMetrics().GetRejectionCount(name)),
						zap.Int64("timeouts", workerPool.GetMetrics().GetTimeoutCount(name)),
						zap.Int64("panics", workerPool.GetMetrics().GetPanicCount(name)),
						zap.Float64("success_rate", workerPool.GetMetrics().GetSuccessRate(name)),
						zap.Duration("avg_execution_time", workerPool.GetMetrics().GetAverageExecutionTime(name)))
				}
			}
			
			// Release all worker pools
			workerPool.Release()
			
			return nil
		},
	})
}

