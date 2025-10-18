package memory

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// HFTMemoryConfig contains memory management configuration
type HFTMemoryConfig struct {
	// Pool settings
	EnableObjectPools   bool `yaml:"enable_object_pools" default:"true"`
	EnableBufferPools   bool `yaml:"enable_buffer_pools" default:"true"`
	EnableStringPools   bool `yaml:"enable_string_pools" default:"true"`
	
	// Memory limits
	MaxHeapSize         int64 `yaml:"max_heap_size" default:"2147483648"`         // 2GB
	GCTargetPercentage  int   `yaml:"gc_target_percentage" default:"200"`        // 200%
	
	// Monitoring
	EnableMemoryMonitoring bool          `yaml:"enable_memory_monitoring" default:"true"`
	MonitoringInterval     time.Duration `yaml:"monitoring_interval" default:"10s"`
	
	// Advanced settings
	EnableMemoryProfiling  bool `yaml:"enable_memory_profiling" default:"false"`
	EnableLeakDetection    bool `yaml:"enable_leak_detection" default:"true"`
	LeakDetectionThreshold int64 `yaml:"leak_detection_threshold" default:"104857600"` // 100MB
}

// MemoryStats contains detailed memory statistics
type MemoryStats struct {
	// Heap statistics
	HeapAlloc     uint64 `json:"heap_alloc"`
	HeapSys       uint64 `json:"heap_sys"`
	HeapIdle      uint64 `json:"heap_idle"`
	HeapInuse     uint64 `json:"heap_inuse"`
	HeapReleased  uint64 `json:"heap_released"`
	HeapObjects   uint64 `json:"heap_objects"`
	
	// Stack statistics
	StackInuse uint64 `json:"stack_inuse"`
	StackSys   uint64 `json:"stack_sys"`
	
	// GC statistics
	NumGC         uint32        `json:"num_gc"`
	PauseTotal    time.Duration `json:"pause_total"`
	LastGC        time.Time     `json:"last_gc"`
	NextGC        uint64        `json:"next_gc"`
	GCCPUFraction float64       `json:"gc_cpu_fraction"`
	
	// Custom statistics
	PoolAllocations   int64 `json:"pool_allocations"`
	PoolDeallocations int64 `json:"pool_deallocations"`
	BufferPoolHits    int64 `json:"buffer_pool_hits"`
	BufferPoolMisses  int64 `json:"buffer_pool_misses"`
	
	Timestamp time.Time `json:"timestamp"`
}

// HFTMemoryManager manages memory for HFT workloads
type HFTMemoryManager struct {
	config *HFTMemoryConfig
	
	// Pools
	objectPools map[string]*sync.Pool
	bufferPools map[int]*BufferPool
	stringPool  *StringPool
	
	// Statistics
	poolAllocations   int64
	poolDeallocations int64
	bufferPoolHits    int64
	bufferPoolMisses  int64
	
	// Monitoring
	stopMonitoring chan struct{}
	
	mu sync.RWMutex
}

// NewHFTMemoryManager creates a new memory manager
func NewHFTMemoryManager(config *HFTMemoryConfig) *HFTMemoryManager {
	if config == nil {
		config = &HFTMemoryConfig{
			EnableObjectPools:      true,
			EnableBufferPools:      true,
			EnableStringPools:      true,
			MaxHeapSize:           2147483648, // 2GB
			GCTargetPercentage:    200,
			EnableMemoryMonitoring: true,
			MonitoringInterval:    10 * time.Second,
			EnableMemoryProfiling: false,
			EnableLeakDetection:   true,
			LeakDetectionThreshold: 104857600, // 100MB
		}
	}
	
	manager := &HFTMemoryManager{
		config:         config,
		objectPools:    make(map[string]*sync.Pool),
		bufferPools:    make(map[int]*BufferPool),
		stringPool:     NewStringPool(),
		stopMonitoring: make(chan struct{}),
	}
	
	// Initialize buffer pools for common sizes
	if config.EnableBufferPools {
		manager.initBufferPools()
	}
	
	// Start monitoring if enabled
	if config.EnableMemoryMonitoring {
		go manager.monitoringLoop()
	}
	
	return manager
}

// initBufferPools initializes buffer pools for common sizes
func (m *HFTMemoryManager) initBufferPools() {
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
	
	for _, size := range sizes {
		m.bufferPools[size] = NewBufferPool(size)
	}
}

// GetObjectPool gets or creates an object pool for a specific type
func (m *HFTMemoryManager) GetObjectPool(name string, newFunc func() interface{}) *sync.Pool {
	m.mu.RLock()
	if pool, exists := m.objectPools[name]; exists {
		m.mu.RUnlock()
		return pool
	}
	m.mu.RUnlock()
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check after acquiring write lock
	if pool, exists := m.objectPools[name]; exists {
		return pool
	}
	
	pool := &sync.Pool{New: newFunc}
	m.objectPools[name] = pool
	return pool
}

// GetBuffer gets a buffer from the appropriate pool
func (m *HFTMemoryManager) GetBuffer(size int) []byte {
	if !m.config.EnableBufferPools {
		return make([]byte, size)
	}
	
	// Find the smallest pool that can accommodate the requested size
	poolSize := m.findBufferPoolSize(size)
	if pool, exists := m.bufferPools[poolSize]; exists {
		buf := pool.Get()
		atomic.AddInt64(&m.bufferPoolHits, 1)
		return buf[:size] // Return slice of requested size
	}
	
	// No suitable pool found, allocate directly
	atomic.AddInt64(&m.bufferPoolMisses, 1)
	return make([]byte, size)
}

// PutBuffer returns a buffer to the appropriate pool
func (m *HFTMemoryManager) PutBuffer(buf []byte) {
	if !m.config.EnableBufferPools || len(buf) == 0 {
		return
	}
	
	// Find the pool that matches the buffer capacity
	capacity := cap(buf)
	if pool, exists := m.bufferPools[capacity]; exists {
		pool.Put(buf)
	}
}

// findBufferPoolSize finds the smallest pool size that can accommodate the requested size
func (m *HFTMemoryManager) findBufferPoolSize(size int) int {
	for poolSize := range m.bufferPools {
		if poolSize >= size {
			return poolSize
		}
	}
	
	// Return the largest pool size if no suitable pool found
	maxSize := 0
	for poolSize := range m.bufferPools {
		if poolSize > maxSize {
			maxSize = poolSize
		}
	}
	return maxSize
}

// GetString gets a string from the string pool
func (m *HFTMemoryManager) GetString(s string) string {
	if !m.config.EnableStringPools {
		return s
	}
	
	return m.stringPool.Get(s)
}

// AllocateObject allocates an object from a pool
func (m *HFTMemoryManager) AllocateObject(poolName string) interface{} {
	if pool, exists := m.objectPools[poolName]; exists {
		atomic.AddInt64(&m.poolAllocations, 1)
		return pool.Get()
	}
	return nil
}

// DeallocateObject returns an object to a pool
func (m *HFTMemoryManager) DeallocateObject(poolName string, obj interface{}) {
	if pool, exists := m.objectPools[poolName]; exists {
		atomic.AddInt64(&m.poolDeallocations, 1)
		pool.Put(obj)
	}
}

// GetMemoryStats returns current memory statistics
func (m *HFTMemoryManager) GetMemoryStats() *MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &MemoryStats{
		HeapAlloc:         memStats.HeapAlloc,
		HeapSys:           memStats.HeapSys,
		HeapIdle:          memStats.HeapIdle,
		HeapInuse:         memStats.HeapInuse,
		HeapReleased:      memStats.HeapReleased,
		HeapObjects:       memStats.HeapObjects,
		StackInuse:        memStats.StackInuse,
		StackSys:          memStats.StackSys,
		NumGC:             memStats.NumGC,
		PauseTotal:        time.Duration(memStats.PauseTotalNs),
		LastGC:            time.Unix(0, int64(memStats.LastGC)),
		NextGC:            memStats.NextGC,
		GCCPUFraction:     memStats.GCCPUFraction,
		PoolAllocations:   atomic.LoadInt64(&m.poolAllocations),
		PoolDeallocations: atomic.LoadInt64(&m.poolDeallocations),
		BufferPoolHits:    atomic.LoadInt64(&m.bufferPoolHits),
		BufferPoolMisses:  atomic.LoadInt64(&m.bufferPoolMisses),
		Timestamp:         time.Now(),
	}
}

// monitoringLoop monitors memory usage and detects leaks
func (m *HFTMemoryManager) monitoringLoop() {
	ticker := time.NewTicker(m.config.MonitoringInterval)
	defer ticker.Stop()
	
	var lastStats *MemoryStats
	
	for {
		select {
		case <-m.stopMonitoring:
			return
		case <-ticker.C:
			stats := m.GetMemoryStats()
			
			// Check for memory leaks
			if m.config.EnableLeakDetection && lastStats != nil {
				m.checkForLeaks(lastStats, stats)
			}
			
			// Log memory statistics
			m.logMemoryStats(stats)
			
			lastStats = stats
		}
	}
}

// checkForLeaks checks for potential memory leaks
func (m *HFTMemoryManager) checkForLeaks(lastStats, currentStats *MemoryStats) {
	heapGrowth := int64(currentStats.HeapAlloc) - int64(lastStats.HeapAlloc)
	
	if heapGrowth > m.config.LeakDetectionThreshold {
		fmt.Printf("[MEMORY LEAK WARNING] Heap grew by %d bytes in %v\n",
			heapGrowth, currentStats.Timestamp.Sub(lastStats.Timestamp))
		
		// Force GC to see if memory can be reclaimed
		runtime.GC()
		
		// Check again after GC
		afterGCStats := m.GetMemoryStats()
		reclaimedMemory := int64(currentStats.HeapAlloc) - int64(afterGCStats.HeapAlloc)
		
		fmt.Printf("[MEMORY LEAK] Reclaimed %d bytes after forced GC\n", reclaimedMemory)
		
		if reclaimedMemory < heapGrowth/2 {
			fmt.Printf("[MEMORY LEAK CRITICAL] Potential memory leak detected!\n")
		}
	}
}

// logMemoryStats logs memory statistics
func (m *HFTMemoryManager) logMemoryStats(stats *MemoryStats) {
	fmt.Printf("[MEMORY] Heap: %d MB, Objects: %d, GC: %d, Pool Hit Rate: %.2f%%\n",
		stats.HeapAlloc/1024/1024,
		stats.HeapObjects,
		stats.NumGC,
		float64(stats.BufferPoolHits)/float64(stats.BufferPoolHits+stats.BufferPoolMisses)*100,
	)
}

// ForceGC forces a garbage collection
func (m *HFTMemoryManager) ForceGC() {
	runtime.GC()
}

// GetPoolStats returns statistics for all object pools
func (m *HFTMemoryManager) GetPoolStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Object pool statistics
	stats["object_pools"] = len(m.objectPools)
	stats["buffer_pools"] = len(m.bufferPools)
	stats["pool_allocations"] = atomic.LoadInt64(&m.poolAllocations)
	stats["pool_deallocations"] = atomic.LoadInt64(&m.poolDeallocations)
	stats["buffer_pool_hits"] = atomic.LoadInt64(&m.bufferPoolHits)
	stats["buffer_pool_misses"] = atomic.LoadInt64(&m.bufferPoolMisses)
	
	// Buffer pool details
	bufferPoolStats := make(map[string]interface{})
	for size, pool := range m.bufferPools {
		bufferPoolStats[fmt.Sprintf("size_%d", size)] = pool.GetStats()
	}
	stats["buffer_pool_details"] = bufferPoolStats
	
	return stats
}

// Close shuts down the memory manager
func (m *HFTMemoryManager) Close() {
	close(m.stopMonitoring)
}

// BufferPool manages a pool of byte buffers
type BufferPool struct {
	pool sync.Pool
	size int
	gets int64
	puts int64
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get gets a buffer from the pool
func (bp *BufferPool) Get() []byte {
	atomic.AddInt64(&bp.gets, 1)
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if cap(buf) != bp.size {
		return // Wrong size, don't return to pool
	}
	
	// Clear the buffer
	for i := range buf {
		buf[i] = 0
	}
	
	atomic.AddInt64(&bp.puts, 1)
	bp.pool.Put(buf[:bp.size]) // Ensure full capacity
}

// GetStats returns buffer pool statistics
func (bp *BufferPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"size": bp.size,
		"gets": atomic.LoadInt64(&bp.gets),
		"puts": atomic.LoadInt64(&bp.puts),
	}
}

// StringPool manages a pool of strings to reduce allocations
type StringPool struct {
	pool sync.Map // map[string]string
}

// NewStringPool creates a new string pool
func NewStringPool() *StringPool {
	return &StringPool{}
}

// Get gets a string from the pool, interning it if necessary
func (sp *StringPool) Get(s string) string {
	if cached, ok := sp.pool.Load(s); ok {
		return cached.(string)
	}
	
	// Store the string in the pool
	sp.pool.Store(s, s)
	return s
}

// MemoryProfiler provides memory profiling capabilities
type MemoryProfiler struct {
	enabled bool
	samples []MemoryStats
	mu      sync.RWMutex
}

// NewMemoryProfiler creates a new memory profiler
func NewMemoryProfiler() *MemoryProfiler {
	return &MemoryProfiler{
		samples: make([]MemoryStats, 0, 1000), // Keep last 1000 samples
	}
}

// Enable enables memory profiling
func (mp *MemoryProfiler) Enable() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.enabled = true
}

// Disable disables memory profiling
func (mp *MemoryProfiler) Disable() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.enabled = false
}

// Sample takes a memory sample
func (mp *MemoryProfiler) Sample(stats MemoryStats) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	if !mp.enabled {
		return
	}
	
	// Add sample
	mp.samples = append(mp.samples, stats)
	
	// Keep only last 1000 samples
	if len(mp.samples) > 1000 {
		mp.samples = mp.samples[1:]
	}
}

// GetSamples returns all memory samples
func (mp *MemoryProfiler) GetSamples() []MemoryStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	
	samples := make([]MemoryStats, len(mp.samples))
	copy(samples, mp.samples)
	return samples
}

// GetMemoryUsage returns current memory usage in bytes
func GetMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// GetObjectSize returns the size of an object in bytes
func GetObjectSize(obj interface{}) uintptr {
	return unsafe.Sizeof(obj)
}

// Global memory manager instance
var GlobalMemoryManager *HFTMemoryManager

// InitMemoryManager initializes the global memory manager
func InitMemoryManager(config *HFTMemoryConfig) {
	GlobalMemoryManager = NewHFTMemoryManager(config)
}
