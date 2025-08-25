package cqrs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QueryHandler is the interface that all query handlers must implement
type QueryHandler interface {
	// Type returns the type of query this handler can process
	Type() reflect.Type
	
	// ResultType returns the type of result this handler produces
	ResultType() reflect.Type
	
	// Handle processes the query and returns a result
	Handle(ctx context.Context, query interface{}) (interface{}, error)
}

// QueryHandlerFunc is a function that handles a query
type QueryHandlerFunc func(ctx context.Context, query interface{}) (interface{}, error)

// QueryMiddleware is a function that wraps a query handler
type QueryMiddleware func(QueryHandlerFunc) QueryHandlerFunc

// QueryBus dispatches queries to their handlers
type QueryBus struct {
	handlers   map[reflect.Type]QueryHandler
	middleware []QueryMiddleware
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewQueryBus creates a new query bus
func NewQueryBus(logger *zap.Logger) *QueryBus {
	return &QueryBus{
		handlers:   make(map[reflect.Type]QueryHandler),
		middleware: []QueryMiddleware{},
		logger:     logger,
	}
}

// RegisterHandler registers a query handler
func (b *QueryBus) RegisterHandler(handler QueryHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	queryType := handler.Type()
	if _, exists := b.handlers[queryType]; exists {
		return fmt.Errorf("handler already registered for query type %v", queryType)
	}
	
	b.handlers[queryType] = handler
	b.logger.Debug("Registered query handler", 
		zap.String("query_type", queryType.String()),
		zap.String("result_type", handler.ResultType().String()))
	
	return nil
}

// RegisterMiddleware registers middleware to be applied to all queries
func (b *QueryBus) RegisterMiddleware(middleware QueryMiddleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.middleware = append(b.middleware, middleware)
}

// Dispatch sends a query to its handler
func (b *QueryBus) Dispatch(ctx context.Context, query interface{}) (interface{}, error) {
	b.mu.RLock()
	queryType := reflect.TypeOf(query)
	handler, exists := b.handlers[queryType]
	middleware := make([]QueryMiddleware, len(b.middleware))
	copy(middleware, b.middleware)
	b.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no handler registered for query type %v", queryType)
	}
	
	// Apply middleware
	next := handler.Handle
	for i := len(middleware) - 1; i >= 0; i-- {
		next = middleware[i](next)
	}
	
	return next(ctx, query)
}

// TypedQueryHandler is a generic query handler for a specific query type
type TypedQueryHandler[Q any, R any] struct {
	handleFunc func(ctx context.Context, query Q) (R, error)
}

// NewTypedQueryHandler creates a new typed query handler
func NewTypedQueryHandler[Q any, R any](handleFunc func(ctx context.Context, query Q) (R, error)) *TypedQueryHandler[Q, R] {
	return &TypedQueryHandler[Q, R]{
		handleFunc: handleFunc,
	}
}

// Type returns the type of query this handler can process
func (h *TypedQueryHandler[Q, R]) Type() reflect.Type {
	var q Q
	return reflect.TypeOf(q)
}

// ResultType returns the type of result this handler produces
func (h *TypedQueryHandler[Q, R]) ResultType() reflect.Type {
	var r R
	return reflect.TypeOf(r)
}

// Handle processes the query
func (h *TypedQueryHandler[Q, R]) Handle(ctx context.Context, query interface{}) (interface{}, error) {
	typedQuery, ok := query.(Q)
	if !ok {
		return nil, fmt.Errorf("invalid query type: expected %T, got %T", *new(Q), query)
	}
	
	return h.handleFunc(ctx, typedQuery)
}

// CachingMiddleware caches query results
func CachingMiddleware(cache QueryCache) QueryMiddleware {
	return func(next QueryHandlerFunc) QueryHandlerFunc {
		return func(ctx context.Context, query interface{}) (interface{}, error) {
			// Check if the query is cacheable
			if cacheable, ok := query.(interface{ CacheKey() string }); ok {
				cacheKey := cacheable.CacheKey()
				
				// Try to get from cache
				if result, found := cache.Get(cacheKey); found {
					return result, nil
				}
				
				// Execute the query
				result, err := next(ctx, query)
				if err != nil {
					return nil, err
				}
				
				// Cache the result
				if ttl, ok := query.(interface{ CacheTTL() time.Duration }); ok {
					cache.Set(cacheKey, result, ttl.CacheTTL())
				} else {
					cache.Set(cacheKey, result, 5*time.Minute) // Default TTL
				}
				
				return result, nil
			}
			
			// Non-cacheable query
			return next(ctx, query)
		}
	}
}

// LoggingMiddleware logs queries before and after execution
func QueryLoggingMiddleware(logger *zap.Logger) QueryMiddleware {
	return func(next QueryHandlerFunc) QueryHandlerFunc {
		return func(ctx context.Context, query interface{}) (interface{}, error) {
			start := time.Now()
			logger.Debug("Executing query", 
				zap.String("query_type", reflect.TypeOf(query).String()))
			
			result, err := next(ctx, query)
			
			logger.Debug("Query executed",
				zap.String("query_type", reflect.TypeOf(query).String()),
				zap.Duration("duration", time.Since(start)),
				zap.Error(err))
			
			return result, err
		}
	}
}

// QueryCache defines the interface for a query cache
type QueryCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
}

// InMemoryQueryCache is a simple in-memory implementation of QueryCache
type InMemoryQueryCache struct {
	items map[string]cacheItem
	mu    sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewInMemoryQueryCache creates a new in-memory query cache
func NewInMemoryQueryCache() *InMemoryQueryCache {
	cache := &InMemoryQueryCache{
		items: make(map[string]cacheItem),
	}
	
	// Start cleanup goroutine
	go cache.cleanup()
	
	return cache
}

// Get retrieves a value from the cache
func (c *InMemoryQueryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	
	// Check if the item has expired
	if time.Now().After(item.expiration) {
		return nil, false
	}
	
	return item.value, true
}

// Set adds a value to the cache with the specified TTL
func (c *InMemoryQueryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache
func (c *InMemoryQueryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.items, key)
}

// cleanup periodically removes expired items from the cache
func (c *InMemoryQueryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
