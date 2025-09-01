package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/performance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestProfiler(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	t.Run("Basic Profiler Functionality", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Verify that the profiler is started
		assert.True(t, profiler.IsStarted())

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)

		// Verify that the profiler is stopped
		assert.False(t, profiler.IsStarted())

		// Verify that profile files were created
		files, err := os.ReadDir(tempDir)
		require.NoError(t, err)
		assert.NotEmpty(t, files)
	})

	t.Run("Profiler Snapshots", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Take a snapshot
		snapshotName := "test-snapshot"
		err = profiler.TakeSnapshot(snapshotName)
		require.NoError(t, err)

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)

		// Verify that snapshot files were created
		snapshotDir := filepath.Join(tempDir, "snapshots", snapshotName)
		files, err := os.ReadDir(snapshotDir)
		require.NoError(t, err)
		assert.NotEmpty(t, files)
	})

	t.Run("CPU Profiling", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir
		options.CPUProfile = true
		options.MemProfile = false
		options.BlockProfile = false
		options.MutexProfile = false

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Do some CPU-intensive work
		for i := 0; i < 1000000; i++ {
			_ = i * i
		}

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)

		// Verify that CPU profile was created
		cpuProfilePath := filepath.Join(tempDir, "cpu.pprof")
		_, err = os.Stat(cpuProfilePath)
		assert.NoError(t, err)
	})

	t.Run("Memory Profiling", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir
		options.CPUProfile = false
		options.MemProfile = true
		options.BlockProfile = false
		options.MutexProfile = false

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Allocate some memory
		data := make([]byte, 10*1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)

		// Verify that memory profile was created
		memProfilePath := filepath.Join(tempDir, "mem.pprof")
		_, err = os.Stat(memProfilePath)
		assert.NoError(t, err)
	})

	t.Run("Block Profiling", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir
		options.CPUProfile = false
		options.MemProfile = false
		options.BlockProfile = true
		options.MutexProfile = false

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Create some blocking operations
		ch := make(chan struct{})
		go func() {
			time.Sleep(100 * time.Millisecond)
			ch <- struct{}{}
		}()
		<-ch

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)

		// Verify that block profile was created
		blockProfilePath := filepath.Join(tempDir, "block.pprof")
		_, err = os.Stat(blockProfilePath)
		assert.NoError(t, err)
	})

	t.Run("Mutex Profiling", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir
		options.CPUProfile = false
		options.MemProfile = false
		options.BlockProfile = false
		options.MutexProfile = true

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Create some mutex contention
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					mu.Lock()
					time.Sleep(1 * time.Millisecond)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)

		// Verify that mutex profile was created
		mutexProfilePath := filepath.Join(tempDir, "mutex.pprof")
		_, err = os.Stat(mutexProfilePath)
		assert.NoError(t, err)
	})

	t.Run("Trace Profiling", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir
		options.CPUProfile = false
		options.MemProfile = false
		options.BlockProfile = false
		options.MutexProfile = false
		options.TraceProfile = true

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Do some work
		for i := 0; i < 1000; i++ {
			_ = i * i
		}

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)

		// Verify that trace profile was created
		traceProfilePath := filepath.Join(tempDir, "trace.out")
		_, err = os.Stat(traceProfilePath)
		assert.NoError(t, err)
	})

	t.Run("CPU Profile Duration", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir
		options.CPUProfile = false

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Create a context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start a CPU profile for a short duration
		err = profiler.StartCPUProfile(ctx, 100*time.Millisecond)
		require.NoError(t, err)

		// Do some CPU-intensive work
		for i := 0; i < 1000000; i++ {
			_ = i * i
		}

		// Wait for the profile to complete
		time.Sleep(200 * time.Millisecond)

		// Verify that CPU profile was created
		files, err := os.ReadDir(tempDir)
		require.NoError(t, err)
		assert.NotEmpty(t, files)
	})

	t.Run("Memory Stats", func(t *testing.T) {
		// Create a profiler
		options := performance.DefaultProfilerOptions()
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Get memory stats
		memStats := profiler.GetMemoryStats()

		// Verify memory stats
		assert.NotNil(t, memStats)
		assert.Contains(t, memStats, "alloc")
		assert.Contains(t, memStats, "totalAlloc")
		assert.Contains(t, memStats, "sys")
		assert.Contains(t, memStats, "numGC")
		assert.Contains(t, memStats, "heapAlloc")
	})

	t.Run("Goroutine Stats", func(t *testing.T) {
		// Create a profiler
		options := performance.DefaultProfilerOptions()
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Get goroutine stats
		goroutineStats := profiler.GetGoroutineStats()

		// Verify goroutine stats
		assert.NotNil(t, goroutineStats)
		assert.Contains(t, goroutineStats, "goroutines")
		assert.Greater(t, goroutineStats["goroutines"].(int), 0)
	})

	t.Run("Profiler Disabled", func(t *testing.T) {
		// Create profiler options with profiling disabled
		options := performance.DefaultProfilerOptions()
		options.Enabled = false

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Verify that the profiler is not started
		assert.False(t, profiler.IsStarted())

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)
	})

	t.Run("Profiler Already Started", func(t *testing.T) {
		// Create a temporary directory for profiles
		tempDir, err := os.MkdirTemp("", "profiler-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create profiler options
		options := performance.DefaultProfilerOptions()
		options.ProfileDir = tempDir

		// Create a profiler
		profiler, err := performance.NewProfiler(options, logger)
		require.NoError(t, err)

		// Start the profiler
		err = profiler.Start()
		require.NoError(t, err)

		// Try to start the profiler again
		err = profiler.Start()
		assert.Error(t, err)

		// Stop the profiler
		err = profiler.Stop()
		require.NoError(t, err)
	})
}

