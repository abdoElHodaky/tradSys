package fx

import (
	"context"
	"time"

	cqrshandlers "github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/handlers"
	eventstore "github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
	eventhandler "github.com/abdoElHodaky/tradSys/internal/eventsourcing/handlers"
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
	NatsConfig cqrshandlers.NatsCQRSConfig

	// WatermillConfig contains configuration for Watermill
	WatermillConfig cqrshandlers.WatermillCQRSConfig

	// EventOrderingGuarantee specifies the required event ordering guarantee
	EventOrderingGuarantee cqrshandlers.EventOrderingGuarantee

	// EventRoutingStrategy specifies the event routing strategy
	EventRoutingStrategy cqrshandlers.EventRoutingStrategy

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
		NatsConfig:             cqrshandlers.DefaultNatsCQRSConfig(),
		WatermillConfig:        cqrshandlers.DefaultWatermillCQRSConfig(),
		EventOrderingGuarantee: cqrshandlers.AggregateOrdering,
		EventRoutingStrategy:   cqrshandlers.SingleBusStrategy,
		CircuitBreakerConfig:   DefaultCircuitBreakerConfig(),
		TracingConfig:          DefaultTracingConfig(),
		ShardingConfig:         DefaultShardingConfig(),
	}
}

// NewEventStore creates a new event store
func NewEventStore(logger *zap.Logger) eventstore.EventStore {
	return eventstore.NewInMemoryEventStore(logger)
}

// NewAggregateRepository creates a new aggregate repository
func NewAggregateRepository(eventStore eventstore.EventStore, logger *zap.Logger) eventhandler.Repository {
	return eventhandler.NewEventSourcedRepository(eventStore, logger)
}

// NewEventBus creates a new event bus
func NewEventBus(
	eventStore eventstore.EventStore,
	logger *zap.Logger,
	lc fx.Lifecycle,
) eventstore.EventBus {
	// Create an in-memory event bus
	bus := eventstore.NewInMemoryEventBus()

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

	return bus
}

// NewCQRSSystem creates a new CQRS system
func NewCQRSSystem(
	eventStore eventstore.EventStore,
	aggregateRepo eventhandler.Repository,
	eventBus eventstore.EventBus,
	logger *zap.Logger,
	lc fx.Lifecycle,
	config CQRSConfig,
) (*cqrshandlers.CQRSSystem, error) {
	// Create a CQRS factory
	factory := cqrshandlers.NewCQRSFactory(
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
				go system.PerformanceMonitor.StartPeriodicLogging(ctx, 10*time.Second)
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
) *cqrshandlers.EventOrderingValidator {
	return cqrshandlers.NewEventOrderingValidator(
		logger,
		config.EventOrderingGuarantee,
	)
}

// NewEventBusRouter creates a new event bus router
func NewEventBusRouter(
	logger *zap.Logger,
	config CQRSConfig,
	lc fx.Lifecycle,
) *cqrshandlers.EventBusRouter {
	// Create the router configuration
	routerConfig := cqrshandlers.EventBusRouterConfig{
		Strategy:        config.EventRoutingStrategy,
		DefaultBus:      cqrshandlers.NatsEventBusType,
		TypeRoutes:      make(map[string]cqrshandlers.EventBusType),
		AggregateRoutes: make(map[string]cqrshandlers.EventBusType),
		PriorityOrder:   []cqrshandlers.EventBusType{cqrshandlers.InMemoryEventBusType, cqrshandlers.NatsEventBusType, cqrshandlers.WatermillEventBusType},
	}

	// Create the router
	router := cqrshandlers.NewEventBusRouter(logger, routerConfig)

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
	system *cqrshandlers.CQRSSystem,
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
