package handlers

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
	"go.uber.org/zap"
)

// Repository provides a repository for aggregates
type Repository interface {
	// Load loads an aggregate from the event store
	Load(ctx context.Context, aggregateID string, aggregate Aggregate) error
	
	// Save saves an aggregate to the event store
	Save(ctx context.Context, aggregate Aggregate) error
}

// RepositoryWithSnapshots provides a repository for aggregates with snapshot support
type RepositoryWithSnapshots interface {
	Repository
	
	// LoadWithSnapshot loads an aggregate from the event store using a snapshot
	LoadWithSnapshot(ctx context.Context, aggregateID string, aggregate Aggregate) error
}

// EventSourcedRepository provides an event-sourced repository for aggregates
type EventSourcedRepository struct {
	store             store.EventStore
	logger            *zap.Logger
	snapshotFrequency int
	aggregateTypes    map[string]reflect.Type
	eventHandlers     map[string]map[string]func(aggregate Aggregate, event *eventsourcing.Event) error
}

// EventSourcedRepositoryOption represents an option for configuring an event-sourced repository
type EventSourcedRepositoryOption func(*EventSourcedRepository)

// WithSnapshotFrequency sets the snapshot frequency for an event-sourced repository
func WithSnapshotFrequency(frequency int) EventSourcedRepositoryOption {
	return func(r *EventSourcedRepository) {
		r.snapshotFrequency = frequency
	}
}

// WithAggregateType registers an aggregate type with an event-sourced repository
func WithAggregateType(aggregateType string, aggregateFactory func() Aggregate) EventSourcedRepositoryOption {
	return func(r *EventSourcedRepository) {
		aggregate := aggregateFactory()
		r.aggregateTypes[aggregateType] = reflect.TypeOf(aggregate).Elem()
	}
}

// WithEventHandler registers an event handler with an event-sourced repository
func WithEventHandler(aggregateType string, eventType string, handler func(aggregate Aggregate, event *eventsourcing.Event) error) EventSourcedRepositoryOption {
	return func(r *EventSourcedRepository) {
		if _, ok := r.eventHandlers[aggregateType]; !ok {
			r.eventHandlers[aggregateType] = make(map[string]func(aggregate Aggregate, event *eventsourcing.Event) error)
		}
		r.eventHandlers[aggregateType][eventType] = handler
	}
}

// NewEventSourcedRepository creates a new event-sourced repository
func NewEventSourcedRepository(eventStore store.EventStore, logger *zap.Logger, options ...EventSourcedRepositoryOption) *EventSourcedRepository {
	repo := &EventSourcedRepository{
		store:             eventStore,
		logger:            logger,
		snapshotFrequency: 100,
		aggregateTypes:    make(map[string]reflect.Type),
		eventHandlers:     make(map[string]map[string]func(aggregate Aggregate, event *eventsourcing.Event) error),
	}
	
	// Apply options
	for _, option := range options {
		option(repo)
	}
	
	return repo
}

// Load loads an aggregate from the event store
func (r *EventSourcedRepository) Load(ctx context.Context, aggregateID string, aggregate Aggregate) error {
	// Get events for the aggregate
	events, err := r.store.GetEvents(ctx, aggregateID, aggregate.GetType(), 0)
	if err != nil {
		return err
	}
	
	// Check if the aggregate exists
	if len(events) == 0 {
		return ErrAggregateNotFound
	}

	// Apply events to the aggregate
	for _, event := range events {
		err := r.applyEventToAggregate(aggregate, event)
		if err != nil {
			return err
		}

		// Update the aggregate version
		aggregate.SetVersion(event.Version)
	}

	return nil
}

// LoadWithSnapshot loads an aggregate from the event store using a snapshot
func (r *EventSourcedRepository) LoadWithSnapshot(ctx context.Context, aggregateID string, aggregate Aggregate) error {
	// Check if the store supports snapshots
	snapshotStore, ok := r.store.(store.SnapshotStore)
	if !ok {
		// Fall back to regular loading
		return r.Load(ctx, aggregateID, aggregate)
	}
	
	// Check if the aggregate supports snapshots
	snapshottable, ok := aggregate.(Snapshottable)
	if !ok {
		// Fall back to regular loading
		return r.Load(ctx, aggregateID, aggregate)
	}

	// Try to get a snapshot
	snapshot, version, err := snapshotStore.GetLatestSnapshot(ctx, aggregateID, aggregate.GetType())
	if err != nil && !errors.Is(err, store.ErrSnapshotNotFound) {
		return err
	}

	// Apply the snapshot if found
	if snapshot != nil {
		// Apply the snapshot to the aggregate
		err := snapshottable.ApplySnapshot(snapshot)
		if err != nil {
			return err
		}

		// Set the aggregate version
		aggregate.SetVersion(version)
	}

	// Get events after the snapshot
	events, err := r.store.GetEvents(ctx, aggregateID, aggregate.GetType(), version)
	if err != nil {
		return err
	}
	
	// Check if the aggregate exists
	if len(events) == 0 && version == 0 {
		return ErrAggregateNotFound
	}

	// Apply events to the aggregate
	for _, event := range events {
		err := r.applyEventToAggregate(aggregate, event)
		if err != nil {
			return err
		}

		// Update the aggregate version
		aggregate.SetVersion(event.Version)
	}

	return nil
}

// Save saves an aggregate to the event store
func (r *EventSourcedRepository) Save(ctx context.Context, aggregate Aggregate) error {
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
	if r.snapshotFrequency > 0 && aggregate.GetVersion() % r.snapshotFrequency == 0 {
		// Check if the store supports snapshots
		snapshotStore, ok := r.store.(store.SnapshotStore)
		if !ok {
			return nil
		}
		
		// Check if the aggregate supports snapshots
		snapshottable, ok := aggregate.(Snapshottable)
		if !ok {
			return nil
		}
		
		// Create a snapshot
		snapshot, err := snapshottable.CreateSnapshot()
		if err != nil {
			r.logger.Error("Failed to create snapshot",
				zap.String("aggregate_id", aggregate.GetID()),
				zap.String("aggregate_type", aggregate.GetType()),
				zap.Int("version", aggregate.GetVersion()),
				zap.Error(err))
			return nil
		}
		
		// Save the snapshot
		err = snapshotStore.SaveSnapshot(ctx, aggregate.GetID(), aggregate.GetType(), aggregate.GetVersion(), snapshot)
		if err != nil {
			r.logger.Error("Failed to save snapshot",
				zap.String("aggregate_id", aggregate.GetID()),
				zap.String("aggregate_type", aggregate.GetType()),
				zap.Int("version", aggregate.GetVersion()),
				zap.Error(err))
		}
	}

	return nil
}

// applyEventToAggregate applies an event to an aggregate
func (r *EventSourcedRepository) applyEventToAggregate(aggregate Aggregate, event *eventsourcing.Event) error {
	// Check if there's a registered handler for this event type
	if handlers, ok := r.eventHandlers[aggregate.GetType()]; ok {
		if handler, ok := handlers[event.EventType]; ok {
			return handler(aggregate, event)
		}
	}
	
	// Fall back to the aggregate's ApplyEvent method
	return aggregate.ApplyEvent(event)
}

// CreateAggregate creates a new aggregate of the specified type
func (r *EventSourcedRepository) CreateAggregate(aggregateType string, aggregateID string) (Aggregate, error) {
	// Check if the aggregate type is registered
	aggregateReflectType, ok := r.aggregateTypes[aggregateType]
	if !ok {
		return nil, fmt.Errorf("aggregate type %s is not registered", aggregateType)
	}
	
	// Create a new aggregate
	aggregateValue := reflect.New(aggregateReflectType)
	aggregate, ok := aggregateValue.Interface().(Aggregate)
	if !ok {
		return nil, fmt.Errorf("failed to create aggregate of type %s", aggregateType)
	}
	
	// Initialize the aggregate
	if initializer, ok := aggregate.(interface{ Initialize(string) }); ok {
		initializer.Initialize(aggregateID)
	}
	
	return aggregate, nil
}

