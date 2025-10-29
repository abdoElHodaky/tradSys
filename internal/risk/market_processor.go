package risk

import (
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"go.uber.org/zap"
)

// processMarketData processes market data updates for risk calculations
func (s *Service) processMarketData() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case update := <-s.marketDataChan:
			s.updateUnrealizedPnL(update.Symbol, update.Price)
		}
	}
}

// updateUnrealizedPnL updates unrealized PnL for all positions in a symbol
func (s *Service) updateUnrealizedPnL(symbol string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update unrealized PnL for all users with positions in this symbol
	for userID, userPositions := range s.Positions {
		if position, exists := userPositions[symbol]; exists && position.Quantity != 0 {
			position.UnrealizedPnL = position.Quantity * (price - position.AveragePrice)
			position.LastUpdated = time.Now()

			// Update cache
			cacheKey := "position:" + userID + ":" + symbol
			s.PositionCache.Set(cacheKey, position, 5*time.Minute)
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
			s.mu.RLock()
			for symbol, breaker := range s.CircuitBreakers {
				if breaker.IsTriggered() {
					// Check if cooldown period has passed
					if time.Since(breaker.LastTriggered()) > breaker.CooldownPeriod {
						breaker.IsTriggeredFlag = false
						s.logger.Info("Circuit breaker reset",
							zap.String("symbol", symbol),
							zap.Time("last_triggered", breaker.LastTriggered()))
					}
				}
			}
			s.mu.RUnlock()
		}
	}
}

// checkCircuitBreaker checks if a circuit breaker should be triggered
func (s *Service) checkCircuitBreaker(symbol string, price float64, timestamp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	breaker, exists := s.CircuitBreakers[symbol]
	if !exists || breaker.IsTriggered() {
		return
	}

	// Check if price change exceeds threshold
	if breaker.LastPrice > 0 {
		priceChange := abs(price-breaker.LastPrice) / breaker.LastPrice
		if priceChange > breaker.PercentageThreshold {
			// Check if within time window
			if timestamp.Sub(breaker.LastTriggered()) < breaker.TimeWindow {
				breaker.IsTriggeredFlag = true
				breaker.LastTriggeredTime = timestamp

				s.logger.Warn("Circuit breaker triggered",
					zap.String("symbol", symbol),
					zap.Float64("price_change", priceChange*100),
					zap.Float64("threshold", breaker.PercentageThreshold*100),
					zap.Float64("old_price", breaker.LastPrice),
					zap.Float64("new_price", price))
			}
		}
	}

	breaker.LastPrice = price
}

// subscribeToTrades subscribes to trades from the order matching engine
func (s *Service) subscribeToTrades() {
	// This would typically subscribe to a message queue or event stream
	// For now, we'll simulate with a ticker
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// In a real implementation, this would receive actual trades
			// For now, we'll skip the simulation
		}
	}
}

// processTrade processes a trade and updates positions
func (s *Service) processTrade(trade *order_matching.Trade) {
	if trade == nil {
		return
	}

	// Get user IDs from the orders
	buyOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.BuyOrderID)
	if err != nil {
		s.logger.Warn("Could not find buy order for trade",
			zap.String("trade_id", trade.ID),
			zap.String("buy_order_id", trade.BuyOrderID),
			zap.Error(err))
		return
	}
	
	sellOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.SellOrderID)
	if err != nil {
		s.logger.Warn("Could not find sell order for trade",
			zap.String("trade_id", trade.ID),
			zap.String("sell_order_id", trade.SellOrderID),
			zap.Error(err))
		return
	}
	
	if buyOrder == nil || sellOrder == nil {
		s.logger.Warn("Could not find orders for trade",
			zap.String("trade_id", trade.ID),
			zap.String("buy_order_id", trade.BuyOrderID),
			zap.String("sell_order_id", trade.SellOrderID))
		return
	}

	// Update positions for both buyer and seller
	s.updatePosition(buyOrder.UserID, trade.Symbol, trade.Quantity, trade.Price)
	s.updatePosition(sellOrder.UserID, trade.Symbol, -trade.Quantity, trade.Price)

	// Check circuit breakers
	s.checkCircuitBreaker(trade.Symbol, trade.Price, trade.Timestamp)

	s.logger.Debug("Trade processed for risk management",
		zap.String("symbol", trade.Symbol),
		zap.Float64("quantity", trade.Quantity),
		zap.Float64("price", trade.Price),
		zap.String("buyer", buyOrder.UserID),
		zap.String("seller", sellOrder.UserID))
}

// updatePosition updates a user's position (thread-safe wrapper)
func (s *Service) updatePosition(userID, symbol string, quantityDelta, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.updatePositionInternal(userID, symbol, quantityDelta, price)
}

// AddCircuitBreaker adds a circuit breaker for a symbol
func (s *Service) AddCircuitBreaker(symbol string, percentageThreshold float64, timeWindow, cooldownPeriod time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	breaker := &riskengine.CircuitBreaker{
		Symbol:              symbol,
		PercentageThreshold: percentageThreshold,
		TimeWindow:          timeWindow,
		CooldownPeriod:      cooldownPeriod,
		LastPrice:           0,
		LastTriggeredTime:   time.Time{},
		IsTriggeredFlag:     false,
		CreatedAt:           time.Now(),
	}

	s.CircuitBreakers[symbol] = breaker

	s.logger.Info("Circuit breaker added",
		zap.String("symbol", symbol),
		zap.Float64("threshold", percentageThreshold*100),
		zap.Duration("time_window", timeWindow),
		zap.Duration("cooldown", cooldownPeriod))

	return nil
}

// IsCircuitBreakerTriggered checks if a circuit breaker is triggered for a symbol
func (s *Service) IsCircuitBreakerTriggered(symbol string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if breaker, exists := s.CircuitBreakers[symbol]; exists {
		return breaker.IsTriggered()
	}
	return false
}

// GetCircuitBreakerStatus returns the status of all circuit breakers
func (s *Service) GetCircuitBreakerStatus() map[string]*riskengine.CircuitBreaker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]*riskengine.CircuitBreaker)
	for symbol, breaker := range s.CircuitBreakers {
		// Create a copy to avoid race conditions
		status[symbol] = &riskengine.CircuitBreaker{
			Symbol:              breaker.Symbol,
			PercentageThreshold: breaker.PercentageThreshold,
			TimeWindow:          breaker.TimeWindow,
			CooldownPeriod:      breaker.CooldownPeriod,
			LastPrice:           breaker.LastPrice,
			LastTriggeredTime:   breaker.LastTriggeredTime,
			IsTriggeredFlag:     breaker.IsTriggeredFlag,
			CreatedAt:           breaker.CreatedAt,
		}
	}
	return status
}

// UpdateMarketData updates market data for risk calculations
func (s *Service) UpdateMarketData(symbol string, price float64) error {
	if symbol == "" {
		return ErrInvalidOrder
	}

	if price <= 0 {
		return ErrInvalidOrder
	}

	update := MarketDataUpdate{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now(),
	}

	select {
	case s.marketDataChan <- update:
		return nil
	default:
		return ErrRiskCheckFailed
	}
}
