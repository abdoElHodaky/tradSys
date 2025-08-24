package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/indicators"
	"go.uber.org/zap"
)

// PluginLoader loads technical indicator plugins
type PluginLoader struct {
	pluginDir string
	plugins   map[string]IndicatorPlugin
	logger    *zap.Logger
	mu        sync.RWMutex
	// Track plugins being loaded to prevent concurrent loading of the same plugin
	loadingPlugins sync.Map
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(pluginDir string, logger *zap.Logger) *PluginLoader {
	return &PluginLoader{
		pluginDir: pluginDir,
		plugins:   make(map[string]IndicatorPlugin),
		logger:    logger,
	}
}

// LoadPlugins loads all plugins from the plugin directory
func (l *PluginLoader) LoadPlugins() error {
	// Check if the plugin directory exists
	if _, err := os.Stat(l.pluginDir); os.IsNotExist(err) {
		l.logger.Warn("Plugin directory does not exist", zap.String("directory", l.pluginDir))
		return nil
	}

	// Find all .so files in the plugin directory
	files, err := filepath.Glob(filepath.Join(l.pluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files: %w", err)
	}

	for _, file := range files {
		// Use a separate function to ensure deferred mutex unlock happens properly
		if err := l.loadPluginFile(file); err != nil {
			l.logger.Error("Failed to load plugin", zap.String("file", file), zap.Error(err))
			continue
		}
	}

	l.logger.Info("Loaded technical indicator plugins", zap.Int("count", len(l.plugins)))
	return nil
}

// loadPluginFile loads a single plugin file with proper locking
func (l *PluginLoader) loadPluginFile(path string) error {
	// Use a loading marker to prevent concurrent loading of the same plugin
	if _, loaded := l.loadingPlugins.LoadOrStore(path, true); loaded {
		// Another goroutine is already loading this plugin
		return fmt.Errorf("plugin %s is already being loaded", path)
	}
	defer l.loadingPlugins.Delete(path)

	// Acquire write lock for the entire loading process
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.loadPlugin(path)
}

// loadPlugin loads a single plugin (must be called with lock held)
func (l *PluginLoader) loadPlugin(path string) error {
	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look up the plugin info
	infoSymbol, err := p.Lookup(PluginInfoSymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s: %w", PluginInfoSymbol, err)
	}

	info, ok := infoSymbol.(*PluginInfo)
	if !ok {
		return fmt.Errorf("plugin info is not of type *PluginInfo")
	}

	// Check if this plugin is already loaded
	if _, exists := l.plugins[info.IndicatorName]; exists {
		return fmt.Errorf("plugin for indicator %s is already loaded", info.IndicatorName)
	}

	// Look up the create indicator function
	createSymbol, err := p.Lookup(CreateIndicatorSymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s: %w", CreateIndicatorSymbol, err)
	}

	createFunc, ok := createSymbol.(func(indicators.IndicatorParams, *zap.Logger) (indicators.Indicator, error))
	if !ok {
		return fmt.Errorf("create indicator function has wrong signature")
	}

	// Create a plugin wrapper
	plugin := &pluginWrapper{
		info:       info,
		createFunc: createFunc,
		path:       path,
	}

	// Register the plugin
	l.plugins[info.IndicatorName] = plugin

	l.logger.Info("Loaded technical indicator plugin",
		zap.String("name", info.Name),
		zap.String("version", info.Version),
		zap.String("author", info.Author),
		zap.String("indicator", info.IndicatorName),
		zap.String("path", path))

	return nil
}

// GetPlugin returns a plugin by indicator name
func (l *PluginLoader) GetPlugin(indicatorName string) (IndicatorPlugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, ok := l.plugins[indicatorName]
	return plugin, ok
}

// GetAvailablePlugins returns a list of available plugins
func (l *PluginLoader) GetAvailablePlugins() []PluginInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var plugins []PluginInfo
	for _, p := range l.plugins {
		plugins = append(plugins, *p.(*pluginWrapper).info)
	}

	return plugins
}

// UnloadPlugin unloads a plugin by indicator name
func (l *PluginLoader) UnloadPlugin(indicatorName string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	plugin, ok := l.plugins[indicatorName]
	if !ok {
		return fmt.Errorf("plugin for indicator %s is not loaded", indicatorName)
	}

	// Call cleanup if the plugin implements it
	if cleanable, ok := plugin.(CleanupablePlugin); ok {
		if err := cleanable.Cleanup(); err != nil {
			l.logger.Warn("Error cleaning up plugin",
				zap.String("indicator", indicatorName),
				zap.Error(err))
		}
	}

	// Remove the plugin from the registry
	delete(l.plugins, indicatorName)

	l.logger.Info("Unloaded technical indicator plugin", zap.String("indicator", indicatorName))
	return nil
}

// pluginWrapper implements the IndicatorPlugin interface
type pluginWrapper struct {
	info       *PluginInfo
	createFunc func(indicators.IndicatorParams, *zap.Logger) (indicators.Indicator, error)
	path       string
}

// GetIndicatorName returns the name of the indicator
func (p *pluginWrapper) GetIndicatorName() string {
	return p.info.IndicatorName
}

// CreateIndicator creates a technical indicator
func (p *pluginWrapper) CreateIndicator(params indicators.IndicatorParams, logger *zap.Logger) (indicators.Indicator, error) {
	// Use panic recovery to prevent plugin failures from crashing the application
	var indicator indicators.Indicator
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in plugin %s: %v", p.info.Name, r)
				logger.Error("Panic in plugin",
					zap.String("plugin", p.info.Name),
					zap.String("indicator", p.info.IndicatorName),
					zap.Any("panic", r))
			}
		}()

		indicator, err = p.createFunc(params, logger)
	}()

	return indicator, err
}

// Cleanup performs cleanup for the plugin
func (p *pluginWrapper) Cleanup() error {
	// No cleanup needed for this plugin wrapper
	return nil
}

// CleanupablePlugin defines a plugin that can be cleaned up
type CleanupablePlugin interface {
	Cleanup() error
}

