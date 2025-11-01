package fx

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CQRSModule provides the CQRS components
var CQRSModule = fx.Options(
	// Provide the event store
	fx.Provide(NewEventStore),

	// Provide the aggregate repository
	fx.Provide(NewAggregateRepository),

	// Provide the event bus
	fx.Provide(NewEventBus),

	// Provide the CQRS system
	fx.Provide(NewCQRSSystem),

	// Provide the event ordering validator
	fx.Provide(NewEventOrderingValidator),

	// Provide the event bus router
	fx.Provide(NewEventBusRouter),

	// Provide the circuit breaker
	fx.Provide(NewCircuitBreaker),

	// Provide the distributed tracer
	fx.Provide(NewDistributedTracer),

	// Provide the event sharding manager
	fx.Provide(NewEventShardingManager),

	// Register lifecycle hooks
	fx.Invoke(registerCQRSHooks),
)

// CQRSConfig contains configuration for the CQRS system
type CQRSConfig struct {
	// UseWatermill determines if Watermill should be used
	UseWatermill bool

	// UseNats determines if NATS should be used
	UseNats bool

	// UseCompatLayer determines if the compatibility layer should be used
	UseCompatLayer bool

	// UseMonitoring determines if performance monitoring should be used
	UseMonitoring bool

	// NatsConfig contains configuration for NATS
	NatsConfig integration.NatsCQRSConfig

	// WatermillConfig contains configuration for Watermill
	WatermillConfig integration.WatermillCQRSConfig

	// EventOrderingGuarantee specifies the required event ordering guarantee
	EventOrderingGuarantee integration.EventOrderingGuarantee

	// EventRoutingStrategy specifies the event routing strategy
	EventRoutingStrategy integration.EventRoutingStrategy

	// CircuitBreakerConfig contains configuration for the circuit breaker
	CircuitBreakerConfig CircuitBreakerConfig

	// TracingConfig contains configuration for distributed tracing
	TracingConfig TracingConfig

	// ShardingConfig contains configuration for event sharding
	ShardingConfig ShardingConfig
}

// DefaultCQRSConfig returns the default CQRS configuration
func DefaultCQRSConfig() CQRSConfig {
	return CQRSConfig{
		UseWatermill:           false,
		UseNats:                true,
		UseCompatLayer:         true,
		UseMonitoring:          true,
		NatsConfig:             integration.DefaultNatsCQRSConfig(),
		WatermillConfig:        integration.DefaultWatermillCQRSConfig(),
		EventOrderingGuarantee: integration.AggregateOrdering,
		EventRoutingStrategy:   integration.SingleBusStrategy,
		CircuitBreakerConfig:   DefaultCircuitBreakerConfig(),
		TracingConfig:          DefaultTracingConfig(),
		ShardingConfig:         DefaultShardingConfig(),
	}
}

// NewEventStore creates a new event store
func NewEventStore() (store.EventStore, error) {
	return store.NewInMemoryEventStore()
}

// NewAggregateRepository creates a new aggregate repository
func NewAggregateRepository(eventStore store.EventStore) aggregate.Repository {
	return aggregate.NewRepository(eventStore)
}

// NewEventBus creates a new event bus
func NewEventBus(
	eventStore store.EventStore,
	logger *zap.Logger,
	lc fx.Lifecycle,
) (eventbus.EventBus, error) {
	// Create an in-memory event bus
	bus := eventbus.NewInMemoryEventBus(eventStore, logger)

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting in-memory event bus")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping in-memory event bus")
			return nil
		},
	})

	return bus, nil
}

// NewCQRSSystem creates a new CQRS system
func NewCQRSSystem(
	eventStore store.EventStore,
	aggregateRepo aggregate.Repository,
	eventBus eventbus.EventBus,
	logger *zap.Logger,
	lc fx.Lifecycle,
	config CQRSConfig,
) (*integration.CQRSSystem, error) {
	// Create a CQRS factory
	factory := integration.NewCQRSFactory(
		logger,
		config.UseWatermill,
		config.UseNats,
		config.UseCompatLayer,
		config.UseMonitoring,
	)

	// Create the CQRS system
	system, err := factory.CreateCQRSSystem()
	if err != nil {
		return nil, err
	}

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting CQRS system")

			// Start the Watermill adapter if enabled
			if system.WatermillAdapter != nil {
				err := system.WatermillAdapter.Start()
				if err != nil {
					return err
				}
			}

			// Start the NATS adapter if enabled
			if system.NatsAdapter != nil {
				err := system.NatsAdapter.Start()
				if err != nil {
					return err
				}
			}

			// Start performance monitoring if enabled
			if system.PerformanceMonitor != nil {
				go system.PerformanceMonitor.StartPeriodicLogging(ctx, config.NatsConfig.ReconnectWait)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping CQRS system")

			// Stop the Watermill adapter if enabled
			if system.WatermillAdapter != nil {
				err := system.WatermillAdapter.Stop()
				if err != nil {
					logger.Error("Failed to stop Watermill adapter", zap.Error(err))
				}
			}

			// Stop the NATS adapter if enabled
			if system.NatsAdapter != nil {
				err := system.NatsAdapter.Stop()
				if err != nil {
					logger.Error("Failed to stop NATS adapter", zap.Error(err))
				}
			}

			return nil
		},
	})

	return system, nil
}

// NewEventOrderingValidator creates a new event ordering validator
func NewEventOrderingValidator(
	logger *zap.Logger,
	config CQRSConfig,
) *integration.EventOrderingValidator {
	return integration.NewEventOrderingValidator(
		logger,
		config.EventOrderingGuarantee,
	)
}

// NewEventBusRouter creates a new event bus router
func NewEventBusRouter(
	logger *zap.Logger,
	config CQRSConfig,
	lc fx.Lifecycle,
) *integration.EventBusRouter {
	// Create the router configuration
	routerConfig := integration.EventBusRouterConfig{
		Strategy:        config.EventRoutingStrategy,
		DefaultBus:      integration.NatsEventBusType,
		TypeRoutes:      make(map[string]integration.EventBusType),
		AggregateRoutes: make(map[string]integration.EventBusType),
		PriorityOrder:   []integration.EventBusType{integration.InMemoryEventBusType, integration.NatsEventBusType, integration.WatermillEventBusType},
	}

	// Create the router
	router := integration.NewEventBusRouter(logger, routerConfig)

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting event bus router")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping event bus router")
			return nil
		},
	})

	return router
}

// registerCQRSHooks registers lifecycle hooks for the CQRS components
func registerCQRSHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	system *integration.CQRSSystem,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting CQRS components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping CQRS components")
			return nil
		},
	})
}
