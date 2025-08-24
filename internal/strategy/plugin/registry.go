package plugin

import (
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// PluginRegistry manages strategy plugins
type PluginRegistry struct {
	loader  *PluginLoader
	logger  *zap.Logger
	factory strategy.StrategyFactory
	mu      sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(loader *PluginLoader, factory strategy.StrategyFactory, logger *zap.Logger) *PluginRegistry {
	return &PluginRegistry{
		loader:  loader,
		logger:  logger,
		factory: factory,
	}
}

// RegisterPlugins registers all available plugins with the strategy factory
func (r *PluginRegistry) RegisterPlugins() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load plugins
	if err := r.loader.LoadPlugins(); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Register each plugin with the strategy factory
	for _, info := range r.loader.GetAvailablePlugins() {
		plugin, _ := r.loader.GetPlugin(info.StrategyType)

		// Register the plugin with the strategy factory
		if factory, ok := r.factory.(interface {
			RegisterStrategyType(string, func(strategy.StrategyConfig) (strategy.Strategy, error))
		}); ok {
			factory.RegisterStrategyType(info.StrategyType, func(config strategy.StrategyConfig) (strategy.Strategy, error) {
				return plugin.CreateStrategy(config, r.logger)
			})

			r.logger.Info("Registered plugin strategy type",
				zap.String("strategy_type", info.StrategyType),
				zap.String("plugin", info.Name))
		} else {
			r.logger.Warn("Strategy factory does not support registering strategy types")
		}
	}

	return nil
}

// GetAvailablePlugins returns a list of available plugins
func (r *PluginRegistry) GetAvailablePlugins() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.loader.GetAvailablePlugins()
}

// CreateStrategy creates a strategy from a plugin
func (r *PluginRegistry) CreateStrategy(config strategy.StrategyConfig) (strategy.Strategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.loader.GetPlugin(config.Type)
	if !ok {
		return nil, fmt.Errorf("plugin not found for strategy type: %s", config.Type)
	}

	return plugin.CreateStrategy(config, r.logger)
}

