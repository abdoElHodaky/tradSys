package eventsourcing

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"go.uber.org/zap"
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
	ApplyEvent(event *Event) error

	// GetUncommittedEvents returns the uncommitted events of the aggregate
	GetUncommittedEvents() []*Event

	// ClearUncommittedEvents clears the uncommitted events of the aggregate
	ClearUncommittedEvents()
}

// BaseAggregate provides a base implementation of the Aggregate interface
type BaseAggregate struct {
	ID                string
	Type              string
	Version           int
	UncommittedEvents []*Event
	mu                sync.RWMutex
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
func (a *BaseAggregate) ApplyEvent(event *Event) error {
	// Apply the event to the aggregate
	// This is a no-op in the base implementation
	return nil
}

// GetUncommittedEvents returns the uncommitted events of the aggregate
func (a *BaseAggregate) GetUncommittedEvents() []*Event {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy of the uncommitted events
	events := make([]*Event, len(a.UncommittedEvents))
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
	event := &Event{
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

// AggregateRepository provides a repository for aggregates
type AggregateRepository struct {
	store             EventStore
	logger            *zap.Logger
	snapshotFrequency int
}

// NewAggregateRepository creates a new aggregate repository
func NewAggregateRepository(store EventStore, logger *zap.Logger, snapshotFrequency int) *AggregateRepository {
	return &AggregateRepository{
		store:             store,
		logger:            logger,
		snapshotFrequency: snapshotFrequency,
	}
}

// Load loads an aggregate from the event store
func (r *AggregateRepository) Load(ctx context.Context, aggregateID string, aggregateType string, aggregate Aggregate) error {
	// Get events for the aggregate
	events, err := r.store.GetEvents(ctx, aggregateID, aggregateType, 0)
	if err != nil {
		return err
	}

	// Apply events to the aggregate
	for _, event := range events {
		err := aggregate.ApplyEvent(event)
		if err != nil {
			return err
		}

		// Update the aggregate version
		aggregate.SetVersion(event.Version)
	}

	return nil
}

// LoadWithSnapshot loads an aggregate from the event store using a snapshot
func (r *AggregateRepository) LoadWithSnapshot(ctx context.Context, aggregateID string, aggregateType string, aggregate Aggregate, snapshotStore interface{}) error {
	// Try to get a snapshot
	snapshot, version, err := r.getLatestSnapshot(snapshotStore, aggregateID)
	if err != nil && !errors.Is(err, ErrSnapshotNotFound) {
		return err
	}

	// Apply the snapshot if found
	if snapshot != nil {
		// Apply the snapshot to the aggregate
		err := r.applySnapshot(aggregate, snapshot)
		if err != nil {
			return err
		}

		// Set the aggregate version
		aggregate.SetVersion(version)
	}

	// Get events after the snapshot
	events, err := r.store.GetEvents(ctx, aggregateID, aggregateType, version)
	if err != nil {
		return err
	}

	// Apply events to the aggregate
	for _, event := range events {
		err := aggregate.ApplyEvent(event)
		if err != nil {
			return err
		}

		// Update the aggregate version
		aggregate.SetVersion(event.Version)
	}

	return nil
}

// Save saves an aggregate to the event store
func (r *AggregateRepository) Save(ctx context.Context, aggregate Aggregate) error {
	// Get uncommitted events
	events := aggregate.GetUncommittedEvents()
	if len(events) == 0 {
		return nil
	}

	// Save events to the store
	err := r.store.SaveEvents(ctx, events)
	if err != nil {
		return err
	}

	// Clear uncommitted events
	aggregate.ClearUncommittedEvents()

	// Check if a snapshot should be created
	if r.snapshotFrequency > 0 && aggregate.GetVersion()%r.snapshotFrequency == 0 {
		// Create a snapshot
		err := r.createSnapshot(aggregate)
		if err != nil {
			r.logger.Error("Failed to create snapshot",
				zap.String("aggregate_id", aggregate.GetID()),
				zap.String("aggregate_type", aggregate.GetType()),
				zap.Int("version", aggregate.GetVersion()),
				zap.Error(err))
		}
	}

	return nil
}

// getLatestSnapshot gets the latest snapshot of an aggregate
func (r *AggregateRepository) getLatestSnapshot(snapshotStore interface{}, aggregateID string) (interface{}, int, error) {
	// Check if the snapshot store implements the required method
	if store, ok := snapshotStore.(interface {
		GetLatestSnapshot(aggregateID string) (interface{}, int, error)
	}); ok {
		return store.GetLatestSnapshot(aggregateID)
	}

	return nil, 0, ErrSnapshotNotFound
}

// applySnapshot applies a snapshot to an aggregate
func (r *AggregateRepository) applySnapshot(aggregate Aggregate, snapshot interface{}) error {
	// Check if the aggregate implements the required method
	if agg, ok := aggregate.(interface {
		ApplySnapshot(snapshot interface{}) error
	}); ok {
		return agg.ApplySnapshot(snapshot)
	}

	// Try to apply the snapshot using reflection
	aggregateValue := reflect.ValueOf(aggregate).Elem()
	snapshotValue := reflect.ValueOf(snapshot).Elem()

	// Check if the types are compatible
	if aggregateValue.Type() != snapshotValue.Type() {
		return fmt.Errorf("snapshot type %s is not compatible with aggregate type %s", snapshotValue.Type(), aggregateValue.Type())
	}

	// Copy the snapshot to the aggregate
	aggregateValue.Set(snapshotValue)

	return nil
}

// createSnapshot creates a snapshot of an aggregate
func (r *AggregateRepository) createSnapshot(aggregate Aggregate) error {
	// Check if the aggregate implements the required method
	if agg, ok := aggregate.(interface {
		CreateSnapshot() (interface{}, error)
	}); ok {
		// Create the snapshot
		snapshot, err := agg.CreateSnapshot()
		if err != nil {
			return err
		}

		// Save the snapshot
		if store, ok := r.store.(interface {
			SaveSnapshot(aggregateID string, version int, snapshot interface{}) error
		}); ok {
			return store.SaveSnapshot(aggregate.GetID(), aggregate.GetVersion(), snapshot)
		}
	}

	return nil
}
