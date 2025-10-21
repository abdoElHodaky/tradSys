package core

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// InMemoryEventStore provides an in-memory event store
type InMemoryEventStore struct {
	events     []*eventsourcing.Event
	mu         sync.RWMutex
	logger     *zap.Logger
	snapshots  map[string]map[string]map[int]interface{} // aggregateType -> aggregateID -> version -> snapshot
	snapshotMu sync.RWMutex
	
	// Cache settings
	cacheEnabled bool
	cacheSize    int
	cacheTTL     time.Duration
	
	// Snapshot settings
	snapshotFrequency int
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(logger *zap.Logger, options ...StoreOption) *InMemoryEventStore {
	store := &InMemoryEventStore{
		events:            make([]*eventsourcing.Event, 0),
		logger:            logger,
		snapshots:         make(map[string]map[string]map[int]interface{}),
		cacheEnabled:      true,
		cacheSize:         1000,
		cacheTTL:          5 * time.Minute,
		snapshotFrequency: 100,
	}
	
	// Apply options
	for _, option := range options {
		option(store)
	}
	
	return store
}

// SetCacheSize sets the cache size
func (s *InMemoryEventStore) SetCacheSize(size int) {
	s.cacheSize = size
}

// SetCacheTTL sets the cache TTL
func (s *InMemoryEventStore) SetCacheTTL(ttl time.Duration) {
	s.cacheTTL = ttl
}

// SetSnapshotFrequency sets the snapshot frequency
func (s *InMemoryEventStore) SetSnapshotFrequency(frequency int) {
	s.snapshotFrequency = frequency
}

// SaveEvents saves events to the store
func (s *InMemoryEventStore) SaveEvents(ctx context.Context, events []*eventsourcing.Event) error {
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
func (s *InMemoryEventStore) GetEvents(ctx context.Context, aggregateID string, aggregateType string, fromVersion int) ([]*eventsourcing.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*eventsourcing.Event, 0)
	for _, event := range s.events {
		if event.AggregateID == aggregateID && event.AggregateType == aggregateType && event.Version > fromVersion {
			events = append(events, event)
		}
	}

	return events, nil
}

// GetEventsByType gets events by type
func (s *InMemoryEventStore) GetEventsByType(ctx context.Context, eventType string, fromTimestamp time.Time, limit int) ([]*eventsourcing.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*eventsourcing.Event, 0)
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
func (s *InMemoryEventStore) GetAggregateEvents(ctx context.Context, aggregateIDs []string, aggregateType string, fromVersion int) ([]*eventsourcing.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a map for faster lookup
	idMap := make(map[string]bool)
	for _, id := range aggregateIDs {
		idMap[id] = true
	}

	events := make([]*eventsourcing.Event, 0)
	for _, event := range s.events {
		if idMap[event.AggregateID] && event.AggregateType == aggregateType && event.Version > fromVersion {
			events = append(events, event)
		}
	}

	return events, nil
}

// GetAllEvents gets all events
func (s *InMemoryEventStore) GetAllEvents(ctx context.Context, fromTimestamp time.Time, limit int) ([]*eventsourcing.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*eventsourcing.Event, 0)
	for _, event := range s.events {
		if event.Timestamp.After(fromTimestamp) {
			events = append(events, event)
			if limit > 0 && len(events) >= limit {
				break
			}
		}
	}

	return events, nil
}

// SaveSnapshot saves a snapshot of an aggregate
func (s *InMemoryEventStore) SaveSnapshot(ctx context.Context, aggregateID string, aggregateType string, version int, snapshot interface{}) error {
	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()

	// Create the aggregate type map if it doesn't exist
	if _, ok := s.snapshots[aggregateType]; !ok {
		s.snapshots[aggregateType] = make(map[string]map[int]interface{})
	}

	// Create the aggregate ID map if it doesn't exist
	if _, ok := s.snapshots[aggregateType][aggregateID]; !ok {
		s.snapshots[aggregateType][aggregateID] = make(map[int]interface{})
	}

	// Save the snapshot
	s.snapshots[aggregateType][aggregateID][version] = snapshot

	return nil
}

// GetLatestSnapshot gets the latest snapshot of an aggregate
func (s *InMemoryEventStore) GetLatestSnapshot(ctx context.Context, aggregateID string, aggregateType string) (interface{}, int, error) {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()

	// Check if the aggregate type has snapshots
	aggregateTypeSnapshots, ok := s.snapshots[aggregateType]
	if !ok {
		return nil, 0, ErrSnapshotNotFound
	}

	// Check if the aggregate ID has snapshots
	aggregateSnapshots, ok := aggregateTypeSnapshots[aggregateID]
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

