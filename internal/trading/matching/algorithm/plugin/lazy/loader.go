package lazy

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching/algorithm/plugin"
	"go.uber.org/zap"
)

// LazyPluginLoader is a lazy-loaded wrapper for the matching algorithm plugin loader
type LazyPluginLoader struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Component name prefix
	componentNamePrefix string
	
	// Configuration
	config plugin.LoaderConfig
	
	// Logger
	logger *zap.Logger
	
	// Lock manager for thread safety
	lockManager *coordination.LockManager
}

// NewLazyPluginLoader creates a new lazy-loaded plugin loader
func NewLazyPluginLoader(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	config plugin.LoaderConfig,
	logger *zap.Logger,
) (*LazyPluginLoader, error) {
	componentNamePrefix := "matching-algorithm-plugin-"
	
	// Register the lock for plugin operations
	lockManager.RegisterLock("matching-algorithm-plugins", &sync.Mutex{})
	
	return &LazyPluginLoader{
		coordinator:         coordinator,
		componentNamePrefix: componentNamePrefix,
		config:              config,
		logger:              logger,
		lockManager:         lockManager,
	}, nil
}

// LoadPlugin loads a plugin
func (l *LazyPluginLoader) LoadPlugin(
	ctx context.Context,
	pluginName string,
) (matching.MatchingAlgorithm, error) {
	componentName := l.componentNamePrefix + pluginName
	
	// Check if the component is already registered
	_, err := l.coordinator.GetComponentInfo(componentName)
	if err != nil {
		// Component not registered, register it
		err = l.registerPlugin(ctx, pluginName, componentName)
		if err != nil {
			return nil, err
		}
	}
	
	// Get the component
	algorithmInterface, err := l.coordinator.GetComponent(ctx, componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual algorithm type
	algorithm, ok := algorithmInterface.(matching.MatchingAlgorithm)
	if !ok {
		return nil, fmt.Errorf("invalid algorithm type for plugin %s", pluginName)
	}
	
	return algorithm, nil
}

// registerPlugin registers a plugin with the coordinator
func (l *LazyPluginLoader) registerPlugin(
	ctx context.Context,
	pluginName string,
	componentName string,
) error {
	// Acquire the lock to prevent concurrent plugin loading
	err := l.lockManager.AcquireLock("matching-algorithm-plugins", "plugin-loader")
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer l.lockManager.ReleaseLock("matching-algorithm-plugins", "plugin-loader")
	
	// Create the provider function
	providerFn := func(log *zap.Logger) (interface{}, error) {
		// Create a regular plugin loader
		loader := plugin.NewLoader(l.config, log)
		
		// Load the plugin
		return loader.LoadPlugin(ctx, pluginName)
	}
	
	// Create the lazy provider
	provider := lazy.NewEnhancedLazyProvider(
		componentName,
		providerFn,
		l.logger,
		nil, // Metrics will be handled by the coordinator
		lazy.WithMemoryEstimate(50*1024*1024), // 50MB estimate
		lazy.WithTimeout(30*time.Second),
		lazy.WithPriority(30), // Medium priority
	)
	
	// Register with the coordinator
	return l.coordinator.RegisterComponent(
		componentName,
		"matching-algorithm",
		provider,
		[]string{}, // No dependencies
	)
}

// UnloadPlugin unloads a plugin
func (l *LazyPluginLoader) UnloadPlugin(
	ctx context.Context,
	pluginName string,
) error {
	componentName := l.componentNamePrefix + pluginName
	
	// Shutdown the component
	return l.coordinator.ShutdownComponent(ctx, componentName)
}

// ListLoadedPlugins lists loaded plugins
func (l *LazyPluginLoader) ListLoadedPlugins() []string {
	components := l.coordinator.ListComponents()
	
	plugins := make([]string, 0)
	for _, component := range components {
		if component.Type == "matching-algorithm" {
			// Extract plugin name from component name
			pluginName := component.Name[len(l.componentNamePrefix):]
			plugins = append(plugins, pluginName)
		}
	}
	
	return plugins
}

// GetPluginInfo gets information about a plugin
func (l *LazyPluginLoader) GetPluginInfo(
	ctx context.Context,
	pluginName string,
) (*plugin.PluginInfo, error) {
	// Load the plugin
	algorithm, err := l.LoadPlugin(ctx, pluginName)
	if err != nil {
		return nil, err
	}
	
	// Get the plugin info
	pluginAlgorithm, ok := algorithm.(plugin.MatchingAlgorithmPlugin)
	if !ok {
		return nil, fmt.Errorf("invalid plugin type for plugin %s", pluginName)
	}
	
	return pluginAlgorithm.GetPluginInfo(), nil
}

