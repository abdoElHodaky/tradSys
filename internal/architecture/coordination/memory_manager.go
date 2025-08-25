package coordination

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MemoryPressureLevel represents the level of memory pressure in the system
type MemoryPressureLevel int

const (
	// MemoryPressureLow indicates low memory pressure
	MemoryPressureLow MemoryPressureLevel = iota
	// MemoryPressureMedium indicates medium memory pressure
	MemoryPressureMedium
	// MemoryPressureHigh indicates high memory pressure
	MemoryPressureHigh
	// MemoryPressureCritical indicates critical memory pressure
	MemoryPressureCritical
)

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

// DefaultMemoryManagerConfig returns the default memory manager configuration
func DefaultMemoryManagerConfig() MemoryManagerConfig {
	return MemoryManagerConfig{
		TotalLimit:        1024 * 1024 * 1024, // 1GB
		LowThreshold:      0.6,                // 60%
		MediumThreshold:   0.75,               // 75%
		HighThreshold:     0.85,               // 85%
		CriticalThreshold: 0.95,               // 95%
		AutoUnloadEnabled: true,
		MinIdleTime:       300,                // 5 minutes
		CheckInterval:     60,                 // 1 minute
	}
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
	
	// Unload callback
	unloadCallback func(ctx context.Context, componentName string) error
	
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

// SetUnloadCallback sets the callback function for unloading components
func (m *MemoryManager) SetUnloadCallback(callback func(ctx context.Context, componentName string) error) {
	m.unloadCallback = callback
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

// StartAutoUnloader starts the automatic component unloader
func (m *MemoryManager) StartAutoUnloader(ctx context.Context) {
	if !m.config.AutoUnloadEnabled {
		m.logger.Info("Automatic component unloading is disabled")
		return
	}
	
	m.logger.Info("Starting automatic component unloader",
		zap.Int("check_interval", m.config.CheckInterval),
		zap.Int("min_idle_time", m.config.MinIdleTime),
	)
	
	go m.runAutoUnloader(ctx)
}

// StopAutoUnloader stops the automatic component unloader
func (m *MemoryManager) StopAutoUnloader() {
	close(m.stopCh)
}

// runAutoUnloader runs the automatic component unloader
func (m *MemoryManager) runAutoUnloader(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.config.CheckInterval) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.checkAndUnloadComponents(ctx)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
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

// checkAndUnloadComponents checks memory pressure and unloads components if necessary
func (m *MemoryManager) checkAndUnloadComponents(ctx context.Context) {
	// Get the current memory pressure level
	pressureLevel := m.GetMemoryPressureLevel()
	
	// If memory pressure is low, do nothing
	if pressureLevel == MemoryPressureLow {
		return
	}
	
	// Get the minimum idle time based on pressure level
	minIdleTime := m.getMinIdleTimeForPressureLevel(pressureLevel)
	
	// Get the maximum number of components to unload based on pressure level
	maxUnload := m.getMaxUnloadForPressureLevel(pressureLevel)
	
	// Get the candidates for unloading
	candidates := m.getUnloadCandidates(minIdleTime, maxUnload)
	
	// Unload the candidates
	for _, candidate := range candidates {
		if m.unloadCallback != nil {
			err := m.unloadCallback(ctx, candidate.Name)
			if err != nil {
				m.logger.Error("Failed to unload component",
					zap.String("component", candidate.Name),
					zap.Error(err),
				)
				continue
			}
			
			m.logger.Info("Automatically unloaded component due to memory pressure",
				zap.String("component", candidate.Name),
				zap.String("type", candidate.Type),
				zap.Int64("memory_freed", candidate.MemoryUsage),
				zap.String("pressure_level", m.pressureLevelToString(pressureLevel)),
			)
		}
	}
}

// getMinIdleTimeForPressureLevel returns the minimum idle time based on pressure level
func (m *MemoryManager) getMinIdleTimeForPressureLevel(level MemoryPressureLevel) time.Duration {
	baseIdleTime := time.Duration(m.config.MinIdleTime) * time.Second
	
	switch level {
	case MemoryPressureMedium:
		return baseIdleTime
	case MemoryPressureHigh:
		return baseIdleTime / 2
	case MemoryPressureCritical:
		return baseIdleTime / 4
	default:
		return baseIdleTime
	}
}

// getMaxUnloadForPressureLevel returns the maximum number of components to unload
func (m *MemoryManager) getMaxUnloadForPressureLevel(level MemoryPressureLevel) int {
	switch level {
	case MemoryPressureMedium:
		return 2
	case MemoryPressureHigh:
		return 5
	case MemoryPressureCritical:
		return 10
	default:
		return 1
	}
}

// getUnloadCandidates returns a list of components that can be unloaded
func (m *MemoryManager) getUnloadCandidates(minIdleTime time.Duration, maxCount int) []*ComponentMemoryInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	now := time.Now()
	candidates := make([]*ComponentMemoryInfo, 0)
	
	// Collect candidates
	for _, info := range m.componentUsage {
		// Skip components that are in use
		if info.InUse {
			continue
		}
		
		// Check if the component has been idle for long enough
		idleTime := now.Sub(info.LastAccess)
		if idleTime < minIdleTime {
			continue
		}
		
		candidates = append(candidates, info)
	}
	
	// Sort candidates by priority (higher priority last) and then by idle time (most idle first)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].LastAccess.Before(candidates[j].LastAccess)
	})
	
	// Limit the number of candidates
	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}
	
	return candidates
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

// CanAllocateMemory checks if memory can be allocated
func (m *MemoryManager) CanAllocateMemory(memoryUsage int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if the memory can be allocated
	return m.totalUsage+memoryUsage <= m.config.TotalLimit
}

// AllocateMemory allocates memory for a component
func (m *MemoryManager) AllocateMemory(name string, memoryUsage int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if the component is already registered
	info, exists := m.componentUsage[name]
	if exists {
		// If the component is already registered, update its usage
		m.totalUsage = m.totalUsage - info.MemoryUsage + memoryUsage
		info.MemoryUsage = memoryUsage
		info.LastAccess = time.Now()
		return nil
	}
	
	// Check if the memory can be allocated
	if m.totalUsage+memoryUsage > m.config.TotalLimit {
		return fmt.Errorf("memory limit exceeded")
	}
	
	// Update total usage
	m.totalUsage += memoryUsage
	
	return nil
}

// FreeMemory tries to free memory by unloading components
func (m *MemoryManager) FreeMemory(ctx context.Context, requiredMemory int64) (bool, error) {
	// If no unload callback is set, we can't free memory
	if m.unloadCallback == nil {
		return false, fmt.Errorf("no unload callback set")
	}
	
	// Get the candidates for unloading
	candidates := m.getUnloadCandidates(0, 100)
	
	// Calculate how much memory we need to free
	m.mu.RLock()
	memoryToFree := requiredMemory - (m.config.TotalLimit - m.totalUsage)
	if memoryToFree <= 0 {
		m.mu.RUnlock()
		return true, nil
	}
	m.mu.RUnlock()
	
	// Try to free memory by unloading components
	freedMemory := int64(0)
	for _, candidate := range candidates {
		// Skip components that are in use
		if candidate.InUse {
			continue
		}
		
		// Try to unload the component
		err := m.unloadCallback(ctx, candidate.Name)
		if err != nil {
			m.logger.Error("Failed to unload component",
				zap.String("component", candidate.Name),
				zap.Error(err),
			)
			continue
		}
		
		// Update freed memory
		freedMemory += candidate.MemoryUsage
		
		m.logger.Info("Unloaded component to free memory",
			zap.String("component", candidate.Name),
			zap.Int64("memory_freed", candidate.MemoryUsage),
			zap.Int64("memory_required", requiredMemory),
			zap.Int64("total_freed", freedMemory),
		)
		
		// Check if we've freed enough memory
		if freedMemory >= memoryToFree {
			return true, nil
		}
	}
	
	// If we couldn't free enough memory, return false
	return false, nil
}

// GetMemoryUsage returns the current memory usage
func (m *MemoryManager) GetMemoryUsage() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.totalUsage
}

// GetMemoryLimit returns the memory limit
func (m *MemoryManager) GetMemoryLimit() int64 {
	return m.config.TotalLimit
}

// GetComponentInfo returns information about a component
func (m *MemoryManager) GetComponentInfo(name string) (*ComponentMemoryInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	info, exists := m.componentUsage[name]
	if !exists {
		return nil, fmt.Errorf("component %s not registered", name)
	}
	
	return info, nil
}

// GetAllComponentInfo returns information about all components
func (m *MemoryManager) GetAllComponentInfo() []*ComponentMemoryInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]*ComponentMemoryInfo, 0, len(m.componentUsage))
	for _, info := range m.componentUsage {
		result = append(result, info)
	}
	
	return result
}

// pressureLevelToString converts a memory pressure level to a string
func (m *MemoryManager) pressureLevelToString(level MemoryPressureLevel) string {
	switch level {
	case MemoryPressureLow:
		return "Low"
	case MemoryPressureMedium:
		return "Medium"
	case MemoryPressureHigh:
		return "High"
	case MemoryPressureCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// GetMemoryStats gets memory statistics
func (m *MemoryManager) GetMemoryStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_usage":    m.totalUsage,
		"total_limit":    m.config.TotalLimit,
		"usage_percent":  float64(m.totalUsage) / float64(m.config.TotalLimit) * 100.0,
		"components":     make(map[string]interface{}),
	}
	
	componentStats := stats["components"].(map[string]interface{})
	
	for name, info := range m.componentUsage {
		componentStats[name] = map[string]interface{}{
			"usage":       info.MemoryUsage,
			"last_access": info.LastAccess,
			"priority":    info.Priority,
			"in_use":      info.InUse,
			"type":        info.Type,
		}
	}
	
	return stats
}

