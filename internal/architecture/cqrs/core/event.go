package core

import (
	"context"
	"sync"
	"time"

	"github.com/segmentio/ksuid"
)

// Event represents a domain event in the CQRS pattern
type Event interface {
	// EventName returns the name of the event
	EventName() string

	// AggregateID returns the ID of the aggregate that emitted the event
	AggregateID() string

	// EventID returns the unique ID of the event
	EventID() string

	// EventTimestamp returns the timestamp when the event occurred
	EventTimestamp() time.Time

	// EventVersion returns the version of the event
	EventVersion() int

	// EventData returns the data associated with the event
	EventData() interface{}
}

// BaseEvent provides a base implementation of the Event interface
type BaseEvent struct {
	ID        string
	Name      string
	Aggregate string
	Timestamp time.Time
	Version   int
	Data      interface{}
}

// EventName returns the name of the event
func (e BaseEvent) EventName() string {
	return e.Name
}

// AggregateID returns the ID of the aggregate that emitted the event
func (e BaseEvent) AggregateID() string {
	return e.Aggregate
}

// EventID returns the unique ID of the event
func (e BaseEvent) EventID() string {
	return e.ID
}

// EventTimestamp returns the timestamp when the event occurred
func (e BaseEvent) EventTimestamp() time.Time {
	return e.Timestamp
}

// EventVersion returns the version of the event
func (e BaseEvent) EventVersion() int {
	return e.Version
}

// EventData returns the data associated with the event
func (e BaseEvent) EventData() interface{} {
	return e.Data
}

// NewEvent creates a new event
func NewEvent(name string, aggregateID string, data interface{}, version int) Event {
	return BaseEvent{
		ID:        ksuid.New().String(),
		Name:      name,
		Aggregate: aggregateID,
		Timestamp: time.Now().UTC(),
		Version:   version,
		Data:      data,
	}
}

// EventStore defines the interface for storing and retrieving events
type EventStore interface {
	// SaveEvents saves events to the event store
	SaveEvents(ctx context.Context, events []Event) error

	// GetEvents retrieves events for an aggregate from the event store
	GetEvents(ctx context.Context, aggregateID string) ([]Event, error)

	// GetEventsByType retrieves events of a specific type from the event store
	GetEventsByType(ctx context.Context, eventType string) ([]Event, error)
}

// InMemoryEventStore implements the EventStore interface using in-memory storage
type InMemoryEventStore struct {
	events map[string][]Event
	mu     sync.RWMutex
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]Event),
	}
}

// SaveEvents saves events to the in-memory event store
func (s *InMemoryEventStore) SaveEvents(ctx context.Context, events []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, event := range events {
		aggregateID := event.AggregateID()
		s.events[aggregateID] = append(s.events[aggregateID], event)
	}

	return nil
}

// GetEvents retrieves events for an aggregate from the in-memory event store
func (s *InMemoryEventStore) GetEvents(ctx context.Context, aggregateID string) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, exists := s.events[aggregateID]
	if !exists {
		return []Event{}, nil
	}

	// Return a copy to avoid race conditions
	result := make([]Event, len(events))
	copy(result, events)
	return result, nil
}

// GetEventsByType retrieves events of a specific type from the in-memory event store
func (s *InMemoryEventStore) GetEventsByType(ctx context.Context, eventType string) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Event
	for _, aggregateEvents := range s.events {
		for _, event := range aggregateEvents {
			if event.EventName() == eventType {
				result = append(result, event)
			}
		}
	}

	return result, nil
}
