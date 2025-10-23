package config

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
)

// HFTGCConfig contains garbage collection configuration for HFT workloads
type HFTGCConfig struct {
	// GC settings
	GCPercent   int   `yaml:"gc_percent" default:"200"`          // Run GC less frequently
	MemoryLimit int64 `yaml:"memory_limit" default:"2147483648"` // 2GB limit
	MaxProcs    int   `yaml:"max_procs" default:"0"`             // 0 = use all CPUs

	// Memory settings
	EnableMemoryLimit bool  `yaml:"enable_memory_limit" default:"true"`
	SoftMemoryLimit   int64 `yaml:"soft_memory_limit" default:"1610612736"` // 1.5GB

	// GC monitoring
	EnableGCMonitoring bool          `yaml:"enable_gc_monitoring" default:"true"`
	GCStatsInterval    time.Duration `yaml:"gc_stats_interval" default:"30s"`

	// Advanced settings
	EnableBallastHeap bool  `yaml:"enable_ballast_heap" default:"false"`
	BallastSize       int64 `yaml:"ballast_size" default:"1073741824"` // 1GB ballast
}

// GCStats contains garbage collection statistics
type GCStats struct {
	NumGC        uint32        `json:"num_gc"`
	PauseTotal   time.Duration `json:"pause_total"`
	PauseNs      []uint64      `json:"pause_ns"`
	LastGC       time.Time     `json:"last_gc"`
	NextGC       uint64        `json:"next_gc"`
	HeapAlloc    uint64        `json:"heap_alloc"`
	HeapSys      uint64        `json:"heap_sys"`
	HeapIdle     uint64        `json:"heap_idle"`
	HeapInuse    uint64        `json:"heap_inuse"`
	HeapReleased uint64        `json:"heap_released"`
	HeapObjects  uint64        `json:"heap_objects"`
	StackInuse   uint64        `json:"stack_inuse"`
	StackSys     uint64        `json:"stack_sys"`
	MSpanInuse   uint64        `json:"mspan_inuse"`
	MSpanSys     uint64        `json:"mspan_sys"`
	MCacheInuse  uint64        `json:"mcache_inuse"`
	MCacheSys    uint64        `json:"mcache_sys"`
	GCSys        uint64        `json:"gc_sys"`
	OtherSys     uint64        `json:"other_sys"`
}

// OptimizeGCForHFT configures garbage collection for HFT workloads
func OptimizeGCForHFT(config *HFTGCConfig) error {
	if config == nil {
		config = &HFTGCConfig{
			GCPercent:          200,
			MemoryLimit:        2147483648, // 2GB
			MaxProcs:           0,          // Use all CPUs
			EnableMemoryLimit:  true,
			SoftMemoryLimit:    1610612736, // 1.5GB
			EnableGCMonitoring: true,
			GCStatsInterval:    30 * time.Second,
			EnableBallastHeap:  false,
			BallastSize:        1073741824, // 1GB
		}
	}

	// Set GC percentage - higher values mean less frequent GC
	debug.SetGCPercent(config.GCPercent)

	// Set memory limit if enabled
	if config.EnableMemoryLimit {
		debug.SetMemoryLimit(config.MemoryLimit)
	}

	// Set GOMAXPROCS
	if config.MaxProcs > 0 {
		runtime.GOMAXPROCS(config.MaxProcs)
	} else {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// Create ballast heap if enabled (helps with GC pacing)
	if config.EnableBallastHeap {
		createBallastHeap(config.BallastSize)
	}

	// Start GC monitoring if enabled
	if config.EnableGCMonitoring {
		go monitorGCStats(config.GCStatsInterval)
	}

	return nil
}

// createBallastHeap creates a ballast heap to improve GC pacing
func createBallastHeap(size int64) {
	// Allocate a large slice that won't be used but helps with GC pacing
	ballast := make([]byte, size)

	// Prevent the ballast from being optimized away
	runtime.KeepAlive(ballast)

	fmt.Printf("[GC] Created ballast heap of %d bytes\n", size)
}

// monitorGCStats monitors garbage collection statistics
func monitorGCStats(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastStats runtime.MemStats
	runtime.ReadMemStats(&lastStats)

	for range ticker.C {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		// Calculate GC frequency and pause times
		gcCount := stats.NumGC - lastStats.NumGC
		if gcCount > 0 {
			// Calculate average pause time for recent GCs
			var totalPause uint64
			for i := uint32(0); i < gcCount && i < 256; i++ {
				idx := (stats.NumGC - 1 - i) % 256
				totalPause += stats.PauseNs[idx]
			}
			avgPause := time.Duration(totalPause / uint64(gcCount))

			fmt.Printf("[GC] Count: %d, Avg Pause: %v, Heap: %d MB, Next GC: %d MB\n",
				gcCount,
				avgPause,
				stats.HeapAlloc/1024/1024,
				stats.NextGC/1024/1024,
			)
		}

		lastStats = stats
	}
}

// GetGCStats returns current garbage collection statistics
func GetGCStats() *GCStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	gcStats := &GCStats{
		NumGC:        stats.NumGC,
		PauseTotal:   time.Duration(stats.PauseTotalNs),
		LastGC:       time.Unix(0, int64(stats.LastGC)),
		NextGC:       stats.NextGC,
		HeapAlloc:    stats.HeapAlloc,
		HeapSys:      stats.HeapSys,
		HeapIdle:     stats.HeapIdle,
		HeapInuse:    stats.HeapInuse,
		HeapReleased: stats.HeapReleased,
		HeapObjects:  stats.HeapObjects,
		StackInuse:   stats.StackInuse,
		StackSys:     stats.StackSys,
		MSpanInuse:   stats.MSpanInuse,
		MSpanSys:     stats.MSpanSys,
		MCacheInuse:  stats.MCacheInuse,
		MCacheSys:    stats.MCacheSys,
		GCSys:        stats.GCSys,
		OtherSys:     stats.OtherSys,
	}

	// Copy recent pause times
	gcStats.PauseNs = make([]uint64, len(stats.PauseNs))
	copy(gcStats.PauseNs, stats.PauseNs[:])

	return gcStats
}

// ForceGC forces a garbage collection cycle
func ForceGC() {
	runtime.GC()
}

// GetMemoryStats returns detailed memory statistics
func GetMemoryStats() map[string]interface{} {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return map[string]interface{}{
		// General statistics
		"alloc":       stats.Alloc,
		"total_alloc": stats.TotalAlloc,
		"sys":         stats.Sys,
		"lookups":     stats.Lookups,
		"mallocs":     stats.Mallocs,
		"frees":       stats.Frees,

		// Heap statistics
		"heap_alloc":    stats.HeapAlloc,
		"heap_sys":      stats.HeapSys,
		"heap_idle":     stats.HeapIdle,
		"heap_inuse":    stats.HeapInuse,
		"heap_released": stats.HeapReleased,
		"heap_objects":  stats.HeapObjects,

		// Stack statistics
		"stack_inuse": stats.StackInuse,
		"stack_sys":   stats.StackSys,

		// MSpan statistics
		"mspan_inuse": stats.MSpanInuse,
		"mspan_sys":   stats.MSpanSys,

		// MCache statistics
		"mcache_inuse": stats.MCacheInuse,
		"mcache_sys":   stats.MCacheSys,

		// GC statistics
		"gc_sys":          stats.GCSys,
		"other_sys":       stats.OtherSys,
		"next_gc":         stats.NextGC,
		"last_gc":         time.Unix(0, int64(stats.LastGC)),
		"pause_total":     time.Duration(stats.PauseTotalNs),
		"num_gc":          stats.NumGC,
		"num_forced_gc":   stats.NumForcedGC,
		"gc_cpu_fraction": stats.GCCPUFraction,

		// Enable/disable statistics
		"enable_gc": stats.EnableGC,
		"debug_gc":  stats.DebugGC,
	}
}

// TuneForLatency optimizes GC settings for low latency
func TuneForLatency() error {
	config := &HFTGCConfig{
		GCPercent:          300,        // Even less frequent GC
		MemoryLimit:        4294967296, // 4GB limit
		MaxProcs:           runtime.NumCPU(),
		EnableMemoryLimit:  true,
		SoftMemoryLimit:    3221225472, // 3GB
		EnableGCMonitoring: true,
		GCStatsInterval:    10 * time.Second,
		EnableBallastHeap:  true,
		BallastSize:        2147483648, // 2GB ballast
	}

	return OptimizeGCForHFT(config)
}

// TuneForThroughput optimizes GC settings for high throughput
func TuneForThroughput() error {
	config := &HFTGCConfig{
		GCPercent:          100,        // More frequent GC for throughput
		MemoryLimit:        8589934592, // 8GB limit
		MaxProcs:           runtime.NumCPU(),
		EnableMemoryLimit:  true,
		SoftMemoryLimit:    6442450944, // 6GB
		EnableGCMonitoring: true,
		GCStatsInterval:    30 * time.Second,
		EnableBallastHeap:  false,
		BallastSize:        0,
	}

	return OptimizeGCForHFT(config)
}

// ValidateGCConfig validates GC configuration
func ValidateGCConfig(config *HFTGCConfig) error {
	if config.GCPercent < 50 || config.GCPercent > 500 {
		return fmt.Errorf("gc_percent must be between 50 and 500")
	}

	if config.MemoryLimit <= 0 {
		return fmt.Errorf("memory_limit must be positive")
	}

	if config.SoftMemoryLimit >= config.MemoryLimit {
		return fmt.Errorf("soft_memory_limit must be less than memory_limit")
	}

	if config.MaxProcs < 0 {
		return fmt.Errorf("max_procs cannot be negative")
	}

	if config.GCStatsInterval <= 0 {
		return fmt.Errorf("gc_stats_interval must be positive")
	}

	return nil
}
