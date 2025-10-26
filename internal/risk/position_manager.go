package risk

import (
	"context"
	"sync"
	"time"

	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// PositionManager handles position tracking and management
type PositionManager struct {
	// Positions is a map of user ID and symbol to position
	Positions map[string]map[string]*riskengine.Position
	// PositionCache is a cache for frequently accessed positions
	PositionCache *cache.Cache
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
}

// NewPositionManager creates a new position manager
func NewPositionManager(logger *zap.Logger) *PositionManager {
	return &PositionManager{
		Positions:     make(map[string]map[string]*riskengine.Position),
		PositionCache: cache.New(5*time.Minute, 10*time.Minute),
		logger:        logger,
	}
}

// UpdatePosition updates a user's position for a symbol
func (pm *PositionManager) UpdatePosition(userID, symbol string, quantityDelta, price float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Initialize user positions if not exists
	if pm.Positions[userID] == nil {
		pm.Positions[userID] = make(map[string]*riskengine.Position)
	}

	// Get or create position
	position, exists := pm.Positions[userID][symbol]
	if !exists {
		position = &riskengine.Position{
			UserID:    userID,
			Symbol:    symbol,
			Quantity:  0,
			AvgPrice:  0,
			UpdatedAt: time.Now(),
		}
		pm.Positions[userID][symbol] = position
	}

	// Update position
	oldQuantity := position.Quantity
	newQuantity := oldQuantity + quantityDelta

	if newQuantity == 0 {
		// Position closed
		position.Quantity = 0
		position.AvgPrice = 0
	} else if oldQuantity == 0 {
		// New position
		position.Quantity = newQuantity
		position.AvgPrice = price
	} else if (oldQuantity > 0 && quantityDelta > 0) || (oldQuantity < 0 && quantityDelta < 0) {
		// Adding to existing position
		totalValue := (oldQuantity * position.AvgPrice) + (quantityDelta * price)
		position.Quantity = newQuantity
		position.AvgPrice = totalValue / newQuantity
	} else {
		// Reducing position
		position.Quantity = newQuantity
		// Keep the same average price when reducing
	}

	position.UpdatedAt = time.Now()

	// Update cache
	cacheKey := userID + ":" + symbol
	pm.PositionCache.Set(cacheKey, position, cache.DefaultExpiration)

	pm.logger.Debug("Position updated",
		zap.String("userID", userID),
		zap.String("symbol", symbol),
		zap.Float64("quantityDelta", quantityDelta),
		zap.Float64("price", price),
		zap.Float64("newQuantity", position.Quantity),
		zap.Float64("avgPrice", position.AvgPrice),
	)
}

// GetPosition retrieves a user's position for a symbol
func (pm *PositionManager) GetPosition(ctx context.Context, userID, symbol string) (*riskengine.Position, error) {
	// Try cache first
	cacheKey := userID + ":" + symbol
	if cached, found := pm.PositionCache.Get(cacheKey); found {
		if position, ok := cached.(*riskengine.Position); ok {
			return position, nil
		}
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	userPositions, exists := pm.Positions[userID]
	if !exists {
		// Return zero position
		return &riskengine.Position{
			UserID:    userID,
			Symbol:    symbol,
			Quantity:  0,
			AvgPrice:  0,
			UpdatedAt: time.Now(),
		}, nil
	}

	position, exists := userPositions[symbol]
	if !exists {
		// Return zero position
		return &riskengine.Position{
			UserID:    userID,
			Symbol:    symbol,
			Quantity:  0,
			AvgPrice:  0,
			UpdatedAt: time.Now(),
		}, nil
	}

	// Update cache
	pm.PositionCache.Set(cacheKey, position, cache.DefaultExpiration)

	return position, nil
}

// GetPositions retrieves all positions for a user
func (pm *PositionManager) GetPositions(ctx context.Context, userID string) ([]*riskengine.Position, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	userPositions, exists := pm.Positions[userID]
	if !exists {
		return []*riskengine.Position{}, nil
	}

	positions := make([]*riskengine.Position, 0, len(userPositions))
	for _, position := range userPositions {
		positions = append(positions, position)
	}

	return positions, nil
}

// UpdateUnrealizedPnL updates unrealized P&L for all positions of a symbol
func (pm *PositionManager) UpdateUnrealizedPnL(symbol string, price float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for userID, userPositions := range pm.Positions {
		if position, exists := userPositions[symbol]; exists && position.Quantity != 0 {
			position.UnrealizedPnL = (price - position.AvgPrice) * position.Quantity
			position.UpdatedAt = time.Now()

			// Update cache
			cacheKey := userID + ":" + symbol
			pm.PositionCache.Set(cacheKey, position, cache.DefaultExpiration)
		}
	}
}
