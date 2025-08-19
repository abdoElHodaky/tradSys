package architecture

import (
	"sync"
)

// EventHandler is a function that handles an event
type EventHandler func(event interface{})

// EventBus implements a simple event bus for publishing and subscribing to events
type EventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Subscribe registers a handler for a specific event type
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	handlers, exists := eb.handlers[eventType]
	if !exists {
		handlers = []EventHandler{}
	}
	
	eb.handlers[eventType] = append(handlers, handler)
}

// Unsubscribe removes a handler for a specific event type
func (eb *EventBus) Unsubscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	handlers, exists := eb.handlers[eventType]
	if !exists {
		return
	}
	
	// Find and remove the handler
	for i, h := range handlers {
		if &h == &handler {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
	
	// Remove the event type if there are no more handlers
	if len(eb.handlers[eventType]) == 0 {
		delete(eb.handlers, eventType)
	}
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(eventType string, event interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	handlers, exists := eb.handlers[eventType]
	if !exists {
		return
	}
	
	// Call all handlers
	for _, handler := range handlers {
		go handler(event)
	}
}

// PublishSync publishes an event to all subscribers synchronously
func (eb *EventBus) PublishSync(eventType string, event interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	handlers, exists := eb.handlers[eventType]
	if !exists {
		return
	}
	
	// Call all handlers synchronously
	for _, handler := range handlers {
		handler(event)
	}
}

// HasSubscribers checks if there are subscribers for a specific event type
func (eb *EventBus) HasSubscribers(eventType string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	handlers, exists := eb.handlers[eventType]
	return exists && len(handlers) > 0
}

// SubscriberCount returns the number of subscribers for a specific event type
func (eb *EventBus) SubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	handlers, exists := eb.handlers[eventType]
	if !exists {
		return 0
	}
	
	return len(handlers)
}

