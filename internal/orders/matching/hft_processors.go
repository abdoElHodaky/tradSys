// 🎯 **HFT Engine Processors**
// Generated using TradSys Code Splitting Standards
//
// This file contains the ultra-critical order matching algorithms and processing logic
// for the High-Frequency Trading Engine. These functions are the most performance-sensitive
// components and require zero-overhead abstractions to maintain <100μs latency.
//
// Performance Requirements: <100μs latency, zero-allocation matching algorithms
// File size limit: 410 lines

package order_matching

import (
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// processOrderFast processes an order with HFT optimizations
func (ob *HFTOrderBook) processOrderFast(order *FastOrder) ([]*Trade, error) {
	trades := make([]*Trade, 0, 4) // Pre-allocate for common case

	// Handle market orders with optimized matching
	if order.Type == OrderTypeMarket {
		if order.Side == OrderSideBuy {
			trades = ob.matchMarketBuyOrder(order, trades)
		} else {
			trades = ob.matchMarketSellOrder(order, trades)
		}
	} else if order.Type == OrderTypeLimit {
		if order.Side == OrderSideBuy {
			trades = ob.matchLimitBuyOrder(order, trades)
		} else {
			trades = ob.matchLimitSellOrder(order, trades)
		}
	}

	// Update order book statistics
	atomic.AddUint64(&ob.orderCount, 1)
	atomic.StoreInt64(&ob.lastUpdated, time.Now().UnixNano())

	return trades, nil
}

// matchMarketBuyOrder matches a market buy order
func (ob *HFTOrderBook) matchMarketBuyOrder(order *FastOrder, trades []*Trade) []*Trade {
	asksPtr := atomic.LoadPointer(&ob.asks)
	asksTree := (*PriceLevelTree)(asksPtr)

	asksTree.mu.Lock()
	defer asksTree.mu.Unlock()

	// Find best ask prices and match
	for order.FilledQuantity < order.Quantity && asksTree.root != nil {
		bestAsk := asksTree.findBestPrice()
		if bestAsk == nil {
			break
		}

		// Match with best ask
		trade := ob.executeTradeOptimized(order, bestAsk.orders[0])
		if trade != nil {
			trades = append(trades, trade)

			// Update last price atomically
			priceUint64 := *(*uint64)(unsafe.Pointer(&trade.Price))
			atomic.StoreUint64(&ob.lastPrice, priceUint64)
		}

		// Remove filled orders
		if bestAsk.orders[0].Status == OrderStatusFilled {
			bestAsk.orders = bestAsk.orders[1:]
			bestAsk.orderCount--

			// Remove price level if empty
			if len(bestAsk.orders) == 0 {
				asksTree.removeNode(bestAsk)
			}
		}
	}

	return trades
}

// matchMarketSellOrder matches a market sell order
func (ob *HFTOrderBook) matchMarketSellOrder(order *FastOrder, trades []*Trade) []*Trade {
	bidsPtr := atomic.LoadPointer(&ob.bids)
	bidsTree := (*PriceLevelTree)(bidsPtr)

	bidsTree.mu.Lock()
	defer bidsTree.mu.Unlock()

	// Find best bid prices and match
	for order.FilledQuantity < order.Quantity && bidsTree.root != nil {
		bestBid := bidsTree.findBestPrice()
		if bestBid == nil {
			break
		}

		// Match with best bid
		trade := ob.executeTradeOptimized(order, bestBid.orders[0])
		if trade != nil {
			trades = append(trades, trade)

			// Update last price atomically
			priceUint64 := *(*uint64)(unsafe.Pointer(&trade.Price))
			atomic.StoreUint64(&ob.lastPrice, priceUint64)
		}

		// Remove filled orders
		if bestBid.orders[0].Status == OrderStatusFilled {
			bestBid.orders = bestBid.orders[1:]
			bestBid.orderCount--

			// Remove price level if empty
			if len(bestBid.orders) == 0 {
				bidsTree.removeNode(bestBid)
			}
		}
	}

	return trades
}

// matchLimitBuyOrder matches a limit buy order
func (ob *HFTOrderBook) matchLimitBuyOrder(order *FastOrder, trades []*Trade) []*Trade {
	asksPtr := atomic.LoadPointer(&ob.asks)
	asksTree := (*PriceLevelTree)(asksPtr)

	asksTree.mu.Lock()
	defer asksTree.mu.Unlock()

	// Match against asks at or below the limit price
	for order.FilledQuantity < order.Quantity {
		bestAsk := asksTree.findBestPrice()
		if bestAsk == nil || bestAsk.price > order.Price {
			break // No more matching opportunities
		}

		// Match with best ask
		trade := ob.executeTradeOptimized(order, bestAsk.orders[0])
		if trade != nil {
			trades = append(trades, trade)

			// Update last price atomically
			priceUint64 := *(*uint64)(unsafe.Pointer(&trade.Price))
			atomic.StoreUint64(&ob.lastPrice, priceUint64)
		}

		// Remove filled orders
		if bestAsk.orders[0].Status == OrderStatusFilled {
			bestAsk.orders = bestAsk.orders[1:]
			bestAsk.orderCount--

			// Remove price level if empty
			if len(bestAsk.orders) == 0 {
				asksTree.removeNode(bestAsk)
			}
		}
	}

	// Add remaining quantity to order book if not fully filled
	if order.FilledQuantity < order.Quantity {
		ob.addOrderToBook(order, asksTree)
	}

	return trades
}

// matchLimitSellOrder matches a limit sell order
func (ob *HFTOrderBook) matchLimitSellOrder(order *FastOrder, trades []*Trade) []*Trade {
	bidsPtr := atomic.LoadPointer(&ob.bids)
	bidsTree := (*PriceLevelTree)(bidsPtr)

	bidsTree.mu.Lock()
	defer bidsTree.mu.Unlock()

	// Match against bids at or above the limit price
	for order.FilledQuantity < order.Quantity {
		bestBid := bidsTree.findBestPrice()
		if bestBid == nil || bestBid.price < order.Price {
			break // No more matching opportunities
		}

		// Match with best bid
		trade := ob.executeTradeOptimized(order, bestBid.orders[0])
		if trade != nil {
			trades = append(trades, trade)

			// Update last price atomically
			priceUint64 := *(*uint64)(unsafe.Pointer(&trade.Price))
			atomic.StoreUint64(&ob.lastPrice, priceUint64)
		}

		// Remove filled orders
		if bestBid.orders[0].Status == OrderStatusFilled {
			bestBid.orders = bestBid.orders[1:]
			bestBid.orderCount--

			// Remove price level if empty
			if len(bestBid.orders) == 0 {
				bidsTree.removeNode(bestBid)
			}
		}
	}

	// Add remaining quantity to order book if not fully filled
	if order.FilledQuantity < order.Quantity {
		ob.addOrderToBook(order, bidsTree)
	}

	return trades
}

// executeTradeOptimized executes a trade between two orders with HFT optimizations
func (ob *HFTOrderBook) executeTradeOptimized(takerOrder *FastOrder, makerOrder *Order) *Trade {
	// Calculate trade quantity (minimum of remaining quantities)
	takerRemaining := takerOrder.Quantity - takerOrder.FilledQuantity
	makerRemaining := makerOrder.Quantity - makerOrder.FilledQuantity
	tradeQuantity := takerRemaining
	if makerRemaining < tradeQuantity {
		tradeQuantity = makerRemaining
	}

	// Use maker's price for trade execution
	tradePrice := makerOrder.Price

	// Create trade with optimized allocation
	trade := &Trade{
		ID:           uuid.New().String(),
		Symbol:       ob.Symbol,
		Price:        tradePrice,
		Quantity:     tradeQuantity,
		TakerOrderID: takerOrder.ID,
		MakerOrderID: makerOrder.ID,
		TakerSide:    takerOrder.Side,
		Timestamp:    time.Now(),
	}

	// Update order quantities
	takerOrder.FilledQuantity += tradeQuantity
	makerOrder.FilledQuantity += tradeQuantity

	// Update order statuses
	if takerOrder.FilledQuantity >= takerOrder.Quantity {
		takerOrder.Status = OrderStatusFilled
	} else {
		takerOrder.Status = OrderStatusPartiallyFilled
	}

	if makerOrder.FilledQuantity >= makerOrder.Quantity {
		makerOrder.Status = OrderStatusFilled
	} else {
		makerOrder.Status = OrderStatusPartiallyFilled
	}

	// Update trade count
	atomic.AddUint64(&ob.tradeCount, 1)

	return trade
}

// addOrderToBook adds an order to the appropriate side of the order book
func (ob *HFTOrderBook) addOrderToBook(order *FastOrder, tree *PriceLevelTree) {
	// Find or create price level
	priceLevel := tree.findOrCreatePriceLevel(order.Price)
	
	// Convert FastOrder to Order for storage
	bookOrder := &Order{
		ID:             order.ID,
		Symbol:         order.Symbol,
		Side:           order.Side,
		Type:           order.Type,
		Quantity:       order.Quantity - order.FilledQuantity, // Remaining quantity
		Price:          order.Price,
		Status:         OrderStatusOpen,
		FilledQuantity: 0, // Reset for book storage
		CreatedAt:      time.Unix(0, order.CreatedAtNano),
		UpdatedAt:      time.Unix(0, order.UpdatedAtNano),
	}

	// Add to price level (FIFO)
	priceLevel.orders = append(priceLevel.orders, bookOrder)
	priceLevel.orderCount++
	priceLevel.totalQuantity += bookOrder.Quantity

	// Store in order lookup map
	ob.orders.Store(order.ID, bookOrder)
}

// cancelOrder cancels an existing order
func (ob *HFTOrderBook) cancelOrder(orderID string) error {
	// Find order in lookup map
	orderInterface, exists := ob.orders.Load(orderID)
	if !exists {
		return ErrOrderNotFound
	}

	order := orderInterface.(*Order)
	
	// Remove from appropriate tree
	var tree *PriceLevelTree
	if order.Side == OrderSideBuy {
		bidsPtr := atomic.LoadPointer(&ob.bids)
		tree = (*PriceLevelTree)(bidsPtr)
	} else {
		asksPtr := atomic.LoadPointer(&ob.asks)
		tree = (*PriceLevelTree)(asksPtr)
	}

	tree.mu.Lock()
	defer tree.mu.Unlock()

	// Find and remove from price level
	priceLevel := tree.findPriceLevel(order.Price)
	if priceLevel != nil {
		for i, levelOrder := range priceLevel.orders {
			if levelOrder.ID == orderID {
				// Remove order from slice
				priceLevel.orders = append(priceLevel.orders[:i], priceLevel.orders[i+1:]...)
				priceLevel.orderCount--
				priceLevel.totalQuantity -= order.Quantity

				// Remove price level if empty
				if len(priceLevel.orders) == 0 {
					tree.removeNode(priceLevel)
				}
				break
			}
		}
	}

	// Remove from lookup map
	ob.orders.Delete(orderID)

	// Update order status
	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()

	return nil
}

// getSnapshot creates a snapshot of the order book
func (ob *HFTOrderBook) getSnapshot(depth int) *HFTOrderBookSnapshot {
	snapshot := &HFTOrderBookSnapshot{
		Symbol:    ob.Symbol,
		Timestamp: time.Now().UnixNano(),
	}

	// Get bids
	bidsPtr := atomic.LoadPointer(&ob.bids)
	bidsTree := (*PriceLevelTree)(bidsPtr)
	bidsTree.mu.RLock()
	snapshot.BidLevels = bidsTree.getTopLevels(depth, true) // Descending for bids
	bidsTree.mu.RUnlock()

	// Get asks
	asksPtr := atomic.LoadPointer(&ob.asks)
	asksTree := (*PriceLevelTree)(asksPtr)
	asksTree.mu.RLock()
	snapshot.AskLevels = asksTree.getTopLevels(depth, false) // Ascending for asks
	asksTree.mu.RUnlock()

	// Calculate best bid/ask and spread
	if len(snapshot.BidLevels) > 0 {
		snapshot.BestBid = snapshot.BidLevels[0].Price
		snapshot.TotalBidQuantity = snapshot.BidLevels[0].Quantity
	}
	if len(snapshot.AskLevels) > 0 {
		snapshot.BestAsk = snapshot.AskLevels[0].Price
		snapshot.TotalAskQuantity = snapshot.AskLevels[0].Quantity
	}
	if snapshot.BestBid > 0 && snapshot.BestAsk > 0 {
		snapshot.Spread = snapshot.BestAsk - snapshot.BestBid
	}

	// Get last trade info
	snapshot.LastTradePrice = ob.GetLastPrice()
	snapshot.LastTradeTime = atomic.LoadInt64(&ob.lastUpdated)

	return snapshot
}

// monitorPerformance monitors engine performance metrics
func (e *HFTEngine) monitorPerformance() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			// Update performance metrics
			stats := e.GetStats()
			
			// Log performance if latency exceeds threshold
			if stats.AvgLatencyNanos > MaxTargetLatencyNanos {
				e.logger.Warn("HFT engine latency exceeds target",
					zap.Uint64("avg_latency_nanos", stats.AvgLatencyNanos),
					zap.Uint64("target_nanos", MaxTargetLatencyNanos),
					zap.Uint64("orders_processed", stats.OrdersProcessed),
					zap.Uint64("trades_executed", stats.TradesExecuted))
			}
		}
	}
}

// processTradesAsync processes trades asynchronously
func (e *HFTEngine) processTradesAsync() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case trade := <-e.TradeChannel:
			// Process trade (e.g., send to risk management, update positions)
			e.logger.Debug("Trade executed",
				zap.String("trade_id", trade.ID),
				zap.String("symbol", trade.Symbol),
				zap.Float64("price", trade.Price),
				zap.Float64("quantity", trade.Quantity),
				zap.String("taker_side", string(trade.TakerSide)))
		}
	}
}

// GetLastPrice returns the last traded price for a symbol
func (ob *HFTOrderBook) GetLastPrice() float64 {
	priceUint64 := atomic.LoadUint64(&ob.lastPrice)
	return *(*float64)(unsafe.Pointer(&priceUint64))
}

// GetOrderCount returns the total number of orders processed
func (ob *HFTOrderBook) GetOrderCount() uint64 {
	return atomic.LoadUint64(&ob.orderCount)
}

// GetTradeCount returns the total number of trades executed
func (ob *HFTOrderBook) GetTradeCount() uint64 {
	return atomic.LoadUint64(&ob.tradeCount)
}
