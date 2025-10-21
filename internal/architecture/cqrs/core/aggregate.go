package aggregate

import (
	"errors"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/event"
	"github.com/segmentio/ksuid"
)

// Aggregate represents a domain aggregate in the CQRS pattern
type Aggregate interface {
	// ID returns the unique identifier of the aggregate
	ID() string
	
	// Type returns the type of the aggregate
	Type() string
	
	// Version returns the current version of the aggregate
	Version() int
	
	// ApplyEvent applies an event to the aggregate
	ApplyEvent(event event.Event) error
	
	// GetUncommittedEvents returns the uncommitted events of the aggregate
	GetUncommittedEvents() []event.Event
	
	// ClearUncommittedEvents clears the uncommitted events of the aggregate
	ClearUncommittedEvents()
}

// BaseAggregate provides a base implementation of the Aggregate interface
type BaseAggregate struct {
	id               string
	aggregateType    string
	version          int
	uncommittedEvents []event.Event
}

// NewBaseAggregate creates a new base aggregate
func NewBaseAggregate(aggregateType string) *BaseAggregate {
	return &BaseAggregate{
		id:               ksuid.New().String(),
		aggregateType:    aggregateType,
		version:          0,
		uncommittedEvents: []event.Event{},
	}
}

// NewBaseAggregateWithID creates a new base aggregate with a specific ID
func NewBaseAggregateWithID(id string, aggregateType string) *BaseAggregate {
	return &BaseAggregate{
		id:               id,
		aggregateType:    aggregateType,
		version:          0,
		uncommittedEvents: []event.Event{},
	}
}

// ID returns the unique identifier of the aggregate
func (a *BaseAggregate) ID() string {
	return a.id
}

// Type returns the type of the aggregate
func (a *BaseAggregate) Type() string {
	return a.aggregateType
}

// Version returns the current version of the aggregate
func (a *BaseAggregate) Version() int {
	return a.version
}

// ApplyEvent applies an event to the aggregate
func (a *BaseAggregate) ApplyEvent(event event.Event) error {
	// Check if the event is for this aggregate
	if event.AggregateID() != a.id {
		return errors.New("event aggregate ID does not match aggregate ID")
	}
	
	// Check if the event version is correct
	if event.EventVersion() != a.version+1 {
		return errors.New("event version does not match expected aggregate version")
	}
	
	// Apply the event to the aggregate
	// This is a base implementation, concrete aggregates should override this method
	
	// Increment the version
	a.version++
	
	return nil
}

// ApplyNewEvent creates and applies a new event to the aggregate
func (a *BaseAggregate) ApplyNewEvent(eventName string, data interface{}, applyFunc func(event event.Event) error) error {
	// Create a new event
	newEvent := event.NewEvent(eventName, a.id, data, a.version+1)
	
	// Apply the event to the aggregate
	if applyFunc != nil {
		if err := applyFunc(newEvent); err != nil {
			return err
		}
	} else {
		if err := a.ApplyEvent(newEvent); err != nil {
			return err
		}
	}
	
	// Add the event to the uncommitted events
	a.uncommittedEvents = append(a.uncommittedEvents, newEvent)
	
	return nil
}

// GetUncommittedEvents returns the uncommitted events of the aggregate
func (a *BaseAggregate) GetUncommittedEvents() []event.Event {
	return a.uncommittedEvents
}

// ClearUncommittedEvents clears the uncommitted events of the aggregate
func (a *BaseAggregate) ClearUncommittedEvents() {
	a.uncommittedEvents = []event.Event{}
}

// LoadFromEvents loads the aggregate state from a series of events
func (a *BaseAggregate) LoadFromEvents(events []event.Event) error {
	for _, event := range events {
		if err := a.ApplyEvent(event); err != nil {
			return err
		}
	}
	
	// Clear uncommitted events after loading
	a.ClearUncommittedEvents()
	
	return nil
}

