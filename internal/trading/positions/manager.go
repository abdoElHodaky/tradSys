package positions

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Position represents a trading position
type Position struct {
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	Quantity     float64   `json:"quantity"`     // Positive for long, negative for short
	AvgPrice     float64   `json:"avg_price"`    // Average entry price
	MarketValue  float64   `json:"market_value"` // Current market value
	UnrealizedPL float64   `json:"unrealized_pl"` // Unrealized P&L
	RealizedPL   float64   `json:"realized_pl"`   // Realized P&L
	LastUpdate   time.Time `json:"last_update"`
	OpenedAt     time.Time `json:"opened_at"`
}

// PositionUpdate represents a position update
type PositionUpdate struct {
	UserID    string  `json:"user_id"`
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	Price     float64 `json:"price"`
	Side      string  `json:"side"` // "buy" or "sell"
	TradeID   string  `json:"trade_id"`
	Timestamp time.Time `json:"timestamp"`
}

// PositionManager manages trading positions with real-time P&L calculation
type PositionManager struct {
	positions       map[string]*Position // key: userID_symbol
	marketPrices    map[string]float64   // key: symbol
	mutex           sync.RWMutex
	metrics         map[string]interface{}
	totalPositions  int64
	totalUpdates    int64
}

// NewPositionManager creates a new position manager
func NewPositionManager() *PositionManager {
	return &PositionManager{
		positions:    make(map[string]*Position),
		marketPrices: make(map[string]float64),
		metrics:      make(map[string]interface{}),
	}
}

// UpdatePosition updates a position based on a trade
func (pm *PositionManager) UpdatePosition(update *PositionUpdate) error {
	if update.UserID == "" || update.Symbol == "" {
		return fmt.Errorf("user ID and symbol are required")
	}
	
	if update.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	
	if update.Price <= 0 {
		return fmt.Errorf("price must be positive")
	}
	
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	positionKey := fmt.Sprintf("%s_%s", update.UserID, update.Symbol)
	position, exists := pm.positions[positionKey]
	
	if !exists {
		// Create new position
		position = &Position{
			UserID:     update.UserID,
			Symbol:     update.Symbol,
			Quantity:   0,
			AvgPrice:   0,
			OpenedAt:   update.Timestamp,
			LastUpdate: update.Timestamp,
		}
		pm.positions[positionKey] = position
		atomic.AddInt64(&pm.totalPositions, 1)
	}
	
	// Calculate new position based on trade side
	var quantityChange float64
	if update.Side == "buy" {
		quantityChange = update.Quantity
	} else {
		quantityChange = -update.Quantity
	}
	
	// Update position
	oldQuantity := position.Quantity
	newQuantity := oldQuantity + quantityChange
	
	// Calculate new average price
	if newQuantity != 0 {
		if (oldQuantity >= 0 && quantityChange > 0) || (oldQuantity <= 0 && quantityChange < 0) {
			// Adding to existing position
			totalValue := (oldQuantity * position.AvgPrice) + (quantityChange * update.Price)
			position.AvgPrice = totalValue / newQuantity
		} else if (oldQuantity > 0 && quantityChange < 0) || (oldQuantity < 0 && quantityChange > 0) {
			// Reducing or closing position
			if newQuantity == 0 {
				// Position closed
				realizedPL := (update.Price - position.AvgPrice) * (-quantityChange)
				if oldQuantity < 0 {
					realizedPL = -realizedPL
				}
				position.RealizedPL += realizedPL
				position.AvgPrice = 0
			} else {
				// Partial close - average price remains the same
				realizedPL := (update.Price - position.AvgPrice) * (-quantityChange)
				if oldQuantity < 0 {
					realizedPL = -realizedPL
				}
				position.RealizedPL += realizedPL
			}
		}
	}
	
	position.Quantity = newQuantity
	position.LastUpdate = update.Timestamp
	
	// Update market value and unrealized P&L
	pm.updatePositionPL(position)
	
	atomic.AddInt64(&pm.totalUpdates, 1)
	pm.updateMetrics()
	
	return nil
}

// updatePositionPL updates the market value and unrealized P&L for a position
func (pm *PositionManager) updatePositionPL(position *Position) {
	marketPrice, exists := pm.marketPrices[position.Symbol]
	if !exists {
		return
	}
	
	position.MarketValue = position.Quantity * marketPrice
	
	if position.Quantity != 0 {
		position.UnrealizedPL = (marketPrice - position.AvgPrice) * position.Quantity
	} else {
		position.UnrealizedPL = 0
	}
}

// UpdateMarketPrice updates the market price for a symbol and recalculates P&L
func (pm *PositionManager) UpdateMarketPrice(symbol string, price float64) {
	if price <= 0 {
		return
	}
	
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	pm.marketPrices[symbol] = price
	
	// Update all positions for this symbol
	for _, position := range pm.positions {
		if position.Symbol == symbol {
			pm.updatePositionPL(position)
		}
	}
}

// GetPosition retrieves a position for a user and symbol
func (pm *PositionManager) GetPosition(userID, symbol string) (*Position, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	positionKey := fmt.Sprintf("%s_%s", userID, symbol)
	position, exists := pm.positions[positionKey]
	
	if exists {
		// Return a copy to avoid race conditions
		positionCopy := *position
		return &positionCopy, true
	}
	
	return nil, false
}

// GetUserPositions returns all positions for a user
func (pm *PositionManager) GetUserPositions(userID string) []*Position {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	var positions []*Position
	for _, position := range pm.positions {
		if position.UserID == userID {
			positionCopy := *position
			positions = append(positions, &positionCopy)
		}
	}
	
	return positions
}

// GetSymbolPositions returns all positions for a symbol
func (pm *PositionManager) GetSymbolPositions(symbol string) []*Position {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	var positions []*Position
	for _, position := range pm.positions {
		if position.Symbol == symbol {
			positionCopy := *position
			positions = append(positions, &positionCopy)
		}
	}
	
	return positions
}

// GetAllPositions returns all positions
func (pm *PositionManager) GetAllPositions() []*Position {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	var positions []*Position
	for _, position := range pm.positions {
		positionCopy := *position
		positions = append(positions, &positionCopy)
	}
	
	return positions
}

// GetUserPL calculates total P&L for a user
func (pm *PositionManager) GetUserPL(userID string) (float64, float64) {
	positions := pm.GetUserPositions(userID)
	
	var totalRealized, totalUnrealized float64
	for _, position := range positions {
		totalRealized += position.RealizedPL
		totalUnrealized += position.UnrealizedPL
	}
	
	return totalRealized, totalUnrealized
}

// GetSymbolExposure calculates total exposure for a symbol
func (pm *PositionManager) GetSymbolExposure(symbol string) float64 {
	positions := pm.GetSymbolPositions(symbol)
	
	var totalExposure float64
	for _, position := range positions {
		totalExposure += position.MarketValue
	}
	
	return totalExposure
}

// ClosePosition closes a position for a user and symbol
func (pm *PositionManager) ClosePosition(userID, symbol string, closePrice float64) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	positionKey := fmt.Sprintf("%s_%s", userID, symbol)
	position, exists := pm.positions[positionKey]
	
	if !exists {
		return fmt.Errorf("position not found for user %s and symbol %s", userID, symbol)
	}
	
	if position.Quantity == 0 {
		return fmt.Errorf("position already closed")
	}
	
	// Calculate realized P&L
	realizedPL := (closePrice - position.AvgPrice) * position.Quantity
	position.RealizedPL += realizedPL
	
	// Close position
	position.Quantity = 0
	position.UnrealizedPL = 0
	position.MarketValue = 0
	position.LastUpdate = time.Now()
	
	return nil
}

// updateMetrics updates internal performance metrics
func (pm *PositionManager) updateMetrics() {
	totalPositions := atomic.LoadInt64(&pm.totalPositions)
	totalUpdates := atomic.LoadInt64(&pm.totalUpdates)
	
	pm.metrics["total_positions"] = totalPositions
	pm.metrics["total_updates"] = totalUpdates
	pm.metrics["active_positions"] = int64(len(pm.positions))
	pm.metrics["tracked_symbols"] = int64(len(pm.marketPrices))
	pm.metrics["last_update"] = time.Now()
}

// GetPerformanceMetrics returns position manager performance metrics
func (pm *PositionManager) GetPerformanceMetrics() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	// Update metrics before returning
	pm.updateMetrics()
	
	metrics := make(map[string]interface{})
	for k, v := range pm.metrics {
		metrics[k] = v
	}
	
	return metrics
}

// GetPositionStats returns position statistics
func (pm *PositionManager) GetPositionStats() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_positions"] = len(pm.positions)
	stats["tracked_symbols"] = len(pm.marketPrices)
	
	// Calculate aggregate statistics
	var totalMarketValue, totalRealizedPL, totalUnrealizedPL float64
	var longPositions, shortPositions int
	
	for _, position := range pm.positions {
		totalMarketValue += position.MarketValue
		totalRealizedPL += position.RealizedPL
		totalUnrealizedPL += position.UnrealizedPL
		
		if position.Quantity > 0 {
			longPositions++
		} else if position.Quantity < 0 {
			shortPositions++
		}
	}
	
	stats["total_market_value"] = totalMarketValue
	stats["total_realized_pl"] = totalRealizedPL
	stats["total_unrealized_pl"] = totalUnrealizedPL
	stats["long_positions"] = longPositions
	stats["short_positions"] = shortPositions
	
	return stats
}

// GetMarketPrice returns the current market price for a symbol
func (pm *PositionManager) GetMarketPrice(symbol string) (float64, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	price, exists := pm.marketPrices[symbol]
	return price, exists
}

