package risk

import (
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// processMarketData processes market data updates
func (s *Service) processMarketData() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case update := <-s.marketDataChan:
			// Update unrealized PnL for all positions in this symbol
			s.updateUnrealizedPnL(update.Symbol, update.Price)

			// Check circuit breakers
			s.checkCircuitBreaker(update.Symbol, update.Price, update.Timestamp)
		}
	}
}

// updateUnrealizedPnL updates the unrealized PnL for all positions in a symbol
func (s *Service) updateUnrealizedPnL(symbol string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for userID, userPositions := range s.Positions {
		position, exists := userPositions[symbol]
		if exists && position.Quantity != 0 {
			// Calculate unrealized PnL
			position.UnrealizedPnL = position.Quantity * (price - position.AveragePrice)
			position.LastUpdateTime = time.Now()

			// Update position in cache
			s.PositionCache.Set(userID+":"+symbol, position, cache.DefaultExpiration)
		}
	}
}

// checkCircuitBreakers periodically checks circuit breakers
func (s *Service) checkCircuitBreakers() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()

			// Check if any triggered circuit breakers should be reset
			for symbol, cb := range s.CircuitBreakers {
				if cb.IsTripped() && now.Sub(cb.GetLastTriggered()) > cb.GetCooldownPeriod() {
					cb.Reset()
					s.logger.Info("Circuit breaker reset",
						zap.String("symbol", symbol),
						zap.Float64("reference_price", cb.GetReferencePrice()))
				}
			}

			s.mu.Unlock()
		}
	}
}

// checkCircuitBreaker checks if a circuit breaker should be triggered
func (s *Service) checkCircuitBreaker(symbol string, price float64, timestamp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cb, exists := s.CircuitBreakers[symbol]
	if !exists {
		return
	}

	// Skip if already triggered
	if cb.IsTripped() {
		return
	}

	// Calculate percentage change
	if cb.GetReferencePrice() == 0 {
		cb.SetReferencePrice(price)
		return
	}

	percentageChange := abs((price - cb.GetReferencePrice()) / cb.GetReferencePrice() * 100)

	// Check if circuit breaker should be triggered
	if percentageChange >= cb.GetPriceChangeThreshold() {
		cb.Trip()

		s.logger.Warn("Circuit breaker triggered",
			zap.String("symbol", symbol),
			zap.Float64("reference_price", cb.GetReferencePrice()),
			zap.Float64("current_price", price),
			zap.Float64("percentage_change", percentageChange),
			zap.Float64("threshold", cb.GetPriceChangeThreshold()))
	}
}

// subscribeToTrades subscribes to trades from the order matching engine
func (s *Service) subscribeToTrades() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case trade := <-s.OrderEngine.TradeChannel:
			// Process trade for risk management
			s.processTrade(trade)
		}
	}
}

// processTrade processes a trade for risk management
func (s *Service) processTrade(trade *orders.Trade) {
	// Get the primary order (the one that initiated the trade)
	primaryOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.OrderID)
	if err != nil {
		s.logger.Error("Failed to get primary order",
			zap.String("order_id", trade.OrderID),
			zap.Error(err))
		return
	}

	// Get the counter order (the one that was matched against)
	counterOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.CounterOrderID)
	if err != nil {
		s.logger.Error("Failed to get counter order",
			zap.String("order_id", trade.CounterOrderID),
			zap.Error(err))
		return
	}

	// Determine buy and sell orders based on trade side
	var buyOrder, sellOrder *orders.Order
	if trade.Side == orders.OrderSideBuy {
		buyOrder = primaryOrder
		sellOrder = counterOrder
	} else {
		buyOrder = counterOrder
		sellOrder = primaryOrder
	}

	// Update buy position
	s.updatePosition(buyOrder.UserID, trade.Symbol, trade.Quantity, trade.Price)

	// Update sell position
	s.updatePosition(sellOrder.UserID, trade.Symbol, -trade.Quantity, trade.Price)

	// Update market data
	s.marketDataChan <- MarketDataUpdate{
		Symbol:    trade.Symbol,
		Price:     trade.Price,
		Timestamp: trade.Timestamp,
	}
}

// updatePosition updates a position
func (s *Service) updatePosition(userID, symbol string, quantityDelta, price float64) {
	// Use batch processing for better performance
	resultCh := make(chan RiskOperationResult, 1)
	s.riskBatchChan <- RiskOperation{
		OpType: "update_position",
		UserID: userID,
		Symbol: symbol,
		Data: map[string]interface{}{
			"quantity_delta": quantityDelta,
			"price":          price,
		},
		ResultCh: resultCh,
	}

	// Wait for result
	<-resultCh
}
