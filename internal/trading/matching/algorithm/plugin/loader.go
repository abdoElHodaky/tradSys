package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin" // Go plugin
	"runtime"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/plugin/adaptive_loader"
	"go.uber.org/zap"
)

// Loader is a loader for matching algorithm plugins
type Loader struct {
	logger         *zap.Logger
	registry       *Registry
	pluginDirs     []string
	adaptiveLoader *adaptive_loader.AdaptivePluginLoader
	mu             sync.RWMutex
}

// NewLoader creates a new loader
func NewLoader(
	logger *zap.Logger,
	registry *Registry,
	pluginDirs []string,
) *Loader {
	return &Loader{
		logger:         logger,
		registry:       registry,
		pluginDirs:     pluginDirs,
		adaptiveLoader: adaptive_loader.NewAdaptivePluginLoader(logger, pluginDirs),
	}
}

// LoadPlugin loads a plugin from a file
func (l *Loader) LoadPlugin(filePath string) (MatchingAlgorithmPlugin, error) {
	l.logger.Info("Loading matching algorithm plugin", zap.String("file", filePath))
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file does not exist: %s", filePath)
	}
	
	// Open the plugin
	plug, err := plugin.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}
	
	// Look up the plugin factory
	factorySym, err := plug.Lookup("CreateMatchingAlgorithmPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export CreateMatchingAlgorithmPlugin: %w", err)
	}
	
	// Assert that the symbol is a factory function
	factoryFunc, ok := factorySym.(func() (MatchingAlgorithmPlugin, error))
	if !ok {
		return nil, fmt.Errorf("CreateMatchingAlgorithmPlugin is not of type func() (MatchingAlgorithmPlugin, error)")
	}
	
	// Create the plugin
	plugin, err := factoryFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}
	
	// Register the plugin
	if err := l.registry.RegisterPlugin(plugin); err != nil {
		return nil, fmt.Errorf("failed to register plugin: %w", err)
	}
	
	return plugin, nil
}

// LoadPlugins loads all plugins from a directory
func (l *Loader) LoadPlugins(dirPath string) ([]MatchingAlgorithmPlugin, error) {
	l.logger.Info("Loading matching algorithm plugins from directory", zap.String("dir", dirPath))
	
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin directory does not exist: %s", dirPath)
	}
	
	// Find all .so files in the directory
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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
	
	// Load plugins in parallel with limited concurrency
	var wg sync.WaitGroup
	maxConcurrent := runtime.NumCPU()
	semaphore := make(chan struct{}, maxConcurrent)
	
	var mu sync.Mutex
	var plugins []MatchingAlgorithmPlugin
	var errors []error
	
	for _, file := range files {
		// Acquire semaphore
		semaphore <- struct{}{}
		
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			plugin, err := l.LoadPlugin(file)
			
			mu.Lock()
			defer mu.Unlock()
			
			if err != nil {
				l.logger.Error("Failed to load plugin",
					zap.String("file", file),
					zap.Error(err))
				errors = append(errors, fmt.Errorf("failed to load %s: %w", file, err))
				return
			}
			
			plugins = append(plugins, plugin)
		}(file)
	}
	
	// Wait for all loads to complete
	wg.Wait()
	
	// Check for errors
	if len(errors) > 0 {
		return plugins, fmt.Errorf("failed to load %d plugins", len(errors))
	}
	
	return plugins, nil
}

// LoadPluginsWithContext loads all plugins from a directory with context
func (l *Loader) LoadPluginsWithContext(ctx context.Context, dirPath string) ([]MatchingAlgorithmPlugin, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		plugins []MatchingAlgorithmPlugin
		err     error
	})
	
	// Load plugins in a goroutine
	go func() {
		plugins, err := l.LoadPlugins(dirPath)
		resultCh <- struct {
			plugins []MatchingAlgorithmPlugin
			err     error
		}{plugins, err}
	}()
	
	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.plugins, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("plugin loading canceled: %w", ctx.Err())
	}
}

// LoadAllPlugins loads all plugins from all configured directories
func (l *Loader) LoadAllPlugins() ([]MatchingAlgorithmPlugin, error) {
	l.mu.Lock()
	dirs := make([]string, len(l.pluginDirs))
	copy(dirs, l.pluginDirs)
	l.mu.Unlock()
	
	var allPlugins []MatchingAlgorithmPlugin
	var allErrors []error
	
	for _, dir := range dirs {
		plugins, err := l.LoadPlugins(dir)
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("failed to load plugins from %s: %w", dir, err))
			continue
		}
		
		allPlugins = append(allPlugins, plugins...)
	}
	
	if len(allErrors) > 0 {
		return allPlugins, fmt.Errorf("failed to load plugins from %d directories", len(allErrors))
	}
	
	return allPlugins, nil
}

// AddPluginDirectory adds a plugin directory
func (l *Loader) AddPluginDirectory(dirPath string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if directory already exists
	for _, dir := range l.pluginDirs {
		if dir == dirPath {
			return
		}
	}
	
	l.pluginDirs = append(l.pluginDirs, dirPath)
	l.adaptiveLoader.AddPluginDirectory(dirPath)
}

// RemovePluginDirectory removes a plugin directory
func (l *Loader) RemovePluginDirectory(dirPath string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Find directory
	for i, dir := range l.pluginDirs {
		if dir == dirPath {
			// Remove directory
			l.pluginDirs = append(l.pluginDirs[:i], l.pluginDirs[i+1:]...)
			l.adaptiveLoader.RemovePluginDirectory(dirPath)
			return
		}
	}
}

// GetPluginDirectories gets the plugin directories
func (l *Loader) GetPluginDirectories() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	dirs := make([]string, len(l.pluginDirs))
	copy(dirs, l.pluginDirs)
	
	return dirs
}

