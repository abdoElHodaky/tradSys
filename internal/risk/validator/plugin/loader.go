package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"go.uber.org/zap"
)

// ValidatorPluginLoader loads validator plugins
type ValidatorPluginLoader struct {
	logger        *zap.Logger
	pluginDirs    []string
	registry      *ValidatorPluginRegistry
	loadedPlugins map[string]*plugin.Plugin
	mu            sync.RWMutex
}

// NewValidatorPluginLoader creates a new validator plugin loader
func NewValidatorPluginLoader(
	logger *zap.Logger,
	registry *ValidatorPluginRegistry,
	pluginDirs []string,
) *ValidatorPluginLoader {
	return &ValidatorPluginLoader{
		logger:        logger,
		pluginDirs:    pluginDirs,
		registry:      registry,
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// LoadPlugins loads all plugins from the configured directories
func (l *ValidatorPluginLoader) LoadPlugins() error {
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
func (l *ValidatorPluginLoader) loadPluginsFromDir(dir string) error {
	// Check if the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("plugin directory does not exist: %s", dir)
	}
	
	// Find all .so files in the directory
	files, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files: %w", err)
	}
	
	l.logger.Info("Found validator plugin files", zap.Strings("files", files))
	
	// Load each plugin
	for _, file := range files {
		if err := l.LoadPlugin(file); err != nil {
			l.logger.Error("Failed to load validator plugin",
				zap.String("file", file),
				zap.Error(err))
		}
	}
	
	return nil
}

// LoadPlugin loads a plugin from a file
func (l *ValidatorPluginLoader) LoadPlugin(file string) error {
	l.logger.Info("Loading validator plugin", zap.String("file", file))
	
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
	
	// Look up the create validator function
	createSymbol, err := plug.Lookup(CreateValidatorSymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s symbol: %w", CreateValidatorSymbol, err)
	}
	
	// Assert that the symbol is a function with the correct signature
	createFunc, ok := createSymbol.(func(ValidatorConfig, *zap.Logger) (RiskValidator, error))
	if !ok {
		return fmt.Errorf("create validator function has incorrect signature")
	}
	
	// Create a plugin wrapper
	wrapper := &ValidatorPluginWrapper{
		Info:       info,
		CreateFunc: createFunc,
		FilePath:   file,
	}
	
	// Register the plugin
	l.registry.RegisterPlugin(info.ValidatorType, wrapper)
	
	// Store the loaded plugin
	l.loadedPlugins[file] = plug
	
	l.logger.Info("Loaded validator plugin",
		zap.String("name", info.Name),
		zap.String("version", info.Version),
		zap.String("validator_type", info.ValidatorType),
	)
	
	return nil
}

// ReloadPlugin reloads a plugin from a file
func (l *ValidatorPluginLoader) ReloadPlugin(file string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if the plugin is loaded
	if _, ok := l.loadedPlugins[file]; ok {
		// Unregister the plugin
		l.registry.UnregisterPluginByFile(file)
		
		// Remove from loaded plugins
		delete(l.loadedPlugins, file)
	}
	
	// Load the plugin
	return l.LoadPlugin(file)
}

// GetLoadedPlugins returns the loaded plugins
func (l *ValidatorPluginLoader) GetLoadedPlugins() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	plugins := make([]string, 0, len(l.loadedPlugins))
	for file := range l.loadedPlugins {
		plugins = append(plugins, file)
	}
	
	return plugins
}

// AddPluginDirectory adds a plugin directory
func (l *ValidatorPluginLoader) AddPluginDirectory(dir string) {
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

// ValidatorPluginWrapper wraps a validator plugin
type ValidatorPluginWrapper struct {
	Info       *PluginInfo
	CreateFunc func(ValidatorConfig, *zap.Logger) (RiskValidator, error)
	FilePath   string
}

// GetValidatorType returns the type of validator provided by this plugin
func (w *ValidatorPluginWrapper) GetValidatorType() string {
	return w.Info.ValidatorType
}

// CreateValidator creates a validator instance
func (w *ValidatorPluginWrapper) CreateValidator(config ValidatorConfig, logger *zap.Logger) (RiskValidator, error) {
	return w.CreateFunc(config, logger)
}

