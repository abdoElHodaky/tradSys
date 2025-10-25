package risk

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"go.uber.org/zap"
)

// Calculator handles risk calculations and metrics
type Calculator struct {
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewCalculator creates a new risk calculator
func NewCalculator(logger *zap.Logger) *Calculator {
	return &Calculator{
		logger: logger,
	}
}

// CalculatePositionRisk calculates risk metrics for a position
func (c *Calculator) CalculatePositionRisk(ctx context.Context, position *Position, currentPrice float64) (*PositionRiskMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if position == nil {
		return nil, ErrInvalidPosition
	}

	metrics := &PositionRiskMetrics{
		Symbol:        position.Symbol,
		UserID:        position.UserID,
		Quantity:      position.Quantity,
		AveragePrice:  position.AveragePrice,
		CurrentPrice:  currentPrice,
		CalculatedAt:  time.Now(),
	}

	// Calculate unrealized P&L
	if position.Quantity != 0 {
		metrics.UnrealizedPnL = position.Quantity * (currentPrice - position.AveragePrice)
		metrics.UnrealizedPnLPercent = (currentPrice - position.AveragePrice) / position.AveragePrice * 100
	}

	// Calculate market value
	metrics.MarketValue = math.Abs(position.Quantity) * currentPrice

	// Calculate risk metrics
	metrics.VaR95 = c.calculateVaR(position, currentPrice, 0.95)
	metrics.VaR99 = c.calculateVaR(position, currentPrice, 0.99)
	metrics.ExpectedShortfall = c.calculateExpectedShortfall(position, currentPrice, 0.95)

	// Calculate Greeks (for options positions)
	if position.InstrumentType == "option" {
		metrics.Delta = c.calculateDelta(position, currentPrice)
		metrics.Gamma = c.calculateGamma(position, currentPrice)
		metrics.Theta = c.calculateTheta(position, currentPrice)
		metrics.Vega = c.calculateVega(position, currentPrice)
	}

	// Determine risk level
	metrics.RiskLevel = c.determineRiskLevel(metrics)

	c.logger.Debug("Position risk calculated",
		zap.String("symbol", position.Symbol),
		zap.String("user_id", position.UserID),
		zap.Float64("unrealized_pnl", metrics.UnrealizedPnL),
		zap.Float64("var_95", metrics.VaR95),
		zap.String("risk_level", string(metrics.RiskLevel)))

	return metrics, nil
}

// CalculateAccountRisk calculates overall account risk
func (c *Calculator) CalculateAccountRisk(ctx context.Context, userID string, positions []*Position, prices map[string]float64) (*AccountRiskMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(positions) == 0 {
		return &AccountRiskMetrics{
			UserID:       userID,
			CalculatedAt: time.Now(),
			RiskLevel:    RiskLevelLow,
		}, nil
	}

	metrics := &AccountRiskMetrics{
		UserID:       userID,
		CalculatedAt: time.Now(),
		Positions:    make([]*PositionRiskMetrics, 0, len(positions)),
	}

	var totalUnrealizedPnL float64
	var totalMarketValue float64
	var totalVaR95 float64
	var totalVaR99 float64
	var maxPositionRisk RiskLevel = RiskLevelLow

	// Calculate risk for each position
	for _, position := range positions {
		currentPrice, exists := prices[position.Symbol]
		if !exists {
			c.logger.Warn("Price not available for symbol", zap.String("symbol", position.Symbol))
			continue
		}

		positionRisk, err := c.CalculatePositionRisk(ctx, position, currentPrice)
		if err != nil {
			c.logger.Error("Failed to calculate position risk",
				zap.String("symbol", position.Symbol),
				zap.Error(err))
			continue
		}

		metrics.Positions = append(metrics.Positions, positionRisk)
		totalUnrealizedPnL += positionRisk.UnrealizedPnL
		totalMarketValue += positionRisk.MarketValue
		totalVaR95 += positionRisk.VaR95
		totalVaR99 += positionRisk.VaR99

		// Track highest risk level
		if c.compareRiskLevels(positionRisk.RiskLevel, maxPositionRisk) > 0 {
			maxPositionRisk = positionRisk.RiskLevel
		}
	}

	// Set aggregate metrics
	metrics.TotalUnrealizedPnL = totalUnrealizedPnL
	metrics.TotalMarketValue = totalMarketValue
	metrics.PortfolioVaR95 = totalVaR95
	metrics.PortfolioVaR99 = totalVaR99

	// Calculate portfolio-level metrics
	if totalMarketValue > 0 {
		metrics.TotalUnrealizedPnLPercent = totalUnrealizedPnL / totalMarketValue * 100
	}

	// Calculate concentration risk
	metrics.ConcentrationRisk = c.calculateConcentrationRisk(metrics.Positions)

	// Calculate correlation risk
	metrics.CorrelationRisk = c.calculateCorrelationRisk(metrics.Positions)

	// Determine overall account risk level
	metrics.RiskLevel = c.determineAccountRiskLevel(metrics, maxPositionRisk)

	c.logger.Debug("Account risk calculated",
		zap.String("user_id", userID),
		zap.Float64("total_unrealized_pnl", metrics.TotalUnrealizedPnL),
		zap.Float64("portfolio_var_95", metrics.PortfolioVaR95),
		zap.String("risk_level", string(metrics.RiskLevel)))

	return metrics, nil
}

// CalculateOrderRisk calculates risk for a new order
func (c *Calculator) CalculateOrderRisk(ctx context.Context, order *orders.Order, currentPosition *Position, currentPrice float64) (*OrderRiskMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if order == nil {
		return nil, ErrInvalidOrder
	}

	metrics := &OrderRiskMetrics{
		OrderID:      order.ID,
		Symbol:       order.Symbol,
		UserID:       order.UserID,
		Side:         string(order.Side),
		Quantity:     order.Quantity,
		Price:        order.Price,
		CurrentPrice: currentPrice,
		CalculatedAt: time.Now(),
	}

	// Calculate order value
	orderPrice := order.Price
	if order.Type == orders.OrderTypeMarket {
		orderPrice = currentPrice
	}
	metrics.OrderValue = order.Quantity * orderPrice

	// Calculate position impact
	if currentPosition != nil {
		metrics.CurrentPosition = currentPosition.Quantity
		
		// Calculate new position after order execution
		newQuantity := currentPosition.Quantity
		if order.Side == orders.OrderSideBuy {
			newQuantity += order.Quantity
		} else {
			newQuantity -= order.Quantity
		}
		metrics.NewPosition = newQuantity

		// Calculate position change
		metrics.PositionChange = newQuantity - currentPosition.Quantity
		metrics.PositionChangePercent = math.Abs(metrics.PositionChange) / math.Max(math.Abs(currentPosition.Quantity), 1) * 100
	} else {
		// New position
		if order.Side == orders.OrderSideBuy {
			metrics.NewPosition = order.Quantity
		} else {
			metrics.NewPosition = -order.Quantity
		}
		metrics.PositionChange = metrics.NewPosition
		metrics.PositionChangePercent = 100 // 100% change for new position
	}

	// Calculate leverage impact
	metrics.LeverageImpact = c.calculateLeverageImpact(order, currentPosition)

	// Calculate margin requirement
	metrics.MarginRequirement = c.calculateMarginRequirement(order, currentPrice)

	// Calculate maximum loss potential
	metrics.MaxLossPotential = c.calculateMaxLossPotential(order, currentPrice)

	// Determine risk level
	metrics.RiskLevel = c.determineOrderRiskLevel(metrics)

	c.logger.Debug("Order risk calculated",
		zap.String("order_id", order.ID),
		zap.String("symbol", order.Symbol),
		zap.Float64("order_value", metrics.OrderValue),
		zap.Float64("max_loss_potential", metrics.MaxLossPotential),
		zap.String("risk_level", string(metrics.RiskLevel)))

	return metrics, nil
}

// calculateVaR calculates Value at Risk for a position
func (c *Calculator) calculateVaR(position *Position, currentPrice float64, confidence float64) float64 {
	// Simplified VaR calculation using historical volatility
	// In production, this would use more sophisticated models
	
	volatility := c.getHistoricalVolatility(position.Symbol)
	if volatility == 0 {
		volatility = 0.02 // Default 2% daily volatility
	}

	// Z-score for confidence level
	var zScore float64
	switch confidence {
	case 0.95:
		zScore = 1.645
	case 0.99:
		zScore = 2.326
	default:
		zScore = 1.645
	}

	marketValue := math.Abs(position.Quantity) * currentPrice
	return marketValue * volatility * zScore
}

// calculateExpectedShortfall calculates Expected Shortfall (Conditional VaR)
func (c *Calculator) calculateExpectedShortfall(position *Position, currentPrice float64, confidence float64) float64 {
	// Simplified ES calculation
	var95 := c.calculateVaR(position, currentPrice, confidence)
	return var95 * 1.3 // Approximate ES as 1.3 times VaR for normal distribution
}

// calculateDelta calculates option delta
func (c *Calculator) calculateDelta(position *Position, currentPrice float64) float64 {
	// Simplified delta calculation - in production would use Black-Scholes
	if position.InstrumentType != "option" {
		return 0
	}
	// Placeholder implementation
	return 0.5
}

// calculateGamma calculates option gamma
func (c *Calculator) calculateGamma(position *Position, currentPrice float64) float64 {
	// Simplified gamma calculation
	if position.InstrumentType != "option" {
		return 0
	}
	// Placeholder implementation
	return 0.01
}

// calculateTheta calculates option theta
func (c *Calculator) calculateTheta(position *Position, currentPrice float64) float64 {
	// Simplified theta calculation
	if position.InstrumentType != "option" {
		return 0
	}
	// Placeholder implementation
	return -0.05
}

// calculateVega calculates option vega
func (c *Calculator) calculateVega(position *Position, currentPrice float64) float64 {
	// Simplified vega calculation
	if position.InstrumentType != "option" {
		return 0
	}
	// Placeholder implementation
	return 0.1
}

// determineRiskLevel determines risk level based on metrics
func (c *Calculator) determineRiskLevel(metrics *PositionRiskMetrics) RiskLevel {
	// Risk level determination logic
	if math.Abs(metrics.UnrealizedPnLPercent) > 20 {
		return RiskLevelCritical
	} else if math.Abs(metrics.UnrealizedPnLPercent) > 10 {
		return RiskLevelHigh
	} else if math.Abs(metrics.UnrealizedPnLPercent) > 5 {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// determineAccountRiskLevel determines account-level risk
func (c *Calculator) determineAccountRiskLevel(metrics *AccountRiskMetrics, maxPositionRisk RiskLevel) RiskLevel {
	// Account risk based on total P&L and concentration
	if math.Abs(metrics.TotalUnrealizedPnLPercent) > 15 || metrics.ConcentrationRisk > 0.8 {
		return RiskLevelCritical
	} else if math.Abs(metrics.TotalUnrealizedPnLPercent) > 8 || metrics.ConcentrationRisk > 0.6 {
		return RiskLevelHigh
	} else if math.Abs(metrics.TotalUnrealizedPnLPercent) > 4 || metrics.ConcentrationRisk > 0.4 {
		return RiskLevelMedium
	}
	
	// Consider maximum position risk
	if maxPositionRisk == RiskLevelCritical {
		return RiskLevelHigh // Downgrade slightly for portfolio effect
	}
	
	return RiskLevelLow
}

// determineOrderRiskLevel determines risk level for an order
func (c *Calculator) determineOrderRiskLevel(metrics *OrderRiskMetrics) RiskLevel {
	// Order risk based on position change and potential loss
	if metrics.PositionChangePercent > 50 || metrics.MaxLossPotential > 10000 {
		return RiskLevelCritical
	} else if metrics.PositionChangePercent > 25 || metrics.MaxLossPotential > 5000 {
		return RiskLevelHigh
	} else if metrics.PositionChangePercent > 10 || metrics.MaxLossPotential > 1000 {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// calculateConcentrationRisk calculates portfolio concentration risk
func (c *Calculator) calculateConcentrationRisk(positions []*PositionRiskMetrics) float64 {
	if len(positions) == 0 {
		return 0
	}

	// Calculate Herfindahl-Hirschman Index for concentration
	var totalValue float64
	for _, pos := range positions {
		totalValue += pos.MarketValue
	}

	if totalValue == 0 {
		return 0
	}

	var hhi float64
	for _, pos := range positions {
		share := pos.MarketValue / totalValue
		hhi += share * share
	}

	return hhi
}

// calculateCorrelationRisk calculates correlation risk between positions
func (c *Calculator) calculateCorrelationRisk(positions []*PositionRiskMetrics) float64 {
	// Simplified correlation risk calculation
	// In production, would use actual correlation matrices
	if len(positions) <= 1 {
		return 0
	}

	// Assume moderate correlation for same-sector positions
	return 0.3
}

// calculateLeverageImpact calculates leverage impact of an order
func (c *Calculator) calculateLeverageImpact(order *orders.Order, currentPosition *Position) float64 {
	// Simplified leverage calculation
	orderValue := order.Quantity * order.Price
	
	if currentPosition != nil {
		currentValue := math.Abs(currentPosition.Quantity) * order.Price
		return orderValue / math.Max(currentValue, 1000) // Avoid division by zero
	}
	
	return orderValue / 1000 // Normalized impact
}

// calculateMarginRequirement calculates margin requirement for an order
func (c *Calculator) calculateMarginRequirement(order *orders.Order, currentPrice float64) float64 {
	// Simplified margin calculation - typically 10-50% of order value
	orderValue := order.Quantity * currentPrice
	marginRate := 0.2 // 20% margin requirement
	
	return orderValue * marginRate
}

// calculateMaxLossPotential calculates maximum potential loss for an order
func (c *Calculator) calculateMaxLossPotential(order *orders.Order, currentPrice float64) float64 {
	// For market orders, assume 2% slippage
	// For limit orders, calculate based on price difference
	
	if order.Type == orders.OrderTypeMarket {
		slippage := 0.02
		return order.Quantity * currentPrice * slippage
	}
	
	// For limit orders, max loss is the difference between limit and current price
	priceDiff := math.Abs(order.Price - currentPrice)
	return order.Quantity * priceDiff
}

// getHistoricalVolatility gets historical volatility for a symbol
func (c *Calculator) getHistoricalVolatility(symbol string) float64 {
	// Placeholder - in production would fetch from market data service
	volatilityMap := map[string]float64{
		"AAPL": 0.25,
		"GOOGL": 0.30,
		"TSLA": 0.45,
		"SPY": 0.15,
	}
	
	if vol, exists := volatilityMap[symbol]; exists {
		return vol
	}
	
	return 0.20 // Default volatility
}

// compareRiskLevels compares two risk levels and returns:
// -1 if a < b, 0 if a == b, 1 if a > b
func (c *Calculator) compareRiskLevels(a, b RiskLevel) int {
	levels := map[RiskLevel]int{
		RiskLevelLow:      1,
		RiskLevelMedium:   2,
		RiskLevelHigh:     3,
		RiskLevelCritical: 4,
	}
	
	levelA := levels[a]
	levelB := levels[b]
	
	if levelA < levelB {
		return -1
	} else if levelA > levelB {
		return 1
	}
	return 0
}
