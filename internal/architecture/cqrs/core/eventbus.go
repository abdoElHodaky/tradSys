package core

import (
	"context"
	"sync"
)

// EventBus interface defines the contract for event bus implementations
type EventBus interface {
	Publish(ctx context.Context, event interface{}) error
	PublishEvents(ctx context.Context, events []Event) error
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventHandler represents a function that handles events
type EventHandler func(ctx context.Context, event interface{}) error

// InMemoryEventBus is a simple in-memory implementation of EventBus
type InMemoryEventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish publishes an event to all registered handlers
func (bus *InMemoryEventBus) Publish(ctx context.Context, event interface{}) error {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Get event type name (simplified)
	eventType := getEventType(event)
	
	handlers, exists := bus.handlers[eventType]
	if !exists {
		return nil // No handlers registered
	}

	// Call all handlers
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			// Log error but continue with other handlers
			continue
		}
	}

	return nil
}

// PublishEvents publishes multiple events
func (bus *InMemoryEventBus) PublishEvents(ctx context.Context, events []Event) error {
	for _, event := range events {
		if err := bus.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a handler for a specific event type
func (bus *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
	return nil
}

// Unsubscribe removes a handler for a specific event type
func (bus *InMemoryEventBus) Unsubscribe(eventType string, handler EventHandler) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlers, exists := bus.handlers[eventType]
	if !exists {
		return nil
	}

	// Remove handler (simplified - in production would need proper comparison)
	bus.handlers[eventType] = handlers[:len(handlers)-1]
	return nil
}

// getEventType returns the type name of an event (simplified implementation)
func getEventType(event interface{}) string {
	// In a real implementation, this would use reflection or type assertions
	// to get the actual type name
	return "Event"
}
