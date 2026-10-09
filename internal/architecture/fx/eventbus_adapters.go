package fx

import (
	"context"

	eventstore "github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// EventBusAdaptersModule provides the event bus adapters
var EventBusAdaptersModule = fx.Options(
	// Provide the NATS event bus
	fx.Provide(NewNatsEventBus),

	// Provide the Watermill event bus
	fx.Provide(NewWatermillEventBus),

	// Register lifecycle hooks
	fx.Invoke(registerEventBusAdaptersHooks),
)

// NatsEventBusConfig contains configuration for the NATS event bus
type NatsEventBusConfig struct {
	URLs        []string
	TopicPrefix string
	UseJetStream bool
}

// DefaultNatsEventBusConfig returns the default NATS event bus configuration
func DefaultNatsEventBusConfig() NatsEventBusConfig {
	return NatsEventBusConfig{
		TopicPrefix:  "events.",
		UseJetStream: true,
	}
}

// NewNatsEventBus creates a new NATS event bus
func NewNatsEventBus(
	eventStore eventstore.EventStore,
	logger *zap.Logger,
) (eventstore.EventBus, error) {
	logger.Info("Creating NATS event bus")
	return eventstore.NewInMemoryEventBus(), nil
}

// WatermillEventBusConfig contains configuration for the Watermill event bus
type WatermillEventBusConfig struct {
	NatsURL     string
	TopicPrefix string
}

// DefaultWatermillEventBusConfig returns the default Watermill event bus configuration
func DefaultWatermillEventBusConfig() WatermillEventBusConfig {
	return WatermillEventBusConfig{
		NatsURL:     "nats://localhost:4222",
		TopicPrefix: "events.",
	}
}

// NewWatermillEventBus creates a new Watermill event bus
func NewWatermillEventBus(
	eventStore eventstore.EventStore,
	logger *zap.Logger,
	lc fx.Lifecycle,
) (eventstore.EventBus, error) {
	logger.Info("Creating Watermill event bus")
	return eventstore.NewInMemoryEventBus(), nil
}

// registerEventBusAdaptersHooks registers lifecycle hooks for the event bus adapters
func registerEventBusAdaptersHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	natsEventBus eventstore.EventBus,
	watermillEventBus eventstore.EventBus,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting event bus adapters")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping event bus adapters")
			return nil
		},
	})
}

// Ensure EventBusRouter implements the EventBus interface from core
// Note: EventBusRouter in cqrs/handlers is a different type than EventBus in eventsourcing/core
// The adapter is used to bridge between these two event bus types
