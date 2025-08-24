package adaptive_loader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// AdaptivePluginLoader is an enhanced plugin loader that addresses
// conflicts and bottlenecks identified in the analysis.
type AdaptivePluginLoader struct {
	logger              *zap.Logger
	pluginDirs          []string
	loadedPlugins       map[string]*LoadedPlugin
	
	// Fine-grained locking for different operations
	configMu            sync.RWMutex // For configuration changes
	pluginMu            sync.RWMutex // For plugin operations
	scannerMu           sync.Mutex   // For scanner operations
	dirsMu              sync.RWMutex // For directory operations
	
	// Concurrency control
	maxConcurrent       int
	loadSemaphore       chan struct{}
	workerPool          *WorkerPool
	
	// Backpressure management
	backpressureEnabled bool
	maxQueuedTasks      int
	currentLoad         int64
	loadMu              sync.RWMutex
	
	// Resource monitoring
	memoryMonitor       *MemoryMonitor
	
	// Configuration
	loadTimeout         time.Duration
	scanInterval        time.Duration
	baseScanInterval    time.Duration
	adaptiveScanEnabled bool
	
	// Scanner state
	scannerRunning      bool
	scannerStopCh       chan struct{}
	
	// Deadlock detection
	lockTimeouts        map[string]time.Duration
	deadlockDetection   bool
}

// LoadedPlugin represents a loaded plugin
type LoadedPlugin struct {
	Path      string
	Type      string
	Name      string
	LoadTime  time.Time
	MemoryUse int64
	Instance  interface{}
}

// NewAdaptivePluginLoader creates a new adaptive plugin loader
func NewAdaptivePluginLoader(
	logger *zap.Logger,
	pluginDirs []string,
) *AdaptivePluginLoader {
	// Default to number of CPUs for concurrent loads
	maxConcurrent := runtime.NumCPU()
	
	// Create worker pool
	workerPool := NewWorkerPool(maxConcurrent, maxConcurrent*10, logger)
	
	// Calculate default max queued tasks based on system resources
	maxQueuedTasks := maxConcurrent * 20
	
	return &AdaptivePluginLoader{
		logger:              logger,
		pluginDirs:          pluginDirs,
		loadedPlugins:       make(map[string]*LoadedPlugin),
		maxConcurrent:       maxConcurrent,
		loadSemaphore:       make(chan struct{}, maxConcurrent),
		workerPool:          workerPool,
		backpressureEnabled: true,
		maxQueuedTasks:      maxQueuedTasks,
		currentLoad:         0,
		memoryMonitor:       NewMemoryMonitor(logger),
		loadTimeout:         30 * time.Second,
		scanInterval:        5 * time.Minute,
		baseScanInterval:    5 * time.Minute,
		adaptiveScanEnabled: false,
		scannerStopCh:       make(chan struct{}),
		lockTimeouts:        make(map[string]time.Duration),
		deadlockDetection:   true,
	}
}

// FindPluginFiles finds all plugin files in the configured directories
func (l *AdaptivePluginLoader) FindPluginFiles() ([]string, error) {
	// Use directory-specific lock
	l.dirsMu.RLock()
	dirs := make([]string, len(l.pluginDirs))
	copy(dirs, l.pluginDirs)
	l.dirsMu.RUnlock()
	
	var pluginFiles []string
	for _, dir := range dirs {
		files, err := l.findPluginFilesInDir(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to find plugin files in %s: %w", dir, err)
		}
		pluginFiles = append(pluginFiles, files...)
	}
	
	return pluginFiles, nil
}

// findPluginFilesInDir finds all plugin files in a directory
func (l *AdaptivePluginLoader) findPluginFilesInDir(dir string) ([]string, error) {
	// Check if the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin directory does not exist: %s", dir)
	}
	
	// Find all .so files in the directory
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check file extension
		if filepath.Ext(path) == ".so" {
			files = append(files, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	
	return files, nil
}

// StartPluginScanner starts a background scanner for plugin changes
func (l *AdaptivePluginLoader) StartPluginScanner(ctx context.Context, scanInterval time.Duration) error {
	// Use scanner-specific lock
	if !l.acquireLock(&l.scannerMu, "scannerMu", 5*time.Second) {
		return fmt.Errorf("failed to acquire scanner lock (potential deadlock)")
	}
	
	// Use config lock for configuration changes
	if !l.acquireLock(&l.configMu, "configMu", 5*time.Second) {
		l.scannerMu.Unlock()
		return fmt.Errorf("failed to acquire config lock (potential deadlock)")
	}
	
	if l.scannerRunning {
		l.configMu.Unlock()
		l.scannerMu.Unlock()
		return fmt.Errorf("plugin scanner already running")
	}
	
	// Start the worker pool if not already running
	if l.workerPool != nil && atomic.LoadInt32(&l.workerPool.running) == 0 {
		l.workerPool.Start()
		l.logger.Info("Started worker pool for plugin operations")
	}
	
	l.scannerRunning = true
	l.scanInterval = scanInterval
	l.baseScanInterval = scanInterval
	l.configMu.Unlock()
	l.scannerMu.Unlock()
	
	// Start the scanner in a goroutine
	go func() {
		// Initial scan interval
		nextInterval := scanInterval
		ticker := time.NewTicker(nextInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				// Create a scan task with priority
				scanTask := NewTask("plugin_scan", func() error {
					return l.scanForNewPlugins(ctx)
				}).WithPriority(5) // Higher priority for scanning
				
				// Submit the task to the worker pool
				if l.workerPool != nil {
					if err := l.workerPool.Submit(scanTask); err != nil {
						l.logger.Error("Failed to submit scan task to worker pool", zap.Error(err))
						
						// Fall back to direct execution if submission fails
						if err := l.scanForNewPlugins(ctx); err != nil {
							l.logger.Error("Failed to scan for new plugins", zap.Error(err))
						}
					}
				} else {
					// Direct execution if worker pool is not available
					if err := l.scanForNewPlugins(ctx); err != nil {
						l.logger.Error("Failed to scan for new plugins", zap.Error(err))
					}
				}
				
				// Adjust scan interval if adaptive scanning is enabled
				l.configMu.RLock()
				adaptiveEnabled := l.adaptiveScanEnabled
				l.configMu.RUnlock()
				
				if adaptiveEnabled {
					nextInterval = l.getNextScanInterval()
					ticker.Reset(nextInterval)
					l.logger.Debug("Adjusted scan interval", 
						zap.Duration("interval", nextInterval),
						zap.Bool("memory_pressure", l.memoryMonitor.IsUnderMemoryPressure()))
				}
			case <-l.scannerStopCh:
				if !l.acquireLock(&l.scannerMu, "scannerMu", 5*time.Second) {
					l.logger.Error("Failed to acquire scanner lock during shutdown (potential deadlock)")
					return
				}
				l.scannerRunning = false
				l.scannerMu.Unlock()
				return
			case <-ctx.Done():
				if !l.acquireLock(&l.scannerMu, "scannerMu", 5*time.Second) {
					l.logger.Error("Failed to acquire scanner lock during context cancellation (potential deadlock)")
					return
				}
				l.scannerRunning = false
				l.scannerMu.Unlock()
				return
			}
		}
	}()
	
	return nil
}

// StopPluginScanner stops the background plugin scanner
func (l *AdaptivePluginLoader) StopPluginScanner() {
	// Use scanner-specific lock with deadlock detection
	if !l.acquireLock(&l.scannerMu, "scannerMu", 5*time.Second) {
		l.logger.Error("Failed to acquire scanner lock during stop (potential deadlock)")
		return
	}
	defer l.scannerMu.Unlock()
	
	if l.scannerRunning {
		close(l.scannerStopCh)
		l.scannerStopCh = make(chan struct{})
		l.scannerRunning = false
	}
	
	// Stop the worker pool if it's running
	if l.workerPool != nil && atomic.LoadInt32(&l.workerPool.running) == 1 {
		l.workerPool.Stop()
		l.logger.Info("Stopped worker pool for plugin operations")
	}
}

// scanForNewPlugins scans for new plugins
func (l *AdaptivePluginLoader) scanForNewPlugins(ctx context.Context) error {
	// Check if we should apply backpressure
	if l.backpressureEnabled {
		// Check current load
		l.loadMu.RLock()
		currentLoad := l.currentLoad
		l.loadMu.RUnlock()
		
		// If we're under heavy load, skip this scan
		if currentLoad > int64(l.maxQueuedTasks) {
			l.logger.Warn("Skipping plugin scan due to high system load",
				zap.Int64("current_load", currentLoad),
				zap.Int("max_queued_tasks", l.maxQueuedTasks))
			return nil
		}
	}
	
	// Find all plugin files
	pluginFiles, err := l.FindPluginFiles()
	if err != nil {
		return fmt.Errorf("failed to find plugin files: %w", err)
	}
	
	// Check for new plugins - use plugin-specific lock
	if !l.acquireLockRO(&l.pluginMu, "pluginMu", 5*time.Second) {
		return fmt.Errorf("failed to acquire plugin lock (potential deadlock)")
	}
	
	var newFiles []string
	for _, file := range pluginFiles {
		if _, ok := l.loadedPlugins[file]; !ok {
			newFiles = append(newFiles, file)
		}
	}
	
	l.pluginMu.RUnlock()
	
	// Log new plugins found
	if len(newFiles) > 0 {
		l.logger.Info("Found new plugins", zap.Int("count", len(newFiles)))
		
		// Load new plugins using worker pool
		if l.workerPool != nil && len(newFiles) > 0 {
			for _, file := range newFiles {
				// Create a load task with file-specific context
				fileCtx, cancel := context.WithTimeout(ctx, l.loadTimeout)
				loadTask := NewTask(fmt.Sprintf("load_plugin_%s", filepath.Base(file)), func() error {
					defer cancel()
					// Increase load counter
					l.loadMu.Lock()
					l.currentLoad++
					l.loadMu.Unlock()
					
					// Load the plugin
					err := l.LoadPlugin(fileCtx, file)
					
					// Decrease load counter
					l.loadMu.Lock()
					l.currentLoad--
					l.loadMu.Unlock()
					
					return err
				}).WithContext(fileCtx)
				
				// Submit the task to the worker pool
				if err := l.workerPool.Submit(loadTask); err != nil {
					l.logger.Error("Failed to submit plugin load task to worker pool",
						zap.String("file", file),
						zap.Error(err))
				}
			}
		}
	}
	
	return nil
}

// AddPluginDirectory adds a plugin directory
func (l *AdaptivePluginLoader) AddPluginDirectory(dirPath string) {
	l.dirsMu.Lock()
	defer l.dirsMu.Unlock()
	
	// Check if directory already exists
	for _, dir := range l.pluginDirs {
		if dir == dirPath {
			return
		}
	}
	
	l.pluginDirs = append(l.pluginDirs, dirPath)
}

// RemovePluginDirectory removes a plugin directory
func (l *AdaptivePluginLoader) RemovePluginDirectory(dirPath string) {
	l.dirsMu.Lock()
	defer l.dirsMu.Unlock()
	
	// Find the directory
	for i, dir := range l.pluginDirs {
		if dir == dirPath {
			// Remove the directory
			l.pluginDirs = append(l.pluginDirs[:i], l.pluginDirs[i+1:]...)
			return
		}
	}
}

// GetPluginDirectories gets the plugin directories
func (l *AdaptivePluginLoader) GetPluginDirectories() []string {
	l.dirsMu.RLock()
	defer l.dirsMu.RUnlock()
	
	dirs := make([]string, len(l.pluginDirs))
	copy(dirs, l.pluginDirs)
	
	return dirs
}

// SetMaxConcurrentLoads sets the maximum number of concurrent plugin loads
func (l *AdaptivePluginLoader) SetMaxConcurrentLoads(max int) {
	// Use config-specific lock with deadlock detection
	if !l.acquireLock(&l.configMu, "configMu", 5*time.Second) {
		l.logger.Error("Failed to acquire config lock (potential deadlock)")
		return
	}
	defer l.configMu.Unlock()
	
	// Create a new semaphore with the new size
	oldSemaphore := l.loadSemaphore
	l.loadSemaphore = make(chan struct{}, max)
	l.maxConcurrent = max
	
	// Close the old semaphore
	close(oldSemaphore)
}

// SetLoadTimeout sets the timeout for plugin loading
func (l *AdaptivePluginLoader) SetLoadTimeout(timeout time.Duration) {
	// Use config-specific lock with deadlock detection
	if !l.acquireLock(&l.configMu, "configMu", 5*time.Second) {
		l.logger.Error("Failed to acquire config lock (potential deadlock)")
		return
	}
	defer l.configMu.Unlock()
	
	l.loadTimeout = timeout
}

// SetScanInterval sets the interval for plugin scanning
func (l *AdaptivePluginLoader) SetScanInterval(interval time.Duration) {
	// Use config-specific lock with deadlock detection
	if !l.acquireLock(&l.configMu, "configMu", 5*time.Second) {
		l.logger.Error("Failed to acquire config lock (potential deadlock)")
		return
	}
	defer l.configMu.Unlock()
	
	l.scanInterval = interval
	l.baseScanInterval = interval
	l.adaptiveScanEnabled = false
}

// SetAdaptiveScanInterval enables adaptive scan intervals with the given base interval
func (l *AdaptivePluginLoader) SetAdaptiveScanInterval(baseInterval time.Duration) {
	// Use config-specific lock with deadlock detection
	if !l.acquireLock(&l.configMu, "configMu", 5*time.Second) {
		l.logger.Error("Failed to acquire config lock (potential deadlock)")
		return
	}
	defer l.configMu.Unlock()
	
	l.baseScanInterval = baseInterval
	l.scanInterval = baseInterval
	l.adaptiveScanEnabled = true
}

// getNextScanInterval calculates the next scan interval based on system load
func (l *AdaptivePluginLoader) getNextScanInterval() time.Duration {
	// Use config-specific lock with deadlock detection
	if !l.acquireLockRO(&l.configMu, "configMu", 5*time.Second) {
		l.logger.Error("Failed to acquire config lock (potential deadlock)")
		return 5 * time.Minute // Default fallback
	}
	
	if !l.adaptiveScanEnabled {
		interval := l.scanInterval
		l.configMu.RUnlock()
		return interval
	}
	
	baseInterval := l.baseScanInterval
	l.configMu.RUnlock()
	
	// Check memory pressure
	if l.memoryMonitor.IsUnderMemoryPressure() {
		// Increase interval under memory pressure (3x longer)
		return baseInterval * 3
	}
	
	// Check CPU load (simplified for now)
	// In a real implementation, this would use OS-specific APIs to get CPU load
	var load float64 = 0.5 // Placeholder
	
	if load > 0.7 { // 70% CPU usage
		// Increase interval under high CPU load (2x longer)
		return baseInterval * 2
	}
	
	// Normal conditions - use base interval
	return baseInterval
}

// GetLoadedPlugins returns the loaded plugins
func (l *AdaptivePluginLoader) GetLoadedPlugins() []*LoadedPlugin {
	// Use plugin-specific lock with deadlock detection
	if !l.acquireLockRO(&l.pluginMu, "pluginMu", 5*time.Second) {
		l.logger.Error("Failed to acquire plugin lock (potential deadlock)")
		return nil
	}
	defer l.pluginMu.RUnlock()
	
	plugins := make([]*LoadedPlugin, 0, len(l.loadedPlugins))
	for _, plugin := range l.loadedPlugins {
		plugins = append(plugins, plugin)
	}
	
	return plugins
}

// GetLoadedPluginCount returns the number of loaded plugins
func (l *AdaptivePluginLoader) GetLoadedPluginCount() int {
	// Use plugin-specific lock with deadlock detection
	if !l.acquireLockRO(&l.pluginMu, "pluginMu", 5*time.Second) {
		l.logger.Error("Failed to acquire plugin lock (potential deadlock)")
		return 0
	}
	defer l.pluginMu.RUnlock()
	
	return len(l.loadedPlugins)
}

// GetTotalMemoryUsage returns the total memory usage of all loaded plugins
func (l *AdaptivePluginLoader) GetTotalMemoryUsage() int64 {
	// Use plugin-specific lock with deadlock detection
	if !l.acquireLockRO(&l.pluginMu, "pluginMu", 5*time.Second) {
		l.logger.Error("Failed to acquire plugin lock (potential deadlock)")
		return 0
	}
	defer l.pluginMu.RUnlock()
	
	var total int64
	for _, plugin := range l.loadedPlugins {
		total += plugin.MemoryUse
	}
	
	return total
}

// MemoryMonitor monitors memory usage
type MemoryMonitor struct {
	logger *zap.Logger
	// Memory pressure threshold (percentage of total memory)
	pressureThreshold float64
	// Last memory check time
	lastCheck time.Time
	// Cache duration for memory stats
	cacheDuration time.Duration
	// Cached memory stats
	cachedAvailable int64
	cachedTotal     int64
	mu              sync.RWMutex
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(logger *zap.Logger) *MemoryMonitor {
	return &MemoryMonitor{
		logger:           logger,
		pressureThreshold: 0.1, // 10% available memory threshold
		cacheDuration:    5 * time.Second,
		lastCheck:        time.Time{}, // Zero time
	}
}

// GetCurrentMemoryUsage returns the current memory usage
func (m *MemoryMonitor) GetCurrentMemoryUsage() int64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int64(memStats.Alloc)
}

// GetAvailableMemory returns the available memory
func (m *MemoryMonitor) GetAvailableMemory() int64 {
	m.mu.RLock()
	if time.Since(m.lastCheck) < m.cacheDuration {
		// Use cached value if recent enough
		available := m.cachedAvailable

// Deadlock detection and lock acquisition helpers

// acquireLock attempts to acquire a mutex with a timeout to prevent deadlocks
func (l *AdaptivePluginLoader) acquireLock(mu *sync.Mutex, name string, timeout time.Duration) bool {
	if !l.deadlockDetection {
		mu.Lock()
		return true
	}
	
	// Create a channel to signal when the lock is acquired
	done := make(chan struct{}, 1)
	
	// Try to acquire the lock in a goroutine
	go func() {
		mu.Lock()
		done <- struct{}{}
	}()
	
	// Wait for the lock with timeout
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		l.logger.Warn("Potential deadlock detected",
			zap.String("lock", name),
			zap.Duration("timeout", timeout))
		return false
	}
}

// acquireLockRO attempts to acquire a read lock with a timeout to prevent deadlocks
func (l *AdaptivePluginLoader) acquireLockRO(mu *sync.RWMutex, name string, timeout time.Duration) bool {
	if !l.deadlockDetection {
		mu.RLock()
		return true
	}
	
	// Create a channel to signal when the lock is acquired
	done := make(chan struct{}, 1)
	
	// Try to acquire the lock in a goroutine
	go func() {
		mu.RLock()
		done <- struct{}{}
	}()
	
	// Wait for the lock with timeout
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		l.logger.Warn("Potential deadlock detected",
			zap.String("lock", name),
			zap.String("mode", "read"),
			zap.Duration("timeout", timeout))
		return false
	}
}

// SetDeadlockDetection enables or disables deadlock detection
func (l *AdaptivePluginLoader) SetDeadlockDetection(enabled bool) {
	// Use config-specific lock with direct locking (to avoid recursion)
	l.configMu.Lock()
	defer l.configMu.Unlock()
	
	l.deadlockDetection = enabled
}

// SetLockTimeout sets the timeout for a specific lock
func (l *AdaptivePluginLoader) SetLockTimeout(lockName string, timeout time.Duration) {
	// Use config-specific lock with direct locking (to avoid recursion)
	l.configMu.Lock()
	defer l.configMu.Unlock()
	
	l.lockTimeouts[lockName] = timeout
}

// GetLockTimeout gets the timeout for a specific lock
func (l *AdaptivePluginLoader) GetLockTimeout(lockName string) time.Duration {
	// Use config-specific lock with direct locking (to avoid recursion)
	l.configMu.RLock()
	defer l.configMu.RUnlock()
	
	timeout, ok := l.lockTimeouts[lockName]
	if !ok {
		return 5 * time.Second // Default timeout
	}
	return timeout
}
		m.mu.RUnlock()
		return available
	}
	m.mu.RUnlock()
	
	// Need to refresh the cache
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check again in case another goroutine updated while we were waiting for the lock
	if time.Since(m.lastCheck) < m.cacheDuration {
		return m.cachedAvailable
	}
	
	// Get system memory info
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		m.logger.Error("Failed to get system info", zap.Error(err))
		// Fallback to a reasonable default if we can't get system info
		return 8 * 1024 * 1024 * 1024 // 8GB
	}
	
	// Update cache
	m.cachedAvailable = int64(info.Freeram)
	m.cachedTotal = int64(info.Totalram)
	m.lastCheck = time.Now()
	
	return m.cachedAvailable
}

// GetTotalSystemMemory returns the total system memory
func (m *MemoryMonitor) GetTotalSystemMemory() int64 {
	m.mu.RLock()
	if time.Since(m.lastCheck) < m.cacheDuration {
		// Use cached value if recent enough
		total := m.cachedTotal
		m.mu.RUnlock()
		return total
	}
	m.mu.RUnlock()
	
	// Need to refresh the cache - GetAvailableMemory also updates total
	m.GetAvailableMemory()
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cachedTotal
}

// IsMemoryAvailable checks if enough memory is available
func (m *MemoryMonitor) IsMemoryAvailable(required int64) bool {
	available := m.GetAvailableMemory()
	return available >= required
}

// IsUnderMemoryPressure checks if the system is under memory pressure
func (m *MemoryMonitor) IsUnderMemoryPressure() bool {
	available := m.GetAvailableMemory()
	total := m.GetTotalSystemMemory()
	
	// Under pressure if available memory is less than threshold percentage of total
	return available < int64(float64(total)*m.pressureThreshold)
}

// SetPressureThreshold sets the memory pressure threshold
func (m *MemoryMonitor) SetPressureThreshold(threshold float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if threshold <= 0 || threshold >= 1 {
		m.logger.Warn("Invalid memory pressure threshold, must be between 0 and 1", 
			zap.Float64("threshold", threshold))
		return
	}
	
	m.pressureThreshold = threshold
}

// SetCacheDuration sets the cache duration for memory stats
func (m *MemoryMonitor) SetCacheDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.cacheDuration = duration
}
