// Package interfaces provides core interfaces for TradSys components
package interfaces

import "github.com/abdoElHodaky/tradSys/internal/trading/types"

// ImpactCalculator defines the interface for market impact calculation strategies
type ImpactCalculator interface {
	// CalculateImpact calculates the market impact for a given order
	// Returns the impact factor that should be applied to the order
	CalculateImpact(order *types.Order, avgTradeSize float64) float64
	
	// GetModelName returns the name of the impact model
	GetModelName() string
	
	// SetLiquidityFactor sets the liquidity factor for the calculation
	SetLiquidityFactor(factor float64)
	
	// GetLiquidityFactor returns the current liquidity factor
	GetLiquidityFactor() float64
}

// ImpactCalculatorFactory creates impact calculators based on model name
type ImpactCalculatorFactory interface {
	// CreateCalculator creates an impact calculator for the specified model
	CreateCalculator(modelName string, liquidityFactor float64) (ImpactCalculator, error)
	
	// GetAvailableModels returns a list of available impact models
	GetAvailableModels() []string
	
	// RegisterModel registers a new impact calculation model
	RegisterModel(modelName string, creator func(liquidityFactor float64) ImpactCalculator) error
}
