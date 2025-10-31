// Package strategies provides impact calculation strategy implementations
package strategies

import (
	"math"
	
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// SqrtImpactCalculator implements square root market impact calculation
// This model is more realistic for large orders as impact grows slower
type SqrtImpactCalculator struct {
	liquidityFactor float64
}

// NewSqrtImpactCalculator creates a new square root impact calculator
func NewSqrtImpactCalculator(liquidityFactor float64) interfaces.ImpactCalculator {
	return &SqrtImpactCalculator{
		liquidityFactor: liquidityFactor,
	}
}

// CalculateImpact calculates market impact using square root model
func (c *SqrtImpactCalculator) CalculateImpact(order *types.Order, avgTradeSize float64) float64 {
	if avgTradeSize <= 0 {
		return 0
	}
	ratio := order.Quantity / avgTradeSize
	return c.liquidityFactor * math.Sqrt(ratio)
}

// GetModelName returns the model name
func (c *SqrtImpactCalculator) GetModelName() string {
	return "sqrt"
}

// SetLiquidityFactor sets the liquidity factor
func (c *SqrtImpactCalculator) SetLiquidityFactor(factor float64) {
	c.liquidityFactor = factor
}

// GetLiquidityFactor returns the current liquidity factor
func (c *SqrtImpactCalculator) GetLiquidityFactor() float64 {
	return c.liquidityFactor
}
