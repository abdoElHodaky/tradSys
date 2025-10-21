package core

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SnapshotStrategy represents a strategy for creating snapshots
type SnapshotStrategy interface {
	// ShouldCreateSnapshot determines if a snapshot should be created
	ShouldCreateSnapshot(aggregateType string, aggregateID string, version int) bool
	
	// GetSnapshotFrequency returns the snapshot frequency for an aggregate type
	GetSnapshotFrequency(aggregateType string) int
}

// DefaultSnapshotStrategy provides a default implementation of the SnapshotStrategy interface
type DefaultSnapshotStrategy struct {
	frequencies map[string]int
	mu          sync.RWMutex
}

// NewDefaultSnapshotStrategy creates a new default snapshot strategy
func NewDefaultSnapshotStrategy() *DefaultSnapshotStrategy {
	return &DefaultSnapshotStrategy{
		frequencies: make(map[string]int),
	}
}

// SetSnapshotFrequency sets the snapshot frequency for an aggregate type
func (s *DefaultSnapshotStrategy) SetSnapshotFrequency(aggregateType string, frequency int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.frequencies[aggregateType] = frequency
}

// ShouldCreateSnapshot determines if a snapshot should be created
func (s *DefaultSnapshotStrategy) ShouldCreateSnapshot(aggregateType string, aggregateID string, version int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Get the frequency for the aggregate type
	frequency, ok := s.frequencies[aggregateType]
	if !ok {
		// Default to no snapshots
		return false
	}
	
	// Check if a snapshot should be created
	return frequency > 0 && version % frequency == 0
}

// GetSnapshotFrequency returns the snapshot frequency for an aggregate type
func (s *DefaultSnapshotStrategy) GetSnapshotFrequency(aggregateType string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Get the frequency for the aggregate type
	frequency, ok := s.frequencies[aggregateType]
	if !ok {
		// Default to no snapshots
		return 0
	}
	
	return frequency
}

// SnapshotManager manages snapshots
type SnapshotManager interface {
	// CreateSnapshot creates a snapshot
	CreateSnapshot(ctx context.Context, aggregateType string, aggregateID string, version int, snapshot interface{}) error
	
	// GetLatestSnapshot gets the latest snapshot for an aggregate
	GetLatestSnapshot(ctx context.Context, aggregateType string, aggregateID string) (interface{}, int, error)
	
	// DeleteSnapshots deletes snapshots for an aggregate
	DeleteSnapshots(ctx context.Context, aggregateType string, aggregateID string) error
}

// DefaultSnapshotManager provides a default implementation of the SnapshotManager interface
type DefaultSnapshotManager struct {
	store    store.SnapshotStore
	strategy SnapshotStrategy
	logger   *zap.Logger
}

// NewDefaultSnapshotManager creates a new default snapshot manager
func NewDefaultSnapshotManager(store store.SnapshotStore, strategy SnapshotStrategy, logger *zap.Logger) *DefaultSnapshotManager {
	return &DefaultSnapshotManager{
		store:    store,
		strategy: strategy,
		logger:   logger,
	}
}

// CreateSnapshot creates a snapshot
func (m *DefaultSnapshotManager) CreateSnapshot(ctx context.Context, aggregateType string, aggregateID string, version int, snapshot interface{}) error {
	// Check if a snapshot should be created
	if !m.strategy.ShouldCreateSnapshot(aggregateType, aggregateID, version) {
		return nil
	}
	
	// Create the snapshot
	return m.store.SaveSnapshot(ctx, aggregateID, aggregateType, version, snapshot)
}

// GetLatestSnapshot gets the latest snapshot for an aggregate
func (m *DefaultSnapshotManager) GetLatestSnapshot(ctx context.Context, aggregateType string, aggregateID string) (interface{}, int, error) {
	return m.store.GetLatestSnapshot(ctx, aggregateID, aggregateType)
}

// DeleteSnapshots deletes snapshots for an aggregate
func (m *DefaultSnapshotManager) DeleteSnapshots(ctx context.Context, aggregateType string, aggregateID string) error {
	// Check if the store supports deleting snapshots
	if deleter, ok := m.store.(interface {
		DeleteSnapshots(ctx context.Context, aggregateID string, aggregateType string) error
	}); ok {
		return deleter.DeleteSnapshots(ctx, aggregateID, aggregateType)
	}
	
	return ErrDeleteSnapshotsNotSupported
}

// SnapshotScheduler schedules snapshot creation
type SnapshotScheduler struct {
	manager      SnapshotManager
	eventStore   store.EventStore
	logger       *zap.Logger
	interval     time.Duration
	stopCh       chan struct{}
	aggregateTypes []string
}

// NewSnapshotScheduler creates a new snapshot scheduler
func NewSnapshotScheduler(manager SnapshotManager, eventStore store.EventStore, logger *zap.Logger, interval time.Duration) *SnapshotScheduler {
	return &SnapshotScheduler{
		manager:      manager,
		eventStore:   eventStore,
		logger:       logger,
		interval:     interval,
		stopCh:       make(chan struct{}),
		aggregateTypes: make([]string, 0),
	}
}

// RegisterAggregateType registers an aggregate type with the scheduler
func (s *SnapshotScheduler) RegisterAggregateType(aggregateType string) {
	s.aggregateTypes = append(s.aggregateTypes, aggregateType)
}

// Start starts the scheduler
func (s *SnapshotScheduler) Start() {
	go s.run()
}

// Stop stops the scheduler
func (s *SnapshotScheduler) Stop() {
	close(s.stopCh)
}

// run runs the scheduler
func (s *SnapshotScheduler) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Create snapshots
			s.createSnapshots()
		case <-s.stopCh:
			return
		}
	}
}

// createSnapshots creates snapshots
func (s *SnapshotScheduler) createSnapshots() {
	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), s.interval/2)
	defer cancel()
	
	// Create snapshots for each aggregate type
	for _, aggregateType := range s.aggregateTypes {
		// Get all events for the aggregate type
		events, err := s.eventStore.GetEventsByType(ctx, aggregateType, time.Time{}, 0)
		if err != nil {
			s.logger.Error("Failed to get events for aggregate type",
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
			// Get the latest version
			latestVersion := 0
			for _, event := range aggregateEvents {
				if event.Version > latestVersion {
					latestVersion = event.Version
				}
			}
			
			// Create a snapshot
			// Note: In a real implementation, we would need to load the aggregate and create a snapshot from it
			// For now, we just log that we would create a snapshot
			s.logger.Info("Would create snapshot",
				zap.String("aggregate_type", aggregateType),
				zap.String("aggregate_id", aggregateID),
				zap.Int("version", latestVersion))
		}
	}
}

// Common errors
var (
	ErrDeleteSnapshotsNotSupported = errors.New("delete snapshots not supported")
)
