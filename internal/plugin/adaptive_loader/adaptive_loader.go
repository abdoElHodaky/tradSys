package adaptive_loader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AdaptivePluginLoader is an enhanced plugin loader that addresses
// conflicts and bottlenecks identified in the analysis.
type AdaptivePluginLoader struct {
	logger         *zap.Logger
	pluginDirs     []string
	loadedPlugins  map[string]*LoadedPlugin
	mu             sync.RWMutex
	maxConcurrent  int
	loadSemaphore  chan struct{}
	memoryMonitor  *MemoryMonitor
	loadTimeout    time.Duration
	scanInterval   time.Duration
	scannerRunning bool
	scannerStopCh  chan struct{}
	dirsMu         sync.RWMutex
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
	
	return &AdaptivePluginLoader{
		logger:        logger,
		pluginDirs:    pluginDirs,
		loadedPlugins: make(map[string]*LoadedPlugin),
		maxConcurrent: maxConcurrent,
		loadSemaphore: make(chan struct{}, maxConcurrent),
		memoryMonitor: NewMemoryMonitor(logger),
		loadTimeout:   30 * time.Second,
		scanInterval:  5 * time.Minute,
		scannerStopCh: make(chan struct{}),
	}
}

// FindPluginFiles finds all plugin files in the configured directories
func (l *AdaptivePluginLoader) FindPluginFiles() ([]string, error) {
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
	l.mu.Lock()
	if l.scannerRunning {
		l.mu.Unlock()
		return fmt.Errorf("plugin scanner already running")
	}
	
	l.scannerRunning = true
	l.scanInterval = scanInterval
	l.mu.Unlock()
	
	// Start the scanner in a goroutine
	go func() {
		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				// Scan for new plugins
				if err := l.scanForNewPlugins(ctx); err != nil {
					l.logger.Error("Failed to scan for new plugins", zap.Error(err))
				}
			case <-l.scannerStopCh:
				l.mu.Lock()
				l.scannerRunning = false
				l.mu.Unlock()
				return
			case <-ctx.Done():
				l.mu.Lock()
				l.scannerRunning = false
				l.mu.Unlock()
				return
			}
		}
	}()
	
	return nil
}

// StopPluginScanner stops the background plugin scanner
func (l *AdaptivePluginLoader) StopPluginScanner() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.scannerRunning {
		close(l.scannerStopCh)
		l.scannerStopCh = make(chan struct{})
	}
}

// scanForNewPlugins scans for new plugins
func (l *AdaptivePluginLoader) scanForNewPlugins(ctx context.Context) error {
	// Find all plugin files
	pluginFiles, err := l.FindPluginFiles()
	if err != nil {
		return fmt.Errorf("failed to find plugin files: %w", err)
	}
	
	// Check for new plugins
	l.mu.Lock()
	
	var newFiles []string
	for _, file := range pluginFiles {
		if _, ok := l.loadedPlugins[file]; !ok {
			newFiles = append(newFiles, file)
		}
	}
	
	l.mu.Unlock()
	
	// Log new plugins found
	if len(newFiles) > 0 {
		l.logger.Info("Found new plugins", zap.Int("count", len(newFiles)))
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
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Create a new semaphore with the new size
	oldSemaphore := l.loadSemaphore
	l.loadSemaphore = make(chan struct{}, max)
	l.maxConcurrent = max
	
	// Close the old semaphore
	close(oldSemaphore)
}

// SetLoadTimeout sets the timeout for plugin loading
func (l *AdaptivePluginLoader) SetLoadTimeout(timeout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.loadTimeout = timeout
}

// SetScanInterval sets the interval for plugin scanning
func (l *AdaptivePluginLoader) SetScanInterval(interval time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.scanInterval = interval
}

// GetLoadedPlugins returns the loaded plugins
func (l *AdaptivePluginLoader) GetLoadedPlugins() []*LoadedPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	plugins := make([]*LoadedPlugin, 0, len(l.loadedPlugins))
	for _, plugin := range l.loadedPlugins {
		plugins = append(plugins, plugin)
	}
	
	return plugins
}

// GetLoadedPluginCount returns the number of loaded plugins
func (l *AdaptivePluginLoader) GetLoadedPluginCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	return len(l.loadedPlugins)
}

// GetTotalMemoryUsage returns the total memory usage of all loaded plugins
func (l *AdaptivePluginLoader) GetTotalMemoryUsage() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	var total int64
	for _, plugin := range l.loadedPlugins {
		total += plugin.MemoryUse
	}
	
	return total
}

// MemoryMonitor monitors memory usage
type MemoryMonitor struct {
	logger *zap.Logger
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(logger *zap.Logger) *MemoryMonitor {
	return &MemoryMonitor{
		logger: logger,
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
	// This is a placeholder for a real implementation
	// In a real system, this would use OS-specific APIs to get available memory
	return 8 * 1024 * 1024 * 1024 // 8GB
}

// IsMemoryAvailable checks if enough memory is available
func (m *MemoryMonitor) IsMemoryAvailable(required int64) bool {
	available := m.GetAvailableMemory()
	return available >= required
}

