package mitigation

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CacheConfig represents the configuration for a cache
type CacheConfig struct {
	// DefaultTTL is the default time-to-live for cache entries
	DefaultTTL time.Duration
	// MaxSize is the maximum number of entries in the cache
	MaxSize int
	// EvictionPolicy is the policy for evicting entries when the cache is full
	EvictionPolicy EvictionPolicy
}

// DefaultCacheConfig returns a default configuration for a cache
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL:     5 * time.Minute,
		MaxSize:        1000,
		EvictionPolicy: LRUEviction,
	}
}

// EvictionPolicy represents a cache eviction policy
type EvictionPolicy int

const (
	// LRUEviction evicts the least recently used entry
	LRUEviction EvictionPolicy = iota
	// LFUEviction evicts the least frequently used entry
	LFUEviction
	// TTLEviction evicts entries based on their TTL
	TTLEviction
	// RandomEviction evicts a random entry
	RandomEviction
)

// CacheEntry represents an entry in the cache
type CacheEntry struct {
	// Key is the key of the entry
	Key string
	// Value is the value of the entry
	Value interface{}
	// Expiration is the time when the entry expires
	Expiration time.Time
	// LastAccessed is the time when the entry was last accessed
	LastAccessed time.Time
	// AccessCount is the number of times the entry was accessed
	AccessCount int
}

// CacheMetrics tracks metrics for the cache
type CacheMetrics struct {
	// Hits is the number of cache hits
	Hits int64
	// Misses is the number of cache misses
	Misses int64
	// Evictions is the number of cache evictions
	Evictions int64
	// Size is the current size of the cache
	Size int
	// HitRate is the cache hit rate (hits / (hits + misses))
	HitRate float64
}

// Cache implements a simple in-memory cache
type Cache struct {
	name      string
	config    CacheConfig
	entries   map[string]*CacheEntry
	metrics   *CacheMetrics
	mutex     sync.RWMutex
	logger    *zap.Logger
	janitor   *time.Ticker
	stopChan  chan struct{}
}

// NewCache creates a new cache with the given name and configuration
func NewCache(name string, config CacheConfig, logger *zap.Logger) *Cache {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	cache := &Cache{
		name:     name,
		config:   config,
		entries:  make(map[string]*CacheEntry),
		metrics: &CacheMetrics{
			Hits:     0,
			Misses:   0,
			Evictions: 0,
			Size:     0,
			HitRate:  0.0,
		},
		logger:   logger.With(zap.String("component", "cache"), zap.String("name", name)),
		janitor:  time.NewTicker(time.Minute),
		stopChan: make(chan struct{}),
	}

	// Start the janitor to clean up expired entries
	go cache.startJanitor()

	return cache
}

// startJanitor starts the janitor to clean up expired entries
func (c *Cache) startJanitor() {
	for {
		select {
		case <-c.janitor.C:
			c.cleanExpired()
		case <-c.stopChan:
			c.janitor.Stop()
			return
		}
	}
}

// cleanExpired removes expired entries from the cache
func (c *Cache) cleanExpired() {
	now := time.Now()
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	for key, entry := range c.entries {
		if entry.Expiration.Before(now) {
			delete(c.entries, key)
			c.metrics.Size--
			c.metrics.Evictions++
		}
	}
}

// Get gets a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	entry, found := c.entries[key]
	c.mutex.RUnlock()
	
	if !found {
		c.mutex.Lock()
		c.metrics.Misses++
		c.updateHitRate()
		c.mutex.Unlock()
		return nil, false
	}
	
	// Check if the entry has expired
	if entry.Expiration.Before(time.Now()) {
		c.mutex.Lock()
		delete(c.entries, key)
		c.metrics.Size--
		c.metrics.Misses++
		c.metrics.Evictions++
		c.updateHitRate()
		c.mutex.Unlock()
		return nil, false
	}
	
	// Update entry metadata
	c.mutex.Lock()
	entry.LastAccessed = time.Now()
	entry.AccessCount++
	c.metrics.Hits++
	c.updateHitRate()
	c.mutex.Unlock()
	
	return entry.Value, true
}

// Set sets a value in the cache with the default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.config.DefaultTTL)
}

// SetWithTTL sets a value in the cache with a specific TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	
	// Check if we need to evict an entry
	if len(c.entries) >= c.config.MaxSize && c.entries[key] == nil {
		c.evict()
	}
	
	// Create or update the entry
	c.entries[key] = &CacheEntry{
		Key:          key,
		Value:        value,
		Expiration:   now.Add(ttl),
		LastAccessed: now,
		AccessCount:  0,
	}
	
	// Update metrics
	if c.entries[key] == nil {
		c.metrics.Size++
	}
}

// evict evicts an entry based on the eviction policy
func (c *Cache) evict() {
	if len(c.entries) == 0 {
		return
	}
	
	var keyToEvict string
	
	switch c.config.EvictionPolicy {
	case LRUEviction:
		// Evict the least recently used entry
		var oldestAccess time.Time
		first := true
		
		for key, entry := range c.entries {
			if first || entry.LastAccessed.Before(oldestAccess) {
				keyToEvict = key
				oldestAccess = entry.LastAccessed
				first = false
			}
		}
	
	case LFUEviction:
		// Evict the least frequently used entry
		var lowestCount int
		first := true
		
		for key, entry := range c.entries {
			if first || entry.AccessCount < lowestCount {
				keyToEvict = key
				lowestCount = entry.AccessCount
				first = false
			}
		}
	
	case TTLEviction:
		// Evict the entry closest to expiration
		var earliestExpiration time.Time
		first := true
		
		for key, entry := range c.entries {
			if first || entry.Expiration.Before(earliestExpiration) {
				keyToEvict = key
				earliestExpiration = entry.Expiration
				first = false
			}
		}
	
	case RandomEviction:
		// Evict a random entry
		for key := range c.entries {
			keyToEvict = key
			break
		}
	}
	
	if keyToEvict != "" {
		delete(c.entries, keyToEvict)
		c.metrics.Size--
		c.metrics.Evictions++
		
		c.logger.Debug("Cache entry evicted",
			zap.String("key", keyToEvict),
			zap.Int("policy", int(c.config.EvictionPolicy)))
	}
}

// Delete deletes a value from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if _, found := c.entries[key]; found {
		delete(c.entries, key)
		c.metrics.Size--
	}
}

// GetWithLoader gets a value from the cache, loading it if not present
func (c *Cache) GetWithLoader(ctx context.Context, key string, loader func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, found := c.Get(key); found {
		return value, nil
	}
	
	// Load the value
	value, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	
	// Cache the value
	c.Set(key, value)
	
	return value, nil
}

// GetMetrics returns a copy of the current metrics
func (c *Cache) GetMetrics() CacheMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return CacheMetrics{
		Hits:      c.metrics.Hits,
		Misses:    c.metrics.Misses,
		Evictions: c.metrics.Evictions,
		Size:      c.metrics.Size,
		HitRate:   c.metrics.HitRate,
	}
}

// updateHitRate updates the cache hit rate
func (c *Cache) updateHitRate() {
	total := c.metrics.Hits + c.metrics.Misses
	if total > 0 {
		c.metrics.HitRate = float64(c.metrics.Hits) / float64(total)
	}
}

// Clear clears all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.entries = make(map[string]*CacheEntry)
	c.metrics.Size = 0
	c.metrics.Evictions += int64(len(c.entries))
	
	c.logger.Info("Cache cleared", zap.String("name", c.name))
}

// Close stops the janitor and cleans up resources
func (c *Cache) Close() {
	close(c.stopChan)
}

