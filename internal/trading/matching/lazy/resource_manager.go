package lazy

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// ResourceManager manages resources for matching components
type ResourceManager struct {
	logger *zap.Logger
	
	// Providers
	engineProvider *MatchingEngineProvider
	orderBookProviders map[string]*OrderBookProvider
	algorithmProviders map[string]*MatchingAlgorithmProvider
	
	// Resource usage tracking
	memoryUsage     int64
	lastAccessTimes map[string]time.Time
	
	// Configuration
	cleanupInterval time.Duration
	idleTimeout     time.Duration
	
	// Synchronization
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewResourceManager creates a new resource manager
func NewResourceManager(
	logger *zap.Logger,
	engineProvider *MatchingEngineProvider,
) *ResourceManager {
	return &ResourceManager{
		logger:              logger,
		engineProvider:      engineProvider,
		orderBookProviders:  make(map[string]*OrderBookProvider),
		algorithmProviders:  make(map[string]*MatchingAlgorithmProvider),
		lastAccessTimes:     make(map[string]time.Time),
		cleanupInterval:     5 * time.Minute,
		idleTimeout:         30 * time.Minute,
		stopChan:            make(chan struct{}),
	}
}

// RegisterOrderBookProvider registers an order book provider
func (m *ResourceManager) RegisterOrderBookProvider(provider *OrderBookProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.orderBookProviders[provider.GetSymbol()] = provider
}

// RegisterAlgorithmProvider registers a matching algorithm provider
func (m *ResourceManager) RegisterAlgorithmProvider(provider *MatchingAlgorithmProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.algorithmProviders[provider.GetAlgorithmType()] = provider
}

// Start starts the resource manager
func (m *ResourceManager) Start() {
	m.wg.Add(1)
	go m.cleanupLoop()
}

// Stop stops the resource manager
func (m *ResourceManager) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

// cleanupLoop periodically checks for idle components and cleans them up
func (m *ResourceManager) cleanupLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.checkAndCleanupIdleComponents()
		case <-m.stopChan:
			return
		}
	}
}

// checkAndCleanupIdleComponents checks for idle components and cleans them up
func (m *ResourceManager) checkAndCleanupIdleComponents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	
	// Check matching engine
	if m.engineProvider.IsInitialized() {
		lastAccess, ok := m.lastAccessTimes["matching-engine"]
		if ok && now.Sub(lastAccess) > m.idleTimeout {
			m.logger.Info("Cleaning up idle matching engine",
				zap.Duration("idle_time", now.Sub(lastAccess)))
			
			// In a real implementation, we would call cleanup methods on the engine
			// For now, we just mark it as cleaned up
			delete(m.lastAccessTimes, "matching-engine")
		}
	}
	
	// Check order books
	for symbol, provider := range m.orderBookProviders {
		if provider.IsInitialized() {
			componentName := "order-book-" + symbol
			lastAccess, ok := m.lastAccessTimes[componentName]
			if ok && now.Sub(lastAccess) > m.idleTimeout {
				m.logger.Info("Cleaning up idle order book",
					zap.String("symbol", symbol),
					zap.Duration("idle_time", now.Sub(lastAccess)))
				
				// In a real implementation, we would call cleanup methods on the order book
				delete(m.lastAccessTimes, componentName)
			}
		}
	}
	
	// Check matching algorithms
	for algorithmType, provider := range m.algorithmProviders {
		if provider.IsInitialized() {
			componentName := "matching-algorithm-" + algorithmType
			lastAccess, ok := m.lastAccessTimes[componentName]
			if ok && now.Sub(lastAccess) > m.idleTimeout {
				m.logger.Info("Cleaning up idle matching algorithm",
					zap.String("type", algorithmType),
					zap.Duration("idle_time", now.Sub(lastAccess)))
				
				// In a real implementation, we would call cleanup methods on the algorithm
				delete(m.lastAccessTimes, componentName)
			}
		}
	}
}

// RecordAccess records access to a component
func (m *ResourceManager) RecordAccess(componentName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.lastAccessTimes[componentName] = time.Now()
}

// RecordMemoryUsage records memory usage for a component
func (m *ResourceManager) RecordMemoryUsage(componentName string, bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// In a real implementation, we would track memory usage per component
	// For now, we just update the total
	m.memoryUsage += bytes
	
	m.logger.Debug("Recorded memory usage",
		zap.String("component", componentName),
		zap.Int64("bytes", bytes),
		zap.Int64("total_bytes", m.memoryUsage))
}

// GetTotalMemoryUsage returns the total memory usage
func (m *ResourceManager) GetTotalMemoryUsage() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.memoryUsage
}

// GetComponentLastAccessTime returns the last access time for a component
func (m *ResourceManager) GetComponentLastAccessTime(componentName string) (time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	lastAccess, ok := m.lastAccessTimes[componentName]
	return lastAccess, ok
}

// SetIdleTimeout sets the idle timeout for components
func (m *ResourceManager) SetIdleTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.idleTimeout = timeout
}

// SetCleanupInterval sets the cleanup interval
func (m *ResourceManager) SetCleanupInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.cleanupInterval = interval
}

