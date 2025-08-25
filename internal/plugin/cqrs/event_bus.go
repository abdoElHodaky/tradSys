package cqrs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Event is the interface that all events must implement
type Event interface {
	// EventType returns the type of the event
	EventType() string
	
	// OccurredAt returns when the event occurred
	OccurredAt() time.Time
}

// BaseEvent provides common functionality for events
type BaseEvent struct {
	Type       string    `json:"type"`
	OccurredOn time.Time `json:"occurred_on"`
}

// EventType returns the type of the event
func (e BaseEvent) EventType() string {
	return e.Type
}

// OccurredAt returns when the event occurred
func (e BaseEvent) OccurredAt() time.Time {
	return e.OccurredOn
}

// NewBaseEvent creates a new base event
func NewBaseEvent(eventType string) BaseEvent {
	return BaseEvent{
		Type:       eventType,
		OccurredOn: time.Now(),
	}
}

// EventHandler is the interface that all event handlers must implement
type EventHandler interface {
	// EventType returns the type of event this handler can process
	EventType() string
	
	// Handle processes the event
	Handle(ctx context.Context, event Event) error
}

// EventHandlerFunc is a function that handles an event
type EventHandlerFunc func(ctx context.Context, event Event) error

// EventMiddleware is a function that wraps an event handler
type EventMiddleware func(EventHandlerFunc) EventHandlerFunc

// EventBus dispatches events to their handlers
type EventBus struct {
	handlers   map[string][]EventHandler
	middleware []EventMiddleware
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus(logger *zap.Logger) *EventBus {
	return &EventBus{
		handlers:   make(map[string][]EventHandler),
		middleware: []EventMiddleware{},
		logger:     logger,
	}
}

// RegisterHandler registers an event handler
func (b *EventBus) RegisterHandler(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	eventType := handler.EventType()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	
	b.logger.Debug("Registered event handler", 
		zap.String("event_type", eventType))
}

// RegisterMiddleware registers middleware to be applied to all events
func (b *EventBus) RegisterMiddleware(middleware EventMiddleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.middleware = append(b.middleware, middleware)
}

// Publish sends an event to all its handlers
func (b *EventBus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	eventType := event.EventType()
	handlers := b.handlers[eventType]
	middleware := make([]EventMiddleware, len(b.middleware))
	copy(middleware, b.middleware)
	b.mu.RUnlock()
	
	if len(handlers) == 0 {
		b.logger.Debug("No handlers registered for event type", 
			zap.String("event_type", eventType))
		return
	}
	
	// Process the event with each handler
	for _, handler := range handlers {
		// Apply middleware
		next := handler.Handle
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i](next)
		}
		
		// Execute the handler in a goroutine
		go func(h EventHandlerFunc) {
			if err := h(ctx, event); err != nil {
				b.logger.Error("Error handling event",
					zap.String("event_type", eventType),
					zap.Error(err))
			}
		}(next)
	}
}

// TypedEventHandler is a generic event handler for a specific event type
type TypedEventHandler[T Event] struct {
	eventType  string
	handleFunc func(ctx context.Context, event T) error
}

// NewTypedEventHandler creates a new typed event handler
func NewTypedEventHandler[T Event](eventType string, handleFunc func(ctx context.Context, event T) error) *TypedEventHandler[T] {
	return &TypedEventHandler[T]{
		eventType:  eventType,
		handleFunc: handleFunc,
	}
}

// EventType returns the type of event this handler can process
func (h *TypedEventHandler[T]) EventType() string {
	return h.eventType
}

// Handle processes the event
func (h *TypedEventHandler[T]) Handle(ctx context.Context, event Event) error {
	typedEvent, ok := event.(T)
	if !ok {
		return fmt.Errorf("invalid event type: expected %T, got %T", *new(T), event)
	}
	
	return h.handleFunc(ctx, typedEvent)
}

// LoggingMiddleware logs events before and after handling
func EventLoggingMiddleware(logger *zap.Logger) EventMiddleware {
	return func(next EventHandlerFunc) EventHandlerFunc {
		return func(ctx context.Context, event Event) error {
			start := time.Now()
			logger.Debug("Handling event", 
				zap.String("event_type", event.EventType()),
				zap.Time("occurred_at", event.OccurredAt()))
			
			err := next(ctx, event)
			
			logger.Debug("Event handled",
				zap.String("event_type", event.EventType()),
				zap.Duration("duration", time.Since(start)),
				zap.Error(err))
			
			return err
		}
	}
}

// RetryMiddleware retries event handling on failure
func RetryMiddleware(maxRetries int, delay time.Duration) EventMiddleware {
	return func(next EventHandlerFunc) EventHandlerFunc {
		return func(ctx context.Context, event Event) error {
			var err error
			
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					// Wait before retrying
					time.Sleep(delay * time.Duration(attempt))
				}
				
				err = next(ctx, event)
				if err == nil {
					return nil
				}
			}
			
			return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
		}
	}
}

// AsyncEventBus is an event bus that processes events asynchronously
type AsyncEventBus struct {
	eventBus *EventBus
	queue    chan asyncEvent
	workers  int
	logger   *zap.Logger
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

type asyncEvent struct {
	ctx   context.Context
	event Event
}

// NewAsyncEventBus creates a new async event bus
func NewAsyncEventBus(eventBus *EventBus, queueSize int, workers int, logger *zap.Logger) *AsyncEventBus {
	return &AsyncEventBus{
		eventBus: eventBus,
		queue:    make(chan asyncEvent, queueSize),
		workers:  workers,
		logger:   logger,
		running:  false,
	}
}

// Start starts the async event bus
func (b *AsyncEventBus) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.running {
		return
	}
	
	b.running = true
	
	// Start worker goroutines
	for i := 0; i < b.workers; i++ {
		b.wg.Add(1)
		go b.worker()
	}
	
	b.logger.Info("Started async event bus", 
		zap.Int("workers", b.workers),
		zap.Int("queue_size", cap(b.queue)))
}

// Stop stops the async event bus
func (b *AsyncEventBus) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.running {
		return
	}
	
	b.running = false
	close(b.queue)
	
	// Wait for all workers to finish
	b.wg.Wait()
	
	b.logger.Info("Stopped async event bus")
}

// Publish sends an event to the queue
func (b *AsyncEventBus) Publish(ctx context.Context, event Event) error {
	b.mu.Lock()
	running := b.running
	b.mu.Unlock()
	
	if !running {
		return fmt.Errorf("async event bus is not running")
	}
	
	// Try to add to queue with timeout
	select {
	case b.queue <- asyncEvent{ctx, event}:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("event queue is full")
	}
}

// worker processes events from the queue
func (b *AsyncEventBus) worker() {
	defer b.wg.Done()
	
	for evt := range b.queue {
		// Create a new context with timeout
		ctx, cancel := context.WithTimeout(evt.ctx, 30*time.Second)
		
		// Process the event
		b.eventBus.Publish(ctx, evt.event)
		
		cancel()
	}
}
