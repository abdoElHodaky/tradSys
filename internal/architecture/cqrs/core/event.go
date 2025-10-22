package core

import (
	"context"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/thefabric-io/eventsourcing"
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
	ID          string
	Name        string
	Aggregate   string
	Timestamp   time.Time
	Version     int
	Data        interface{}
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
		ID:          ksuid.New().String(),
		Name:        name,
		Aggregate:   aggregateID,
		Timestamp:   time.Now().UTC(),
		Version:     version,
		Data:        data,
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

// PostgresEventStore implements the EventStore interface using PostgreSQL
type PostgresEventStore struct {
	store *eventsourcing.EventStore
}

// NewPostgresEventStore creates a new PostgreSQL event store
func NewPostgresEventStore(store *eventsourcing.EventStore) *PostgresEventStore {
	return &PostgresEventStore{
		store: store,
	}
}

// SaveEvents saves events to the PostgreSQL event store
func (s *PostgresEventStore) SaveEvents(ctx context.Context, events []Event) error {
	// Convert our events to thefabric-io/eventsourcing events
	esEvents := make([]eventsourcing.Event, len(events))
	for i, event := range events {
		esEvents[i] = eventsourcing.Event{
			ID:          event.EventID(),
			AggregateID: event.AggregateID(),
			Type:        event.EventName(),
			Version:     event.EventVersion(),
			Payload:     event.EventData(),
			CreatedAt:   event.EventTimestamp(),
		}
	}
	
	// Save events to the event store
	return s.store.Save(ctx, esEvents)
}

// GetEvents retrieves events for an aggregate from the PostgreSQL event store
func (s *PostgresEventStore) GetEvents(ctx context.Context, aggregateID string) ([]Event, error) {
	// Get events from the event store
	esEvents, err := s.store.GetByAggregateID(ctx, aggregateID)
	if err != nil {
		return nil, err
	}
	
	// Convert thefabric-io/eventsourcing events to our events
	events := make([]Event, len(esEvents))
	for i, esEvent := range esEvents {
		events[i] = BaseEvent{
			ID:          esEvent.ID,
			Name:        esEvent.Type,
			Aggregate:   esEvent.AggregateID,
			Timestamp:   esEvent.CreatedAt,
			Version:     esEvent.Version,
			Data:        esEvent.Payload,
		}
	}
	
	return events, nil
}

// GetEventsByType retrieves events of a specific type from the PostgreSQL event store
func (s *PostgresEventStore) GetEventsByType(ctx context.Context, eventType string) ([]Event, error) {
	// Get events from the event store
	esEvents, err := s.store.GetByType(ctx, eventType)
	if err != nil {
		return nil, err
	}
	
	// Convert thefabric-io/eventsourcing events to our events
	events := make([]Event, len(esEvents))
	for i, esEvent := range esEvents {
		events[i] = BaseEvent{
			ID:          esEvent.ID,
			Name:        esEvent.Type,
			Aggregate:   esEvent.AggregateID,
			Timestamp:   esEvent.CreatedAt,
			Version:     esEvent.Version,
			Data:        esEvent.Payload,
		}
	}
	
	return events, nil
}
