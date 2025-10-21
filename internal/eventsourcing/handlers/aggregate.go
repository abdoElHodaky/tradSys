package aggregate

import (
	"errors"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
)

// Aggregate represents an aggregate in the event sourcing pattern
type Aggregate interface {
	// GetID returns the ID of the aggregate
	GetID() string

	// GetType returns the type of the aggregate
	GetType() string

	// GetVersion returns the version of the aggregate
	GetVersion() int

	// SetVersion sets the version of the aggregate
	SetVersion(version int)

	// ApplyEvent applies an event to the aggregate
	ApplyEvent(event *eventsourcing.Event) error

	// GetUncommittedEvents returns the uncommitted events of the aggregate
	GetUncommittedEvents() []*eventsourcing.Event

	// ClearUncommittedEvents clears the uncommitted events of the aggregate
	ClearUncommittedEvents()
}

// Snapshottable represents an aggregate that can create and apply snapshots
type Snapshottable interface {
	// CreateSnapshot creates a snapshot of the aggregate
	CreateSnapshot() (interface{}, error)
	
	// ApplySnapshot applies a snapshot to the aggregate
	ApplySnapshot(snapshot interface{}) error
}

// BaseAggregate provides a base implementation of the Aggregate interface
type BaseAggregate struct {
	ID               string
	Type             string
	Version          int
	UncommittedEvents []*eventsourcing.Event
	mu               sync.RWMutex
}

// NewBaseAggregate creates a new base aggregate
func NewBaseAggregate(id string, aggregateType string) *BaseAggregate {
	return &BaseAggregate{
		ID:                id,
		Type:              aggregateType,
		Version:           0,
		UncommittedEvents: make([]*eventsourcing.Event, 0),
	}
}

// GetID returns the ID of the aggregate
func (a *BaseAggregate) GetID() string {
	return a.ID
}

// GetType returns the type of the aggregate
func (a *BaseAggregate) GetType() string {
	return a.Type
}

// GetVersion returns the version of the aggregate
func (a *BaseAggregate) GetVersion() int {
	return a.Version
}

// SetVersion sets the version of the aggregate
func (a *BaseAggregate) SetVersion(version int) {
	a.Version = version
}

// ApplyEvent applies an event to the aggregate
func (a *BaseAggregate) ApplyEvent(event *eventsourcing.Event) error {
	// Apply the event to the aggregate
	// This is a no-op in the base implementation
	return nil
}

// GetUncommittedEvents returns the uncommitted events of the aggregate
func (a *BaseAggregate) GetUncommittedEvents() []*eventsourcing.Event {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy of the uncommitted events
	events := make([]*eventsourcing.Event, len(a.UncommittedEvents))
	copy(events, a.UncommittedEvents)

	return events
}

// ClearUncommittedEvents clears the uncommitted events of the aggregate
func (a *BaseAggregate) ClearUncommittedEvents() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.UncommittedEvents = nil
}

// AddEvent adds an event to the aggregate
func (a *BaseAggregate) AddEvent(eventType string, payload map[string]interface{}, metadata map[string]interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create the event
	event := &eventsourcing.Event{
		AggregateID:   a.ID,
		AggregateType: a.Type,
		EventType:     eventType,
		Version:       a.Version + 1,
		Payload:       payload,
		Metadata:      metadata,
	}

	// Add the event to the uncommitted events
	a.UncommittedEvents = append(a.UncommittedEvents, event)

	// Increment the version
	a.Version++
}

// Common errors
var (
	ErrAggregateNotFound = errors.New("aggregate not found")
	ErrInvalidAggregate  = errors.New("invalid aggregate")
	ErrInvalidEvent      = errors.New("invalid event")
	ErrInvalidSnapshot   = errors.New("invalid snapshot")
)

