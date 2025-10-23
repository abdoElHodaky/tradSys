package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// BulkheadModule provides the bulkhead components
var BulkheadModule = fx.Options(
	// Provide the bulkhead factory
	fx.Provide(NewBulkheadFactory),

	// Register lifecycle hooks
	fx.Invoke(registerBulkheadHooks),
)

// BulkheadConfig contains configuration for bulkheads
type BulkheadConfig struct {
	// DefaultMaxConcurrency is the default maximum number of concurrent calls
	DefaultMaxConcurrency int64

	// DefaultMaxWaitingQueue is the default maximum size of the waiting queue
	DefaultMaxWaitingQueue int64
}

// DefaultBulkheadConfig returns the default bulkhead configuration
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		DefaultMaxConcurrency:  10,
		DefaultMaxWaitingQueue: 100,
	}
}

// BulkheadFactory creates bulkheads
type BulkheadFactory struct {
	logger *zap.Logger
	config BulkheadConfig
}

// NewBulkheadFactory creates a new bulkhead factory
func NewBulkheadFactory(logger *zap.Logger) *BulkheadFactory {
	return &BulkheadFactory{
		logger: logger,
		config: DefaultBulkheadConfig(),
	}
}

// CreateBulkhead creates a new bulkhead with the given name
func (f *BulkheadFactory) CreateBulkhead(name string) *architecture.Bulkhead {
	return architecture.NewBulkhead(architecture.BulkheadOptions{
		Name:            name,
		MaxConcurrency:  f.config.DefaultMaxConcurrency,
		MaxWaitingQueue: f.config.DefaultMaxWaitingQueue,
	})
}

// CreateCustomBulkhead creates a new bulkhead with custom options
func (f *BulkheadFactory) CreateCustomBulkhead(options architecture.BulkheadOptions) *architecture.Bulkhead {
	return architecture.NewBulkhead(options)
}

// registerBulkheadHooks registers lifecycle hooks for the bulkhead components
func registerBulkheadHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting bulkhead components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping bulkhead components")
			return nil
		},
	})
}
