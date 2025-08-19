package performance

import (
	"context"
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
		SampleRate:    1000,
	}
}

// NewProfiler creates a new profiler
func NewProfiler(logger *zap.Logger, options ProfilerOptions) *Profiler {
	return &Profiler{
		enabled:       options.Enabled,
		cpuProfile:    options.CPUProfile,
		memProfile:    options.MemProfile,
		blockProfile:  options.BlockProfile,
		mutexProfile:  options.MutexProfile,
		traceProfile:  options.TraceProfile,
		profileDir:    options.ProfileDir,
		sampleRate:    options.SampleRate,
		logger:        logger,
	}
}

// Start starts profiling
func (p *Profiler) Start() error {
	if !p.enabled {
		return nil
	}

	// Create profile directory if it doesn't exist
	err := os.MkdirAll(p.profileDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Start CPU profiling
	if p.cpuProfile {
		cpuFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("cpu-%s.pprof", time.Now().Format("20060102-150405"))))
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %w", err)
		}

		p.cpuFile = cpuFile
		err = pprof.StartCPUProfile(cpuFile)
		if err != nil {
			cpuFile.Close()
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}

		p.logger.Info("Started CPU profiling")
	}

	// Set block profiling rate
	if p.blockProfile {
		runtime.SetBlockProfileRate(p.sampleRate)
		p.logger.Info("Started block profiling",
			zap.Int("rate", p.sampleRate))
	}

	// Set mutex profiling fraction
	if p.mutexProfile {
		runtime.SetMutexProfileFraction(p.sampleRate)
		p.logger.Info("Started mutex profiling",
			zap.Int("fraction", p.sampleRate))
	}

	return nil
}

// Stop stops profiling
func (p *Profiler) Stop() error {
	if !p.enabled {
		return nil
	}

	// Stop CPU profiling
	if p.cpuProfile && p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
		p.cpuFile = nil
		p.logger.Info("Stopped CPU profiling")
	}

	// Write memory profile
	if p.memProfile {
		memFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("mem-%s.pprof", time.Now().Format("20060102-150405"))))
		if err != nil {
			return fmt.Errorf("failed to create memory profile file: %w", err)
		}
		defer memFile.Close()

		runtime.GC() // Get up-to-date statistics
		err = pprof.WriteHeapProfile(memFile)
		if err != nil {
			return fmt.Errorf("failed to write memory profile: %w", err)
		}

		p.logger.Info("Wrote memory profile")
	}

	// Write block profile
	if p.blockProfile {
		blockFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("block-%s.pprof", time.Now().Format("20060102-150405"))))
		if err != nil {
			return fmt.Errorf("failed to create block profile file: %w", err)
		}
		defer blockFile.Close()

		err = pprof.Lookup("block").WriteTo(blockFile, 0)
		if err != nil {
			return fmt.Errorf("failed to write block profile: %w", err)
		}

		// Reset block profiling rate
		runtime.SetBlockProfileRate(0)
		p.logger.Info("Wrote block profile and reset rate")
	}

	// Write mutex profile
	if p.mutexProfile {
		mutexFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("mutex-%s.pprof", time.Now().Format("20060102-150405"))))
		if err != nil {
			return fmt.Errorf("failed to create mutex profile file: %w", err)
		}
		defer mutexFile.Close()

		err = pprof.Lookup("mutex").WriteTo(mutexFile, 0)
		if err != nil {
			return fmt.Errorf("failed to write mutex profile: %w", err)
		}

		// Reset mutex profiling fraction
		runtime.SetMutexProfileFraction(0)
		p.logger.Info("Wrote mutex profile and reset fraction")
	}

	return nil
}

// CaptureProfile captures a profile on demand
func (p *Profiler) CaptureProfile(ctx context.Context, profileType string, duration time.Duration) error {
	if !p.enabled {
		return nil
	}

	// Create profile directory if it doesn't exist
	err := os.MkdirAll(p.profileDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	switch profileType {
	case "cpu":
		// Create CPU profile file
		cpuFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("cpu-capture-%s.pprof", time.Now().Format("20060102-150405"))))
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %w", err)
		}
		defer cpuFile.Close()

		// Start CPU profiling
		err = pprof.StartCPUProfile(cpuFile)
		if err != nil {
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}

		p.logger.Info("Started CPU profile capture",
			zap.Duration("duration", duration))

		// Wait for the specified duration or context cancellation
		select {
		case <-time.After(duration):
		case <-ctx.Done():
			p.logger.Info("CPU profile capture cancelled")
		}

		// Stop CPU profiling
		pprof.StopCPUProfile()
		p.logger.Info("Completed CPU profile capture")

	case "heap":
		// Create heap profile file
		heapFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("heap-capture-%s.pprof", time.Now().Format("20060102-150405"))))
		if err != nil {
			return fmt.Errorf("failed to create heap profile file: %w", err)
		}
		defer heapFile.Close()

		// Run garbage collection to get accurate memory statistics
		runtime.GC()

		// Write heap profile
		err = pprof.WriteHeapProfile(heapFile)
		if err != nil {
			return fmt.Errorf("failed to write heap profile: %w", err)
		}

		p.logger.Info("Captured heap profile")

	case "goroutine":
		// Create goroutine profile file
		goroutineFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("goroutine-capture-%s.pprof", time.Now().Format("20060102-150405"))))
		if err != nil {
			return fmt.Errorf("failed to create goroutine profile file: %w", err)
		}
		defer goroutineFile.Close()

		// Write goroutine profile
		err = pprof.Lookup("goroutine").WriteTo(goroutineFile, 0)
		if err != nil {
			return fmt.Errorf("failed to write goroutine profile: %w", err)
		}

		p.logger.Info("Captured goroutine profile")

	default:
		return fmt.Errorf("unsupported profile type: %s", profileType)
	}

	return nil
}

// EnableContinuousMemoryProfiling enables continuous memory profiling
func (p *Profiler) EnableContinuousMemoryProfiling(ctx context.Context, interval time.Duration) {
	if !p.enabled || !p.memProfile {
		return
	}

	p.logger.Info("Starting continuous memory profiling",
		zap.Duration("interval", interval))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Create memory profile file
				memFile, err := os.Create(filepath.Join(p.profileDir, fmt.Sprintf("mem-continuous-%s.pprof", time.Now().Format("20060102-150405"))))
				if err != nil {
					p.logger.Error("Failed to create memory profile file",
						zap.Error(err))
					continue
				}

				// Run garbage collection to get accurate memory statistics
				runtime.GC()

				// Write memory profile
				err = pprof.WriteHeapProfile(memFile)
				if err != nil {
					p.logger.Error("Failed to write memory profile",
						zap.Error(err))
				}

				memFile.Close()
				p.logger.Debug("Wrote continuous memory profile")

			case <-ctx.Done():
				p.logger.Info("Stopping continuous memory profiling")
				return
			}
		}
	}()
}

