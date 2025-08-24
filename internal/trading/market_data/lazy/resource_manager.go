package lazy

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"go.uber.org/zap"
)

// ResourceManager manages resources for market data components
type ResourceManager struct {
	logger            *zap.Logger
	metrics           *lazy.AdaptiveMetrics
	cleanupInterval   time.Duration
	idleTimeout       time.Duration
	resources         map[string]*resourceInfo
	mu                sync.RWMutex
	cleanupRunning    bool
	cleanupStopCh     chan struct{}
	memoryThreshold   int64
	currentMemoryUsed int64
	memoryMu          sync.RWMutex
}

// resourceInfo tracks information about a resource
type resourceInfo struct {
	key           string
	lastAccessed  time.Time
	memoryUsage   int64
	cleanupFunc   func() error
	isInitialized bool
}

// NewResourceManager creates a new resource manager
func NewResourceManager(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
) *ResourceManager {
	return &ResourceManager{
		logger:          logger,
		metrics:         metrics,
		cleanupInterval: 5 * time.Minute,
		idleTimeout:     30 * time.Minute,
		resources:       make(map[string]*resourceInfo),
		cleanupStopCh:   make(chan struct{}),
		memoryThreshold: 2 * 1024 * 1024 * 1024, // 2GB default
	}
}

// RegisterResource registers a resource with the manager
func (m *ResourceManager) RegisterResource(
	key string,
	memoryUsage int64,
	cleanupFunc func() error,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if resource already exists
	if info, ok := m.resources[key]; ok {
		// Update existing resource
		info.lastAccessed = time.Now()
		info.memoryUsage = memoryUsage
		info.cleanupFunc = cleanupFunc
		info.isInitialized = true
		return
	}
	
	// Create new resource info
	m.resources[key] = &resourceInfo{
		key:           key,
		lastAccessed:  time.Now(),
		memoryUsage:   memoryUsage,
		cleanupFunc:   cleanupFunc,
		isInitialized: true,
	}
	
	// Update memory usage
	m.updateMemoryUsage(memoryUsage)
	
	m.logger.Debug("Registered resource",
		zap.String("key", key),
		zap.Int64("memory_usage", memoryUsage))
}

// AccessResource marks a resource as accessed
func (m *ResourceManager) AccessResource(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	info, ok := m.resources[key]
	if !ok {
		return false
	}
	
	info.lastAccessed = time.Now()
	return true
}

// UnregisterResource unregisters a resource
func (m *ResourceManager) UnregisterResource(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	info, ok := m.resources[key]
	if !ok {
		return nil
	}
	
	// Call cleanup function
	var err error
	if info.cleanupFunc != nil {
		err = info.cleanupFunc()
	}
	
	// Update memory usage
	m.updateMemoryUsage(-info.memoryUsage)
	
	// Remove resource
	delete(m.resources, key)
	
	m.logger.Debug("Unregistered resource",
		zap.String("key", key),
		zap.Int64("memory_usage", info.memoryUsage))
	
	return err
}

// StartCleanup starts the cleanup goroutine
func (m *ResourceManager) StartCleanup(ctx context.Context) {
	m.mu.Lock()
	if m.cleanupRunning {
		m.mu.Unlock()
		return
	}
	
	m.cleanupRunning = true
	m.mu.Unlock()
	
	go func() {
		ticker := time.NewTicker(m.cleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				m.cleanup()
			case <-m.cleanupStopCh:
				m.mu.Lock()
				m.cleanupRunning = false
				m.mu.Unlock()
				return
			case <-ctx.Done():
				m.mu.Lock()
				m.cleanupRunning = false
				m.mu.Unlock()
				return
			}
		}
	}()
}

// StopCleanup stops the cleanup goroutine
func (m *ResourceManager) StopCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.cleanupRunning {
		close(m.cleanupStopCh)
		m.cleanupStopCh = make(chan struct{})
	}
}

// cleanup cleans up idle resources
func (m *ResourceManager) cleanup() {
	now := time.Now()
	
	// Get resources to clean up
	var toCleanup []string
	
	m.mu.RLock()
	for key, info := range m.resources {
		if now.Sub(info.lastAccessed) > m.idleTimeout {
			toCleanup = append(toCleanup, key)
		}
	}
	m.mu.RUnlock()
	
	// Clean up resources
	for _, key := range toCleanup {
		m.logger.Debug("Cleaning up idle resource", zap.String("key", key))
		if err := m.UnregisterResource(key); err != nil {
			m.logger.Error("Failed to clean up resource",
				zap.String("key", key),
				zap.Error(err))
		}
	}
	
	m.logger.Debug("Cleanup completed",
		zap.Int("cleaned_up", len(toCleanup)),
		zap.Int64("memory_used", m.GetMemoryUsage()))
}

// SetCleanupInterval sets the cleanup interval
func (m *ResourceManager) SetCleanupInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.cleanupInterval = interval
}

// SetIdleTimeout sets the idle timeout
func (m *ResourceManager) SetIdleTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.idleTimeout = timeout
}

// SetMemoryThreshold sets the memory threshold
func (m *ResourceManager) SetMemoryThreshold(threshold int64) {
	m.memoryMu.Lock()
	defer m.memoryMu.Unlock()
	
	m.memoryThreshold = threshold
}

// GetMemoryUsage gets the current memory usage
func (m *ResourceManager) GetMemoryUsage() int64 {
	m.memoryMu.RLock()
	defer m.memoryMu.RUnlock()
	
	return m.currentMemoryUsed
}

// GetMemoryThreshold gets the memory threshold
func (m *ResourceManager) GetMemoryThreshold() int64 {
	m.memoryMu.RLock()
	defer m.memoryMu.RUnlock()
	
	return m.memoryThreshold
}

// updateMemoryUsage updates the current memory usage
func (m *ResourceManager) updateMemoryUsage(delta int64) {
	m.memoryMu.Lock()
	defer m.memoryMu.Unlock()
	
	m.currentMemoryUsed += delta
	
	// Ensure we don't go negative
	if m.currentMemoryUsed < 0 {
		m.currentMemoryUsed = 0
	}
	
	// Record metrics
	m.metrics.RecordMemoryUsage("market-data", m.currentMemoryUsed)
}

// GetResourceCount gets the number of resources
func (m *ResourceManager) GetResourceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.resources)
}

// IsMemoryAvailable checks if memory is available for a new resource
func (m *ResourceManager) IsMemoryAvailable(memoryNeeded int64) bool {
	m.memoryMu.RLock()
	defer m.memoryMu.RUnlock()
	
	return m.currentMemoryUsed+memoryNeeded <= m.memoryThreshold
}

// ForceCleanup forces cleanup of resources to free memory
func (m *ResourceManager) ForceCleanup(memoryNeeded int64) int64 {
	// Get resources sorted by last accessed time
	type resourceWithTime struct {
		key          string
		lastAccessed time.Time
		memoryUsage  int64
	}
	
	var resources []resourceWithTime
	
	m.mu.RLock()
	for key, info := range m.resources {
		resources = append(resources, resourceWithTime{
			key:          key,
			lastAccessed: info.lastAccessed,
			memoryUsage:  info.memoryUsage,
		})
	}
	m.mu.RUnlock()
	
	// Sort by last accessed time (oldest first)
	for i := 0; i < len(resources)-1; i++ {
		for j := i + 1; j < len(resources); j++ {
			if resources[i].lastAccessed.After(resources[j].lastAccessed) {
				resources[i], resources[j] = resources[j], resources[i]
			}
		}
	}
	
	// Clean up resources until we have enough memory
	var freedMemory int64
	for _, res := range resources {
		if m.IsMemoryAvailable(memoryNeeded) {
			break
		}
		
		m.logger.Info("Forcing cleanup of resource to free memory",
			zap.String("key", res.key),
			zap.Int64("memory_usage", res.memoryUsage))
		
		if err := m.UnregisterResource(res.key); err != nil {
			m.logger.Error("Failed to clean up resource",
				zap.String("key", res.key),
				zap.Error(err))
			continue
		}
		
		freedMemory += res.memoryUsage
	}
	
	return freedMemory
}

