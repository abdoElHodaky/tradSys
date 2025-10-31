// Package strategies provides impact calculation strategy implementations
package strategies

import (
	"math"
	
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// LogImpactCalculator implements logarithmic market impact calculation
// This model is suitable for very large orders where impact grows very slowly
type LogImpactCalculator struct {
	liquidityFactor float64
}

// NewLogImpactCalculator creates a new logarithmic impact calculator
func NewLogImpactCalculator(liquidityFactor float64) interfaces.ImpactCalculator {
	return &LogImpactCalculator{
		liquidityFactor: liquidityFactor,
	}
}

// CalculateImpact calculates market impact using logarithmic model
func (c *LogImpactCalculator) CalculateImpact(order *types.Order, avgTradeSize float64) float64 {
	if avgTradeSize <= 0 {
		return 0
	}
	ratio := order.Quantity / avgTradeSize
	if ratio <= 1 {
		// For small orders, use linear approximation to avoid negative log
		return c.liquidityFactor * ratio
	}
	return c.liquidityFactor * math.Log(ratio)
}

// GetModelName returns the model name
func (c *LogImpactCalculator) GetModelName() string {
	return "log"
}

// SetLiquidityFactor sets the liquidity factor
func (c *LogImpactCalculator) SetLiquidityFactor(factor float64) {
	c.liquidityFactor = factor
}

// GetLiquidityFactor returns the current liquidity factor
func (c *LogImpactCalculator) GetLiquidityFactor() float64 {
	return c.liquidityFactor
}
