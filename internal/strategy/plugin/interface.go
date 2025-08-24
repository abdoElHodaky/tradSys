package plugin

import (
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// StrategyPlugin defines the interface for a strategy plugin
type StrategyPlugin interface {
	// GetStrategyType returns the type of strategy provided by this plugin
	GetStrategyType() string
	
	// CreateStrategy creates a strategy instance
	CreateStrategy(config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error)
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
	
	// StrategyType is the type of strategy provided by this plugin
	StrategyType string `json:"strategy_type"`
}

// PluginSymbols defines the symbols that must be exported by a plugin
const (
	// PluginInfoSymbol is the name of the exported plugin info symbol
	PluginInfoSymbol = "PluginInfo"
	
	// CreateStrategySymbol is the name of the exported function to create a strategy
	CreateStrategySymbol = "CreateStrategy"
)

