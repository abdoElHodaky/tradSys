package historical

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
)

// Config contains configuration for the historical data service
type Config struct {
	// MaxCacheSize is the maximum number of items to keep in the cache
	MaxCacheSize int
	
	// CacheTTL is the time-to-live for cache entries
	CacheTTL time.Duration
	
	// MaxConcurrentRequests is the maximum number of concurrent requests
	MaxConcurrentRequests int
}

// Service provides historical market data
type Service struct {
	logger    *zap.Logger
	config    Config
	
	// Cache for historical data
	cache     map[string]*CacheEntry
	cacheMu   sync.RWMutex
	
	// Semaphore for limiting concurrent requests
	semaphore chan struct{}
	
	// Track memory usage
	memoryUsage int64
	memoryMu    sync.RWMutex
}

// CacheEntry represents a cached historical data entry
type CacheEntry struct {
	Data      *marketdata.HistoricalDataResponse
	Timestamp time.Time
	Size      int64
}

// NewService creates a new historical data service
func NewService(config Config, logger *zap.Logger) *Service {
	if config.MaxCacheSize <= 0 {
		config.MaxCacheSize = 1000
	}
	
	if config.CacheTTL <= 0 {
		config.CacheTTL = 24 * time.Hour
	}
	
	if config.MaxConcurrentRequests <= 0 {
		config.MaxConcurrentRequests = 10
	}
	
	service := &Service{
		logger:    logger,
		config:    config,
		cache:     make(map[string]*CacheEntry),
		semaphore: make(chan struct{}, config.MaxConcurrentRequests),
	}
	
	// Start cache cleanup goroutine
	go service.cleanupCache()
	
	return service
}

// GetHistoricalData gets historical market data
func (s *Service) GetHistoricalData(ctx context.Context, request *marketdata.HistoricalDataRequest) (*marketdata.HistoricalDataResponse, error) {
	// Generate cache key
	cacheKey := s.generateCacheKey(request)
	
	// Check cache first
	if data := s.getFromCache(cacheKey); data != nil {
		s.logger.Debug("Cache hit for historical data",
			zap.String("symbol", request.Symbol),
			zap.String("timeframe", request.Timeframe),
			zap.Time("from", time.Unix(request.From, 0)),
			zap.Time("to", time.Unix(request.To, 0)))
		return data, nil
	}
	
	// Acquire a semaphore slot
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled while waiting for semaphore: %w", ctx.Err())
	}
	
	// Fetch the data
	data, err := s.fetchHistoricalData(ctx, request)
	if err != nil {
		return nil, err
	}
	
	// Cache the data
	s.cacheData(cacheKey, data)
	
	return data, nil
}

// generateCacheKey generates a cache key for a request
func (s *Service) generateCacheKey(request *marketdata.HistoricalDataRequest) string {
	return fmt.Sprintf("%s-%s-%d-%d", request.Symbol, request.Timeframe, request.From, request.To)
}

// getFromCache gets data from the cache
func (s *Service) getFromCache(key string) *marketdata.HistoricalDataResponse {
	s.cacheMu.RLock()
	entry, exists := s.cache[key]
	s.cacheMu.RUnlock()
	
	if !exists {
		return nil
	}
	
	// Check if the entry has expired
	if time.Since(entry.Timestamp) > s.config.CacheTTL {
		// Remove expired entry
		s.cacheMu.Lock()
		if entry, stillExists := s.cache[key]; stillExists {
			s.decreaseMemoryUsage(entry.Size)
			delete(s.cache, key)
		}
		s.cacheMu.Unlock()
		return nil
	}
	
	return entry.Data
}

// cacheData caches data
func (s *Service) cacheData(key string, data *marketdata.HistoricalDataResponse) {
	// Estimate size of data
	size := s.estimateSize(data)
	
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	
	// Check if we need to evict entries to make room
	s.evictIfNeeded(size)
	
	// Add to cache
	entry := &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		Size:      size,
	}
	s.cache[key] = entry
	s.increaseMemoryUsage(size)
}

// fetchHistoricalData fetches historical data
func (s *Service) fetchHistoricalData(ctx context.Context, request *marketdata.HistoricalDataRequest) (*marketdata.HistoricalDataResponse, error) {
	// This would be implemented to fetch data from a data provider
	// For now, we'll just return a placeholder
	s.logger.Info("Fetching historical data",
		zap.String("symbol", request.Symbol),
		zap.String("timeframe", request.Timeframe),
		zap.Time("from", time.Unix(request.From, 0)),
		zap.Time("to", time.Unix(request.To, 0)))
	
	// Simulate a delay
	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled while fetching data: %w", ctx.Err())
	}
	
	// Return placeholder data
	return &marketdata.HistoricalDataResponse{
		Symbol:    request.Symbol,
		Timeframe: request.Timeframe,
		From:      request.From,
		To:        request.To,
		Candles:   make([]*marketdata.Candle, 0),
	}, nil
}

// estimateSize estimates the size of a data response
func (s *Service) estimateSize(data *marketdata.HistoricalDataResponse) int64 {
	// Base size for the response
	size := int64(100)
	
	// Add size for each candle
	size += int64(len(data.Candles) * 50)
	
	return size
}

// evictIfNeeded evicts cache entries if needed to make room
func (s *Service) evictIfNeeded(newEntrySize int64) {
	// Check if adding the new entry would exceed the cache size limit
	for s.memoryUsage+newEntrySize > int64(s.config.MaxCacheSize) && len(s.cache) > 0 {
		// Find the oldest entry
		var oldestKey string
		var oldestTime time.Time
		var oldestSize int64
		
		for key, entry := range s.cache {
			if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.Timestamp
				oldestSize = entry.Size
			}
		}
		
		// Evict the oldest entry
		if oldestKey != "" {
			s.logger.Debug("Evicting cache entry to make room",
				zap.String("key", oldestKey),
				zap.Time("timestamp", oldestTime))
			
			s.decreaseMemoryUsage(oldestSize)
			delete(s.cache, oldestKey)
		}
	}
}

// cleanupCache periodically cleans up expired cache entries
func (s *Service) cleanupCache() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.cacheMu.Lock()
		
		now := time.Now()
		for key, entry := range s.cache {
			if now.Sub(entry.Timestamp) > s.config.CacheTTL {
				s.logger.Debug("Removing expired cache entry",
					zap.String("key", key),
					zap.Time("timestamp", entry.Timestamp),
					zap.Duration("age", now.Sub(entry.Timestamp)))
				
				s.decreaseMemoryUsage(entry.Size)
				delete(s.cache, key)
			}
		}
		
		s.cacheMu.Unlock()
	}
}

// increaseMemoryUsage increases the tracked memory usage
func (s *Service) increaseMemoryUsage(size int64) {
	s.memoryMu.Lock()
	defer s.memoryMu.Unlock()
	
	s.memoryUsage += size
}

// decreaseMemoryUsage decreases the tracked memory usage
func (s *Service) decreaseMemoryUsage(size int64) {
	s.memoryMu.Lock()
	defer s.memoryMu.Unlock()
	
	s.memoryUsage -= size
	if s.memoryUsage < 0 {
		s.memoryUsage = 0
	}
}

// GetMemoryUsage gets the current memory usage
func (s *Service) GetMemoryUsage() int64 {
	s.memoryMu.RLock()
	defer s.memoryMu.RUnlock()
	
	return s.memoryUsage
}

// ClearCache clears the cache
func (s *Service) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	
	s.cache = make(map[string]*CacheEntry)
	
	s.memoryMu.Lock()
	s.memoryUsage = 0
	s.memoryMu.Unlock()
	
	s.logger.Info("Cache cleared")
}

// GetCacheStats gets cache statistics
func (s *Service) GetCacheStats() map[string]interface{} {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	
	s.memoryMu.RLock()
	defer s.memoryMu.RUnlock()
	
	return map[string]interface{}{
		"entries":      len(s.cache),
		"memory_usage": s.memoryUsage,
		"max_size":     s.config.MaxCacheSize,
	}
}

