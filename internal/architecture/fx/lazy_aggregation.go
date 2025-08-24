package fx

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/integration"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyAggregationModule provides lazily loaded aggregation components
var LazyAggregationModule = fx.Options(
	// Provide lazily loaded aggregation components
	provideLazyEventShardingManager,
	provideLazyEventOrderingValidator,
	provideLazyShardingEventBusDecorator,
	provideLazyOrderingEventBusDecorator,
	
	// Register lifecycle hooks
	fx.Invoke(registerLazyAggregationHooks),
)

// provideLazyEventShardingManager provides a lazily loaded event sharding manager
func provideLazyEventShardingManager(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"event-sharding-manager",
		func(
			logger *zap.Logger,
			conn *nats.Conn,
			js nats.JetStreamContext,
		) (*integration.EventShardingManager, error) {
			logger.Info("Lazily initializing event sharding manager")
			config := integration.DefaultShardingConfig()
			manager := integration.NewEventShardingManager(logger, config, conn, js)
			
			// Initialize the manager
			if err := manager.Initialize(nil); err != nil {
				return nil, err
			}
			
			return manager, nil
		},
		logger,
		metrics,
	)
}

// provideLazyEventOrderingValidator provides a lazily loaded event ordering validator
func provideLazyEventOrderingValidator(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"event-ordering-validator",
		func(
			logger *zap.Logger,
		) (*integration.EventOrderingValidator, error) {
			logger.Info("Lazily initializing event ordering validator")
			return integration.NewEventOrderingValidator(
				logger,
				integration.AggregateOrdering, // Default to aggregate ordering
			), nil
		},
		logger,
		metrics,
	)
}

// provideLazyShardingEventBusDecorator provides a lazily loaded sharding event bus decorator
func provideLazyShardingEventBusDecorator(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"sharding-event-bus-decorator",
		func(
			logger *zap.Logger,
			eventBus integration.EventBus,
			shardingManagerProvider *lazy.LazyProvider,
		) (*integration.ShardingEventBusDecorator, error) {
			logger.Info("Lazily initializing sharding event bus decorator")
			
			// Get the sharding manager
			instance, err := shardingManagerProvider.Get()
			if err != nil {
				return nil, err
			}
			
			shardingManager := instance.(*integration.EventShardingManager)
			
			// Create the decorator
			return integration.NewShardingEventBusDecorator(
				eventBus,
				shardingManager,
				logger,
			), nil
		},
		logger,
		metrics,
	)
}

// provideLazyOrderingEventBusDecorator provides a lazily loaded ordering event bus decorator
func provideLazyOrderingEventBusDecorator(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"ordering-event-bus-decorator",
		func(
			logger *zap.Logger,
			eventBus integration.EventBus,
			validatorProvider *lazy.LazyProvider,
		) (*integration.OrderingEventBusDecorator, error) {
			logger.Info("Lazily initializing ordering event bus decorator")
			
			// Get the validator
			instance, err := validatorProvider.Get()
			if err != nil {
				return nil, err
			}
			
			validator := instance.(*integration.EventOrderingValidator)
			
			// Create the decorator
			return integration.NewOrderingEventBusDecorator(
				eventBus,
				validator,
				logger,
				true, // Add handler
			), nil
		},
		logger,
		metrics,
	)
}

// registerLazyAggregationHooks registers lifecycle hooks for the lazy aggregation components
func registerLazyAggregationHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	shardingManagerProvider *lazy.LazyProvider,
	validatorProvider *lazy.LazyProvider,
) {
	logger.Info("Registering lazy aggregation component hooks")
}

// GetEventShardingManager gets the event sharding manager, initializing it if necessary
func GetEventShardingManager(provider *lazy.LazyProvider) (*integration.EventShardingManager, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*integration.EventShardingManager), nil
}

// GetEventOrderingValidator gets the event ordering validator, initializing it if necessary
func GetEventOrderingValidator(provider *lazy.LazyProvider) (*integration.EventOrderingValidator, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*integration.EventOrderingValidator), nil
}

// GetShardingEventBusDecorator gets the sharding event bus decorator, initializing it if necessary
func GetShardingEventBusDecorator(provider *lazy.LazyProvider) (*integration.ShardingEventBusDecorator, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*integration.ShardingEventBusDecorator), nil
}

// GetOrderingEventBusDecorator gets the ordering event bus decorator, initializing it if necessary
func GetOrderingEventBusDecorator(provider *lazy.LazyProvider) (*integration.OrderingEventBusDecorator, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*integration.OrderingEventBusDecorator), nil
}

