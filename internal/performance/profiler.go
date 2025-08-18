package performance

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Profiler provides performance profiling capabilities
type Profiler struct {
	logger *zap.Logger
}

// NewProfiler creates a new profiler
func NewProfiler(logger *zap.Logger) *Profiler {
	return &Profiler{
		logger: logger,
	}
}

// RegisterHTTPEndpoints registers HTTP endpoints for profiling
func (p *Profiler) RegisterHTTPEndpoints(router *gin.Engine) {
	// Create a profiling group
	profiling := router.Group("/debug/pprof")
	{
		profiling.GET("/", gin.WrapF(pprof.Index))
		profiling.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		profiling.GET("/profile", gin.WrapF(pprof.Profile))
		profiling.POST("/symbol", gin.WrapF(pprof.Symbol))
		profiling.GET("/symbol", gin.WrapF(pprof.Symbol))
		profiling.GET("/trace", gin.WrapF(pprof.Trace))
		profiling.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		profiling.GET("/block", gin.WrapH(pprof.Handler("block")))
		profiling.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		profiling.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		profiling.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		profiling.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}

	// Add custom profiling endpoints
	router.GET("/debug/performance/memory", p.memoryStats)
	router.GET("/debug/performance/gc", p.gcStats)
	router.GET("/debug/performance/cpu", p.cpuProfile)
}

// memoryStats returns memory statistics
func (p *Profiler) memoryStats(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.JSON(http.StatusOK, gin.H{
		"alloc":        m.Alloc,
		"total_alloc":  m.TotalAlloc,
		"sys":          m.Sys,
		"lookups":      m.Lookups,
		"mallocs":      m.Mallocs,
		"frees":        m.Frees,
		"heap_alloc":   m.HeapAlloc,
		"heap_sys":     m.HeapSys,
		"heap_idle":    m.HeapIdle,
		"heap_inuse":   m.HeapInuse,
		"heap_objects": m.HeapObjects,
		"stack_inuse":  m.StackInuse,
		"stack_sys":    m.StackSys,
		"mspan_inuse":  m.MSpanInuse,
		"mspan_sys":    m.MSpanSys,
		"mcache_inuse": m.MCacheInuse,
		"mcache_sys":   m.MCacheSys,
		"buck_hash_sys": m.BuckHashSys,
		"gc_sys":       m.GCSys,
		"other_sys":    m.OtherSys,
		"next_gc":      m.NextGC,
		"last_gc":      m.LastGC,
		"pause_total_ns": m.PauseTotalNs,
		"num_gc":       m.NumGC,
		"num_forced_gc": m.NumForcedGC,
	})
}

// gcStats returns garbage collection statistics
func (p *Profiler) gcStats(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.JSON(http.StatusOK, gin.H{
		"next_gc":      m.NextGC,
		"last_gc":      m.LastGC,
		"pause_total_ns": m.PauseTotalNs,
		"num_gc":       m.NumGC,
		"num_forced_gc": m.NumForcedGC,
		"gc_cpu_fraction": m.GCCPUFraction,
	})
}

// cpuProfile generates a CPU profile
func (p *Profiler) cpuProfile(c *gin.Context) {
	// Get duration from query parameter (default: 30 seconds)
	durationStr := c.DefaultQuery("duration", "30")
	duration, err := time.ParseDuration(durationStr + "s")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration"})
		return
	}

	// Create a temporary file for the CPU profile
	f, err := os.CreateTemp("", "cpu-profile-*.pprof")
	if err != nil {
		p.logger.Error("Failed to create CPU profile file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create CPU profile file"})
		return
	}
	defer f.Close()

	// Start CPU profiling
	if err := pprof.StartCPUProfile(f); err != nil {
		p.logger.Error("Failed to start CPU profile", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start CPU profile"})
		return
	}

	// Stop CPU profiling after the specified duration
	time.Sleep(duration)
	pprof.StopCPUProfile()

	// Return the profile file
	c.File(f.Name())
}

// StartCPUProfile starts CPU profiling to a file
func (p *Profiler) StartCPUProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	p.logger.Info("CPU profiling started", zap.String("filename", filename))
	return nil
}

// StopCPUProfile stops CPU profiling
func (p *Profiler) StopCPUProfile() {
	pprof.StopCPUProfile()
	p.logger.Info("CPU profiling stopped")
}

// WriteHeapProfile writes a heap profile to a file
func (p *Profiler) WriteHeapProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create heap profile file: %w", err)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}

	p.logger.Info("Heap profile written", zap.String("filename", filename))
	return nil
}

// MemoryStats returns memory statistics
func (p *Profiler) MemoryStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// ForceGC forces a garbage collection
func (p *Profiler) ForceGC() {
	runtime.GC()
	p.logger.Info("Forced garbage collection")
}
