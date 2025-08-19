package eventsourcing

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Event represents an event in the event sourcing pattern
type Event struct {
	ID            string                 `json:"id"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	EventType     string                 `json:"event_type"`
	Version       int                    `json:"version"`
	Timestamp     time.Time              `json:"timestamp"`
	Payload       map[string]interface{} `json:"payload"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// EventStore provides event storage functionality
type EventStore interface {
	// SaveEvents saves events to the store
	SaveEvents(ctx context.Context, events []*Event) error

	// GetEvents gets events for an aggregate
	GetEvents(ctx context.Context, aggregateID string, aggregateType string, fromVersion int) ([]*Event, error)

	// GetEventsByType gets events by type
	GetEventsByType(ctx context.Context, eventType string, fromTimestamp time.Time, limit int) ([]*Event, error)

	// GetAggregateEvents gets events for multiple aggregates
	GetAggregateEvents(ctx context.Context, aggregateIDs []string, aggregateType string, fromVersion int) ([]*Event, error)
}

// InMemoryEventStore provides an in-memory event store
type InMemoryEventStore struct {
	events     []*Event
	mu         sync.RWMutex
	logger     *zap.Logger
	snapshots  map[string]map[int]interface{}
	snapshotMu sync.RWMutex
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(logger *zap.Logger) *InMemoryEventStore {
	return &InMemoryEventStore{
		events:    make([]*Event, 0),
		logger:    logger,
		snapshots: make(map[string]map[int]interface{}),
	}
}

// SaveEvents saves events to the store
func (s *InMemoryEventStore) SaveEvents(ctx context.Context, events []*Event) error {
	if len(events) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for optimistic concurrency conflicts
	aggregateID := events[0].AggregateID
	aggregateType := events[0].AggregateType
	expectedVersion := events[0].Version - 1

	// Get the current version of the aggregate
	currentVersion := 0
	for _, event := range s.events {
		if event.AggregateID == aggregateID && event.AggregateType == aggregateType && event.Version > currentVersion {
			currentVersion = event.Version
		}
	}

	// Check for concurrency conflicts
	if currentVersion != expectedVersion {
		return ErrConcurrencyConflict
	}

	// Add the events to the store
	for _, event := range events {
		// Generate an ID if not provided
		if event.ID == "" {
			event.ID = uuid.New().String()
		}

		// Set the timestamp if not provided
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}

		s.events = append(s.events, event)
	}

	return nil
}

// GetEvents gets events for an aggregate
func (s *InMemoryEventStore) GetEvents(ctx context.Context, aggregateID string, aggregateType string, fromVersion int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*Event, 0)
	for _, event := range s.events {
		if event.AggregateID == aggregateID && event.AggregateType == aggregateType && event.Version > fromVersion {
			events = append(events, event)
		}
	}

	return events, nil
}

// GetEventsByType gets events by type
func (s *InMemoryEventStore) GetEventsByType(ctx context.Context, eventType string, fromTimestamp time.Time, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*Event, 0)
	for _, event := range s.events {
		if event.EventType == eventType && event.Timestamp.After(fromTimestamp) {
			events = append(events, event)
			if limit > 0 && len(events) >= limit {
				break
			}
		}
	}

	return events, nil
}

// GetAggregateEvents gets events for multiple aggregates
func (s *InMemoryEventStore) GetAggregateEvents(ctx context.Context, aggregateIDs []string, aggregateType string, fromVersion int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a map for faster lookup
	idMap := make(map[string]bool)
	for _, id := range aggregateIDs {
		idMap[id] = true
	}

	events := make([]*Event, 0)
	for _, event := range s.events {
		if idMap[event.AggregateID] && event.AggregateType == aggregateType && event.Version > fromVersion {
			events = append(events, event)
		}
	}

	return events, nil
}

// SaveSnapshot saves a snapshot of an aggregate
func (s *InMemoryEventStore) SaveSnapshot(aggregateID string, version int, snapshot interface{}) error {
	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()

	// Create the aggregate map if it doesn't exist
	if _, ok := s.snapshots[aggregateID]; !ok {
		s.snapshots[aggregateID] = make(map[int]interface{})
	}

	// Save the snapshot
	s.snapshots[aggregateID][version] = snapshot

	return nil
}

// GetLatestSnapshot gets the latest snapshot of an aggregate
func (s *InMemoryEventStore) GetLatestSnapshot(aggregateID string) (interface{}, int, error) {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()

	// Check if the aggregate has snapshots
	aggregateSnapshots, ok := s.snapshots[aggregateID]
	if !ok {
		return nil, 0, ErrSnapshotNotFound
	}

	// Find the latest version
	latestVersion := 0
	for version := range aggregateSnapshots {
		if version > latestVersion {
			latestVersion = version
		}
	}

	// Check if a snapshot was found
	if latestVersion == 0 {
		return nil, 0, ErrSnapshotNotFound
	}

	return aggregateSnapshots[latestVersion], latestVersion, nil
}

// BatchEventStore provides a batched event store
type BatchEventStore struct {
	store       EventStore
	batchSize   int
	batchEvents map[string][]*Event
	mu          sync.Mutex
	logger      *zap.Logger
}

// NewBatchEventStore creates a new batched event store
func NewBatchEventStore(store EventStore, batchSize int, logger *zap.Logger) *BatchEventStore {
	return &BatchEventStore{
		store:       store,
		batchSize:   batchSize,
		batchEvents: make(map[string][]*Event),
		logger:      logger,
	}
}

// SaveEvent saves an event to the store
func (s *BatchEventStore) SaveEvent(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add the event to the batch
	key := event.AggregateType + ":" + event.AggregateID
	s.batchEvents[key] = append(s.batchEvents[key], event)

	// Check if the batch is full
	if len(s.batchEvents[key]) >= s.batchSize {
		// Save the batch
		err := s.store.SaveEvents(ctx, s.batchEvents[key])
		if err != nil {
			return err
		}

		// Clear the batch
		s.batchEvents[key] = nil
	}

	return nil
}

// Flush flushes all batched events to the store
func (s *BatchEventStore) Flush(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Save all batched events
	for key, events := range s.batchEvents {
		if len(events) > 0 {
			err := s.store.SaveEvents(ctx, events)
			if err != nil {
				return err
			}

			// Clear the batch
			s.batchEvents[key] = nil
		}
	}

	return nil
}

// GetEvents gets events for an aggregate
func (s *BatchEventStore) GetEvents(ctx context.Context, aggregateID string, aggregateType string, fromVersion int) ([]*Event, error) {
	// Flush any pending events
	err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	// Get events from the store
	return s.store.GetEvents(ctx, aggregateID, aggregateType, fromVersion)
}

// GetEventsByType gets events by type
func (s *BatchEventStore) GetEventsByType(ctx context.Context, eventType string, fromTimestamp time.Time, limit int) ([]*Event, error) {
	// Flush any pending events
	err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	// Get events from the store
	return s.store.GetEventsByType(ctx, eventType, fromTimestamp, limit)
}

// GetAggregateEvents gets events for multiple aggregates
func (s *BatchEventStore) GetAggregateEvents(ctx context.Context, aggregateIDs []string, aggregateType string, fromVersion int) ([]*Event, error) {
	// Flush any pending events
	err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	// Get events from the store
	return s.store.GetAggregateEvents(ctx, aggregateIDs, aggregateType, fromVersion)
}

// ErrConcurrencyConflict is returned when there is a concurrency conflict
var ErrConcurrencyConflict = errors.New("concurrency conflict")

// ErrSnapshotNotFound is returned when a snapshot is not found
var ErrSnapshotNotFound = errors.New("snapshot not found")

