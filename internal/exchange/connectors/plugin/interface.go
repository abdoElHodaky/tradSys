package plugin

import (
	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	"go.uber.org/zap"
)

// ExchangeConnectorPlugin defines the interface for an exchange connector plugin
type ExchangeConnectorPlugin interface {
	// GetExchangeName returns the name of the exchange
	GetExchangeName() string
	
	// CreateConnector creates an exchange connector
	CreateConnector(config connectors.ExchangeConfig, logger *zap.Logger) (connectors.ExchangeConnector, error)
}

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
	
	// APIVersion is the API version the plugin is compatible with
	APIVersion string `json:"api_version"`
	
	// MinCoreVersion is the minimum core version the plugin is compatible with
	MinCoreVersion string `json:"min_core_version"`
	
	// MaxCoreVersion is the maximum core version the plugin is compatible with
	// An empty string means compatible with any future version
	MaxCoreVersion string `json:"max_core_version"`
	
	// Dependencies is a list of other plugins this plugin depends on
	Dependencies []string `json:"dependencies"`
}

// PluginSymbols defines the symbols that must be exported by a plugin
const (
	// PluginInfoSymbol is the name of the exported plugin info symbol
	PluginInfoSymbol = "PluginInfo"
	
	// CreateConnectorSymbol is the name of the exported function to create a connector
	CreateConnectorSymbol = "CreateConnector"
	
	// InitializePluginSymbol is the name of the exported function to initialize the plugin
	InitializePluginSymbol = "InitializePlugin"
	
	// ShutdownPluginSymbol is the name of the exported function to shutdown the plugin
	ShutdownPluginSymbol = "ShutdownPlugin"
)

// CoreVersion is the current core version of the application
// This should be updated when making breaking changes to the plugin API
const CoreVersion = "1.0.0"

