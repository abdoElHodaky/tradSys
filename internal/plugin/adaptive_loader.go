package plugin

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
	registry       *EnhancedPluginRegistry
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
}

// LoadedPlugin represents a loaded plugin
type LoadedPlugin struct {
	Path      string
	Type      string
	Name      string
	Info      *PluginInfo
	LoadTime  time.Time
	MemoryUse int64
	Instance  interface{}
}

// NewAdaptivePluginLoader creates a new adaptive plugin loader
func NewAdaptivePluginLoader(
	logger *zap.Logger,
	registry *EnhancedPluginRegistry,
	pluginDirs []string,
) *AdaptivePluginLoader {
	// Default to number of CPUs for concurrent loads
	maxConcurrent := runtime.NumCPU()
	
	return &AdaptivePluginLoader{
		logger:        logger,
		registry:      registry,
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

// LoadPlugins loads all plugins from the configured directories
func (l *AdaptivePluginLoader) LoadPlugins(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	var wg sync.WaitGroup
	errCh := make(chan error, 100) // Buffer for errors
	
	// Find all plugin files
	var pluginFiles []string
	for _, dir := range l.pluginDirs {
		files, err := l.findPluginFiles(dir)
		if err != nil {
			return fmt.Errorf("failed to find plugin files in %s: %w", dir, err)
		}
		pluginFiles = append(pluginFiles, files...)
	}
	
	l.logger.Info("Found plugin files", zap.Int("count", len(pluginFiles)))
	
	// Load plugins in parallel with limited concurrency
	for _, file := range pluginFiles {
		// Skip if already loaded
		if _, ok := l.loadedPlugins[file]; ok {
			continue
		}
		
		// Check if context is canceled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue
		}
		
		// Acquire semaphore
		l.loadSemaphore <- struct{}{}
		
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			defer func() { <-l.loadSemaphore }()
			
			// Load the plugin with timeout
			loadCtx, cancel := context.WithTimeout(ctx, l.loadTimeout)
			defer cancel()
			
			if err := l.loadPluginWithContext(loadCtx, file); err != nil {
				errCh <- fmt.Errorf("failed to load plugin %s: %w", file, err)
			}
		}(file)
	}
	
	// Wait for all loads to complete
	wg.Wait()
	close(errCh)
	
	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	
	// Return combined error if any
	if len(errs) > 0 {
		return fmt.Errorf("failed to load %d plugins: %v", len(errs), errs)
	}
	
	return nil
}

// findPluginFiles finds all plugin files in a directory
func (l *AdaptivePluginLoader) findPluginFiles(dir string) ([]string, error) {
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

// loadPluginWithContext loads a plugin with context
func (l *AdaptivePluginLoader) loadPluginWithContext(ctx context.Context, file string) error {
	// Create a channel for the result
	resultCh := make(chan error, 1)
	
	// Load the plugin in a goroutine
	go func() {
		// Record memory before loading
		memBefore := l.memoryMonitor.GetCurrentMemoryUsage()
		
		startTime := time.Now()
		l.logger.Debug("Loading plugin", zap.String("file", file))
		
		// Load the plugin
		loadedPlugin, err := l.loadPlugin(file)
		if err != nil {
			resultCh <- err
			return
		}
		
		// Record memory after loading
		memAfter := l.memoryMonitor.GetCurrentMemoryUsage()
		memoryUse := memAfter - memBefore
		if memoryUse < 0 {
			memoryUse = 0 // Avoid negative values due to GC
		}
		
		// Update memory usage
		loadedPlugin.MemoryUse = memoryUse
		
		// Store the loaded plugin
		l.loadedPlugins[file] = loadedPlugin
		
		duration := time.Since(startTime)
		l.logger.Info("Loaded plugin",
			zap.String("file", file),
			zap.String("type", loadedPlugin.Type),
			zap.String("name", loadedPlugin.Name),
			zap.Duration("duration", duration),
			zap.Int64("memory_use", memoryUse))
		
		resultCh <- nil
	}()
	
	// Wait for the result or context cancellation
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("plugin loading canceled: %w", ctx.Err())
	}
}

// loadPlugin loads a plugin
func (l *AdaptivePluginLoader) loadPlugin(file string) (*LoadedPlugin, error) {
	// This is a placeholder for the actual plugin loading logic
	// In a real implementation, this would use Go's plugin package
	// or another mechanism to load the plugin
	
	// For now, we'll just create a dummy loaded plugin
	loadedPlugin := &LoadedPlugin{
		Path:     file,
		Type:     "dummy",
		Name:     filepath.Base(file),
		Info:     &PluginInfo{Name: filepath.Base(file), Version: "1.0.0"},
		LoadTime: time.Now(),
	}
	
	return loadedPlugin, nil
}

// UnloadPlugin unloads a plugin
func (l *AdaptivePluginLoader) UnloadPlugin(file string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if the plugin is loaded
	loadedPlugin, ok := l.loadedPlugins[file]
	if !ok {
		return fmt.Errorf("plugin not loaded: %s", file)
	}
	
	// Unregister from registry
	if err := l.registry.UnregisterPlugin(loadedPlugin.Type, loadedPlugin.Name); err != nil {
		return fmt.Errorf("failed to unregister plugin: %w", err)
	}
	
	// Remove from loaded plugins
	delete(l.loadedPlugins, file)
	
	l.logger.Info("Unloaded plugin",
		zap.String("file", file),
		zap.String("type", loadedPlugin.Type),
		zap.String("name", loadedPlugin.Name))
	
	return nil
}

// ReloadPlugin reloads a plugin
func (l *AdaptivePluginLoader) ReloadPlugin(ctx context.Context, file string) error {
	l.mu.Lock()
	
	// Check if the plugin is loaded
	_, ok := l.loadedPlugins[file]
	if !ok {
		l.mu.Unlock()
		return fmt.Errorf("plugin not loaded: %s", file)
	}
	
	// Unload the plugin
	if err := l.UnloadPlugin(file); err != nil {
		l.mu.Unlock()
		return fmt.Errorf("failed to unload plugin: %w", err)
	}
	
	l.mu.Unlock()
	
	// Load the plugin
	if err := l.loadPluginWithContext(ctx, file); err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}
	
	return nil
}

// StartPluginScanner starts a background scanner for plugin changes
func (l *AdaptivePluginLoader) StartPluginScanner(ctx context.Context) error {
	l.mu.Lock()
	if l.scannerRunning {
		l.mu.Unlock()
		return fmt.Errorf("plugin scanner already running")
	}
	
	l.scannerRunning = true
	l.mu.Unlock()
	
	// Start the scanner in a goroutine
	go func() {
		ticker := time.NewTicker(l.scanInterval)
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
	var pluginFiles []string
	for _, dir := range l.pluginDirs {
		files, err := l.findPluginFiles(dir)
		if err != nil {
			return fmt.Errorf("failed to find plugin files in %s: %w", dir, err)
		}
		pluginFiles = append(pluginFiles, files...)
	}
	
	// Check for new plugins
	l.mu.Lock()
	defer l.mu.Unlock()
	
	var newFiles []string
	for _, file := range pluginFiles {
		if _, ok := l.loadedPlugins[file]; !ok {
			newFiles = append(newFiles, file)
		}
	}
	
	// Load new plugins
	if len(newFiles) > 0 {
		l.logger.Info("Found new plugins", zap.Int("count", len(newFiles)))
		
		// Unlock while loading
		l.mu.Unlock()
		
		var wg sync.WaitGroup
		errCh := make(chan error, len(newFiles))
		
		for _, file := range newFiles {
			// Acquire semaphore
			l.loadSemaphore <- struct{}{}
			
			wg.Add(1)
			go func(file string) {
				defer wg.Done()
				defer func() { <-l.loadSemaphore }()
				
				// Load the plugin with timeout
				loadCtx, cancel := context.WithTimeout(ctx, l.loadTimeout)
				defer cancel()
				
				if err := l.loadPluginWithContext(loadCtx, file); err != nil {
					errCh <- fmt.Errorf("failed to load plugin %s: %w", file, err)
				}
			}(file)
		}
		
		// Wait for all loads to complete
		wg.Wait()
		close(errCh)
		
		// Collect errors
		var errs []error
		for err := range errCh {
			errs = append(errs, err)
		}
		
		// Lock again
		l.mu.Lock()
		
		// Return combined error if any
		if len(errs) > 0 {
			return fmt.Errorf("failed to load %d plugins: %v", len(errs), errs)
		}
	}
	
	return nil
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

