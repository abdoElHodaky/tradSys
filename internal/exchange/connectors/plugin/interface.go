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
}

// PluginSymbols defines the symbols that must be exported by a plugin
const (
	// PluginInfoSymbol is the name of the exported plugin info symbol
	PluginInfoSymbol = "PluginInfo"
	
	// CreateConnectorSymbol is the name of the exported function to create a connector
	CreateConnectorSymbol = "CreateConnector"
)

