package plugin

import (
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// StrategyPluginRegistry manages strategy plugins
type StrategyPluginRegistry struct {
	logger  *zap.Logger
	plugins map[string]StrategyPlugin
	fileMap map[string]string // Maps file paths to strategy types
	mu      sync.RWMutex
}

// NewStrategyPluginRegistry creates a new strategy plugin registry
func NewStrategyPluginRegistry(logger *zap.Logger) *StrategyPluginRegistry {
	return &StrategyPluginRegistry{
		logger:  logger,
		plugins: make(map[string]StrategyPlugin),
		fileMap: make(map[string]string),
	}
}

// RegisterPlugin registers a strategy plugin
func (r *StrategyPluginRegistry) RegisterPlugin(strategyType string, plugin StrategyPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.plugins[strategyType] = plugin
	
	// If the plugin has a file path, map it
	if wrapper, ok := plugin.(*StrategyPluginWrapper); ok {
		r.fileMap[wrapper.FilePath] = strategyType
	}
	
	r.logger.Info("Registered strategy plugin",
		zap.String("strategy_type", strategyType))
}

// UnregisterPlugin unregisters a strategy plugin
func (r *StrategyPluginRegistry) UnregisterPlugin(strategyType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Find the file path if it exists
	var filePath string
	for path, sType := range r.fileMap {
		if sType == strategyType {
			filePath = path
			break
		}
	}
	
	// Remove from plugins
	delete(r.plugins, strategyType)
	
	// Remove from file map if found
	if filePath != "" {
		delete(r.fileMap, filePath)
	}
	
	r.logger.Info("Unregistered strategy plugin",
		zap.String("strategy_type", strategyType))
}

// UnregisterPluginByFile unregisters a strategy plugin by file path
func (r *StrategyPluginRegistry) UnregisterPluginByFile(filePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Find the strategy type
	strategyType, ok := r.fileMap[filePath]
	if !ok {
		return
	}
	
	// Remove from plugins
	delete(r.plugins, strategyType)
	
	// Remove from file map
	delete(r.fileMap, filePath)
	
	r.logger.Info("Unregistered strategy plugin by file",
		zap.String("file", filePath),
		zap.String("strategy_type", strategyType))
}

// GetPlugin gets a strategy plugin by type
func (r *StrategyPluginRegistry) GetPlugin(strategyType string) (StrategyPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugin, ok := r.plugins[strategyType]
	return plugin, ok
}

// CreateStrategy creates a strategy instance from a plugin
func (r *StrategyPluginRegistry) CreateStrategy(strategyType string, config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error) {
	r.mu.RLock()
	plugin, ok := r.plugins[strategyType]
	r.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("strategy plugin not found: %s", strategyType)
	}
	
	return plugin.CreateStrategy(config, logger)
}

// ListPlugins lists all registered plugins
func (r *StrategyPluginRegistry) ListPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugins := make([]string, 0, len(r.plugins))
	for strategyType := range r.plugins {
		plugins = append(plugins, strategyType)
	}
	
	return plugins
}

// GetPluginInfo gets information about a plugin
func (r *StrategyPluginRegistry) GetPluginInfo(strategyType string) (*PluginInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugin, ok := r.plugins[strategyType]
	if !ok {
		return nil, false
	}
	
	// If it's a wrapper, return the info
	if wrapper, ok := plugin.(*StrategyPluginWrapper); ok {
		return wrapper.Info, true
	}
	
	return nil, false
}

// GetPluginByFile gets a plugin by file path
func (r *StrategyPluginRegistry) GetPluginByFile(filePath string) (StrategyPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	strategyType, ok := r.fileMap[filePath]
	if !ok {
		return nil, false
	}
	
	plugin, ok := r.plugins[strategyType]
	return plugin, ok
}

// GetPluginByFile gets a plugin wrapper by file path without bool return
// This is a convenience method for cleanup operations
func (r *StrategyPluginRegistry) GetPluginByFile(filePath string) *StrategyPluginWrapper {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	strategyType, ok := r.fileMap[filePath]
	if !ok {
		return nil
	}
	
	plugin, ok := r.plugins[strategyType]
	if !ok {
		return nil
	}
	
	// If it's a wrapper, return it
	if wrapper, ok := plugin.(*StrategyPluginWrapper); ok {
		return wrapper
	}
	
	return nil
}
