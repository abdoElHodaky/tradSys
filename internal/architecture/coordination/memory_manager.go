package coordination

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MemoryManager manages memory allocation for components
type MemoryManager struct {
	// Total memory limit
	totalLimit int64
	
	// Current memory usage
	totalUsage int64
	
	// Component memory usage
	componentUsage map[string]*ComponentMemoryInfo
	
	// Mutex for protecting memory state
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
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
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.componentUsage[name] = &ComponentMemoryInfo{
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
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
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
	
	return nil
}

// ReleaseMemory releases memory for a component
func (m *MemoryManager) ReleaseMemory(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if component exists
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
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
	
	return nil
}

// UpdateUsage updates the memory usage for a component
func (m *MemoryManager) UpdateUsage(name string, usage int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if component exists
	info, exists := m.componentUsage[name]
	if !exists {
		return fmt.Errorf("component %s not registered", name)
	}
	
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
	
	return nil
}

// CanAllocate checks if memory can be allocated for a component
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
}

