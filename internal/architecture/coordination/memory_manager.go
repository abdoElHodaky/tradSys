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
	// Component name
	Name string
	
	// Component type
	Type string
	
	// Estimated memory usage
	MemoryUsage int64
	
	// Component priority (lower is higher priority)
	Priority int
	
	// Last access time
	LastAccess time.Time
	
	// Whether the component is currently in use
	InUse bool
}

// MemoryManagerConfig contains configuration for the memory manager
type MemoryManagerConfig struct {
	// Total memory limit
	TotalLimit int64
	
	// Low memory threshold (percentage of total limit)
	LowThreshold float64
	
	// Medium memory threshold (percentage of total limit)
	MediumThreshold float64
	
	// High memory threshold (percentage of total limit)
	HighThreshold float64
	
	// Critical memory threshold (percentage of total limit)
	CriticalThreshold float64
	
	// Automatic unloading enabled
	AutoUnloadEnabled bool
	
	// Minimum idle time before a component can be unloaded (seconds)
	MinIdleTime int
	
	// Check interval for automatic unloading (seconds)
	CheckInterval int
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

// MemoryManager manages memory allocation and tracking for components
type MemoryManager struct {
	// Configuration
	config MemoryManagerConfig
	
	// Total memory usage
//=======
// MemoryManager manages memory allocation for components
type MemoryManager struct {
	// Total memory limit
	totalLimit int64
	
	// Current memory usage
//>>>>>>> main
	totalUsage int64
	
	// Component memory usage
	componentUsage map[string]*ComponentMemoryInfo
	
//<<<<<<< codegen-bot/integrate-coordination-system
	// Mutex for thread safety
//=======
	// Mutex for protecting memory state
//>>>>>>> main
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
//<<<<<<< codegen-bot/integrate-coordination-system
	
	// Unload callback
	unloadCallback func(ctx context.Context, componentName string) error
	
	// Stop channel for the background checker
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
//=======
}

// ComponentMemoryInfo contains memory information for a component
type ComponentMemoryInfo struct {
	// Current memory usage
	Usage int64
	
	// Estimated memory usage
	Estimate int64
	
	// Last access time
	LastAccess time.Time
	
	// Priority (lower is higher priority)
	Priority int
	
	// Whether the component is currently in use
	InUse bool
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(totalLimit int64, logger *zap.Logger) *MemoryManager {
	return &MemoryManager{
		totalLimit:     totalLimit,
		totalUsage:     0,
		componentUsage: make(map[string]*ComponentMemoryInfo),
		logger:         logger,
	}
}

// RegisterComponent registers a component with the memory manager
func (m *MemoryManager) RegisterComponent(name string, memoryEstimate int64) {
//>>>>>>> main
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.componentUsage[name] = &ComponentMemoryInfo{
//<<<<<<< codegen-bot/integrate-coordination-system
		Name:        name,
		Type:        componentType,
		MemoryUsage: memoryUsage,
		Priority:    priority,
		LastAccess:  time.Now(),
		InUse:       false,
	}
}

// UpdateComponentUsage updates the memory usage of a component
func (m *MemoryManager) UpdateComponentUsage(name string, memoryUsage int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
//=======
		Usage:      0,
		Estimate:   memoryEstimate,
		LastAccess: time.Now(),
		Priority:   50, // Default priority
		InUse:      false,
	}
}

// AllocateMemory allocates memory for a component
func (m *MemoryManager) AllocateMemory(name string, size int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if component exists
//>>>>>>> main
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
//<<<<<<< codegen-bot/integrate-coordination-system
	// Update total usage
	m.totalUsage = m.totalUsage - info.MemoryUsage + memoryUsage
	
	// Update component usage
	info.MemoryUsage = memoryUsage
	info.LastAccess = time.Now()
//=======
	// Check if there's enough memory
	if m.totalUsage+size > m.totalLimit {
		return fmt.Errorf("memory limit exceeded: requested %d, available %d", size, m.totalLimit-m.totalUsage)
	}
	
	// Allocate memory
	m.totalUsage += size
	info.Usage += size
	info.LastAccess = time.Now()
	info.InUse = true
	
	m.logger.Debug("Memory allocated",
		zap.String("component", name),
		zap.Int64("size", size),
		zap.Int64("total_usage", m.totalUsage),
		zap.Int64("total_limit", m.totalLimit),
	)
//>>>>>>> main
	
	return nil
}

//<<<<<<< codegen-bot/integrate-coordination-system
// MarkComponentAccessed marks a component as accessed
func (m *MemoryManager) MarkComponentAccessed(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
//=======
// ReleaseMemory releases memory for a component
func (m *MemoryManager) ReleaseMemory(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if component exists
//>>>>>>> main
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
//<<<<<<< codegen-bot/integrate-coordination-system
	info.LastAccess = time.Now()
	
	return nil
}

// MarkComponentInUse marks a component as in use or not in use
func (m *MemoryManager) MarkComponentInUse(name string, inUse bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
	info.InUse = inUse
	info.LastAccess = time.Now()
//=======
	// Release memory
	m.totalUsage -= info.Usage
	
	// Ensure we don't go negative
	if m.totalUsage < 0 {
		m.totalUsage = 0
	}
	
	m.logger.Debug("Memory released",
		zap.String("component", name),
		zap.Int64("size", info.Usage),
		zap.Int64("total_usage", m.totalUsage),
		zap.Int64("total_limit", m.totalLimit),
	)
	
	// Reset component usage
	info.Usage = 0
	info.LastAccess = time.Now()
	info.InUse = false
//>>>>>>> main
	
	return nil
}

//<<<<<<< codegen-bot/integrate-coordination-system
// UnregisterComponent unregisters a component from the memory manager
func (m *MemoryManager) UnregisterComponent(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
//=======
// UpdateUsage updates the memory usage for a component
func (m *MemoryManager) UpdateUsage(name string, usage int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if component exists
//>>>>>>> main
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
//<<<<<<< codegen-bot/integrate-coordination-system
	// Update total usage
	m.totalUsage -= info.MemoryUsage
	
	// Remove component
	delete(m.componentUsage, name)
//=======
	// Calculate the difference
	diff := usage - info.Usage
	
	// Check if there's enough memory
	if diff > 0 && m.totalUsage+diff > m.totalLimit {
		return fmt.Errorf("memory limit exceeded: requested %d, available %d", diff, m.totalLimit-m.totalUsage)
	}
	
	// Update usage
	m.totalUsage += diff
	info.Usage = usage
	info.LastAccess = time.Now()
//>>>>>>> main
	
	return nil
}

// CanAllocate checks if memory can be allocated for a component
//<<<<<<< codegen-bot/integrate-coordination-system
func (m *MemoryManager) CanAllocate(name string, memoryUsage int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if the component is already registered
	info, exists := m.componentUsage[name]
	if exists {
		// If the component is already registered, check if the new usage is higher
		if memoryUsage <= info.MemoryUsage {
			return true
		}
		
		// Check if the additional memory can be allocated
		additionalUsage := memoryUsage - info.MemoryUsage
		return m.totalUsage+additionalUsage <= m.config.TotalLimit
	}
	
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

// GetMemoryPressureLevel returns the current memory pressure level
func (m *MemoryManager) GetMemoryPressureLevel() MemoryPressureLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Calculate the memory usage percentage
	usagePercentage := float64(m.totalUsage) / float64(m.config.TotalLimit)
	
	// Determine the pressure level
	if usagePercentage >= m.config.CriticalThreshold {
		return MemoryPressureCritical
	} else if usagePercentage >= m.config.HighThreshold {
		return MemoryPressureHigh
	} else if usagePercentage >= m.config.MediumThreshold {
		return MemoryPressureMedium
	} else {
		return MemoryPressureLow
	}
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
//=======
func (m *MemoryManager) CanAllocate(name string, size int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.totalUsage+size <= m.totalLimit
}

// FreeMemory attempts to free memory by releasing unused components
func (m *MemoryManager) FreeMemory(requiredSize int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if we already have enough memory
	if m.totalUsage+requiredSize <= m.totalLimit {
		return true, nil
	}
	
	// Calculate how much memory we need to free
	toFree := (m.totalUsage + requiredSize) - m.totalLimit
	
	// Get a list of components sorted by priority and last access time
	type componentToFree struct {
		Name      string
		Info      *ComponentMemoryInfo
		FreeScore float64 // Higher score means more likely to be freed
	}
	
	candidates := make([]componentToFree, 0, len(m.componentUsage))
	
	now := time.Now()
	for name, info := range m.componentUsage {
		// Skip components that are in use
		if info.InUse {
			continue
		}
		
		// Skip components with no memory usage
		if info.Usage == 0 {
			continue
		}
		
		// Calculate a score based on priority and idle time
		idleTime := now.Sub(info.LastAccess).Seconds()
		priorityFactor := float64(100 - info.Priority) / 100.0 // Higher priority (lower number) = lower score
		
		// Score is primarily based on idle time, with priority as a secondary factor
		score := idleTime * priorityFactor
		
		candidates = append(candidates, componentToFree{
			Name:      name,
			Info:      info,
			FreeScore: score,
		})
	}
	
	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].FreeScore > candidates[j].FreeScore
	})
	
	// Try to free memory
	freed := int64(0)
	freedComponents := make([]string, 0)
	
	for _, candidate := range candidates {
		// Skip if this component wouldn't help
		if candidate.Info.Usage == 0 {
			continue
		}
		
		// Free this component
		freed += candidate.Info.Usage
		freedComponents = append(freedComponents, candidate.Name)
		
		// Reset component usage
		m.totalUsage -= candidate.Info.Usage
		candidate.Info.Usage = 0
		candidate.Info.InUse = false
		
		// Check if we've freed enough
		if freed >= toFree {
			break
		}
	}
	
	// Log what we freed
	if len(freedComponents) > 0 {
		m.logger.Info("Freed memory from components",
			zap.Int64("freed", freed),
			zap.Int64("required", toFree),
			zap.Strings("components", freedComponents),
		)
	}
	
	// Check if we freed enough
	return freed >= toFree, nil
}

// GetComponentUsage gets the memory usage for a component
func (m *MemoryManager) GetComponentUsage(name string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	info, exists := m.componentUsage[name]
	if !exists {
		return 0, fmt.Errorf("component %s not registered", name)
	}
	
	return info.Usage, nil
}

// GetTotalUsage gets the total memory usage
func (m *MemoryManager) GetTotalUsage() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.totalUsage
}

// GetTotalLimit gets the total memory limit
func (m *MemoryManager) GetTotalLimit() int64 {
	return m.totalLimit
}

// SetComponentPriority sets the priority for a component
func (m *MemoryManager) SetComponentPriority(name string, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
	info.Priority = priority
	return nil
}

// GetMemoryStats gets memory statistics
func (m *MemoryManager) GetMemoryStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_usage": m.totalUsage,
		"total_limit": m.totalLimit,
		"usage_percent": float64(m.totalUsage) / float64(m.totalLimit) * 100.0,
		"components": make(map[string]interface{}),
	}
	
	componentStats := stats["components"].(map[string]interface{})
	
	for name, info := range m.componentUsage {
		componentStats[name] = map[string]interface{}{
			"usage":       info.Usage,
			"estimate":    info.Estimate,
			"last_access": info.LastAccess,
			"priority":    info.Priority,
			"in_use":      info.InUse,
		}
	}
	
	return stats
//>>>>>>> main
}
