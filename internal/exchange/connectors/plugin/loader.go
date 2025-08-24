package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	"go.uber.org/zap"
)

// PluginLoader loads exchange connector plugins
type PluginLoader struct {
	pluginDir string
	plugins   map[string]ExchangeConnectorPlugin
	logger    *zap.Logger
	mu        sync.RWMutex
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(pluginDir string, logger *zap.Logger) *PluginLoader {
	return &PluginLoader{
		pluginDir: pluginDir,
		plugins:   make(map[string]ExchangeConnectorPlugin),
		logger:    logger,
	}
}

// LoadPlugins loads all plugins from the plugin directory
func (l *PluginLoader) LoadPlugins() error {
	l.mu.Lock()
	defer l.mu.Unlock()

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
		if err := l.loadPlugin(file); err != nil {
			l.logger.Error("Failed to load plugin", zap.String("file", file), zap.Error(err))
			continue
		}
	}

	l.logger.Info("Loaded exchange connector plugins", zap.Int("count", len(l.plugins)))
	return nil
}

// loadPlugin loads a single plugin
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

	// Look up the create connector function
	createSymbol, err := p.Lookup(CreateConnectorSymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s: %w", CreateConnectorSymbol, err)
	}

	createFunc, ok := createSymbol.(func(connectors.ExchangeConfig, *zap.Logger) (connectors.ExchangeConnector, error))
	if !ok {
		return fmt.Errorf("create connector function has wrong signature")
	}

	// Create a plugin wrapper
	plugin := &pluginWrapper{
		info:       info,
		createFunc: createFunc,
	}

	// Register the plugin
	l.plugins[info.ExchangeName] = plugin

	l.logger.Info("Loaded exchange connector plugin",
		zap.String("name", info.Name),
		zap.String("version", info.Version),
		zap.String("author", info.Author),
		zap.String("exchange", info.ExchangeName))

	return nil
}

// GetPlugin returns a plugin by exchange name
func (l *PluginLoader) GetPlugin(exchangeName string) (ExchangeConnectorPlugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, ok := l.plugins[exchangeName]
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

// pluginWrapper implements the ExchangeConnectorPlugin interface
type pluginWrapper struct {
	info       *PluginInfo
	createFunc func(connectors.ExchangeConfig, *zap.Logger) (connectors.ExchangeConnector, error)
}

// GetExchangeName returns the name of the exchange
func (p *pluginWrapper) GetExchangeName() string {
	return p.info.ExchangeName
}

// CreateConnector creates an exchange connector
func (p *pluginWrapper) CreateConnector(config connectors.ExchangeConfig, logger *zap.Logger) (connectors.ExchangeConnector, error) {
	return p.createFunc(config, logger)
}

