package plugin

import (
	"github.com/abdoElHodaky/tradSys/internal/peerjs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ModuleParams contains parameters for the plugin module
type ModuleParams struct {
	fx.In

	Logger     *zap.Logger
	PeerServer *peerjs.PeerServer `optional:"true"`
}

// Module provides the PeerJS plugin components
var Module = fx.Options(
	// Provide the plugin loader
	fx.Provide(providePluginLoader),
	
	// Register lifecycle hooks
	fx.Invoke(registerHooks),
)

// providePluginLoader provides a plugin loader
func providePluginLoader(params ModuleParams) *PluginLoader {
	// Get the plugin directory from environment or use a default
	pluginDir := "/etc/tradsys/peerjs/plugins"
	
	return NewPluginLoader(pluginDir, params.PeerServer, params.Logger)
}

// registerHooks registers lifecycle hooks for the plugin components
func registerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	loader *PluginLoader,
) {
	logger.Info("Registering PeerJS plugin component hooks")
	
	// Register plugins when the application starts
	if err := loader.LoadPlugins(); err != nil {
		logger.Error("Failed to load PeerJS plugins", zap.Error(err))
	}
}

