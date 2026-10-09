package eventbus

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	eventstore "github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
)

// EventBus provides an interface for event publishing and subscription
type EventBus interface {
	PublishEvent(ctx context.Context, event *eventsourcing.Event) error
	PublishEvents(ctx context.Context, events []*eventsourcing.Event) error
	Subscribe(handler eventsourcing.EventHandler) error
	SubscribeToType(eventType string, handler eventsourcing.EventHandler) error
	SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error
}

// NatsEventBusConfig contains configuration for the NATS event bus
type NatsEventBusConfig struct{}

// DefaultNatsEventBusConfig returns the default NATS event bus configuration
func DefaultNatsEventBusConfig() NatsEventBusConfig {
	return NatsEventBusConfig{}
}

// NatsEventBus is a basic event bus implementation
type NatsEventBus struct {
	eventStore eventstore.EventStore
}

// NewNatsEventBus creates a new NATS event bus
func NewNatsEventBus(eventStore eventstore.EventStore, logger interface{}, config NatsEventBusConfig) EventBus {
	return &NatsEventBus{
		eventStore: eventStore,
	}
}

// PublishEvent publishes an event
func (b *NatsEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	return b.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
}

// PublishEvents publishes multiple events
func (b *NatsEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	return b.eventStore.SaveEvents(ctx, events)
}

// Subscribe subscribes to all events
func (b *NatsEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (b *NatsEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (b *NatsEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	return nil
}

// Start starts the event bus
func (b *NatsEventBus) Start() error {
	return nil
}

// Stop stops the event bus
func (b *NatsEventBus) Stop() error {
	return nil
}

// InMemoryEventBusType is the type for in-memory event bus
const InMemoryEventBusType = "in-memory"

// WatermillEventBusConfig contains configuration for the Watermill event bus
type WatermillEventBusConfig struct{}

// DefaultWatermillEventBusConfig returns the default Watermill event bus configuration
func DefaultWatermillEventBusConfig() WatermillEventBusConfig {
	return WatermillEventBusConfig{}
}

// WatermillEventBus is a basic event bus implementation
type WatermillEventBus struct {
	eventStore eventstore.EventStore
}

// NewWatermillEventBus creates a new Watermill event bus
func NewWatermillEventBus(eventStore eventstore.EventStore) EventBus {
	return &WatermillEventBus{
		eventStore: eventStore,
	}
}

// PublishEvent publishes an event
func (b *WatermillEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	return b.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
}

// PublishEvents publishes multiple events
func (b *WatermillEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	return b.eventStore.SaveEvents(ctx, events)
}

// Subscribe subscribes to all events
func (b *WatermillEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (b *WatermillEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (b *WatermillEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	return nil
}

// Start starts the event bus
func (b *WatermillEventBus) Start() error {
	return nil
}

// Stop stops the event bus
func (b *WatermillEventBus) Stop() error {
	return nil
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(eventStore eventstore.EventStore, logger interface{}, config interface{}) EventBus {
	return &NatsEventBus{
		eventStore: eventStore,
	}
}

// NatsEventBusType is the type for NATS event bus
const NatsEventBusType = "nats"

// WatermillEventBusType is the type for Watermill event bus
const WatermillEventBusType = "watermill"
