package eventbus

import (
	"context"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	"go.uber.org/zap"
)

// EventBus is the interface for an event bus
type EventBus interface {
	// PublishEvent publishes an event
	PublishEvent(ctx context.Context, event *eventsourcing.Event) error
	
	// PublishEvents publishes multiple events
	PublishEvents(ctx context.Context, events []*eventsourcing.Event) error
	
	// Subscribe subscribes to all events
	Subscribe(handler eventsourcing.EventHandler) error
	
	// SubscribeToType subscribes to events of a specific type
	SubscribeToType(eventType string, handler eventsourcing.EventHandler) error
	
	// SubscribeToAggregate subscribes to events of a specific aggregate type
	SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error
}

// InMemoryEventBus provides an in-memory implementation of the EventBus interface
type InMemoryEventBus struct {
	eventStore   store.EventStore
	handlers     []eventsourcing.EventHandler
	typeHandlers map[string][]eventsourcing.EventHandler
	aggHandlers  map[string][]eventsourcing.EventHandler
	logger       *zap.Logger
	mu           sync.RWMutex
}

// NewInMemoryEventBus creates a new InMemoryEventBus
func NewInMemoryEventBus(eventStore store.EventStore, logger *zap.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		eventStore:   eventStore,
		handlers:     make([]eventsourcing.EventHandler, 0),
		typeHandlers: make(map[string][]eventsourcing.EventHandler),
		aggHandlers:  make(map[string][]eventsourcing.EventHandler),
		logger:       logger,
	}
}

// PublishEvent publishes an event
func (b *InMemoryEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Save the event to the store
	err := b.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return err
	}
	
	// Notify handlers
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Notify all handlers
	for _, handler := range b.handlers {
		err := handler.HandleEvent(event)
		if err != nil {
			b.logger.Error("Failed to handle event",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.String("aggregate_type", event.AggregateType),
				zap.Error(err))
		}
	}
	
	// Notify type handlers
	if handlers, ok := b.typeHandlers[event.EventType]; ok {
		for _, handler := range handlers {
			err := handler.HandleEvent(event)
			if err != nil {
				b.logger.Error("Failed to handle event",
					zap.String("event_type", event.EventType),
					zap.String("aggregate_id", event.AggregateID),
					zap.String("aggregate_type", event.AggregateType),
					zap.Error(err))
			}
		}
	}
	
	// Notify aggregate handlers
	if handlers, ok := b.aggHandlers[event.AggregateType]; ok {
		for _, handler := range handlers {
			err := handler.HandleEvent(event)
			if err != nil {
				b.logger.Error("Failed to handle event",
					zap.String("event_type", event.EventType),
					zap.String("aggregate_id", event.AggregateID),
					zap.String("aggregate_type", event.AggregateType),
					zap.Error(err))
			}
		}
	}
	
	return nil
}

// PublishEvents publishes multiple events
func (b *InMemoryEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	if len(events) == 0 {
		return nil
	}
	
	// Save the events to the store
	err := b.eventStore.SaveEvents(ctx, events)
	if err != nil {
		return err
	}
	
	// Notify handlers for each event
	for _, event := range events {
		b.mu.RLock()
		
		// Notify all handlers
		for _, handler := range b.handlers {
			err := handler.HandleEvent(event)
			if err != nil {
				b.logger.Error("Failed to handle event",
					zap.String("event_type", event.EventType),
					zap.String("aggregate_id", event.AggregateID),
					zap.String("aggregate_type", event.AggregateType),
					zap.Error(err))
			}
		}
		
		// Notify type handlers
		if handlers, ok := b.typeHandlers[event.EventType]; ok {
			for _, handler := range handlers {
				err := handler.HandleEvent(event)
				if err != nil {
					b.logger.Error("Failed to handle event",
						zap.String("event_type", event.EventType),
						zap.String("aggregate_id", event.AggregateID),
						zap.String("aggregate_type", event.AggregateType),
						zap.Error(err))
				}
			}
		}
		
		// Notify aggregate handlers
		if handlers, ok := b.aggHandlers[event.AggregateType]; ok {
			for _, handler := range handlers {
				err := handler.HandleEvent(event)
				if err != nil {
					b.logger.Error("Failed to handle event",
						zap.String("event_type", event.EventType),
						zap.String("aggregate_id", event.AggregateID),
						zap.String("aggregate_type", event.AggregateType),
						zap.Error(err))
				}
			}
		}
		
		b.mu.RUnlock()
	}
	
	return nil
}

// Subscribe subscribes to all events
func (b *InMemoryEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.handlers = append(b.handlers, handler)
	
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (b *InMemoryEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.typeHandlers[eventType]; !ok {
		b.typeHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.typeHandlers[eventType] = append(b.typeHandlers[eventType], handler)
	
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (b *InMemoryEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.aggHandlers[aggregateType]; !ok {
		b.aggHandlers[aggregateType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.aggHandlers[aggregateType] = append(b.aggHandlers[aggregateType], handler)
	
	return nil
}

