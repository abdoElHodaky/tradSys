package performance

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"go.uber.org/zap"
)

// Profiler provides profiling functionality
type Profiler struct {
	logger       *zap.Logger
	cpuProfile   *os.File
	memProfile   *os.File
	profilesDir  string
	isRunning    bool
	startTime    time.Time
	sampleRate   int
	memThreshold uint64
}

// ProfilerOptions contains options for the profiler
type ProfilerOptions struct {
	ProfilesDir  string
	SampleRate   int
	MemThreshold uint64
}

// DefaultProfilerOptions returns default profiler options
func DefaultProfilerOptions() ProfilerOptions {
	return ProfilerOptions{
		ProfilesDir:  "profiles",
		SampleRate:   100,
		MemThreshold: 1024 * 1024 * 100, // 100 MB
	}
}

// NewProfiler creates a new profiler
func NewProfiler(logger *zap.Logger, options ProfilerOptions) (*Profiler, error) {
	// Create the profiles directory if it doesn't exist
	if err := os.MkdirAll(options.ProfilesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profiles directory: %w", err)
	}

	return &Profiler{
		logger:       logger,
		profilesDir:  options.ProfilesDir,
		sampleRate:   options.SampleRate,
		memThreshold: options.MemThreshold,
	}, nil
}

// StartCPUProfiling starts CPU profiling
func (p *Profiler) StartCPUProfiling() error {
	if p.isRunning {
		return fmt.Errorf("profiler is already running")
	}

	// Create a CPU profile file
	cpuProfilePath := filepath.Join(p.profilesDir, fmt.Sprintf("cpu-%s.pprof", time.Now().Format("20060102-150405")))
	cpuProfile, err := os.Create(cpuProfilePath)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}

	// Start CPU profiling
	if err := pprof.StartCPUProfile(cpuProfile); err != nil {
		cpuProfile.Close()
		return fmt.Errorf("failed to start CPU profiling: %w", err)
	}

	p.cpuProfile = cpuProfile
	p.isRunning = true
	p.startTime = time.Now()

	p.logger.Info("Started CPU profiling", zap.String("file", cpuProfilePath))
	return nil
}

// StopCPUProfiling stops CPU profiling
func (p *Profiler) StopCPUProfiling() error {
	if !p.isRunning || p.cpuProfile == nil {
		return fmt.Errorf("CPU profiling is not running")
	}

	// Stop CPU profiling
	pprof.StopCPUProfile()
	p.cpuProfile.Close()
	p.cpuProfile = nil
	p.isRunning = false

	duration := time.Since(p.startTime)
	p.logger.Info("Stopped CPU profiling", zap.Duration("duration", duration))
	return nil
}

// CaptureHeapProfile captures a heap profile
func (p *Profiler) CaptureHeapProfile() error {
	// Create a heap profile file
	heapProfilePath := filepath.Join(p.profilesDir, fmt.Sprintf("heap-%s.pprof", time.Now().Format("20060102-150405")))
	heapProfile, err := os.Create(heapProfilePath)
	if err != nil {
		return fmt.Errorf("failed to create heap profile file: %w", err)
	}
	defer heapProfile.Close()

	// Write heap profile
	if err := pprof.WriteHeapProfile(heapProfile); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}

	p.logger.Info("Captured heap profile", zap.String("file", heapProfilePath))
	return nil
}

// StartMemoryMonitoring starts monitoring memory usage
func (p *Profiler) StartMemoryMonitoring(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Log memory stats
			p.logger.Info("Memory stats",
				zap.Uint64("alloc", m.Alloc),
				zap.Uint64("total_alloc", m.TotalAlloc),
				zap.Uint64("sys", m.Sys),
				zap.Uint64("heap_alloc", m.HeapAlloc),
				zap.Uint64("heap_sys", m.HeapSys),
				zap.Uint64("heap_idle", m.HeapIdle),
				zap.Uint64("heap_inuse", m.HeapInuse),
				zap.Uint64("heap_released", m.HeapReleased),
				zap.Uint64("heap_objects", m.HeapObjects),
				zap.Uint64("num_gc", uint64(m.NumGC)),
			)

			// Capture heap profile if memory usage exceeds threshold
			if m.Alloc > p.memThreshold {
				p.logger.Warn("Memory usage exceeds threshold, capturing heap profile",
					zap.Uint64("alloc", m.Alloc),
					zap.Uint64("threshold", p.memThreshold),
				)
				if err := p.CaptureHeapProfile(); err != nil {
					p.logger.Error("Failed to capture heap profile", zap.Error(err))
				}
			}
		}
	}()

	p.logger.Info("Started memory monitoring", zap.Duration("interval", interval))
}

