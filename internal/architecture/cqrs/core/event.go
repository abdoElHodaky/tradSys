package core

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
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
		ID:        generateEventID(),
		Name:      name,
		Aggregate: aggregateID,
		Timestamp: time.Now().UTC(),
		Version:   version,
		Data:      data,
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().UTC().Format("20060102150405") + "-" + string(rune(time.Now().UnixNano()%1000000))
}

// EventStoreAdapter adapts the eventsourcing.EventStore to our EventStore interface
type EventStoreAdapter struct {
	store eventsourcing.EventStore
}

// NewEventStoreAdapter creates a new event store adapter
func NewEventStoreAdapter(store eventsourcing.EventStore) *EventStoreAdapter {
	return &EventStoreAdapter{store: store}
}

// SaveEvents saves events to the event store
func (a *EventStoreAdapter) SaveEvents(ctx context.Context, events []Event) error {
	// Convert our events to eventsourcing events
	esEvents := make([]*eventsourcing.Event, len(events))
	for i, event := range events {
		esEvents[i] = &eventsourcing.Event{
			ID:            event.EventID(),
			AggregateID:   event.AggregateID(),
			EventType:     event.EventName(),
			Version:       event.EventVersion(),
			Payload:       convertToMap(event.EventData()),
			Timestamp:     event.EventTimestamp(),
			AggregateType: "", // Will be filled if aggregate type is known
		}
	}

	return a.store.SaveEvents(ctx, esEvents)
}

// GetEvents retrieves events for an aggregate from the event store
func (a *EventStoreAdapter) GetEvents(ctx context.Context, aggregateID string) ([]Event, error) {
	// Get events from the event store (need to know aggregate type and version)
	esEvents, err := a.store.GetEvents(ctx, aggregateID, "", 0)
	if err != nil {
		return nil, err
	}

	// Convert eventsourcing events to our events
	events := make([]Event, len(esEvents))
	for i, esEvent := range esEvents {
		events[i] = BaseEvent{
			ID:        esEvent.ID,
			Name:      esEvent.EventType,
			Aggregate: esEvent.AggregateID,
			Timestamp: esEvent.Timestamp,
			Version:   esEvent.Version,
			Data:      esEvent.Payload,
		}
	}

	return events, nil
}

// GetEventsByType retrieves events of a specific type from the event store
func (a *EventStoreAdapter) GetEventsByType(ctx context.Context, eventType string) ([]Event, error) {
	// Get events from the event store
	esEvents, err := a.store.GetEventsByType(ctx, eventType, time.Time{}, 0)
	if err != nil {
		return nil, err
	}

	// Convert eventsourcing events to our events
	events := make([]Event, len(esEvents))
	for i, esEvent := range esEvents {
		events[i] = BaseEvent{
			ID:        esEvent.ID,
			Name:      esEvent.EventType,
			Aggregate: esEvent.AggregateID,
			Timestamp: esEvent.Timestamp,
			Version:   esEvent.Version,
			Data:      esEvent.Payload,
		}
	}

	return events, nil
}

// convertToMap converts interface{} to map[string]interface{}
func convertToMap(data interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	if m, ok := data.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{"data": data}
}