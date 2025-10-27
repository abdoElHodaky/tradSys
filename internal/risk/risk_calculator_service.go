package risk

import (
	"time"

	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/patrickmn/go-cache"
)

// processUpdatePositionBatch processes a batch of update position operations
func (s *Service) processUpdatePositionBatch(ops []RiskOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		userID := op.UserID
		symbol := op.Symbol
		data := op.Data.(map[string]interface{})

		// Get position
		var position *riskengine.Position
		userPositions, exists := s.Positions[userID]
		if !exists {
			userPositions = make(map[string]*riskengine.Position)
			s.Positions[userID] = userPositions
		}

		position, exists = userPositions[symbol]
		if !exists {
			position = &riskengine.Position{
				Symbol:         symbol,
				Quantity:       0,
				AveragePrice:   0,
				UnrealizedPnL:  0,
				RealizedPnL:    0,
				LastUpdateTime: time.Now(),
			}
			userPositions[symbol] = position
		}

		// Update position
		quantityDelta, ok := data["quantity_delta"].(float64)
		if ok {
			price, ok := data["price"].(float64)
			if ok {
				// Calculate realized PnL for reducing positions
				var realizedPnL float64
				if (position.Quantity > 0 && quantityDelta < 0) || (position.Quantity < 0 && quantityDelta > 0) {
					// Reducing position
					reduceQuantity := min(abs(position.Quantity), abs(quantityDelta))
					if position.Quantity > 0 {
						reduceQuantity = -reduceQuantity
					}
					realizedPnL = reduceQuantity * (price - position.AveragePrice)
					position.RealizedPnL += realizedPnL
				}

				// Update average entry price for increasing positions
				if (position.Quantity >= 0 && quantityDelta > 0) || (position.Quantity <= 0 && quantityDelta < 0) {
					// Increasing position
					oldValue := position.Quantity * position.AveragePrice
					newValue := quantityDelta * price
					position.AveragePrice = (oldValue + newValue) / (position.Quantity + quantityDelta)
				}

				// Update quantity
				position.Quantity += quantityDelta

				// If position is flat (zero), reset average entry price
				if position.Quantity == 0 {
					position.AveragePrice = 0
				}

				// Update last updated time
				position.LastUpdateTime = time.Now()

				// Update position in cache
				s.PositionCache.Set(userID+":"+symbol, position, cache.DefaultExpiration)
			}
		}

		// Send result
		op.ResultCh <- RiskOperationResult{
			Success: true,
			Data:    position,
		}
	}
}
