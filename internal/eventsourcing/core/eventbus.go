package core

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
)

// EventBus defines the interface for event publishing and subscription
type EventBus interface {
	// PublishEvent publishes a single event to all subscribers
	PublishEvent(ctx context.Context, event *eventsourcing.Event) error

	// PublishEvents publishes multiple events to all subscribers
	PublishEvents(ctx context.Context, events []*eventsourcing.Event) error

	// Subscribe subscribes to all events
	Subscribe(handler eventsourcing.EventHandler) error

	// SubscribeToType subscribes to events of a specific type
	SubscribeToType(eventType string, handler eventsourcing.EventHandler) error

	// SubscribeToAggregate subscribes to events of a specific aggregate type
	SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error
}

// InMemoryEventBus provides an in-memory implementation of EventBus
type InMemoryEventBus struct {
	eventBus map[string][]eventsourcing.EventHandler
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		eventBus: make(map[string][]eventsourcing.EventHandler),
	}
}

// PublishEvent publishes an event to matching handlers
func (eb *InMemoryEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Publish to all event type handlers
	eventType := event.EventType
	for _, handler := range eb.eventBus[eventType] {
		go handler.HandleEvent(event)
	}

	// Also publish to aggregate type handlers
	aggregateType := event.AggregateType
	if aggregateType != "" {
		for _, handler := range eb.eventBus[aggregateType] {
			go handler.HandleEvent(event)
		}
	}

	return nil
}

// PublishEvents publishes multiple events
func (eb *InMemoryEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	for _, event := range events {
		if err := eb.PublishEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe subscribes to all events
func (eb *InMemoryEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	// Use wildcard key for subscribing to all events
	eb.eventBus["*"] = append(eb.eventBus["*"], handler)
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (eb *InMemoryEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	eb.eventBus[eventType] = append(eb.eventBus[eventType], handler)
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (eb *InMemoryEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	eb.eventBus["aggregate:"+aggregateType] = append(eb.eventBus["aggregate:"+aggregateType], handler)
	return nil
}