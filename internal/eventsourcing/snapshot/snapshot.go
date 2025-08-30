package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	"go.uber.org/zap"
)

// SnapshotConfig represents the configuration for the snapshot system
type SnapshotConfig struct {
	// Frequency is how often snapshots are created
	Frequency time.Duration
	// EventThreshold is the number of events after which a snapshot is created
	EventThreshold int
	// Retention is how long snapshots are kept
	Retention time.Duration
	// MaxSnapshotsPerAggregate is the maximum number of snapshots kept per aggregate
	MaxSnapshotsPerAggregate int
}

// DefaultSnapshotConfig returns the default snapshot configuration
func DefaultSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{
		Frequency:               1 * time.Hour,
		EventThreshold:          100,
		Retention:               30 * 24 * time.Hour, // 30 days
		MaxSnapshotsPerAggregate: 10,
	}
}

// Snapshot represents a point-in-time state of an aggregate
type Snapshot struct {
	ID           string
	AggregateID  string
	AggregateType string
	Version      int
	State        []byte
	CreatedAt    time.Time
	EventCount   int
}

// SnapshotStore is the interface for storing and retrieving snapshots
type SnapshotStore interface {
	// Save saves a snapshot
	Save(ctx context.Context, snapshot *Snapshot) error
	// Get gets the latest snapshot for an aggregate
	Get(ctx context.Context, aggregateID string) (*Snapshot, error)
	// GetByVersion gets a snapshot for an aggregate at a specific version
	GetByVersion(ctx context.Context, aggregateID string, version int) (*Snapshot, error)
	// List lists all snapshots for an aggregate
	List(ctx context.Context, aggregateID string) ([]*Snapshot, error)
	// Delete deletes a snapshot
	Delete(ctx context.Context, snapshotID string) error
	// DeleteByAggregateID deletes all snapshots for an aggregate
	DeleteByAggregateID(ctx context.Context, aggregateID string) error
	// Prune deletes old snapshots based on retention policy
	Prune(ctx context.Context, retention time.Duration) error
}

// SnapshotManager manages the creation and retrieval of snapshots
type SnapshotManager struct {
	config       SnapshotConfig
	snapshotStore SnapshotStore
	eventStore   store.EventStore
	logger       *zap.Logger
	
	// Snapshot handlers by aggregate type
	handlers     map[string]SnapshotHandler
	mu           sync.RWMutex
}

// SnapshotHandler is the interface for creating and applying snapshots for a specific aggregate type
type SnapshotHandler interface {
	// AggregateType returns the type of aggregate this handler is for
	AggregateType() string
	// CreateSnapshot creates a snapshot from an aggregate
	CreateSnapshot(aggregate eventsourcing.Aggregate) (*Snapshot, error)
	// ApplySnapshot applies a snapshot to an aggregate
	ApplySnapshot(snapshot *Snapshot, aggregate eventsourcing.Aggregate) error
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(config SnapshotConfig, snapshotStore SnapshotStore, eventStore store.EventStore, logger *zap.Logger) *SnapshotManager {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	
	return &SnapshotManager{
		config:       config,
		snapshotStore: snapshotStore,
		eventStore:   eventStore,
		logger:       logger.With(zap.String("component", "snapshot_manager")),
		handlers:     make(map[string]SnapshotHandler),
	}
}

// RegisterHandler registers a snapshot handler for an aggregate type
func (m *SnapshotManager) RegisterHandler(handler SnapshotHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	aggregateType := handler.AggregateType()
	m.handlers[aggregateType] = handler
	m.logger.Info("Registered snapshot handler", zap.String("aggregate_type", aggregateType))
}

// GetHandler gets a snapshot handler for an aggregate type
func (m *SnapshotManager) GetHandler(aggregateType string) (SnapshotHandler, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	handler, ok := m.handlers[aggregateType]
	return handler, ok
}

// CreateSnapshot creates a snapshot for an aggregate
func (m *SnapshotManager) CreateSnapshot(ctx context.Context, aggregate eventsourcing.Aggregate) (*Snapshot, error) {
	aggregateType := aggregate.AggregateType()
	
	handler, ok := m.GetHandler(aggregateType)
	if !ok {
		return nil, fmt.Errorf("no snapshot handler registered for aggregate type: %s", aggregateType)
	}
	
	snapshot, err := handler.CreateSnapshot(aggregate)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}
	
	if err := m.snapshotStore.Save(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}
	
	m.logger.Info("Created snapshot",
		zap.String("aggregate_id", aggregate.AggregateID()),
		zap.String("aggregate_type", aggregateType),
		zap.Int("version", snapshot.Version))
	
	return snapshot, nil
}

// GetLatestSnapshot gets the latest snapshot for an aggregate
func (m *SnapshotManager) GetLatestSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error) {
	snapshot, err := m.snapshotStore.Get(ctx, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	
	return snapshot, nil
}

// ApplySnapshot applies a snapshot to an aggregate
func (m *SnapshotManager) ApplySnapshot(ctx context.Context, snapshot *Snapshot, aggregate eventsourcing.Aggregate) error {
	handler, ok := m.GetHandler(snapshot.AggregateType)
	if !ok {
		return fmt.Errorf("no snapshot handler registered for aggregate type: %s", snapshot.AggregateType)
	}
	
	if err := handler.ApplySnapshot(snapshot, aggregate); err != nil {
		return fmt.Errorf("failed to apply snapshot: %w", err)
	}
	
	m.logger.Info("Applied snapshot",
		zap.String("aggregate_id", aggregate.AggregateID()),
		zap.String("aggregate_type", aggregate.AggregateType()),
		zap.Int("version", snapshot.Version))
	
	return nil
}

// ShouldCreateSnapshot determines if a snapshot should be created based on the configuration
func (m *SnapshotManager) ShouldCreateSnapshot(eventCount int, lastSnapshotTime time.Time) bool {
	// Check if event threshold is reached
	if eventCount >= m.config.EventThreshold {
		return true
	}
	
	// Check if frequency threshold is reached
	if time.Since(lastSnapshotTime) >= m.config.Frequency {
		return true
	}
	
	return false
}

// PruneSnapshots prunes old snapshots based on retention policy
func (m *SnapshotManager) PruneSnapshots(ctx context.Context) error {
	return m.snapshotStore.Prune(ctx, m.config.Retention)
}

// CreateSnapshotsForAllAggregates creates snapshots for all aggregates
func (m *SnapshotManager) CreateSnapshotsForAllAggregates(ctx context.Context) error {
	m.mu.RLock()
	handlers := make([]SnapshotHandler, 0, len(m.handlers))
	for _, handler := range m.handlers {
		handlers = append(handlers, handler)
	}
	m.mu.RUnlock()
	
	for _, handler := range handlers {
		aggregateType := handler.AggregateType()
		
		// Get all events for this aggregate type
		events, err := m.eventStore.GetByAggregateType(ctx, aggregateType)
		if err != nil {
			m.logger.Error("Failed to get events for aggregate type",
				zap.String("aggregate_type", aggregateType),
				zap.Error(err))
			continue
		}
		
		// Group events by aggregate ID
		eventsByAggregateID := make(map[string][]*eventsourcing.Event)
		for _, event := range events {
			eventsByAggregateID[event.AggregateID] = append(eventsByAggregateID[event.AggregateID], event)
		}
		
		// Create snapshots for each aggregate
		for aggregateID, aggregateEvents := range eventsByAggregateID {
			// Get the latest snapshot for this aggregate
			latestSnapshot, err := m.snapshotStore.Get(ctx, aggregateID)
			if err != nil && err.Error() != "snapshot not found" {
				m.logger.Error("Failed to get latest snapshot",
					zap.String("aggregate_id", aggregateID),
					zap.Error(err))
				continue
			}
			
			var lastSnapshotTime time.Time
			var lastSnapshotVersion int
			if latestSnapshot != nil {
				lastSnapshotTime = latestSnapshot.CreatedAt
				lastSnapshotVersion = latestSnapshot.Version
			}
			
			// Filter events after the last snapshot
			var eventsAfterSnapshot []*eventsourcing.Event
			for _, event := range aggregateEvents {
				if event.Version > lastSnapshotVersion {
					eventsAfterSnapshot = append(eventsAfterSnapshot, event)
				}
			}
			
			// Check if we should create a snapshot
			if !m.ShouldCreateSnapshot(len(eventsAfterSnapshot), lastSnapshotTime) {
				continue
			}
			
			// Reconstruct the aggregate
			aggregate, err := m.ReconstructAggregate(ctx, aggregateID, aggregateType)
			if err != nil {
				m.logger.Error("Failed to reconstruct aggregate",
					zap.String("aggregate_id", aggregateID),
					zap.String("aggregate_type", aggregateType),
					zap.Error(err))
				continue
			}
			
			// Create a snapshot
			_, err = m.CreateSnapshot(ctx, aggregate)
			if err != nil {
				m.logger.Error("Failed to create snapshot",
					zap.String("aggregate_id", aggregateID),
					zap.String("aggregate_type", aggregateType),
					zap.Error(err))
				continue
			}
			
			// Prune old snapshots if we have too many
			snapshots, err := m.snapshotStore.List(ctx, aggregateID)
			if err != nil {
				m.logger.Error("Failed to list snapshots",
					zap.String("aggregate_id", aggregateID),
					zap.Error(err))
				continue
			}
			
			if len(snapshots) > m.config.MaxSnapshotsPerAggregate {
				// Sort snapshots by version (descending)
				// Keep only the most recent ones
				// Delete the rest
				// ...
			}
		}
	}
	
	return nil
}

// ReconstructAggregate reconstructs an aggregate from events
func (m *SnapshotManager) ReconstructAggregate(ctx context.Context, aggregateID string, aggregateType string) (eventsourcing.Aggregate, error) {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Get the latest snapshot
	// 2. Create a new aggregate
	// 3. Apply the snapshot
	// 4. Get events after the snapshot
	// 5. Apply the events
	
	return nil, fmt.Errorf("not implemented")
}

// MemorySnapshotStore is an in-memory implementation of SnapshotStore
type MemorySnapshotStore struct {
	snapshots map[string]*Snapshot // snapshotID -> snapshot
	byAggregate map[string][]*Snapshot // aggregateID -> snapshots
	mu        sync.RWMutex
}

// NewMemorySnapshotStore creates a new in-memory snapshot store
func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{
		snapshots: make(map[string]*Snapshot),
		byAggregate: make(map[string][]*Snapshot),
	}
}

// Save saves a snapshot
func (s *MemorySnapshotStore) Save(ctx context.Context, snapshot *Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.snapshots[snapshot.ID] = snapshot
	s.byAggregate[snapshot.AggregateID] = append(s.byAggregate[snapshot.AggregateID], snapshot)
	
	return nil
}

// Get gets the latest snapshot for an aggregate
func (s *MemorySnapshotStore) Get(ctx context.Context, aggregateID string) (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	snapshots, ok := s.byAggregate[aggregateID]
	if !ok || len(snapshots) == 0 {
		return nil, fmt.Errorf("snapshot not found")
	}
	
	// Find the snapshot with the highest version
	var latestSnapshot *Snapshot
	for _, snapshot := range snapshots {
		if latestSnapshot == nil || snapshot.Version > latestSnapshot.Version {
			latestSnapshot = snapshot
		}
	}
	
	return latestSnapshot, nil
}

// GetByVersion gets a snapshot for an aggregate at a specific version
func (s *MemorySnapshotStore) GetByVersion(ctx context.Context, aggregateID string, version int) (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	snapshots, ok := s.byAggregate[aggregateID]
	if !ok {
		return nil, fmt.Errorf("snapshot not found")
	}
	
	for _, snapshot := range snapshots {
		if snapshot.Version == version {
			return snapshot, nil
		}
	}
	
	return nil, fmt.Errorf("snapshot not found for version: %d", version)
}

// List lists all snapshots for an aggregate
func (s *MemorySnapshotStore) List(ctx context.Context, aggregateID string) ([]*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	snapshots, ok := s.byAggregate[aggregateID]
	if !ok {
		return []*Snapshot{}, nil
	}
	
	// Return a copy to avoid race conditions
	result := make([]*Snapshot, len(snapshots))
	copy(result, snapshots)
	
	return result, nil
}

// Delete deletes a snapshot
func (s *MemorySnapshotStore) Delete(ctx context.Context, snapshotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	snapshot, ok := s.snapshots[snapshotID]
	if !ok {
		return nil // Already deleted
	}
	
	// Remove from snapshots map
	delete(s.snapshots, snapshotID)
	
	// Remove from byAggregate map
	snapshots := s.byAggregate[snapshot.AggregateID]
	for i, s := range snapshots {
		if s.ID == snapshotID {
			// Remove this snapshot
			snapshots = append(snapshots[:i], snapshots[i+1:]...)
			break
		}
	}
	s.byAggregate[snapshot.AggregateID] = snapshots
	
	return nil
}

// DeleteByAggregateID deletes all snapshots for an aggregate
func (s *MemorySnapshotStore) DeleteByAggregateID(ctx context.Context, aggregateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	snapshots, ok := s.byAggregate[aggregateID]
	if !ok {
		return nil // No snapshots for this aggregate
	}
	
	// Remove from snapshots map
	for _, snapshot := range snapshots {
		delete(s.snapshots, snapshot.ID)
	}
	
	// Remove from byAggregate map
	delete(s.byAggregate, aggregateID)
	
	return nil
}

// Prune deletes old snapshots based on retention policy
func (s *MemorySnapshotStore) Prune(ctx context.Context, retention time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-retention)
	
	for aggregateID, snapshots := range s.byAggregate {
		var newSnapshots []*Snapshot
		
		for _, snapshot := range snapshots {
			if snapshot.CreatedAt.After(cutoff) {
				newSnapshots = append(newSnapshots, snapshot)
			} else {
				delete(s.snapshots, snapshot.ID)
			}
		}
		
		if len(newSnapshots) == 0 {
			delete(s.byAggregate, aggregateID)
		} else {
			s.byAggregate[aggregateID] = newSnapshots
		}
	}
	
	return nil
}

// JSONSnapshotHandler is a snapshot handler that uses JSON serialization
type JSONSnapshotHandler struct {
	aggregateType string
	newAggregate  func() eventsourcing.Aggregate
}

// NewJSONSnapshotHandler creates a new JSON snapshot handler
func NewJSONSnapshotHandler(aggregateType string, newAggregate func() eventsourcing.Aggregate) *JSONSnapshotHandler {
	return &JSONSnapshotHandler{
		aggregateType: aggregateType,
		newAggregate:  newAggregate,
	}
}

// AggregateType returns the type of aggregate this handler is for
func (h *JSONSnapshotHandler) AggregateType() string {
	return h.aggregateType
}

// CreateSnapshot creates a snapshot from an aggregate
func (h *JSONSnapshotHandler) CreateSnapshot(aggregate eventsourcing.Aggregate) (*Snapshot, error) {
	state, err := json.Marshal(aggregate)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal aggregate: %w", err)
	}
	
	return &Snapshot{
		ID:           fmt.Sprintf("%s-%d-%d", aggregate.AggregateID(), aggregate.Version(), time.Now().UnixNano()),
		AggregateID:  aggregate.AggregateID(),
		AggregateType: aggregate.AggregateType(),
		Version:      aggregate.Version(),
		State:        state,
		CreatedAt:    time.Now(),
		EventCount:   aggregate.Version(), // Assuming version is the event count
	}, nil
}

// ApplySnapshot applies a snapshot to an aggregate
func (h *JSONSnapshotHandler) ApplySnapshot(snapshot *Snapshot, aggregate eventsourcing.Aggregate) error {
	if err := json.Unmarshal(snapshot.State, aggregate); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}
	
	return nil
}

