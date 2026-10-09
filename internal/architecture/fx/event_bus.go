package fx

import (
	"context"

	eventstore "github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// EventBusModule provides the event bus components
var EventBusModule = fx.Options(
	// Provide the event bus factory
	fx.Provide(NewEventBusFactory),

	// Register lifecycle hooks
	fx.Invoke(registerEventBusHooks),
)

// EventBusConfig contains configuration for event buses
type EventBusConfig struct {
	// DefaultQueueSize is the default size of the event queue
	DefaultQueueSize int
}

// DefaultEventBusConfig returns the default event bus configuration
func DefaultEventBusConfig() EventBusConfig {
	return EventBusConfig{
		DefaultQueueSize: 1000,
	}
}

// EventBusFactory creates event buses
type EventBusFactory struct {
	logger *zap.Logger
	config EventBusConfig
}

// NewEventBusFactory creates a new event bus factory
func NewEventBusFactory(logger *zap.Logger) *EventBusFactory {
	return &EventBusFactory{
		logger: logger,
		config: DefaultEventBusConfig(),
	}
}

// CreateInMemoryEventBus creates a new in-memory event bus
func (f *EventBusFactory) CreateInMemoryEventBus() eventstore.EventBus {
	return eventstore.NewInMemoryEventBus()
}

// registerEventBusHooks registers lifecycle hooks for the event bus components
func registerEventBusHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting event bus components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping event bus components")
			return nil
		},
	})
}
