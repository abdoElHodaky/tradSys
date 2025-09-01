package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Profiler provides profiling functionality
type Profiler struct {
	// Configuration
	enabled       bool
	cpuProfile    bool
	memProfile    bool
	blockProfile  bool
	mutexProfile  bool
	traceProfile  bool
	profileDir    string
	sampleRate    int
	
	// State
	cpuFile       *os.File
	traceFile     *os.File
	started       bool
	startTime     time.Time
	mu            sync.Mutex
	logger        *zap.Logger
}

// ProfilerOptions contains options for the profiler
type ProfilerOptions struct {
	Enabled       bool
	CPUProfile    bool
	MemProfile    bool
	BlockProfile  bool
	MutexProfile  bool
	TraceProfile  bool
	ProfileDir    string
	SampleRate    int
}

// DefaultProfilerOptions returns default profiler options
func DefaultProfilerOptions() ProfilerOptions {
	return ProfilerOptions{
		Enabled:       true,
		CPUProfile:    true,
		MemProfile:    true,
		BlockProfile:  true,
		MutexProfile:  true,
		TraceProfile:  false,
		ProfileDir:    "profiles",
		SampleRate:    1,
	}
}

// NewProfiler creates a new profiler
func NewProfiler(options ProfilerOptions, logger *zap.Logger) (*Profiler, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create the profile directory if it doesn't exist
	if options.Enabled && options.ProfileDir != "" {
		if err := os.MkdirAll(options.ProfileDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create profile directory: %w", err)
		}
	}

	return &Profiler{
		enabled:      options.Enabled,
		cpuProfile:   options.CPUProfile,
		memProfile:   options.MemProfile,
		blockProfile: options.BlockProfile,
		mutexProfile: options.MutexProfile,
		traceProfile: options.TraceProfile,
		profileDir:   options.ProfileDir,
		sampleRate:   options.SampleRate,
		logger:       logger,
	}, nil
}

// Start starts profiling
func (p *Profiler) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled {
		return nil
	}

	if p.started {
		return fmt.Errorf("profiler already started")
	}

	p.logger.Info("Starting profiler",
		zap.Bool("cpuProfile", p.cpuProfile),
		zap.Bool("memProfile", p.memProfile),
		zap.Bool("blockProfile", p.blockProfile),
		zap.Bool("mutexProfile", p.mutexProfile),
		zap.Bool("traceProfile", p.traceProfile),
		zap.String("profileDir", p.profileDir),
	)

	// Start CPU profiling
	if p.cpuProfile {
		cpuFile, err := os.Create(filepath.Join(p.profileDir, "cpu.pprof"))
		if err != nil {
			return fmt.Errorf("failed to create CPU profile: %w", err)
		}
		p.cpuFile = cpuFile

		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			cpuFile.Close()
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}

		p.logger.Debug("Started CPU profiling")
	}

	// Configure block profiling
	if p.blockProfile {
		runtime.SetBlockProfileRate(p.sampleRate)
		p.logger.Debug("Enabled block profiling",
			zap.Int("sampleRate", p.sampleRate),
		)
	}

	// Configure mutex profiling
	if p.mutexProfile {
		runtime.SetMutexProfileFraction(p.sampleRate)
		p.logger.Debug("Enabled mutex profiling",
			zap.Int("sampleRate", p.sampleRate),
		)
	}

	// Start trace profiling
	if p.traceProfile {
		traceFile, err := os.Create(filepath.Join(p.profileDir, "trace.out"))
		if err != nil {
			return fmt.Errorf("failed to create trace profile: %w", err)
		}
		p.traceFile = traceFile

		if err := trace.Start(traceFile); err != nil {
			traceFile.Close()
			return fmt.Errorf("failed to start trace profile: %w", err)
		}

		p.logger.Debug("Started trace profiling")
	}

	p.started = true
	p.startTime = time.Now()

	return nil
}

// Stop stops profiling and writes profiles to disk
func (p *Profiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled || !p.started {
		return nil
	}

	p.logger.Info("Stopping profiler",
		zap.Duration("duration", time.Since(p.startTime)),
	)

	// Stop CPU profiling
	if p.cpuProfile && p.cpuFile != nil {
		pprof.StopCPUProfile()
		if err := p.cpuFile.Close(); err != nil {
			p.logger.Error("Failed to close CPU profile file",
				zap.Error(err),
			)
		}
		p.cpuFile = nil
		p.logger.Debug("Stopped CPU profiling")
	}

	// Write memory profile
	if p.memProfile {
		memFile, err := os.Create(filepath.Join(p.profileDir, "mem.pprof"))
		if err != nil {
			p.logger.Error("Failed to create memory profile file",
				zap.Error(err),
			)
		} else {
			defer memFile.Close()
			runtime.GC() // Get up-to-date statistics
			if err := pprof.WriteHeapProfile(memFile); err != nil {
				p.logger.Error("Failed to write memory profile",
					zap.Error(err),
				)
			} else {
				p.logger.Debug("Wrote memory profile")
			}
		}
	}

	// Write goroutine profile
	goroutineFile, err := os.Create(filepath.Join(p.profileDir, "goroutine.pprof"))
	if err != nil {
		p.logger.Error("Failed to create goroutine profile file",
			zap.Error(err),
		)
	} else {
		defer goroutineFile.Close()
		if err := pprof.Lookup("goroutine").WriteTo(goroutineFile, 0); err != nil {
			p.logger.Error("Failed to write goroutine profile",
				zap.Error(err),
			)
		} else {
			p.logger.Debug("Wrote goroutine profile")
		}
	}

	// Write block profile
	if p.blockProfile {
		blockFile, err := os.Create(filepath.Join(p.profileDir, "block.pprof"))
		if err != nil {
			p.logger.Error("Failed to create block profile file",
				zap.Error(err),
			)
		} else {
			defer blockFile.Close()
			if err := pprof.Lookup("block").WriteTo(blockFile, 0); err != nil {
				p.logger.Error("Failed to write block profile",
					zap.Error(err),
				)
			} else {
				p.logger.Debug("Wrote block profile")
			}
		}
		runtime.SetBlockProfileRate(0) // Disable block profiling
	}

	// Write mutex profile
	if p.mutexProfile {
		mutexFile, err := os.Create(filepath.Join(p.profileDir, "mutex.pprof"))
		if err != nil {
			p.logger.Error("Failed to create mutex profile file",
				zap.Error(err),
			)
		} else {
			defer mutexFile.Close()
			if err := pprof.Lookup("mutex").WriteTo(mutexFile, 0); err != nil {
				p.logger.Error("Failed to write mutex profile",
					zap.Error(err),
				)
			} else {
				p.logger.Debug("Wrote mutex profile")
			}
		}
		runtime.SetMutexProfileFraction(0) // Disable mutex profiling
	}

	// Stop trace profiling
	if p.traceProfile && p.traceFile != nil {
		trace.Stop()
		if err := p.traceFile.Close(); err != nil {
			p.logger.Error("Failed to close trace profile file",
				zap.Error(err),
			)
		}
		p.traceFile = nil
		p.logger.Debug("Stopped trace profiling")
	}

	p.started = false

	return nil
}

// TakeSnapshot takes a snapshot of the current profiles
func (p *Profiler) TakeSnapshot(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled || !p.started {
		return nil
	}

	p.logger.Info("Taking profile snapshot",
		zap.String("name", name),
	)

	// Create snapshot directory
	snapshotDir := filepath.Join(p.profileDir, "snapshots", name)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Write memory profile
	if p.memProfile {
		memFile, err := os.Create(filepath.Join(snapshotDir, "mem.pprof"))
		if err != nil {
			p.logger.Error("Failed to create memory profile snapshot",
				zap.Error(err),
			)
		} else {
			defer memFile.Close()
			runtime.GC() // Get up-to-date statistics
			if err := pprof.WriteHeapProfile(memFile); err != nil {
				p.logger.Error("Failed to write memory profile snapshot",
					zap.Error(err),
				)
			}
		}
	}

	// Write goroutine profile
	goroutineFile, err := os.Create(filepath.Join(snapshotDir, "goroutine.pprof"))
	if err != nil {
		p.logger.Error("Failed to create goroutine profile snapshot",
			zap.Error(err),
		)
	} else {
		defer goroutineFile.Close()
		if err := pprof.Lookup("goroutine").WriteTo(goroutineFile, 0); err != nil {
			p.logger.Error("Failed to write goroutine profile snapshot",
				zap.Error(err),
			)
		}
	}

	// Write block profile
	if p.blockProfile {
		blockFile, err := os.Create(filepath.Join(snapshotDir, "block.pprof"))
		if err != nil {
			p.logger.Error("Failed to create block profile snapshot",
				zap.Error(err),
			)
		} else {
			defer blockFile.Close()
			if err := pprof.Lookup("block").WriteTo(blockFile, 0); err != nil {
				p.logger.Error("Failed to write block profile snapshot",
					zap.Error(err),
				)
			}
		}
	}

	// Write mutex profile
	if p.mutexProfile {
		mutexFile, err := os.Create(filepath.Join(snapshotDir, "mutex.pprof"))
		if err != nil {
			p.logger.Error("Failed to create mutex profile snapshot",
				zap.Error(err),
			)
		} else {
			defer mutexFile.Close()
			if err := pprof.Lookup("mutex").WriteTo(mutexFile, 0); err != nil {
				p.logger.Error("Failed to write mutex profile snapshot",
					zap.Error(err),
				)
			}
		}
	}

	return nil
}

// StartCPUProfile starts CPU profiling
func (p *Profiler) StartCPUProfile(ctx context.Context, duration time.Duration) error {
	if !p.enabled {
		return nil
	}

	p.logger.Info("Starting CPU profile",
		zap.Duration("duration", duration),
	)

	// Create profile file
	cpuFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("cpu_%s.pprof", time.Now().Format("20060102_150405"))))
	if err != nil {
		return fmt.Errorf("failed to create CPU profile: %w", err)
	}

	// Start CPU profiling
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		cpuFile.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	// Stop CPU profiling after duration
	go func() {
		select {
		case <-time.After(duration):
			pprof.StopCPUProfile()
			cpuFile.Close()
			p.logger.Info("CPU profile completed",
				zap.Duration("duration", duration),
			)
		case <-ctx.Done():
			pprof.StopCPUProfile()
			cpuFile.Close()
			p.logger.Info("CPU profile cancelled",
				zap.Error(ctx.Err()),
			)
		}
	}()

	return nil
}

// GetMemoryStats returns memory statistics
func (p *Profiler) GetMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":        m.Alloc,
		"totalAlloc":   m.TotalAlloc,
		"sys":          m.Sys,
		"numGC":        m.NumGC,
		"pauseTotalNs": m.PauseTotalNs,
		"heapAlloc":    m.HeapAlloc,
		"heapSys":      m.HeapSys,
		"heapIdle":     m.HeapIdle,
		"heapInuse":    m.HeapInuse,
		"stackInuse":   m.StackInuse,
		"stackSys":     m.StackSys,
	}
}

// GetGoroutineStats returns goroutine statistics
func (p *Profiler) GetGoroutineStats() map[string]interface{} {
	return map[string]interface{}{
		"goroutines": runtime.NumGoroutine(),
	}
}

// IsEnabled returns whether profiling is enabled
func (p *Profiler) IsEnabled() bool {
	return p.enabled
}

// IsStarted returns whether profiling is started
func (p *Profiler) IsStarted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.started
}

// SetEnabled sets whether profiling is enabled
func (p *Profiler) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started && !enabled {
		p.logger.Warn("Cannot disable profiler while it is running")
		return
	}

	p.enabled = enabled
}

