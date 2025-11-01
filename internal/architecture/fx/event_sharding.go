package fx

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ShardingModule provides the event sharding components
var ShardingModule = fx.Options(
	// Provide the event sharding manager
	fx.Provide(NewEventShardingManager),

	// Register lifecycle hooks
	fx.Invoke(registerShardingHooks),
)

// ShardingConfig contains configuration for event sharding
type ShardingConfig struct {
	// Strategy determines the sharding strategy
	Strategy string

	// ShardCount is the number of shards
	ShardCount int
}

// DefaultShardingConfig returns the default sharding configuration
func DefaultShardingConfig() ShardingConfig {
	return ShardingConfig{
		Strategy:   "aggregate",
		ShardCount: 10,
	}
}

// NewEventShardingManager creates a new event sharding manager
func NewEventShardingManager(
	logger *zap.Logger,
	conn *nats.Conn,
	js nats.JetStreamContext,
) *integration.EventShardingManager {
	// Create the sharding configuration
	config := integration.DefaultShardingConfig()

	// Create the event sharding manager
	return integration.NewEventShardingManager(logger, config, conn, js)
}

// registerShardingHooks registers lifecycle hooks for the event sharding manager
func registerShardingHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	manager *integration.EventShardingManager,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting event sharding manager")
			return manager.Initialize(ctx)
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping event sharding manager")
			return nil
		},
	})
}
