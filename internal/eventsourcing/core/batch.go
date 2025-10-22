package core

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"go.uber.org/zap"
)

// BatchEventStore provides a batched event store
type BatchEventStore struct {
	store         EventStore
	batchSize     int
	flushInterval time.Duration
	batchEvents   map[string][]*eventsourcing.Event // aggregateType:aggregateID -> events
	mu            sync.Mutex
	logger        *zap.Logger
	flushTimer    *time.Timer
	stopCh        chan struct{}
}

// BatchEventStoreOptions contains options for the batched event store
type BatchEventStoreOptions struct {
	BatchSize     int
	FlushInterval time.Duration
}

// DefaultBatchEventStoreOptions returns default batched event store options
func DefaultBatchEventStoreOptions() BatchEventStoreOptions {
	return BatchEventStoreOptions{
		BatchSize:     100,
		FlushInterval: 100 * time.Millisecond,
	}
}

// NewBatchEventStore creates a new batched event store
func NewBatchEventStore(store EventStore, logger *zap.Logger, options ...StoreOption) *BatchEventStore {
	batchStore := &BatchEventStore{
		store:         store,
		batchSize:     100,
		flushInterval: 100 * time.Millisecond,
		batchEvents:   make(map[string][]*eventsourcing.Event),
		logger:        logger,
		stopCh:        make(chan struct{}),
	}
	
	// Apply options
	for _, option := range options {
		option(batchStore)
	}
	
	// Start the flush timer
	batchStore.flushTimer = time.AfterFunc(batchStore.flushInterval, batchStore.onFlushTimer)
	
	return batchStore
}

// SetBatchSize sets the batch size
func (s *BatchEventStore) SetBatchSize(size int) {
	s.batchSize = size
}

// SetFlushInterval sets the flush interval
func (s *BatchEventStore) SetFlushInterval(interval time.Duration) {
	s.flushInterval = interval
	if s.flushTimer != nil {
		s.flushTimer.Reset(interval)
	}
}

// onFlushTimer is called when the flush timer expires
func (s *BatchEventStore) onFlushTimer() {
	// Flush the batched events
	err := s.Flush(context.Background())
	if err != nil {
		s.logger.Error("Failed to flush batched events",
			zap.Error(err))
	}
	
	// Reset the timer
	select {
	case <-s.stopCh:
		// Stop the timer
		return
	default:
		// Reset the timer
		s.flushTimer.Reset(s.flushInterval)
	}
}

// SaveEvent saves an event to the store
func (s *BatchEventStore) SaveEvent(ctx context.Context, event *eventsourcing.Event) error {
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

// SaveEvents saves events to the store
func (s *BatchEventStore) SaveEvents(ctx context.Context, events []*eventsourcing.Event) error {
	if len(events) == 0 {
		return nil
	}
	
	// If the number of events is greater than the batch size, save them directly
	if len(events) >= s.batchSize {
		return s.store.SaveEvents(ctx, events)
	}
	
	// Otherwise, add them to the batch
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Group events by aggregate
	eventsByAggregate := make(map[string][]*eventsourcing.Event)
	for _, event := range events {
		key := event.AggregateType + ":" + event.AggregateID
		eventsByAggregate[key] = append(eventsByAggregate[key], event)
	}
	
	// Add events to the batch
	for key, aggregateEvents := range eventsByAggregate {
		s.batchEvents[key] = append(s.batchEvents[key], aggregateEvents...)
		
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
func (s *BatchEventStore) GetEvents(ctx context.Context, aggregateID string, aggregateType string, fromVersion int) ([]*eventsourcing.Event, error) {
	// Flush any pending events
	err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	// Get events from the store
	return s.store.GetEvents(ctx, aggregateID, aggregateType, fromVersion)
}

// GetEventsByType gets events by type
func (s *BatchEventStore) GetEventsByType(ctx context.Context, eventType string, fromTimestamp time.Time, limit int) ([]*eventsourcing.Event, error) {
	// Flush any pending events
	err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	// Get events from the store
	return s.store.GetEventsByType(ctx, eventType, fromTimestamp, limit)
}

// GetAggregateEvents gets events for multiple aggregates
func (s *BatchEventStore) GetAggregateEvents(ctx context.Context, aggregateIDs []string, aggregateType string, fromVersion int) ([]*eventsourcing.Event, error) {
	// Flush any pending events
	err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	// Get events from the store
	return s.store.GetAggregateEvents(ctx, aggregateIDs, aggregateType, fromVersion)
}

// GetAllEvents gets all events
func (s *BatchEventStore) GetAllEvents(ctx context.Context, fromTimestamp time.Time, limit int) ([]*eventsourcing.Event, error) {
	// Flush any pending events
	err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}
	
	// Get events from the store
	return s.store.GetAllEvents(ctx, fromTimestamp, limit)
}

// Close closes the batched event store
func (s *BatchEventStore) Close() error {
	// Stop the flush timer
	close(s.stopCh)
	s.flushTimer.Stop()
	
	// Flush any pending events
	return s.Flush(context.Background())
}

