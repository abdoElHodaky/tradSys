package coordination

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MemoryManagerConfig contains configuration for the memory manager
type MemoryManagerConfig struct {
	// TotalLimit is the total memory limit in bytes
	TotalLimit int64
	
	// Memory pressure thresholds
	LowThreshold      float64
	MediumThreshold   float64
	HighThreshold     float64
	CriticalThreshold float64
	
	// Auto unload configuration
	AutoUnloadEnabled bool
	MinIdleTime       int
	CheckInterval     int
}

// ComponentMemoryInfo contains memory usage information for a component
type ComponentMemoryInfo struct {
	// Component identity
	Name string
	Type string
	
	// Memory usage
	MemoryUsage int64
	
	// Component priority (higher priority components are unloaded last)
	Priority int
	
	// Component state
	LastAccess time.Time
	InUse      bool
}

// MemoryManager manages memory usage and unloads components when memory pressure is high
type MemoryManager struct {
	// Configuration
	config MemoryManagerConfig
	
	// Memory usage
	totalUsage     int64
	componentUsage map[string]*ComponentMemoryInfo
	
	// Synchronization
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
	
	// Background monitoring
	stopCh chan struct{}
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(config MemoryManagerConfig, logger *zap.Logger) *MemoryManager {
	return &MemoryManager{
		config:         config,
		totalUsage:     0,
		componentUsage: make(map[string]*ComponentMemoryInfo),
		logger:         logger,
		stopCh:         make(chan struct{}),
	}
}

// Start starts the memory manager
func (m *MemoryManager) Start() {
	// Start background monitoring if auto unload is enabled
	if m.config.AutoUnloadEnabled {
		go m.monitorMemoryUsage()
	}
}

// Stop stops the memory manager
func (m *MemoryManager) Stop() {
	close(m.stopCh)
}

// RegisterComponent registers a component with the memory manager
func (m *MemoryManager) RegisterComponent(name, componentType string, memoryUsage int64, priority int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.componentUsage[name] = &ComponentMemoryInfo{
		Name:        name,
		Type:        componentType,
		MemoryUsage: memoryUsage,
		Priority:    priority,
		LastAccess:  time.Now(),
		InUse:       false,
	}
	
	m.totalUsage += memoryUsage
}

// UnregisterComponent unregisters a component from the memory manager
func (m *MemoryManager) UnregisterComponent(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if info, exists := m.componentUsage[name]; exists {
		m.totalUsage -= info.MemoryUsage
		delete(m.componentUsage, name)
	}
}

// UpdateComponentUsage updates the memory usage of a component
func (m *MemoryManager) UpdateComponentUsage(name string, memoryUsage int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if info, exists := m.componentUsage[name]; exists {
		m.totalUsage -= info.MemoryUsage
		info.MemoryUsage = memoryUsage
		m.totalUsage += memoryUsage
	}
}

// MarkComponentInUse marks a component as in use
func (m *MemoryManager) MarkComponentInUse(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if info, exists := m.componentUsage[name]; exists {
		info.InUse = true
		info.LastAccess = time.Now()
	}
}

// MarkComponentNotInUse marks a component as not in use
func (m *MemoryManager) MarkComponentNotInUse(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if info, exists := m.componentUsage[name]; exists {
		info.InUse = false
		info.LastAccess = time.Now()
	}
}

// GetMemoryPressureLevel returns the current memory pressure level
func (m *MemoryManager) GetMemoryPressureLevel() MemoryPressureLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usagePercentage := float64(m.totalUsage) / float64(m.config.TotalLimit)
	
	if usagePercentage >= m.config.CriticalThreshold {
		return MemoryPressureCritical
	} else if usagePercentage >= m.config.HighThreshold {
		return MemoryPressureHigh
	} else if usagePercentage >= m.config.MediumThreshold {
		return MemoryPressureMedium
	}
	
	return MemoryPressureLow
}

// GetTotalMemoryUsage returns the total memory usage
func (m *MemoryManager) GetTotalMemoryUsage() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.totalUsage
}

// GetComponentMemoryUsage returns the memory usage of a component
func (m *MemoryManager) GetComponentMemoryUsage(name string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if info, exists := m.componentUsage[name]; exists {
		return info.MemoryUsage, nil
	}
	
	return 0, fmt.Errorf("component %s not registered", name)
}

// GetAllComponentMemoryUsage returns the memory usage of all components
func (m *MemoryManager) GetAllComponentMemoryUsage() map[string]*ComponentMemoryInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy of the component usage map
	componentUsage := make(map[string]*ComponentMemoryInfo, len(m.componentUsage))
	for name, info := range m.componentUsage {
		componentUsage[name] = &ComponentMemoryInfo{
			Name:        info.Name,
			Type:        info.Type,
			MemoryUsage: info.MemoryUsage,
			Priority:    info.Priority,
			LastAccess:  info.LastAccess,
			InUse:       info.InUse,
		}
	}
	
	return componentUsage
}

// UnloadComponent unloads a component
func (m *MemoryManager) UnloadComponent(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if info, exists := m.componentUsage[name]; exists {
		if info.InUse {
			return fmt.Errorf("component %s is in use", name)
		}
		
		m.totalUsage -= info.MemoryUsage
		delete(m.componentUsage, name)
		
		return nil
	}
	
	return fmt.Errorf("component %s not registered", name)
}

// UnloadIdleComponents unloads idle components
func (m *MemoryManager) UnloadIdleComponents(minIdleTime int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Get all idle components
	var idleComponents []*ComponentMemoryInfo
	for _, info := range m.componentUsage {
		if !info.InUse && time.Since(info.LastAccess).Seconds() >= float64(minIdleTime) {
			idleComponents = append(idleComponents, info)
		}
	}
	
	// Sort idle components by priority (lower priority first)
	sort.Slice(idleComponents, func(i, j int) bool {
		return idleComponents[i].Priority < idleComponents[j].Priority
	})
	
	// Unload idle components
	unloadedCount := 0
	for _, info := range idleComponents {
		m.totalUsage -= info.MemoryUsage
		delete(m.componentUsage, info.Name)
		unloadedCount++
	}
	
	return unloadedCount
}

// monitorMemoryUsage monitors memory usage and unloads idle components when memory pressure is high
func (m *MemoryManager) monitorMemoryUsage() {
	ticker := time.NewTicker(time.Duration(m.config.CheckInterval) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Check memory pressure
			pressureLevel := m.GetMemoryPressureLevel()
			
			// Unload idle components if memory pressure is high
			if pressureLevel >= MemoryPressureMedium {
				m.UnloadIdleComponents(m.config.MinIdleTime)
			}
		case <-m.stopCh:
			return
		}
	}
}

