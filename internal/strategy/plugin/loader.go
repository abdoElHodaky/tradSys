package plugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// StrategyPluginLoader loads strategy plugins
type StrategyPluginLoader struct {
	logger         *zap.Logger
	pluginDirs     []string
	registry       *StrategyPluginRegistry
	loadedPlugins  map[string]*plugin.Plugin
	mu             sync.RWMutex
	
	// Plugin metadata cache
	metadataCache  map[string]*PluginMetadata
	cacheMu        sync.RWMutex
	
	// Validation settings
	maxValidators  int
	validationSem  chan struct{}
	
	// Retry settings
	maxRetries     int
	retryDelay     time.Duration
}

// NewStrategyPluginLoader creates a new strategy plugin loader
func NewStrategyPluginLoader(
	logger *zap.Logger,
	registry *StrategyPluginRegistry,
	pluginDirs []string,
) *StrategyPluginLoader {
	// Default to number of CPUs for concurrent validation
	maxValidators := runtime.NumCPU()
	
	return &StrategyPluginLoader{
		logger:         logger,
		pluginDirs:     pluginDirs,
		registry:       registry,
		loadedPlugins:  make(map[string]*plugin.Plugin),
		metadataCache:  make(map[string]*PluginMetadata),
		maxValidators:  maxValidators,
		validationSem:  make(chan struct{}, maxValidators),
		maxRetries:     3,
		retryDelay:     100 * time.Millisecond,
	}
}

// LoadPlugins loads all plugins from the configured directories
func (l *StrategyPluginLoader) LoadPlugins() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Create a context for the loading operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Track errors
	var loadErrors []error
	
	// Use a wait group to track parallel operations
	var wg sync.WaitGroup
	
	// Create a channel for collecting errors
	errChan := make(chan error, len(l.pluginDirs)*10) // Buffer size is an estimate
	
	// Start a goroutine to collect errors
	go func() {
		for err := range errChan {
			loadErrors = append(loadErrors, err)
		}
	}()
	
	for _, dir := range l.pluginDirs {
		wg.Add(1)
		go func(directory string) {
			defer wg.Done()
			
			if err := l.loadPluginsFromDir(ctx, directory); err != nil {
				l.logger.Error("Failed to load plugins from directory",
					zap.String("directory", directory),
					zap.Error(err))
				errChan <- fmt.Errorf("failed to load plugins from directory %s: %w", directory, err)
			}
		}(dir)
	}
	
	// Wait for all loading operations to complete
	wg.Wait()
	close(errChan)
	
	// If there were errors, return a combined error
	if len(loadErrors) > 0 {
		return fmt.Errorf("encountered %d errors while loading plugins", len(loadErrors))
	}
	
	return nil
}

// loadPluginsFromDir loads all plugins from a directory
func (l *StrategyPluginLoader) loadPluginsFromDir(ctx context.Context, dir string) error {
	// Check if the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("plugin directory does not exist: %s", dir)
	}
	
	// Find all .so files in the directory
	files, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files: %w", err)
	}
	
	l.logger.Info("Found plugin files", zap.Strings("files", files))
	
	// Use a wait group to track parallel operations
	var wg sync.WaitGroup
	
	// Create a channel for collecting errors
	errChan := make(chan error, len(files))
	
	// Track errors
	var loadErrors []error
	
	// Start a goroutine to collect errors
	go func() {
		for err := range errChan {
			loadErrors = append(loadErrors, err)
		}
	}()
	
	// Load each plugin in parallel
	for _, file := range files {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue with loading
		}
		
		// Acquire a validation semaphore
		l.validationSem <- struct{}{}
		
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			defer func() { <-l.validationSem }() // Release semaphore
			
			// Try to load the plugin with retries
			var loadErr error
			for attempt := 0; attempt < l.maxRetries; attempt++ {
				if attempt > 0 {
					// Log retry attempt
					l.logger.Info("Retrying plugin load",
						zap.String("file", filePath),
						zap.Int("attempt", attempt+1),
						zap.Int("max_attempts", l.maxRetries))
					
					// Wait before retrying
					select {
					case <-ctx.Done():
						errChan <- ctx.Err()
						return
					case <-time.After(l.retryDelay * time.Duration(attempt+1)):
						// Exponential backoff
					}
				}
				
				// Try to load the plugin
				if err := l.LoadPlugin(filePath); err != nil {
					loadErr = err
					l.logger.Warn("Failed to load plugin, may retry",
						zap.String("file", filePath),
						zap.Error(err),
						zap.Int("attempt", attempt+1),
						zap.Int("max_attempts", l.maxRetries))
					continue
				}
				
				// If we get here, the plugin loaded successfully
				loadErr = nil
				break
			}
			
			// If we still have an error after all retries, log it
			if loadErr != nil {
				l.logger.Error("Failed to load plugin after all retries",
					zap.String("file", filePath),
					zap.Error(loadErr),
					zap.Int("max_attempts", l.maxRetries))
				errChan <- fmt.Errorf("failed to load plugin %s after %d attempts: %w", 
					filePath, l.maxRetries, loadErr)
			}
		}(file)
	}
	
	// Wait for all loading operations to complete
	wg.Wait()
	close(errChan)
	
	// If there were errors, return a combined error
	if len(loadErrors) > 0 {
		return fmt.Errorf("encountered %d errors while loading plugins from directory %s", 
			len(loadErrors), dir)
	}
	
	return nil
}

// LoadPlugin loads a plugin from a file
func (l *StrategyPluginLoader) LoadPlugin(file string) error {
	startTime := time.Now()
	l.logger.Info("Loading plugin", zap.String("file", file))
	
	// Check if we have cached metadata for this plugin
	l.cacheMu.RLock()
	metadata, hasCachedMetadata := l.metadataCache[file]
	l.cacheMu.RUnlock()
	
	// If we have cached metadata, check if the file has been modified
	if hasCachedMetadata {
		modified, err := metadata.IsModified()
		if err != nil {
			l.logger.Warn("Failed to check if plugin is modified, will reload",
				zap.String("file", file),
				zap.Error(err))
		} else if !modified {
			// File hasn't changed, we can use the cached metadata
			l.logger.Debug("Using cached plugin metadata",
				zap.String("file", file),
				zap.String("name", metadata.Info.Name),
				zap.String("version", metadata.Info.Version))
			
			// If the plugin is already loaded and registered, we're done
			l.mu.RLock()
			_, alreadyLoaded := l.loadedPlugins[file]
			l.mu.RUnlock()
			
			if alreadyLoaded {
				return nil
			}
			
			// Otherwise, we need to load and register the plugin
			// but we can skip some validation steps
		}
	}
	
	// Create or update metadata
	if !hasCachedMetadata {
		var err error
		metadata, err = NewPluginMetadata(file)
		if err != nil {
			return fmt.Errorf("failed to create plugin metadata: %w", err)
		}
	} else {
		// Update existing metadata
		if err := metadata.UpdateMetadata(); err != nil {
			return fmt.Errorf("failed to update plugin metadata: %w", err)
		}
	}
	
	// Validate the plugin file
	if err := l.validatePluginFile(file); err != nil {
		return fmt.Errorf("plugin file validation failed: %w", err)
	}
	
	// Open the plugin
	plug, err := plugin.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}
	
	// Look up the plugin info
	infoSymbol, err := plug.Lookup(PluginInfoSymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s symbol: %w", PluginInfoSymbol, err)
	}
	
	// Assert that the symbol is a *PluginInfo
	info, ok := infoSymbol.(*PluginInfo)
	if !ok {
		return fmt.Errorf("plugin info is not of type *PluginInfo")
	}
	
	// Look up the create strategy function
	createSymbol, err := plug.Lookup(CreateStrategySymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s symbol: %w", CreateStrategySymbol, err)
	}
	
	// Assert that the symbol is a function with the correct signature
	createFunc, ok := createSymbol.(func(strategy.StrategyConfig, *zap.Logger) (strategy.Strategy, error))
	if !ok {
		return fmt.Errorf("create strategy function has incorrect signature")
	}
	
	// Look up the cleanup function (optional)
	var cleanupFunc func() error
	if cleanupSymbol, err := plug.Lookup(CleanupSymbol); err == nil {
		if cf, ok := cleanupSymbol.(func() error); ok {
			cleanupFunc = cf
		}
	}
	
	// Create a plugin wrapper
	wrapper := &StrategyPluginWrapper{
		Info:         info,
		CreateFunc:   createFunc,
		CleanupFunc:  cleanupFunc,
		FilePath:     file,
	}
	
	// Register the plugin
	l.registry.RegisterPlugin(info.StrategyType, wrapper)
	
	// Store the loaded plugin
	l.mu.Lock()
	l.loadedPlugins[file] = plug
	l.mu.Unlock()
	
	// Update metadata
	metadata.Info = info
	metadata.Validated = true
	metadata.ValidatedAt = time.Now()
	metadata.LoadDuration = time.Since(startTime)
	
	// Store updated metadata in cache
	l.cacheMu.Lock()
	l.metadataCache[file] = metadata
	l.cacheMu.Unlock()
	
	l.logger.Info("Loaded plugin",
		zap.String("name", info.Name),
		zap.String("version", info.Version),
		zap.String("strategy_type", info.StrategyType),
		zap.Duration("load_duration", metadata.LoadDuration),
	)
	
	return nil
}

// validatePluginFile validates a plugin file
func (l *StrategyPluginLoader) validatePluginFile(file string) error {
	// Check if the file exists
	fileInfo, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to stat plugin file: %w", err)
	}
	
	// Check if it's a regular file
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("plugin file is not a regular file")
	}
	
	// Check if it has the correct extension
	if filepath.Ext(file) != ".so" {
		return fmt.Errorf("plugin file does not have .so extension")
	}
	
	// Check if it's readable
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open plugin file: %w", err)
	}
	f.Close()
	
	return nil
}

// ReloadPlugin reloads a plugin from a file
func (l *StrategyPluginLoader) ReloadPlugin(file string) error {
	l.mu.Lock()
	
	// Check if the plugin is loaded
	if plug, ok := l.loadedPlugins[file]; ok {
		// Get the plugin wrapper from registry
		wrapper := l.registry.GetPluginByFile(file)
		if wrapper != nil {
			// Perform cleanup
			if err := wrapper.Cleanup(); err != nil {
				l.logger.Warn("Error during plugin cleanup",
					zap.String("file", file),
					zap.Error(err))
			}
		}
		
		// Unregister the plugin
		l.registry.UnregisterPluginByFile(file)
		
		// Remove from loaded plugins
		delete(l.loadedPlugins, file)
		
		// Remove from metadata cache
		l.cacheMu.Lock()
		delete(l.metadataCache, file)
		l.cacheMu.Unlock()
		
		// Suggest garbage collection
		plug = nil
		runtime.GC()
	}
	
	// Unlock before loading to prevent deadlocks
	l.mu.Unlock()
	
	// Load the plugin with retry logic
	var loadErr error
	for attempt := 0; attempt < l.maxRetries; attempt++ {
		if attempt > 0 {
			// Log retry attempt
			l.logger.Info("Retrying plugin reload",
				zap.String("file", file),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", l.maxRetries))
			
			// Wait before retrying with exponential backoff
			time.Sleep(l.retryDelay * time.Duration(attempt+1))
		}
		
		// Try to load the plugin
		if err := l.LoadPlugin(file); err != nil {
			loadErr = err
			l.logger.Warn("Failed to reload plugin, may retry",
				zap.String("file", file),
				zap.Error(err),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", l.maxRetries))
			continue
		}
		
		// If we get here, the plugin loaded successfully
		return nil
	}
	
	// If we get here, all retries failed
	return fmt.Errorf("failed to reload plugin %s after %d attempts: %w", 
		file, l.maxRetries, loadErr)
}

// GetLoadedPlugins returns the loaded plugins
func (l *StrategyPluginLoader) GetLoadedPlugins() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	plugins := make([]string, 0, len(l.loadedPlugins))
	for file := range l.loadedPlugins {
		plugins = append(plugins, file)
	}
	
	return plugins
}

// GetPluginMetadata returns metadata for a loaded plugin
func (l *StrategyPluginLoader) GetPluginMetadata(file string) (*PluginMetadata, bool) {
	l.cacheMu.RLock()
	defer l.cacheMu.RUnlock()
	
	metadata, ok := l.metadataCache[file]
	return metadata, ok
}

// GetAllPluginMetadata returns metadata for all loaded plugins
func (l *StrategyPluginLoader) GetAllPluginMetadata() map[string]*PluginMetadata {
	l.cacheMu.RLock()
	defer l.cacheMu.RUnlock()
	
	// Create a copy of the metadata map
	result := make(map[string]*PluginMetadata, len(l.metadataCache))
	for file, metadata := range l.metadataCache {
		result[file] = metadata
	}
	
	return result
}

// SetMaxRetries sets the maximum number of retries for plugin loading
func (l *StrategyPluginLoader) SetMaxRetries(maxRetries int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if maxRetries < 1 {
		maxRetries = 1
	}
	
	l.maxRetries = maxRetries
}

// SetRetryDelay sets the delay between retries for plugin loading
func (l *StrategyPluginLoader) SetRetryDelay(delay time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if delay < 0 {
		delay = 0
	}
	
	l.retryDelay = delay
}

// SetMaxValidators sets the maximum number of concurrent validators
func (l *StrategyPluginLoader) SetMaxValidators(max int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if max < 1 {
		max = 1
	}
	
	// Create a new semaphore with the new size
	oldSemaphore := l.validationSem
	l.validationSem = make(chan struct{}, max)
	l.maxValidators = max
	
	// Close the old semaphore
	close(oldSemaphore)
}

// AddPluginDirectory adds a plugin directory
func (l *StrategyPluginLoader) AddPluginDirectory(dir string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if the directory is already in the list
	for _, existingDir := range l.pluginDirs {
		if existingDir == dir {
			return
		}
	}
	
	l.pluginDirs = append(l.pluginDirs, dir)
}

// StrategyPluginWrapper wraps a strategy plugin
type StrategyPluginWrapper struct {
	Info       *PluginInfo
	CreateFunc func(strategy.StrategyConfig, *zap.Logger) (strategy.Strategy, error)
	CleanupFunc func() error
	FilePath   string
	mu         sync.Mutex
}

// GetStrategyType returns the type of strategy provided by this plugin
func (w *StrategyPluginWrapper) GetStrategyType() string {
	return w.Info.StrategyType
}

// CreateStrategy creates a strategy instance
func (w *StrategyPluginWrapper) CreateStrategy(config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error) {
	// Wrap the call with panic recovery
	var strat strategy.Strategy
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in plugin %s: %v", w.Info.Name, r)
			}
		}()
		strat, err = w.CreateFunc(config, logger)
	}()
	
	return strat, err
}

// Cleanup performs cleanup for this plugin
func (w *StrategyPluginWrapper) Cleanup() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.CleanupFunc != nil {
		if err := w.CleanupFunc(); err != nil {
			return fmt.Errorf("plugin cleanup error: %w", err)
		}
	}
	
	// Clear references to help GC
	w.CreateFunc = nil
	w.CleanupFunc = nil
	
	return nil
}
