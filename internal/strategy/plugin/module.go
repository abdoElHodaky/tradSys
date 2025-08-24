package plugin

import (
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ModuleParams contains parameters for the plugin module
type ModuleParams struct {
	fx.In

	Logger        *zap.Logger
	StrategyFactory strategy.StrategyFactory `optional:"true"`
}

// Module provides the plugin components
var Module = fx.Options(
	// Provide the plugin loader
	fx.Provide(providePluginLoader),
	
	// Provide the plugin registry
	fx.Provide(providePluginRegistry),
	
	// Register lifecycle hooks
	fx.Invoke(registerHooks),
)

// providePluginLoader provides a plugin loader
func providePluginLoader(params ModuleParams) *PluginLoader {
	// Get the plugin directory from environment or use a default
	pluginDir := "/etc/tradsys/plugins"
	
	return NewPluginLoader(pluginDir, params.Logger)
}

// providePluginRegistry provides a plugin registry
func providePluginRegistry(loader *PluginLoader, params ModuleParams) *PluginRegistry {
	return NewPluginRegistry(loader, params.StrategyFactory, params.Logger)
}

// registerHooks registers lifecycle hooks for the plugin components
func registerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	registry *PluginRegistry,
) {
	logger.Info("Registering plugin component hooks")
	
	// Register plugins when the application starts
	if err := registry.RegisterPlugins(); err != nil {
		logger.Error("Failed to register plugins", zap.Error(err))
	}
}

