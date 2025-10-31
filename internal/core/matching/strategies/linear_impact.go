// Package strategies provides impact calculation strategy implementations
package strategies

import (
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// LinearImpactCalculator implements linear market impact calculation
type LinearImpactCalculator struct {
	liquidityFactor float64
}

// NewLinearImpactCalculator creates a new linear impact calculator
func NewLinearImpactCalculator(liquidityFactor float64) interfaces.ImpactCalculator {
	return &LinearImpactCalculator{
		liquidityFactor: liquidityFactor,
	}
}

// CalculateImpact calculates market impact using linear model
func (c *LinearImpactCalculator) CalculateImpact(order *types.Order, avgTradeSize float64) float64 {
	if avgTradeSize <= 0 {
		return 0
	}
	return c.liquidityFactor * (order.Quantity / avgTradeSize)
}

// GetModelName returns the model name
func (c *LinearImpactCalculator) GetModelName() string {
	return "linear"
}

// SetLiquidityFactor sets the liquidity factor
func (c *LinearImpactCalculator) SetLiquidityFactor(factor float64) {
	c.liquidityFactor = factor
}

// GetLiquidityFactor returns the current liquidity factor
func (c *LinearImpactCalculator) GetLiquidityFactor() float64 {
	return c.liquidityFactor
}
