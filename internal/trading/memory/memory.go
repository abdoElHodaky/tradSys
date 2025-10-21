package memory

import (
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// Manager manages memory for high-frequency trading operations
type Manager struct {
	mu              sync.RWMutex
	maxMemoryUsage  uint64
	currentUsage    uint64
	gcTargetPercent int
	lastGC          time.Time
	stats           *Stats
}

// Stats represents memory usage statistics
type Stats struct {
	TotalAlloc      uint64
	Sys             uint64
	Mallocs         uint64
	Frees           uint64
	HeapAlloc       uint64
	HeapSys         uint64
	HeapIdle        uint64
	HeapInuse       uint64
	HeapReleased    uint64
	HeapObjects     uint64
	StackInuse      uint64
	StackSys        uint64
	MSpanInuse      uint64
	MSpanSys        uint64
	MCacheInuse     uint64
	MCacheSys       uint64
	BuckHashSys     uint64
	GCSys           uint64
	OtherSys        uint64
	NextGC          uint64
	LastGC          time.Time
	PauseTotalNs    uint64
	NumGC           uint32
	NumForcedGC     uint32
	GCCPUFraction   float64
}

// NewManager creates a new memory manager
func NewManager(maxMemoryUsage uint64, gcTargetPercent int) *Manager {
	return &Manager{
		maxMemoryUsage:  maxMemoryUsage,
		gcTargetPercent: gcTargetPercent,
		stats:           &Stats{},
	}
}

// Start starts the memory manager
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Set GC target percentage
	debug.SetGCPercent(m.gcTargetPercent)
	
	// Start monitoring goroutine
	go m.monitor()
}

// Stop stops the memory manager
func (m *Manager) Stop() {
	// Reset GC to default
	debug.SetGCPercent(100)
}

// monitor continuously monitors memory usage
func (m *Manager) monitor() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		m.updateStats()
		m.checkMemoryPressure()
	}
}

// updateStats updates memory statistics
func (m *Manager) updateStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	m.stats.TotalAlloc = memStats.TotalAlloc
	m.stats.Sys = memStats.Sys
	m.stats.Mallocs = memStats.Mallocs
	m.stats.Frees = memStats.Frees
	m.stats.HeapAlloc = memStats.HeapAlloc
	m.stats.HeapSys = memStats.HeapSys
	m.stats.HeapIdle = memStats.HeapIdle
	m.stats.HeapInuse = memStats.HeapInuse
	m.stats.HeapReleased = memStats.HeapReleased
	m.stats.HeapObjects = memStats.HeapObjects
	m.stats.StackInuse = memStats.StackInuse
	m.stats.StackSys = memStats.StackSys
	m.stats.MSpanInuse = memStats.MSpanInuse
	m.stats.MSpanSys = memStats.MSpanSys
	m.stats.MCacheInuse = memStats.MCacheInuse
	m.stats.MCacheSys = memStats.MCacheSys
	m.stats.BuckHashSys = memStats.BuckHashSys
	m.stats.GCSys = memStats.GCSys
	m.stats.OtherSys = memStats.OtherSys
	m.stats.NextGC = memStats.NextGC
	m.stats.LastGC = time.Unix(0, int64(memStats.LastGC))
	m.stats.PauseTotalNs = memStats.PauseTotalNs
	m.stats.NumGC = memStats.NumGC
	m.stats.NumForcedGC = memStats.NumForcedGC
	m.stats.GCCPUFraction = memStats.GCCPUFraction
	
	m.currentUsage = memStats.HeapAlloc
}

// checkMemoryPressure checks if memory usage is too high and triggers GC if needed
func (m *Manager) checkMemoryPressure() {
	m.mu.RLock()
	usage := m.currentUsage
	maxUsage := m.maxMemoryUsage
	m.mu.RUnlock()
	
	if usage > maxUsage*80/100 { // 80% threshold
		m.forceGC()
	}
}

// forceGC forces garbage collection
func (m *Manager) forceGC() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	if now.Sub(m.lastGC) > time.Millisecond*100 { // Don't GC more than once per 100ms
		runtime.GC()
		m.lastGC = now
	}
}

// GetStats returns current memory statistics
func (m *Manager) GetStats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	statsCopy := *m.stats
	return &statsCopy
}

// GetCurrentUsage returns current memory usage in bytes
func (m *Manager) GetCurrentUsage() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentUsage
}

// GetMaxUsage returns maximum allowed memory usage in bytes
func (m *Manager) GetMaxUsage() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maxMemoryUsage
}

// SetMaxUsage sets maximum allowed memory usage
func (m *Manager) SetMaxUsage(maxUsage uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxMemoryUsage = maxUsage
}

// SetGCTargetPercent sets the GC target percentage
func (m *Manager) SetGCTargetPercent(percent int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcTargetPercent = percent
	debug.SetGCPercent(percent)
}

// IsMemoryPressureHigh returns true if memory pressure is high
func (m *Manager) IsMemoryPressureHigh() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentUsage > m.maxMemoryUsage*80/100
}

// Global memory manager instance
var globalManager *Manager

// InitGlobalManager initializes the global memory manager
func InitGlobalManager(maxMemoryUsage uint64, gcTargetPercent int) {
	globalManager = NewManager(maxMemoryUsage, gcTargetPercent)
	globalManager.Start()
}

// GetGlobalManager returns the global memory manager
func GetGlobalManager() *Manager {
	return globalManager
}

// GetStats returns stats from the global manager
func GetStats() *Stats {
	if globalManager != nil {
		return globalManager.GetStats()
	}
	return &Stats{}
}

// ForceGC forces garbage collection using the global manager
func ForceGC() {
	if globalManager != nil {
		globalManager.forceGC()
	}
}
