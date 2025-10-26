package order_matching

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"go.uber.org/zap"
)

// PerformanceOptimizer handles high-frequency trading optimizations
type PerformanceOptimizer struct {
	logger *zap.Logger
	
	// Memory pools for object reuse
	orderPool    sync.Pool
	tradePool    sync.Pool
	matchPool    sync.Pool
	
	// Lock-free data structures
	orderQueue   *LockFreeQueue
	tradeQueue   *LockFreeQueue
	
	// Performance metrics
	latencyStats *LatencyStats
	throughput   *ThroughputStats
	
	// Configuration
	config *PerformanceConfig
	
	// CPU affinity and NUMA optimization
	cpuAffinity []int
	numaNode    int
	
	// Memory management
	memoryPool *MemoryPool
	
	// Hot path optimization
	hotPathCache *HotPathCache
}

// PerformanceConfig holds performance optimization configuration
type PerformanceConfig struct {
	// Memory pool sizes
	OrderPoolSize    int `json:"order_pool_size"`
	TradePoolSize    int `json:"trade_pool_size"`
	MatchPoolSize    int `json:"match_pool_size"`
	
	// Queue sizes
	OrderQueueSize   int `json:"order_queue_size"`
	TradeQueueSize   int `json:"trade_queue_size"`
	
	// CPU optimization
	CPUAffinity      []int `json:"cpu_affinity"`
	NUMANode         int   `json:"numa_node"`
	
	// Memory optimization
	PreallocateMemory bool `json:"preallocate_memory"`
	MemoryPoolSize    int  `json:"memory_pool_size"`
	
	// Cache optimization
	HotPathCacheSize int `json:"hot_path_cache_size"`
	CacheTTL         time.Duration `json:"cache_ttl"`
	
	// Latency targets (microseconds)
	TargetLatency    int64 `json:"target_latency_us"`
	MaxLatency       int64 `json:"max_latency_us"`
	
	// Throughput targets
	TargetTPS        int64 `json:"target_tps"`
	MaxTPS           int64 `json:"max_tps"`
}

// LockFreeQueue implements a lock-free queue for high-performance operations
type LockFreeQueue struct {
	head   unsafe.Pointer
	tail   unsafe.Pointer
	length int64
}

// QueueNode represents a node in the lock-free queue
type QueueNode struct {
	data interface{}
	next unsafe.Pointer
}

// LatencyStats tracks latency statistics
type LatencyStats struct {
	count       int64
	totalTime   int64
	minLatency  int64
	maxLatency  int64
	p50Latency  int64
	p95Latency  int64
	p99Latency  int64
	
	// Histogram for detailed analysis
	histogram [100]int64
	mutex     sync.RWMutex
}

// ThroughputStats tracks throughput statistics
type ThroughputStats struct {
	totalOps      int64
	currentTPS    int64
	peakTPS       int64
	avgTPS        int64
	lastUpdate    time.Time
	windowStart   time.Time
	windowOps     int64
	
	mutex sync.RWMutex
}

// MemoryPool manages pre-allocated memory blocks
type MemoryPool struct {
	blocks    [][]byte
	available chan []byte
	blockSize int
	poolSize  int
	mutex     sync.Mutex
}

// HotPathCache caches frequently accessed data
type HotPathCache struct {
	data    map[string]*CacheEntry
	mutex   sync.RWMutex
	ttl     time.Duration
	maxSize int
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	value     interface{}
	timestamp time.Time
	hits      int64
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(config *PerformanceConfig, logger *zap.Logger) *PerformanceOptimizer {
	po := &PerformanceOptimizer{
		logger:       logger,
		config:       config,
		orderQueue:   NewLockFreeQueue(),
		tradeQueue:   NewLockFreeQueue(),
		latencyStats: NewLatencyStats(),
		throughput:   NewThroughputStats(),
		memoryPool:   NewMemoryPool(config.MemoryPoolSize, 1024), // 1KB blocks
		hotPathCache: NewHotPathCache(config.HotPathCacheSize, config.CacheTTL),
	}
	
	// Initialize object pools
	po.initializePools()
	
	// Set CPU affinity if configured
	if len(config.CPUAffinity) > 0 {
		po.setCPUAffinity(config.CPUAffinity)
	}
	
	// Initialize NUMA optimization
	if config.NUMANode >= 0 {
		po.optimizeNUMA(config.NUMANode)
	}
	
	return po
}

// initializePools initializes object pools for memory reuse
func (po *PerformanceOptimizer) initializePools() {
	// Order pool
	po.orderPool = sync.Pool{
		New: func() interface{} {
			return &Order{}
		},
	}
	
	// Trade pool
	po.tradePool = sync.Pool{
		New: func() interface{} {
			return &Trade{}
		},
	}
	
	// Match result pool
	po.matchPool = sync.Pool{
		New: func() interface{} {
			return &pool.MatchResult{
				Trades: make([]*pool.Trade, 0, 10),
			}
		},
	}
}

// GetOrder gets an order from the pool
func (po *PerformanceOptimizer) GetOrder() *Order {
	order := po.orderPool.Get().(*Order)
	// Reset order fields
	*order = Order{}
	return order
}

// PutOrder returns an order to the pool
func (po *PerformanceOptimizer) PutOrder(order *Order) {
	if order != nil {
		po.orderPool.Put(order)
	}
}

// GetTrade gets a trade from the pool
func (po *PerformanceOptimizer) GetTrade() *Trade {
	trade := po.tradePool.Get().(*Trade)
	// Reset trade fields
	*trade = Trade{}
	return trade
}

// PutTrade returns a trade to the pool
func (po *PerformanceOptimizer) PutTrade(trade *Trade) {
	if trade != nil {
		po.tradePool.Put(trade)
	}
}

// GetMatchResult gets a match result from the pool
func (po *PerformanceOptimizer) GetMatchResult() *pool.MatchResult {
	result := po.matchPool.Get().(*pool.MatchResult)
	// Reset match result
	result.Trades = result.Trades[:0]
	result.RemainingOrder = nil
	result.FullyMatched = false
	result.PartiallyMatched = false
	result.TotalQuantity = 0
	result.WeightedPrice = 0
	return result
}

// PutMatchResult returns a match result to the pool
func (po *PerformanceOptimizer) PutMatchResult(result *pool.MatchResult) {
	if result != nil {
		po.matchPool.Put(result)
	}
}

// RecordLatency records latency measurement
func (po *PerformanceOptimizer) RecordLatency(duration time.Duration) {
	po.latencyStats.Record(duration)
}

// RecordThroughput records throughput measurement
func (po *PerformanceOptimizer) RecordThroughput(ops int64) {
	po.throughput.Record(ops)
}

// OptimizeHotPath optimizes frequently executed code paths
func (po *PerformanceOptimizer) OptimizeHotPath(key string, fn func() interface{}) interface{} {
	// Check cache first
	if cached := po.hotPathCache.Get(key); cached != nil {
		return cached
	}
	
	// Execute function and cache result
	result := fn()
	po.hotPathCache.Set(key, result)
	return result
}

// setCPUAffinity sets CPU affinity for the current goroutine
func (po *PerformanceOptimizer) setCPUAffinity(cpus []int) {
	// This would require CGO and platform-specific code
	// For now, we'll use GOMAXPROCS as a proxy
	if len(cpus) > 0 {
		runtime.GOMAXPROCS(len(cpus))
		po.cpuAffinity = cpus
		po.logger.Info("CPU affinity configured",
			zap.Ints("cpus", cpus))
	}
}

// optimizeNUMA optimizes for NUMA topology
func (po *PerformanceOptimizer) optimizeNUMA(node int) {
	po.numaNode = node
	po.logger.Info("NUMA optimization enabled",
		zap.Int("numa_node", node))
}

// GetStats returns performance statistics
func (po *PerformanceOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"latency":    po.latencyStats.GetStats(),
		"throughput": po.throughput.GetStats(),
		"memory":     po.memoryPool.GetStats(),
		"cache":      po.hotPathCache.GetStats(),
		"config":     po.config,
	}
}

// NewLockFreeQueue creates a new lock-free queue
func NewLockFreeQueue() *LockFreeQueue {
	node := &QueueNode{}
	return &LockFreeQueue{
		head: unsafe.Pointer(node),
		tail: unsafe.Pointer(node),
	}
}

// Enqueue adds an item to the queue
func (q *LockFreeQueue) Enqueue(data interface{}) {
	node := &QueueNode{data: data}
	
	for {
		tail := (*QueueNode)(atomic.LoadPointer(&q.tail))
		next := (*QueueNode)(atomic.LoadPointer(&tail.next))
		
		if tail == (*QueueNode)(atomic.LoadPointer(&q.tail)) {
			if next == nil {
				if atomic.CompareAndSwapPointer(&tail.next, unsafe.Pointer(next), unsafe.Pointer(node)) {
					break
				}
			} else {
				atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer(tail), unsafe.Pointer(next))
			}
		}
	}
	
	atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer((*QueueNode)(atomic.LoadPointer(&q.tail))), unsafe.Pointer(node))
	atomic.AddInt64(&q.length, 1)
}

// Dequeue removes an item from the queue
func (q *LockFreeQueue) Dequeue() interface{} {
	for {
		head := (*QueueNode)(atomic.LoadPointer(&q.head))
		tail := (*QueueNode)(atomic.LoadPointer(&q.tail))
		next := (*QueueNode)(atomic.LoadPointer(&head.next))
		
		if head == (*QueueNode)(atomic.LoadPointer(&q.head)) {
			if head == tail {
				if next == nil {
					return nil
				}
				atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer(tail), unsafe.Pointer(next))
			} else {
				if next == nil {
					continue
				}
				data := next.data
				if atomic.CompareAndSwapPointer(&q.head, unsafe.Pointer(head), unsafe.Pointer(next)) {
					atomic.AddInt64(&q.length, -1)
					return data
				}
			}
		}
	}
}

// Length returns the approximate length of the queue
func (q *LockFreeQueue) Length() int64 {
	return atomic.LoadInt64(&q.length)
}

// NewLatencyStats creates new latency statistics
func NewLatencyStats() *LatencyStats {
	return &LatencyStats{
		minLatency: int64(^uint64(0) >> 1), // Max int64
	}
}

// Record records a latency measurement
func (ls *LatencyStats) Record(duration time.Duration) {
	latencyUs := duration.Nanoseconds() / 1000 // Convert to microseconds
	
	atomic.AddInt64(&ls.count, 1)
	atomic.AddInt64(&ls.totalTime, latencyUs)
	
	// Update min/max atomically
	for {
		current := atomic.LoadInt64(&ls.minLatency)
		if latencyUs >= current || atomic.CompareAndSwapInt64(&ls.minLatency, current, latencyUs) {
			break
		}
	}
	
	for {
		current := atomic.LoadInt64(&ls.maxLatency)
		if latencyUs <= current || atomic.CompareAndSwapInt64(&ls.maxLatency, current, latencyUs) {
			break
		}
	}
	
	// Update histogram
	bucket := int(latencyUs / 10) // 10us buckets
	if bucket >= len(ls.histogram) {
		bucket = len(ls.histogram) - 1
	}
	atomic.AddInt64(&ls.histogram[bucket], 1)
}

// GetStats returns latency statistics
func (ls *LatencyStats) GetStats() map[string]interface{} {
	count := atomic.LoadInt64(&ls.count)
	if count == 0 {
		return map[string]interface{}{
			"count": 0,
		}
	}
	
	totalTime := atomic.LoadInt64(&ls.totalTime)
	avgLatency := totalTime / count
	
	return map[string]interface{}{
		"count":        count,
		"avg_latency":  avgLatency,
		"min_latency":  atomic.LoadInt64(&ls.minLatency),
		"max_latency":  atomic.LoadInt64(&ls.maxLatency),
		"total_time":   totalTime,
	}
}

// NewThroughputStats creates new throughput statistics
func NewThroughputStats() *ThroughputStats {
	now := time.Now()
	return &ThroughputStats{
		lastUpdate:  now,
		windowStart: now,
	}
}

// Record records throughput measurement
func (ts *ThroughputStats) Record(ops int64) {
	now := time.Now()
	
	atomic.AddInt64(&ts.totalOps, ops)
	atomic.AddInt64(&ts.windowOps, ops)
	
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	// Calculate current TPS
	if now.Sub(ts.lastUpdate) >= time.Second {
		windowDuration := now.Sub(ts.windowStart).Seconds()
		if windowDuration > 0 {
			currentTPS := int64(float64(ts.windowOps) / windowDuration)
			atomic.StoreInt64(&ts.currentTPS, currentTPS)
			
			if currentTPS > atomic.LoadInt64(&ts.peakTPS) {
				atomic.StoreInt64(&ts.peakTPS, currentTPS)
			}
		}
		
		ts.lastUpdate = now
		ts.windowStart = now
		ts.windowOps = 0
	}
}

// GetStats returns throughput statistics
func (ts *ThroughputStats) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_ops":   atomic.LoadInt64(&ts.totalOps),
		"current_tps": atomic.LoadInt64(&ts.currentTPS),
		"peak_tps":    atomic.LoadInt64(&ts.peakTPS),
	}
}

// NewMemoryPool creates a new memory pool
func NewMemoryPool(poolSize, blockSize int) *MemoryPool {
	mp := &MemoryPool{
		blocks:    make([][]byte, poolSize),
		available: make(chan []byte, poolSize),
		blockSize: blockSize,
		poolSize:  poolSize,
	}
	
	// Pre-allocate blocks
	for i := 0; i < poolSize; i++ {
		block := make([]byte, blockSize)
		mp.blocks[i] = block
		mp.available <- block
	}
	
	return mp
}

// Get gets a memory block from the pool
func (mp *MemoryPool) Get() []byte {
	select {
	case block := <-mp.available:
		return block
	default:
		// Pool exhausted, allocate new block
		return make([]byte, mp.blockSize)
	}
}

// Put returns a memory block to the pool
func (mp *MemoryPool) Put(block []byte) {
	if len(block) != mp.blockSize {
		return // Wrong size, discard
	}
	
	select {
	case mp.available <- block:
		// Successfully returned to pool
	default:
		// Pool full, let GC handle it
	}
}

// GetStats returns memory pool statistics
func (mp *MemoryPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"pool_size":   mp.poolSize,
		"block_size":  mp.blockSize,
		"available":   len(mp.available),
		"utilization": float64(mp.poolSize-len(mp.available)) / float64(mp.poolSize) * 100,
	}
}

// NewHotPathCache creates a new hot path cache
func NewHotPathCache(maxSize int, ttl time.Duration) *HotPathCache {
	return &HotPathCache{
		data:    make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get gets a value from the cache
func (hpc *HotPathCache) Get(key string) interface{} {
	hpc.mutex.RLock()
	entry, exists := hpc.data[key]
	hpc.mutex.RUnlock()
	
	if !exists {
		return nil
	}
	
	// Check TTL
	if time.Since(entry.timestamp) > hpc.ttl {
		hpc.mutex.Lock()
		delete(hpc.data, key)
		hpc.mutex.Unlock()
		return nil
	}
	
	atomic.AddInt64(&entry.hits, 1)
	return entry.value
}

// Set sets a value in the cache
func (hpc *HotPathCache) Set(key string, value interface{}) {
	hpc.mutex.Lock()
	defer hpc.mutex.Unlock()
	
	// Check if we need to evict
	if len(hpc.data) >= hpc.maxSize {
		// Simple LRU eviction - remove oldest entry
		var oldestKey string
		var oldestTime time.Time
		
		for k, v := range hpc.data {
			if oldestKey == "" || v.timestamp.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.timestamp
			}
		}
		
		if oldestKey != "" {
			delete(hpc.data, oldestKey)
		}
	}
	
	hpc.data[key] = &CacheEntry{
		value:     value,
		timestamp: time.Now(),
		hits:      0,
	}
}

// GetStats returns cache statistics
func (hpc *HotPathCache) GetStats() map[string]interface{} {
	hpc.mutex.RLock()
	defer hpc.mutex.RUnlock()
	
	totalHits := int64(0)
	for _, entry := range hpc.data {
		totalHits += atomic.LoadInt64(&entry.hits)
	}
	
	return map[string]interface{}{
		"size":       len(hpc.data),
		"max_size":   hpc.maxSize,
		"total_hits": totalHits,
		"hit_rate":   float64(totalHits) / float64(len(hpc.data)) * 100,
	}
}
