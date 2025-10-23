package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// QueryCache provides caching for database queries
type QueryCache struct {
	cache       *cache.Cache
	logger      *zap.Logger
	metrics     *CacheMetrics
	mutex       sync.RWMutex
	defaultTTL  time.Duration
	cleanupTime time.Duration
}

// CacheMetrics tracks cache metrics
type CacheMetrics struct {
	Hits              int64
	Misses            int64
	Errors            int64
	Evictions         int64
	Size              int64
	TotalItems        int64
	TotalOperations   int64
	AverageAccessTime time.Duration
	mutex             sync.RWMutex
}

// CacheMetricsSnapshot represents a snapshot of cache metrics without mutex
type CacheMetricsSnapshot struct {
	Hits              int64
	Misses            int64
	Errors            int64
	Evictions         int64
	Size              int64
	TotalItems        int64
	TotalOperations   int64
	AverageAccessTime time.Duration
}

// QueryCacheOptions contains options for the query cache
type QueryCacheOptions struct {
	DefaultTTL  time.Duration
	CleanupTime time.Duration
}

// NewQueryCache creates a new query cache
func NewQueryCache(logger *zap.Logger, options QueryCacheOptions) *QueryCache {
	// Set default values if not provided
	if options.DefaultTTL == 0 {
		options.DefaultTTL = 5 * time.Minute
	}
	if options.CleanupTime == 0 {
		options.CleanupTime = 10 * time.Minute
	}

	c := cache.New(options.DefaultTTL, options.CleanupTime)

	metrics := &CacheMetrics{}

	qc := &QueryCache{
		cache:       c,
		logger:      logger,
		metrics:     metrics,
		defaultTTL:  options.DefaultTTL,
		cleanupTime: options.CleanupTime,
	}

	// Register eviction callback
	c.OnEvicted(func(key string, value interface{}) {
		qc.metrics.mutex.Lock()
		qc.metrics.Evictions++
		qc.metrics.mutex.Unlock()

		qc.logger.Debug("Cache item evicted", zap.String("key", key))
	})

	logger.Info("Query cache initialized",
		zap.Duration("default_ttl", options.DefaultTTL),
		zap.Duration("cleanup_time", options.CleanupTime),
	)

	// Start metrics collection
	go qc.collectMetrics()

	return qc
}

// collectMetrics periodically collects cache metrics
func (qc *QueryCache) collectMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		items := qc.cache.Items()

		qc.metrics.mutex.Lock()
		qc.metrics.TotalItems = int64(len(items))
		qc.metrics.Size = 0
		for _, item := range items {
			if data, ok := item.Object.([]byte); ok {
				qc.metrics.Size += int64(len(data))
			}
		}
		qc.metrics.mutex.Unlock()

		qc.logger.Debug("Query cache metrics",
			zap.Int64("total_items", qc.metrics.TotalItems),
			zap.Int64("size_bytes", qc.metrics.Size),
			zap.Int64("hits", qc.metrics.Hits),
			zap.Int64("misses", qc.metrics.Misses),
			zap.Int64("errors", qc.metrics.Errors),
			zap.Int64("evictions", qc.metrics.Evictions),
			zap.Int64("total_operations", qc.metrics.TotalOperations),
		)
	}
}

// Get retrieves an item from the cache
func (qc *QueryCache) Get(ctx context.Context, key string, dest interface{}) bool {
	startTime := time.Now()

	// Check if context is canceled
	if ctx.Err() != nil {
		return false
	}

	// Get item from cache
	item, found := qc.cache.Get(key)

	// Track metrics
	qc.metrics.mutex.Lock()
	qc.metrics.TotalOperations++
	if found {
		qc.metrics.Hits++
	} else {
		qc.metrics.Misses++
	}
	qc.metrics.AverageAccessTime = (qc.metrics.AverageAccessTime*time.Duration(qc.metrics.TotalOperations-1) + time.Since(startTime)) / time.Duration(qc.metrics.TotalOperations)
	qc.metrics.mutex.Unlock()

	// Return if not found
	if !found {
		return false
	}

	// Unmarshal data
	data, ok := item.([]byte)
	if !ok {
		qc.logger.Error("Invalid cache item type", zap.String("key", key))
		return false
	}

	err := json.Unmarshal(data, dest)
	if err != nil {
		qc.metrics.mutex.Lock()
		qc.metrics.Errors++
		qc.metrics.mutex.Unlock()

		qc.logger.Error("Failed to unmarshal cache item",
			zap.Error(err),
			zap.String("key", key),
		)
		return false
	}

	return true
}

// Set stores an item in the cache
func (qc *QueryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Check if context is canceled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Marshal data
	data, err := json.Marshal(value)
	if err != nil {
		qc.metrics.mutex.Lock()
		qc.metrics.Errors++
		qc.metrics.mutex.Unlock()

		qc.logger.Error("Failed to marshal cache item",
			zap.Error(err),
			zap.String("key", key),
		)
		return fmt.Errorf("failed to marshal cache item: %w", err)
	}

	// Use default TTL if not provided
	if ttl == 0 {
		ttl = qc.defaultTTL
	}

	// Set item in cache
	qc.cache.Set(key, data, ttl)

	qc.metrics.mutex.Lock()
	qc.metrics.TotalOperations++
	qc.metrics.mutex.Unlock()

	return nil
}

// Delete removes an item from the cache
func (qc *QueryCache) Delete(ctx context.Context, key string) {
	// Check if context is canceled
	if ctx.Err() != nil {
		return
	}

	// Delete item from cache
	qc.cache.Delete(key)

	qc.metrics.mutex.Lock()
	qc.metrics.TotalOperations++
	qc.metrics.mutex.Unlock()
}

// Flush removes all items from the cache
func (qc *QueryCache) Flush() {
	qc.cache.Flush()

	qc.metrics.mutex.Lock()
	qc.metrics.TotalOperations++
	qc.metrics.mutex.Unlock()

	qc.logger.Info("Query cache flushed")
}

// GetMetrics returns the current cache metrics
func (qc *QueryCache) GetMetrics() CacheMetricsSnapshot {
	qc.metrics.mutex.RLock()
	defer qc.metrics.mutex.RUnlock()

	return CacheMetricsSnapshot{
		Hits:              qc.metrics.Hits,
		Misses:            qc.metrics.Misses,
		Errors:            qc.metrics.Errors,
		Evictions:         qc.metrics.Evictions,
		Size:              qc.metrics.Size,
		TotalItems:        qc.metrics.TotalItems,
		TotalOperations:   qc.metrics.TotalOperations,
		AverageAccessTime: qc.metrics.AverageAccessTime,
	}
}

// ResetMetrics resets the cache metrics
func (qc *QueryCache) ResetMetrics() {
	qc.metrics.mutex.Lock()
	defer qc.metrics.mutex.Unlock()

	qc.metrics.Hits = 0
	qc.metrics.Misses = 0
	qc.metrics.Errors = 0
	qc.metrics.Evictions = 0
	qc.metrics.TotalOperations = 0
	qc.metrics.AverageAccessTime = 0

	qc.logger.Info("Query cache metrics reset")
}

// GetCacheKey generates a cache key from a query and args
func GetCacheKey(query string, args ...interface{}) string {
	if len(args) == 0 {
		return query
	}

	// Marshal args to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%s-%v", query, args)
	}

	return fmt.Sprintf("%s-%s", query, argsJSON)
}

// WithCache executes a function with caching
func (qc *QueryCache) WithCache(ctx context.Context, key string, dest interface{}, ttl time.Duration, fn func() error) error {
	// Try to get from cache
	if qc.Get(ctx, key, dest) {
		return nil
	}

	// Execute function
	err := fn()
	if err != nil {
		return err
	}

	// Store in cache
	return qc.Set(ctx, key, dest, ttl)
}
