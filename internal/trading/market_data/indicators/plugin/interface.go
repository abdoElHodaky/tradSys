package plugin

import (
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/indicators"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
)

// IndicatorPlugin defines the interface for a technical indicator plugin
type IndicatorPlugin interface {
	// GetIndicatorName returns the name of the indicator
	GetIndicatorName() string
	
	// CreateIndicator creates a technical indicator
	CreateIndicator(params indicators.IndicatorParams, logger *zap.Logger) (indicators.Indicator, error)
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
	
	// IndicatorName is the name of the indicator
	IndicatorName string `json:"indicator_name"`
	
	// DefaultParams are the default parameters for the indicator
	DefaultParams map[string]interface{} `json:"default_params"`
	
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
	
	// CreateIndicatorSymbol is the name of the exported function to create an indicator
	CreateIndicatorSymbol = "CreateIndicator"
	
	// InitializePluginSymbol is the name of the exported function to initialize the plugin
	InitializePluginSymbol = "InitializePlugin"
	
	// ShutdownPluginSymbol is the name of the exported function to shutdown the plugin
	ShutdownPluginSymbol = "ShutdownPlugin"
)

// CoreVersion is the current core version of the application
// This should be updated when making breaking changes to the plugin API
const CoreVersion = "1.0.0"

