package plugin

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
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
	
	// ExchangeName is the name of the exchange
	ExchangeName string `json:"exchange_name"`
	
	// APIVersion is the version of the exchange API
	APIVersion string `json:"api_version"`
	
	// MinCoreVersion is the minimum core version required by this plugin
	MinCoreVersion string `json:"min_core_version"`
	
	// MaxCoreVersion is the maximum core version supported by this plugin
	MaxCoreVersion string `json:"max_core_version"`
	
	// Dependencies is a list of other plugins that this plugin depends on
	Dependencies []string `json:"dependencies"`
}

// ExchangeConnectorPlugin is the interface for exchange connector plugins
type ExchangeConnectorPlugin interface {
	// GetPluginInfo returns information about the plugin
	GetPluginInfo() *PluginInfo
	
	// CreateConnector creates an exchange connector
	CreateConnector(config connectors.ExchangeConfig, logger *zap.Logger) (connectors.ExchangeConnector, error)
	
	// Initialize initializes the plugin
	Initialize(ctx context.Context) error
	
	// Shutdown shuts down the plugin
	Shutdown(ctx context.Context) error
}

// ExchangeConnectorPluginFactory is a factory for creating exchange connector plugins
type ExchangeConnectorPluginFactory interface {
	// CreatePlugin creates a plugin
	CreatePlugin() (ExchangeConnectorPlugin, error)
}

// ExchangeConnectorPluginRegistry is a registry for exchange connector plugins
type ExchangeConnectorPluginRegistry interface {
	// RegisterPlugin registers a plugin
	RegisterPlugin(plugin ExchangeConnectorPlugin) error
	
	// GetPlugin gets a plugin by exchange name
	GetPlugin(exchangeName string) (ExchangeConnectorPlugin, error)
	
	// ListPlugins lists all plugins
	ListPlugins() []ExchangeConnectorPlugin
	
	// UnregisterPlugin unregisters a plugin
	UnregisterPlugin(exchangeName string) error
}

// ExchangeConnectorPluginLoader is a loader for exchange connector plugins
type ExchangeConnectorPluginLoader interface {
	// LoadPlugin loads a plugin from a file
	LoadPlugin(filePath string) (ExchangeConnectorPlugin, error)
	
	// LoadPlugins loads all plugins from a directory
	LoadPlugins(dirPath string) ([]ExchangeConnectorPlugin, error)
}

// ExchangeConnectorPluginManager manages exchange connector plugins
type ExchangeConnectorPluginManager interface {
	// RegisterPlugin registers a plugin
	RegisterPlugin(plugin ExchangeConnectorPlugin) error
	
	// GetPlugin gets a plugin by exchange name
	GetPlugin(exchangeName string) (ExchangeConnectorPlugin, error)
	
	// ListPlugins lists all plugins
	ListPlugins() []ExchangeConnectorPlugin
	
	// UnregisterPlugin unregisters a plugin
	UnregisterPlugin(exchangeName string) error
	
	// LoadPlugin loads a plugin from a file
	LoadPlugin(filePath string) (ExchangeConnectorPlugin, error)
	
	// LoadPlugins loads all plugins from a directory
	LoadPlugins(dirPath string) ([]ExchangeConnectorPlugin, error)
	
	// CreateConnector creates an exchange connector
	CreateConnector(exchangeName string, config connectors.ExchangeConfig, logger *zap.Logger) (connectors.ExchangeConnector, error)
	
	// Initialize initializes all plugins
	Initialize(ctx context.Context) error
	
	// Shutdown shuts down all plugins
	Shutdown(ctx context.Context) error
}

