package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
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
	// EnableCompression enables compression of snapshots
	EnableCompression bool
	// CompressionLevel is the compression level (1-9, higher is more compression)
	CompressionLevel int
	// EnableConcurrentSnapshots enables concurrent snapshot creation
	EnableConcurrentSnapshots bool
	// MaxConcurrentSnapshots is the maximum number of concurrent snapshots
	MaxConcurrentSnapshots int
}

// DefaultSnapshotConfig returns the default snapshot configuration
func DefaultSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{
		Frequency:                1 * time.Hour,
		EventThreshold:           100,
		Retention:                30 * 24 * time.Hour, // 30 days
		MaxSnapshotsPerAggregate: 10,
		EnableCompression:        true,
		CompressionLevel:         6, // Default compression level
		EnableConcurrentSnapshots: true,
		MaxConcurrentSnapshots:   5,
	}
}

// Snapshot represents a point-in-time state of an aggregate
type Snapshot struct {
	ID            string
	AggregateID   string
	AggregateType string
	Version       int
	State         []byte
	CreatedAt     time.Time
	EventCount    int
	Compressed    bool
}

// SnapshotStore is the interface for storing and retrieving snapshots
type SnapshotStore interface {
	// Save saves a snapshot
	Save(ctx context.Context, snapshot *Snapshot) error
	// Get gets the latest snapshot for an aggregate
	Get(ctx context.Context, aggregateType, aggregateID string) (*Snapshot, error)
	// GetByVersion gets a snapshot for an aggregate at a specific version
	GetByVersion(ctx context.Context, aggregateType, aggregateID string, version int) (*Snapshot, error)
	// List lists snapshots for an aggregate
	List(ctx context.Context, aggregateType, aggregateID string) ([]*Snapshot, error)
	// Delete deletes a snapshot
	Delete(ctx context.Context, snapshotID string) error
	// DeleteByAggregate deletes all snapshots for an aggregate
	DeleteByAggregate(ctx context.Context, aggregateType, aggregateID string) error
	// Cleanup cleans up old snapshots
	Cleanup(ctx context.Context, retention time.Duration) error
}

// SnapshotManager manages snapshots for aggregates
type SnapshotManager struct {
	// Configuration
	config SnapshotConfig

	// Stores
	snapshotStore SnapshotStore
	eventStore    store.EventStore

	// Snapshot creation tracking
	snapshotCreationLock  sync.Mutex
	snapshotCreationCount int64
	snapshotSemaphore     chan struct{}

	// Statistics
	snapshotsCreated int64
	snapshotsLoaded  int64
	snapshotsDeleted int64
	eventsProcessed  int64

	// Context for cancellation
	ctx        context.Context
	cancelFunc context.CancelFunc

	// Wait group for goroutines
	wg sync.WaitGroup

	// Logger
	logger *zap.Logger
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(config SnapshotConfig, snapshotStore SnapshotStore, eventStore store.EventStore, logger *zap.Logger) *SnapshotManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm := &SnapshotManager{
		config:            config,
		snapshotStore:     snapshotStore,
		eventStore:        eventStore,
		snapshotSemaphore: make(chan struct{}, config.MaxConcurrentSnapshots),
		ctx:               ctx,
		cancelFunc:        cancel,
		logger:            logger,
	}

	// Start the cleanup goroutine
	sm.wg.Add(1)
	go sm.cleanupLoop()

	return sm
}

// CreateSnapshot creates a snapshot for an aggregate
func (sm *SnapshotManager) CreateSnapshot(ctx context.Context, aggregate eventsourcing.Aggregate) error {
	// Serialize the aggregate state
	state, err := json.Marshal(aggregate)
	if err != nil {
		return fmt.Errorf("failed to marshal aggregate state: %w", err)
	}

	// Compress the state if enabled
	compressed := false
	if sm.config.EnableCompression && len(state) > 1024 {
		compressedState, err := sm.compressData(state)
		if err != nil {
			sm.logger.Warn("Failed to compress snapshot",
				zap.Error(err),
				zap.String("aggregateID", aggregate.GetID()),
				zap.String("aggregateType", aggregate.GetType()),
			)
		} else {
			state = compressedState
			compressed = true
		}
	}

	// Create the snapshot
	snapshot := &Snapshot{
		ID:            fmt.Sprintf("%s-%s-%d", aggregate.GetType(), aggregate.GetID(), aggregate.GetVersion()),
		AggregateID:   aggregate.GetID(),
		AggregateType: aggregate.GetType(),
		Version:       aggregate.GetVersion(),
		State:         state,
		CreatedAt:     time.Now(),
		EventCount:    aggregate.GetEventCount(),
		Compressed:    compressed,
	}

	// Save the snapshot
	if err := sm.snapshotStore.Save(ctx, snapshot); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	// Update statistics
	atomic.AddInt64(&sm.snapshotsCreated, 1)

	sm.logger.Debug("Created snapshot",
		zap.String("snapshotID", snapshot.ID),
		zap.String("aggregateID", snapshot.AggregateID),
		zap.String("aggregateType", snapshot.AggregateType),
		zap.Int("version", snapshot.Version),
		zap.Int("eventCount", snapshot.EventCount),
		zap.Bool("compressed", snapshot.Compressed),
		zap.Int("stateSize", len(state)),
	)

	// Clean up old snapshots if we have too many
	if err := sm.cleanupAggregateSnapshots(ctx, aggregate.GetType(), aggregate.GetID()); err != nil {
		sm.logger.Warn("Failed to clean up old snapshots",
			zap.Error(err),
			zap.String("aggregateID", aggregate.GetID()),
			zap.String("aggregateType", aggregate.GetType()),
		)
	}

	return nil
}

// LoadSnapshot loads the latest snapshot for an aggregate
func (sm *SnapshotManager) LoadSnapshot(ctx context.Context, aggregate eventsourcing.Aggregate) error {
	// Get the latest snapshot
	snapshot, err := sm.snapshotStore.Get(ctx, aggregate.GetType(), aggregate.GetID())
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}

	if snapshot == nil {
		return nil // No snapshot found
	}

	// Decompress the state if needed
	state := snapshot.State
	if snapshot.Compressed {
		decompressedState, err := sm.decompressData(state)
		if err != nil {
			return fmt.Errorf("failed to decompress snapshot: %w", err)
		}
		state = decompressedState
	}

	// Deserialize the state
	if err := json.Unmarshal(state, aggregate); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot state: %w", err)
	}

	// Update statistics
	atomic.AddInt64(&sm.snapshotsLoaded, 1)

	sm.logger.Debug("Loaded snapshot",
		zap.String("snapshotID", snapshot.ID),
		zap.String("aggregateID", snapshot.AggregateID),
		zap.String("aggregateType", snapshot.AggregateType),
		zap.Int("version", snapshot.Version),
		zap.Int("eventCount", snapshot.EventCount),
		zap.Bool("compressed", snapshot.Compressed),
		zap.Int("stateSize", len(state)),
	)

	return nil
}

// LoadSnapshotByVersion loads a snapshot for an aggregate at a specific version
func (sm *SnapshotManager) LoadSnapshotByVersion(ctx context.Context, aggregate eventsourcing.Aggregate, version int) error {
	// Get the snapshot at the specified version
	snapshot, err := sm.snapshotStore.GetByVersion(ctx, aggregate.GetType(), aggregate.GetID(), version)
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}

	if snapshot == nil {
		return nil // No snapshot found
	}

	// Decompress the state if needed
	state := snapshot.State
	if snapshot.Compressed {
		decompressedState, err := sm.decompressData(state)
		if err != nil {
			return fmt.Errorf("failed to decompress snapshot: %w", err)
		}
		state = decompressedState
	}

	// Deserialize the state
	if err := json.Unmarshal(state, aggregate); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot state: %w", err)
	}

	// Update statistics
	atomic.AddInt64(&sm.snapshotsLoaded, 1)

	sm.logger.Debug("Loaded snapshot by version",
		zap.String("snapshotID", snapshot.ID),
		zap.String("aggregateID", snapshot.AggregateID),
		zap.String("aggregateType", snapshot.AggregateType),
		zap.Int("version", snapshot.Version),
		zap.Int("eventCount", snapshot.EventCount),
		zap.Bool("compressed", snapshot.Compressed),
		zap.Int("stateSize", len(state)),
	)

	return nil
}

// ProcessEvents processes events for an aggregate and creates snapshots as needed
func (sm *SnapshotManager) ProcessEvents(ctx context.Context, aggregate eventsourcing.Aggregate, events []eventsourcing.Event) error {
	// Update statistics
	atomic.AddInt64(&sm.eventsProcessed, int64(len(events)))

	// Check if we need to create a snapshot
	if len(events) > 0 && sm.shouldCreateSnapshot(aggregate, events) {
		// Create the snapshot
		if sm.config.EnableConcurrentSnapshots {
			// Create the snapshot concurrently
			sm.createSnapshotConcurrently(ctx, aggregate)
		} else {
			// Create the snapshot synchronously
			if err := sm.CreateSnapshot(ctx, aggregate); err != nil {
				sm.logger.Warn("Failed to create snapshot",
					zap.Error(err),
					zap.String("aggregateID", aggregate.GetID()),
					zap.String("aggregateType", aggregate.GetType()),
				)
			}
		}
	}

	return nil
}

// shouldCreateSnapshot checks if a snapshot should be created
func (sm *SnapshotManager) shouldCreateSnapshot(aggregate eventsourcing.Aggregate, events []eventsourcing.Event) bool {
	// Check if we have enough events
	if aggregate.GetEventCount() >= sm.config.EventThreshold {
		return true
	}

	// Check if it's been long enough since the last snapshot
	lastSnapshot, err := sm.snapshotStore.Get(context.Background(), aggregate.GetType(), aggregate.GetID())
	if err != nil {
		sm.logger.Warn("Failed to get last snapshot",
			zap.Error(err),
			zap.String("aggregateID", aggregate.GetID()),
			zap.String("aggregateType", aggregate.GetType()),
		)
		return false
	}

	if lastSnapshot != nil {
		timeSinceLastSnapshot := time.Since(lastSnapshot.CreatedAt)
		if timeSinceLastSnapshot >= sm.config.Frequency {
			return true
		}
	} else {
		// No previous snapshot, create one if we have events
		return len(events) > 0
	}

	return false
}

// createSnapshotConcurrently creates a snapshot concurrently
func (sm *SnapshotManager) createSnapshotConcurrently(ctx context.Context, aggregate eventsourcing.Aggregate) {
	// Check if we're already creating too many snapshots
	currentCount := atomic.LoadInt64(&sm.snapshotCreationCount)
	if currentCount >= int64(sm.config.MaxConcurrentSnapshots) {
		sm.logger.Debug("Too many concurrent snapshots, skipping",
			zap.Int64("currentCount", currentCount),
			zap.Int("maxConcurrent", sm.config.MaxConcurrentSnapshots),
		)
		return
	}

	// Increment the snapshot creation count
	atomic.AddInt64(&sm.snapshotCreationCount, 1)

	// Create a copy of the aggregate to avoid race conditions
	aggregateCopy := aggregate.Copy()

	// Create the snapshot in a goroutine
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		defer atomic.AddInt64(&sm.snapshotCreationCount, -1)

		// Create a new context with timeout
		snapshotCtx, cancel := context.WithTimeout(sm.ctx, 30*time.Second)
		defer cancel()

		// Create the snapshot
		if err := sm.CreateSnapshot(snapshotCtx, aggregateCopy); err != nil {
			sm.logger.Warn("Failed to create snapshot concurrently",
				zap.Error(err),
				zap.String("aggregateID", aggregateCopy.GetID()),
				zap.String("aggregateType", aggregateCopy.GetType()),
			)
		}
	}()
}

// cleanupAggregateSnapshots cleans up old snapshots for an aggregate
func (sm *SnapshotManager) cleanupAggregateSnapshots(ctx context.Context, aggregateType, aggregateID string) error {
	// Get all snapshots for the aggregate
	snapshots, err := sm.snapshotStore.List(ctx, aggregateType, aggregateID)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Sort snapshots by version (descending)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Version > snapshots[j].Version
	})

	// Keep only the most recent snapshots
	if len(snapshots) > sm.config.MaxSnapshotsPerAggregate {
		for i := sm.config.MaxSnapshotsPerAggregate; i < len(snapshots); i++ {
			if err := sm.snapshotStore.Delete(ctx, snapshots[i].ID); err != nil {
				sm.logger.Warn("Failed to delete old snapshot",
					zap.Error(err),
					zap.String("snapshotID", snapshots[i].ID),
				)
				continue
			}

			// Update statistics
			atomic.AddInt64(&sm.snapshotsDeleted, 1)

			sm.logger.Debug("Deleted old snapshot",
				zap.String("snapshotID", snapshots[i].ID),
				zap.String("aggregateID", snapshots[i].AggregateID),
				zap.String("aggregateType", snapshots[i].AggregateType),
				zap.Int("version", snapshots[i].Version),
			)
		}
	}

	return nil
}

// cleanupLoop runs the cleanup process periodically
func (sm *SnapshotManager) cleanupLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(24 * time.Hour) // Run cleanup once a day
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Create a new context with timeout
			cleanupCtx, cancel := context.WithTimeout(sm.ctx, 1*time.Hour)

			// Run cleanup
			if err := sm.snapshotStore.Cleanup(cleanupCtx, sm.config.Retention); err != nil {
				sm.logger.Warn("Failed to clean up old snapshots",
					zap.Error(err),
				)
			}

			cancel()
		case <-sm.ctx.Done():
			return
		}
	}
}

// GetStats gets statistics about the snapshot manager
func (sm *SnapshotManager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["snapshotsCreated"] = atomic.LoadInt64(&sm.snapshotsCreated)
	stats["snapshotsLoaded"] = atomic.LoadInt64(&sm.snapshotsLoaded)
	stats["snapshotsDeleted"] = atomic.LoadInt64(&sm.snapshotsDeleted)
	stats["eventsProcessed"] = atomic.LoadInt64(&sm.eventsProcessed)
	stats["concurrentSnapshots"] = atomic.LoadInt64(&sm.snapshotCreationCount)

	return stats
}

// Shutdown shuts down the snapshot manager
func (sm *SnapshotManager) Shutdown() {
	// Cancel the context
	sm.cancelFunc()

	// Wait for goroutines to finish
	sm.wg.Wait()

	sm.logger.Info("Snapshot manager shutdown complete")
}

// compressData compresses data
func (sm *SnapshotManager) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, sm.config.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write data to gzip writer: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// decompressData decompresses data
func (sm *SnapshotManager) decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, fmt.Errorf("failed to read data from gzip reader: %w", err)
	}

	return buf.Bytes(), nil
}
