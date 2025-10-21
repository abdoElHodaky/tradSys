package core

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
)

// EventStore provides event storage functionality
type EventStore interface {
	// SaveEvents saves events to the store
	SaveEvents(ctx context.Context, events []*eventsourcing.Event) error

	// GetEvents gets events for an aggregate
	GetEvents(ctx context.Context, aggregateID string, aggregateType string, fromVersion int) ([]*eventsourcing.Event, error)

	// GetEventsByType gets events by type
	GetEventsByType(ctx context.Context, eventType string, fromTimestamp time.Time, limit int) ([]*eventsourcing.Event, error)

	// GetAggregateEvents gets events for multiple aggregates
	GetAggregateEvents(ctx context.Context, aggregateIDs []string, aggregateType string, fromVersion int) ([]*eventsourcing.Event, error)
	
	// GetAllEvents gets all events
	GetAllEvents(ctx context.Context, fromTimestamp time.Time, limit int) ([]*eventsourcing.Event, error)
}

// SnapshotStore provides snapshot storage functionality
type SnapshotStore interface {
	// SaveSnapshot saves a snapshot
	SaveSnapshot(ctx context.Context, aggregateID string, aggregateType string, version int, snapshot interface{}) error

	// GetLatestSnapshot gets the latest snapshot for an aggregate
	GetLatestSnapshot(ctx context.Context, aggregateID string, aggregateType string) (interface{}, int, error)
}

// EventStoreWithSnapshots provides event storage with snapshot functionality
type EventStoreWithSnapshots interface {
	EventStore
	SnapshotStore
}

// StoreOption represents an option for configuring a store
type StoreOption func(interface{}) error

// WithBatchSize sets the batch size for a store
func WithBatchSize(batchSize int) StoreOption {
	return func(store interface{}) error {
		if s, ok := store.(interface{ SetBatchSize(int) }); ok {
			s.SetBatchSize(batchSize)
			return nil
		}
		return errors.New("store does not support batch size")
	}
}

// WithFlushInterval sets the flush interval for a store
func WithFlushInterval(interval time.Duration) StoreOption {
	return func(store interface{}) error {
		if s, ok := store.(interface{ SetFlushInterval(time.Duration) }); ok {
			s.SetFlushInterval(interval)
			return nil
		}
		return errors.New("store does not support flush interval")
	}
}

// WithCacheSize sets the cache size for a store
func WithCacheSize(cacheSize int) StoreOption {
	return func(store interface{}) error {
		if s, ok := store.(interface{ SetCacheSize(int) }); ok {
			s.SetCacheSize(cacheSize)
			return nil
		}
		return errors.New("store does not support cache size")
	}
}

// WithCacheTTL sets the cache TTL for a store
func WithCacheTTL(ttl time.Duration) StoreOption {
	return func(store interface{}) error {
		if s, ok := store.(interface{ SetCacheTTL(time.Duration) }); ok {
			s.SetCacheTTL(ttl)
			return nil
		}
		return errors.New("store does not support cache TTL")
	}
}

// WithSnapshotFrequency sets the snapshot frequency for a store
func WithSnapshotFrequency(frequency int) StoreOption {
	return func(store interface{}) error {
		if s, ok := store.(interface{ SetSnapshotFrequency(int) }); ok {
			s.SetSnapshotFrequency(frequency)
			return nil
		}
		return errors.New("store does not support snapshot frequency")
	}
}

// Common errors
var (
	ErrConcurrencyConflict = errors.New("concurrency conflict")
	ErrSnapshotNotFound    = errors.New("snapshot not found")
	ErrAggregateNotFound   = errors.New("aggregate not found")
	ErrEventNotFound       = errors.New("event not found")
)

