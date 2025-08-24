package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/abdoElHodaky/tradSys/internal/plugin"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/zap"
)

// Registry is a registry for indicator plugins
type Registry struct {
	logger         *zap.Logger
	plugins        map[string]IndicatorPlugin
	pluginRegistry *plugin.EnhancedPluginRegistry
	mu             sync.RWMutex
	coreVersion    string
}

// NewRegistry creates a new registry
func NewRegistry(logger *zap.Logger, coreVersion string) *Registry {
	return &Registry{
		logger:         logger,
		plugins:        make(map[string]IndicatorPlugin),
		pluginRegistry: plugin.NewEnhancedPluginRegistry(logger, coreVersion),
		coreVersion:    coreVersion,
	}
}

// RegisterPlugin registers a plugin
func (r *Registry) RegisterPlugin(plugin IndicatorPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	info := plugin.GetPluginInfo()
	
	// Check if plugin already exists
	if _, ok := r.plugins[info.IndicatorType]; ok {
		return fmt.Errorf("plugin already registered for indicator type: %s", info.IndicatorType)
	}
	
	// Validate core version compatibility
	if err := r.validateCoreVersionCompatibility(info); err != nil {
		return fmt.Errorf("core version compatibility check failed: %w", err)
	}
	
	// Register plugin
	r.plugins[info.IndicatorType] = plugin
	
	// Register with plugin registry
	err := r.pluginRegistry.RegisterPlugin(
		"indicator",
		info.IndicatorType,
		plugin,
		&plugin.PluginInfo{
			Name:           info.Name,
			Version:        info.Version,
			Author:         info.Author,
			Description:    info.Description,
			Type:           "indicator",
			MinCoreVersion: info.MinCoreVersion,
			MaxCoreVersion: info.MaxCoreVersion,
			Dependencies:   []plugin.PluginDependency{},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register with plugin registry: %w", err)
	}
	
	r.logger.Info("Registered indicator plugin",
		zap.String("indicator_type", info.IndicatorType),
		zap.String("name", info.Name),
		zap.String("version", info.Version))
	
	return nil
}

// GetPlugin gets a plugin by indicator type
func (r *Registry) GetPlugin(indicatorType string) (IndicatorPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugin, ok := r.plugins[indicatorType]
	if !ok {
		return nil, fmt.Errorf("plugin not found for indicator type: %s", indicatorType)
	}
	
	return plugin, nil
}

// ListPlugins lists all plugins
func (r *Registry) ListPlugins() []IndicatorPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugins := make([]IndicatorPlugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	
	return plugins
}

// UnregisterPlugin unregisters a plugin
func (r *Registry) UnregisterPlugin(indicatorType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if plugin exists
	if _, ok := r.plugins[indicatorType]; !ok {
		return fmt.Errorf("plugin not found for indicator type: %s", indicatorType)
	}
	
	// Unregister from plugin registry
	err := r.pluginRegistry.UnregisterPlugin("indicator", indicatorType)
	if err != nil {
		return fmt.Errorf("failed to unregister from plugin registry: %w", err)
	}
	
	// Remove from plugins map
	delete(r.plugins, indicatorType)
	
	r.logger.Info("Unregistered indicator plugin",
		zap.String("indicator_type", indicatorType))
	
	return nil
}

// CreateIndicator creates an indicator
func (r *Registry) CreateIndicator(
	indicatorType string,
	config market_data.IndicatorConfig,
	logger *zap.Logger,
) (market_data.Indicator, error) {
	plugin, err := r.GetPlugin(indicatorType)
	if err != nil {
		return nil, err
	}
	
	return plugin.CreateIndicator(config, logger)
}

// Initialize initializes all plugins
func (r *Registry) Initialize(ctx context.Context) error {
	r.mu.RLock()
	plugins := make([]IndicatorPlugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()
	
	// Initialize plugins
	for _, plugin := range plugins {
		info := plugin.GetPluginInfo()
		r.logger.Info("Initializing indicator plugin",
			zap.String("indicator_type", info.IndicatorType),
			zap.String("name", info.Name))
		
		if err := plugin.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", info.Name, err)
		}
	}
	
	return nil
}

// Shutdown shuts down all plugins
func (r *Registry) Shutdown(ctx context.Context) error {
	r.mu.RLock()
	plugins := make([]IndicatorPlugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()
	
	// Shutdown plugins
	for _, plugin := range plugins {
		info := plugin.GetPluginInfo()
		r.logger.Info("Shutting down indicator plugin",
			zap.String("indicator_type", info.IndicatorType),
			zap.String("name", info.Name))
		
		if err := plugin.Shutdown(ctx); err != nil {
			r.logger.Error("Failed to shutdown plugin",
				zap.String("indicator_type", info.IndicatorType),
				zap.String("name", info.Name),
				zap.Error(err))
			// Continue shutting down other plugins
		}
	}
	
	return nil
}

// validateCoreVersionCompatibility validates that a plugin is compatible with the core version
func (r *Registry) validateCoreVersionCompatibility(info *PluginInfo) error {
	// If no constraints are specified, assume compatibility
	if info.MinCoreVersion == "" && info.MaxCoreVersion == "" {
		return nil
	}
	
	// Parse core version
	coreVer, err := semver.NewVersion(r.coreVersion)
	if err != nil {
		return fmt.Errorf("invalid core version: %w", err)
	}
	
	// Check minimum core version
	if info.MinCoreVersion != "" {
		minVer, err := semver.NewVersion(info.MinCoreVersion)
		if err != nil {
			return fmt.Errorf("invalid minimum core version: %w", err)
		}
		
		if coreVer.LessThan(minVer) {
			return fmt.Errorf("core version %s is less than minimum required version %s",
				r.coreVersion, info.MinCoreVersion)
		}
	}
	
	// Check maximum core version
	if info.MaxCoreVersion != "" {
		maxVer, err := semver.NewVersion(info.MaxCoreVersion)
		if err != nil {
			return fmt.Errorf("invalid maximum core version: %w", err)
		}
		
		if coreVer.GreaterThan(maxVer) {
			return fmt.Errorf("core version %s is greater than maximum supported version %s",
				r.coreVersion, info.MaxCoreVersion)
		}
	}
	
	return nil
}

// SetCoreVersion sets the core version
func (r *Registry) SetCoreVersion(version string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.coreVersion = version
	r.pluginRegistry.SetCoreVersion(version)
}

// GetCoreVersion gets the core version
func (r *Registry) GetCoreVersion() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.coreVersion
}

// ValidateAllPlugins validates all plugins
func (r *Registry) ValidateAllPlugins() error {
	return r.pluginRegistry.ValidateAllPlugins()
}

