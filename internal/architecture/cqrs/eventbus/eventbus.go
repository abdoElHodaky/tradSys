package eventbus

import (
	"context"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	"go.uber.org/zap"
)

// EventBus represents an event bus for publishing and subscribing to events
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

// DefaultEventBus provides a default implementation of the EventBus interface
type DefaultEventBus struct {
	eventStore      store.EventStore
	handlers        []eventsourcing.EventHandler
	typeHandlers    map[string][]eventsourcing.EventHandler
	aggregateHandlers map[string][]eventsourcing.EventHandler
	logger          *zap.Logger
	mu              sync.RWMutex
}

// NewDefaultEventBus creates a new default event bus
func NewDefaultEventBus(eventStore store.EventStore, logger *zap.Logger) *DefaultEventBus {
	return &DefaultEventBus{
		eventStore:      eventStore,
		handlers:        make([]eventsourcing.EventHandler, 0),
		typeHandlers:    make(map[string][]eventsourcing.EventHandler),
		aggregateHandlers: make(map[string][]eventsourcing.EventHandler),
		logger:          logger,
	}
}

// PublishEvent publishes an event
func (b *DefaultEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Save the event to the store
	err := b.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return err
	}
	
	// Notify handlers
	b.notifyHandlers(ctx, event)
	
	return nil
}

// PublishEvents publishes multiple events
func (b *DefaultEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	if len(events) == 0 {
		return nil
	}
	
	// Save the events to the store
	err := b.eventStore.SaveEvents(ctx, events)
	if err != nil {
		return err
	}
	
	// Notify handlers
	for _, event := range events {
		b.notifyHandlers(ctx, event)
	}
	
	return nil
}

// Subscribe subscribes to all events
func (b *DefaultEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.handlers = append(b.handlers, handler)
	
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (b *DefaultEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.typeHandlers[eventType]; !ok {
		b.typeHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.typeHandlers[eventType] = append(b.typeHandlers[eventType], handler)
	
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (b *DefaultEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.aggregateHandlers[aggregateType]; !ok {
		b.aggregateHandlers[aggregateType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.aggregateHandlers[aggregateType] = append(b.aggregateHandlers[aggregateType], handler)
	
	return nil
}

// notifyHandlers notifies handlers of an event
func (b *DefaultEventBus) notifyHandlers(ctx context.Context, event *eventsourcing.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Notify all handlers
	for _, handler := range b.handlers {
		go func(h eventsourcing.EventHandler) {
			err := h.HandleEvent(event)
			if err != nil {
				b.logger.Error("Failed to handle event",
					zap.String("event_type", event.EventType),
					zap.String("aggregate_id", event.AggregateID),
					zap.String("aggregate_type", event.AggregateType),
					zap.Error(err))
			}
		}(handler)
	}
	
	// Notify type handlers
	if handlers, ok := b.typeHandlers[event.EventType]; ok {
		for _, handler := range handlers {
			go func(h eventsourcing.EventHandler) {
				err := h.HandleEvent(event)
				if err != nil {
					b.logger.Error("Failed to handle event",
						zap.String("event_type", event.EventType),
						zap.String("aggregate_id", event.AggregateID),
						zap.String("aggregate_type", event.AggregateType),
						zap.Error(err))
				}
			}(handler)
		}
	}
	
	// Notify aggregate handlers
	if handlers, ok := b.aggregateHandlers[event.AggregateType]; ok {
		for _, handler := range handlers {
			go func(h eventsourcing.EventHandler) {
				err := h.HandleEvent(event)
				if err != nil {
					b.logger.Error("Failed to handle event",
						zap.String("event_type", event.EventType),
						zap.String("aggregate_id", event.AggregateID),
						zap.String("aggregate_type", event.AggregateType),
						zap.Error(err))
				}
			}(handler)
		}
	}
}

// AsyncEventBus provides an asynchronous implementation of the EventBus interface
type AsyncEventBus struct {
	eventStore      store.EventStore
	handlers        []eventsourcing.EventHandler
	typeHandlers    map[string][]eventsourcing.EventHandler
	aggregateHandlers map[string][]eventsourcing.EventHandler
	logger          *zap.Logger
	mu              sync.RWMutex
	eventChan       chan *eventsourcing.Event
	stopChan        chan struct{}
	workerCount     int
}

// NewAsyncEventBus creates a new asynchronous event bus
func NewAsyncEventBus(eventStore store.EventStore, logger *zap.Logger, workerCount int) *AsyncEventBus {
	return &AsyncEventBus{
		eventStore:      eventStore,
		handlers:        make([]eventsourcing.EventHandler, 0),
		typeHandlers:    make(map[string][]eventsourcing.EventHandler),
		aggregateHandlers: make(map[string][]eventsourcing.EventHandler),
		logger:          logger,
		eventChan:       make(chan *eventsourcing.Event, 1000),
		stopChan:        make(chan struct{}),
		workerCount:     workerCount,
	}
}

// Start starts the event bus
func (b *AsyncEventBus) Start() {
	// Start workers
	for i := 0; i < b.workerCount; i++ {
		go b.worker()
	}
}

// Stop stops the event bus
func (b *AsyncEventBus) Stop() {
	close(b.stopChan)
}

// worker processes events
func (b *AsyncEventBus) worker() {
	for {
		select {
		case event := <-b.eventChan:
			// Notify handlers
			b.notifyHandlers(context.Background(), event)
		case <-b.stopChan:
			return
		}
	}
}

// PublishEvent publishes an event
func (b *AsyncEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Save the event to the store
	err := b.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return err
	}
	
	// Send the event to the channel
	select {
	case b.eventChan <- event:
		// Event sent
	default:
		// Channel is full, log and drop the event
		b.logger.Warn("Event channel is full, dropping event",
			zap.String("event_type", event.EventType),
			zap.String("aggregate_id", event.AggregateID),
			zap.String("aggregate_type", event.AggregateType))
	}
	
	return nil
}

// PublishEvents publishes multiple events
func (b *AsyncEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	if len(events) == 0 {
		return nil
	}
	
	// Save the events to the store
	err := b.eventStore.SaveEvents(ctx, events)
	if err != nil {
		return err
	}
	
	// Send the events to the channel
	for _, event := range events {
		select {
		case b.eventChan <- event:
			// Event sent
		default:
			// Channel is full, log and drop the event
			b.logger.Warn("Event channel is full, dropping event",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.String("aggregate_type", event.AggregateType))
		}
	}
	
	return nil
}

// Subscribe subscribes to all events
func (b *AsyncEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.handlers = append(b.handlers, handler)
	
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (b *AsyncEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.typeHandlers[eventType]; !ok {
		b.typeHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.typeHandlers[eventType] = append(b.typeHandlers[eventType], handler)
	
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (b *AsyncEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.aggregateHandlers[aggregateType]; !ok {
		b.aggregateHandlers[aggregateType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.aggregateHandlers[aggregateType] = append(b.aggregateHandlers[aggregateType], handler)
	
	return nil
}

// notifyHandlers notifies handlers of an event
func (b *AsyncEventBus) notifyHandlers(ctx context.Context, event *eventsourcing.Event) {
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
	if handlers, ok := b.aggregateHandlers[event.AggregateType]; ok {
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
}

