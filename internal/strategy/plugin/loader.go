package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// StrategyPluginLoader loads strategy plugins
type StrategyPluginLoader struct {
	logger       *zap.Logger
	pluginDirs   []string
	registry     *StrategyPluginRegistry
	loadedPlugins map[string]*plugin.Plugin
	mu           sync.RWMutex
}

// NewStrategyPluginLoader creates a new strategy plugin loader
func NewStrategyPluginLoader(
	logger *zap.Logger,
	registry *StrategyPluginRegistry,
	pluginDirs []string,
) *StrategyPluginLoader {
	return &StrategyPluginLoader{
		logger:       logger,
		pluginDirs:   pluginDirs,
		registry:     registry,
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// LoadPlugins loads all plugins from the configured directories
func (l *StrategyPluginLoader) LoadPlugins() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for _, dir := range l.pluginDirs {
		if err := l.loadPluginsFromDir(dir); err != nil {
			l.logger.Error("Failed to load plugins from directory",
				zap.String("directory", dir),
				zap.Error(err))
		}
	}
	
	return nil
}

// loadPluginsFromDir loads all plugins from a directory
func (l *StrategyPluginLoader) loadPluginsFromDir(dir string) error {
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
	
	// Load each plugin
	for _, file := range files {
		if err := l.LoadPlugin(file); err != nil {
			l.logger.Error("Failed to load plugin",
				zap.String("file", file),
				zap.Error(err))
		}
	}
	
	return nil
}

// LoadPlugin loads a plugin from a file
func (l *StrategyPluginLoader) LoadPlugin(file string) error {
	l.logger.Info("Loading plugin", zap.String("file", file))
	
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
	
	// Create a plugin wrapper
	wrapper := &StrategyPluginWrapper{
		Info:         info,
		CreateFunc:   createFunc,
		FilePath:     file,
	}
	
	// Register the plugin
	l.registry.RegisterPlugin(info.StrategyType, wrapper)
	
	// Store the loaded plugin
	l.loadedPlugins[file] = plug
	
	l.logger.Info("Loaded plugin",
		zap.String("name", info.Name),
		zap.String("version", info.Version),
		zap.String("strategy_type", info.StrategyType),
	)
	
	return nil
}

// ReloadPlugin reloads a plugin from a file
func (l *StrategyPluginLoader) ReloadPlugin(file string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if the plugin is loaded
	if _, ok := l.loadedPlugins[file]; ok {
		// Unregister the plugin
		// Note: This is a simplistic approach. In a real implementation,
		// we would need to handle active strategies created by this plugin.
		l.registry.UnregisterPluginByFile(file)
		
		// Remove from loaded plugins
		delete(l.loadedPlugins, file)
	}
	
	// Load the plugin
	return l.LoadPlugin(file)
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
	FilePath   string
}

// GetStrategyType returns the type of strategy provided by this plugin
func (w *StrategyPluginWrapper) GetStrategyType() string {
	return w.Info.StrategyType
}

// CreateStrategy creates a strategy instance
func (w *StrategyPluginWrapper) CreateStrategy(config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error) {
	return w.CreateFunc(config, logger)
}

