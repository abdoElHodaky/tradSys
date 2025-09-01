package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// StrategyPlugin represents a loaded strategy plugin
type StrategyPlugin struct {
	Path       string
	Handle     *plugin.Plugin
	Strategy   strategy.Strategy
	Initialized bool
}

// StrategyLoader is responsible for loading strategy plugins
type StrategyLoader struct {
	plugins     map[string]*StrategyPlugin
	pluginsDir  string
	logger      *zap.Logger
	mu          sync.RWMutex
}

// NewStrategyLoader creates a new strategy loader
func NewStrategyLoader(pluginsDir string, logger *zap.Logger) *StrategyLoader {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &StrategyLoader{
		plugins:    make(map[string]*StrategyPlugin),
		pluginsDir: pluginsDir,
		logger:     logger,
	}
}

// LoadPlugin loads a strategy plugin from the specified path
func (l *StrategyLoader) LoadPlugin(ctx context.Context, pluginName string) (strategy.Strategy, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if plugin is already loaded
	if p, exists := l.plugins[pluginName]; exists && p.Initialized {
		l.logger.Debug("Plugin already loaded", zap.String("plugin", pluginName))
		return p.Strategy, nil
	}

	// Construct plugin path
	pluginPath := filepath.Join(l.pluginsDir, pluginName+".so")
	l.logger.Debug("Loading plugin", zap.String("path", pluginPath))

	// Check if plugin file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file not found: %s", pluginPath)
	}

	// Open the plugin
	plug, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look up the strategy symbol
	sym, err := plug.Lookup("Strategy")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup 'Strategy' symbol: %w", err)
	}

	// Assert that the symbol is a strategy.Strategy
	var s strategy.Strategy
	var ok bool
	if s, ok = sym.(strategy.Strategy); !ok {
		return nil, fmt.Errorf("plugin symbol is not a strategy.Strategy")
	}

	// Initialize the strategy
	if err := s.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// Store the loaded plugin
	l.plugins[pluginName] = &StrategyPlugin{
		Path:       pluginPath,
		Handle:     plug,
		Strategy:   s,
		Initialized: true,
	}

	l.logger.Info("Successfully loaded and initialized plugin",
		zap.String("plugin", pluginName),
		zap.String("strategy", s.GetName()),
	)

	return s, nil
}

// UnloadPlugin unloads a strategy plugin
func (l *StrategyLoader) UnloadPlugin(ctx context.Context, pluginName string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	p, exists := l.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not loaded: %s", pluginName)
	}

	// Shutdown the strategy
	if p.Initialized && p.Strategy != nil {
		if err := p.Strategy.Shutdown(ctx); err != nil {
			return fmt.Errorf("plugin cleanup error: %w", err)
		}
	}

	// Remove from loaded plugins
	delete(l.plugins, pluginName)
	l.logger.Info("Unloaded plugin", zap.String("plugin", pluginName))

	return nil
}

// GetLoadedPlugins returns a list of loaded plugin names
func (l *StrategyLoader) GetLoadedPlugins() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugins := make([]string, 0, len(l.plugins))
	for name := range l.plugins {
		plugins = append(plugins, name)
	}

	return plugins
}

// GetStrategy returns a loaded strategy by name
func (l *StrategyLoader) GetStrategy(pluginName string) (strategy.Strategy, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	p, exists := l.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin not loaded: %s", pluginName)
	}

	return p.Strategy, nil
}

// Cleanup unloads all plugins
func (l *StrategyLoader) Cleanup(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var lastErr error
	for name, p := range l.plugins {
		if p.Initialized && p.Strategy != nil {
			if err := p.Strategy.Shutdown(ctx); err != nil {
				l.logger.Error("Failed to shutdown strategy",
					zap.String("plugin", name),
					zap.Error(err),
				)
				lastErr = err
			}
		}
	}

	// Clear the plugins map
	l.plugins = make(map[string]*StrategyPlugin)
	
	if lastErr != nil {
		return fmt.Errorf("errors occurred during cleanup: %w", lastErr)
	}
	
	return nil
}

