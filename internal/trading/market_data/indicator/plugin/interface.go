package plugin

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/zap"
)

// PluginInfo contains information about a plugin
type PluginInfo struct {
	// Name is the name of the plugin
	Name string `json:"name"`
	
	// Version is the version of the plugin
	Version string `json:"version"`
	
	// Author is the author of the plugin
	Author string `json:"author"`
	
	// Description is a description of the plugin
	Description string `json:"description"`
	
	// IndicatorType is the type of indicator
	IndicatorType string `json:"indicator_type"`
	
	// APIVersion is the version of the indicator API
	APIVersion string `json:"api_version"`
	
	// MinCoreVersion is the minimum core version required by this plugin
	MinCoreVersion string `json:"min_core_version"`
	
	// MaxCoreVersion is the maximum core version supported by this plugin
	MaxCoreVersion string `json:"max_core_version"`
	
	// Dependencies is a list of other plugins that this plugin depends on
	Dependencies []string `json:"dependencies"`
}

// IndicatorPlugin is the interface for indicator plugins
type IndicatorPlugin interface {
	// GetPluginInfo returns information about the plugin
	GetPluginInfo() *PluginInfo
	
	// CreateIndicator creates an indicator
	CreateIndicator(config market_data.IndicatorConfig, logger *zap.Logger) (market_data.Indicator, error)
	
	// Initialize initializes the plugin
	Initialize(ctx context.Context) error
	
	// Shutdown shuts down the plugin
	Shutdown(ctx context.Context) error
}

// IndicatorPluginFactory is a factory for creating indicator plugins
type IndicatorPluginFactory interface {
	// CreatePlugin creates a plugin
	CreatePlugin() (IndicatorPlugin, error)
}

// IndicatorPluginRegistry is a registry for indicator plugins
type IndicatorPluginRegistry interface {
	// RegisterPlugin registers a plugin
	RegisterPlugin(plugin IndicatorPlugin) error
	
	// GetPlugin gets a plugin by indicator type
	GetPlugin(indicatorType string) (IndicatorPlugin, error)
	
	// ListPlugins lists all plugins
	ListPlugins() []IndicatorPlugin
	
	// UnregisterPlugin unregisters a plugin
	UnregisterPlugin(indicatorType string) error
}

// IndicatorPluginLoader is a loader for indicator plugins
type IndicatorPluginLoader interface {
	// LoadPlugin loads a plugin from a file
	LoadPlugin(filePath string) (IndicatorPlugin, error)
	
	// LoadPlugins loads all plugins from a directory
	LoadPlugins(dirPath string) ([]IndicatorPlugin, error)
}

// IndicatorPluginManager manages indicator plugins
type IndicatorPluginManager interface {
	// RegisterPlugin registers a plugin
	RegisterPlugin(plugin IndicatorPlugin) error
	
	// GetPlugin gets a plugin by indicator type
	GetPlugin(indicatorType string) (IndicatorPlugin, error)
	
	// ListPlugins lists all plugins
	ListPlugins() []IndicatorPlugin
	
	// UnregisterPlugin unregisters a plugin
	UnregisterPlugin(indicatorType string) error
	
	// LoadPlugin loads a plugin from a file
	LoadPlugin(filePath string) (IndicatorPlugin, error)
	
	// LoadPlugins loads all plugins from a directory
	LoadPlugins(dirPath string) ([]IndicatorPlugin, error)
	
	// CreateIndicator creates an indicator
	CreateIndicator(indicatorType string, config market_data.IndicatorConfig, logger *zap.Logger) (market_data.Indicator, error)
	
	// Initialize initializes all plugins
	Initialize(ctx context.Context) error
	
	// Shutdown shuts down all plugins
	Shutdown(ctx context.Context) error
}

