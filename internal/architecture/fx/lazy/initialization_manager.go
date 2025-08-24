package lazy

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// InitializationManager manages the initialization of lazy-loaded components
// to address conflicts and bottlenecks identified in the analysis.
type InitializationManager struct {
	logger           *zap.Logger
	providers        map[string]*EnhancedLazyProvider
	initQueue        []*EnhancedLazyProvider
	mu               sync.RWMutex
	maxConcurrent    int
	memoryThreshold  int64
	currentMemory    int64
	memoryMu         sync.RWMutex
	initSemaphore    chan struct{}
	warmupInProgress bool
}

// NewInitializationManager creates a new initialization manager
func NewInitializationManager(logger *zap.Logger) *InitializationManager {
	// Default to number of CPUs for concurrent initializations
	maxConcurrent := runtime.NumCPU()
	
	return &InitializationManager{
		logger:          logger,
		providers:       make(map[string]*EnhancedLazyProvider),
		maxConcurrent:   maxConcurrent,
		memoryThreshold: 1024 * 1024 * 1024, // 1GB default
		initSemaphore:   make(chan struct{}, maxConcurrent),
	}
}

// RegisterProvider registers a provider with the manager
func (m *InitializationManager) RegisterProvider(provider *EnhancedLazyProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.providers[provider.GetName()] = provider
	m.rebuildInitQueue()
}

// UnregisterProvider unregisters a provider
func (m *InitializationManager) UnregisterProvider(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.providers, name)
	m.rebuildInitQueue()
}

// rebuildInitQueue rebuilds the initialization queue based on priorities
func (m *InitializationManager) rebuildInitQueue() {
	m.initQueue = make([]*EnhancedLazyProvider, 0, len(m.providers))
	
	for _, provider := range m.providers {
		m.initQueue = append(m.initQueue, provider)
	}
	
	// Sort by priority (lower is higher priority)
	sort.Slice(m.initQueue, func(i, j int) bool {
		return m.initQueue[i].GetPriority() < m.initQueue[j].GetPriority()
	})
}

// SetMaxConcurrentInitializations sets the maximum number of concurrent initializations
func (m *InitializationManager) SetMaxConcurrentInitializations(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create a new semaphore with the new size
	oldSemaphore := m.initSemaphore
	m.initSemaphore = make(chan struct{}, max)
	m.maxConcurrent = max
	
	// Close the old semaphore
	close(oldSemaphore)
}

// SetMemoryThreshold sets the memory threshold for initialization
func (m *InitializationManager) SetMemoryThreshold(bytes int64) {
	m.memoryMu.Lock()
	defer m.memoryMu.Unlock()
	
	m.memoryThreshold = bytes
}

// WarmupComponents initializes high-priority components in the background
func (m *InitializationManager) WarmupComponents(ctx context.Context) {
	m.mu.Lock()
	if m.warmupInProgress {
		m.mu.Unlock()
		return
	}
	m.warmupInProgress = true
	
	// Make a copy of the queue to avoid holding the lock
	queue := make([]*EnhancedLazyProvider, len(m.initQueue))
	copy(queue, m.initQueue)
	m.mu.Unlock()
	
	// Initialize components in priority order
	for _, provider := range queue {
		// Skip if already initialized
		if provider.IsInitialized() {
			continue
		}
		
		// Check if we should continue warming up
		select {
		case <-ctx.Done():
			m.logger.Info("Warmup canceled", zap.Error(ctx.Err()))
			m.mu.Lock()
			m.warmupInProgress = false
			m.mu.Unlock()
			return
		default:
			// Continue
		}
		
		// Check memory threshold
		if !m.checkMemoryThreshold(provider.GetMemoryEstimate()) {
			m.logger.Warn("Skipping warmup due to memory threshold",
				zap.String("component", provider.GetName()),
				zap.Int64("memory_estimate", provider.GetMemoryEstimate()),
				zap.Int64("current_memory", m.getCurrentMemory()),
				zap.Int64("threshold", m.getMemoryThreshold()))
			continue
		}
		
		// Acquire semaphore
		m.initSemaphore <- struct{}{}
		
		// Initialize in background
		go func(p *EnhancedLazyProvider) {
			defer func() { <-m.initSemaphore }()
			
			m.logger.Info("Warming up component", zap.String("component", p.GetName()))
			
			// Initialize with timeout
			initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			
			_, err := p.GetWithContext(initCtx)
			if err != nil {
				m.logger.Warn("Failed to warm up component",
					zap.String("component", p.GetName()),
					zap.Error(err))
			} else {
				m.logger.Info("Component warmed up successfully",
					zap.String("component", p.GetName()))
				
				// Update memory usage
				m.updateMemoryUsage(p.GetMemoryEstimate())
			}
		}(provider)
	}
	
	// Wait for all initializations to complete or context to be canceled
	go func() {
		// Wait for context cancellation or completion
		<-ctx.Done()
		
		m.mu.Lock()
		m.warmupInProgress = false
		m.mu.Unlock()
		
		m.logger.Info("Warmup completed or canceled")
	}()
}

// checkMemoryThreshold checks if initializing a component would exceed the memory threshold
func (m *InitializationManager) checkMemoryThreshold(estimate int64) bool {
	// If estimate is unknown or zero, assume it's safe
	if estimate <= 0 {
		return true
	}
	
	m.memoryMu.RLock()
	defer m.memoryMu.RUnlock()
	
	return m.currentMemory+estimate <= m.memoryThreshold
}

// updateMemoryUsage updates the current memory usage
func (m *InitializationManager) updateMemoryUsage(delta int64) {
	// If estimate is unknown or zero, don't update
	if delta <= 0 {
		return
	}
	
	m.memoryMu.Lock()
	defer m.memoryMu.Unlock()
	
	m.currentMemory += delta
	
	// Ensure we don't go negative
	if m.currentMemory < 0 {
		m.currentMemory = 0
	}
}

// getCurrentMemory gets the current memory usage
func (m *InitializationManager) getCurrentMemory() int64 {
	m.memoryMu.RLock()
	defer m.memoryMu.RUnlock()
	
	return m.currentMemory
}

// getMemoryThreshold gets the memory threshold
func (m *InitializationManager) getMemoryThreshold() int64 {
	m.memoryMu.RLock()
	defer m.memoryMu.RUnlock()
	
	return m.memoryThreshold
}

// ResetComponent resets a component, forcing reinitialization on next use
func (m *InitializationManager) ResetComponent(name string) {
	m.mu.RLock()
	provider, ok := m.providers[name]
	m.mu.RUnlock()
	
	if ok {
		provider.Reset()
		
		// Update memory usage (subtract the estimate)
		m.updateMemoryUsage(-provider.GetMemoryEstimate())
	}
}

// GetInitializedComponentCount returns the number of initialized components
func (m *InitializationManager) GetInitializedComponentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	count := 0
	for _, provider := range m.providers {
		if provider.IsInitialized() {
			count++
		}
	}
	
	return count
}

// GetTotalComponentCount returns the total number of registered components
func (m *InitializationManager) GetTotalComponentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.providers)
}

// GetComponentNames returns the names of all registered components
func (m *InitializationManager) GetComponentNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	
	return names
}

// GetInitializedComponentNames returns the names of initialized components
func (m *InitializationManager) GetInitializedComponentNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0)
	for name, provider := range m.providers {
		if provider.IsInitialized() {
			names = append(names, name)
		}
	}
	
	return names
}

