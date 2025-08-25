package plugin

import (
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// Plugin symbol names
const (
	PluginInfoSymbol     = "PluginInfo"
	CreateStrategySymbol = "CreateStrategy"
	CleanupSymbol        = "Cleanup"
)

// PluginInfo contains information about a plugin
type PluginInfo struct {
	Name         string
	Version      string
	StrategyType string
	Description  string
	Author       string
}

// StrategyPlugin is the interface for strategy plugins
type StrategyPlugin interface {
	// GetStrategyType returns the type of strategy provided by this plugin
	GetStrategyType() string
	
	// CreateStrategy creates a strategy instance
	CreateStrategy(config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error)
}
