package architecture

import (
	"sync"
	"time"
)

// Cache implements a simple in-memory cache with expiration
type Cache struct {
	items map[string]cacheItem
	mu    sync.RWMutex
}

// cacheItem represents an item in the cache
type cacheItem struct {
	value      interface{}
	expiration int64 // Unix timestamp
}

// NewCache creates a new cache
func NewCache() *Cache {
	cache := &Cache{
		items: make(map[string]cacheItem),
	}

	// Start a background goroutine to clean up expired items
	go cache.janitor()

	return cache
}

// Set adds an item to the cache with the specified expiration
func (c *Cache) Set(key string, value interface{}, expiration time.Duration) {
	var exp int64

	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:      value,
		expiration: exp,
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Check if the item has expired
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return nil, false
	}

	return item.value, true
}

// GetWithExpiration retrieves an item from the cache along with its expiration time
func (c *Cache) GetWithExpiration(key string) (interface{}, time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, time.Time{}, false
	}

	// Check if the item has expired
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return nil, time.Time{}, false
	}

	var expiration time.Time
	if item.expiration > 0 {
		expiration = time.Unix(0, item.expiration)
	}

	return item.value, expiration, true
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
}

// Count returns the number of items in the cache
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// janitor periodically cleans up expired items
func (c *Cache) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.deleteExpired()
	}
}

// deleteExpired deletes expired items from the cache
func (c *Cache) deleteExpired() {
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.expiration > 0 && now > item.expiration {
			delete(c.items, key)
		}
	}
}
