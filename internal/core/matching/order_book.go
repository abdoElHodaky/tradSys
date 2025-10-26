package order_matching

import (
	"container/heap"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OrderBook represents a complete order book implementation
type OrderBook struct {
	Symbol    string
	Bids      *OrderHeap
	Asks      *OrderHeap
	mutex     sync.RWMutex
	logger    *zap.Logger
	
	// Performance tracking
	lastTrade     *Trade
	totalTrades   int64
	totalVolume   float64
	lastUpdate    time.Time
	
	// Order tracking
	orders        map[string]*Order
	ordersByPrice map[float64][]*Order
}

// OrderHeap implements a heap of orders
type OrderHeap struct {
	Orders    []*Order
	IsMaxHeap bool
	mutex     sync.RWMutex
}

// NewOrderBook creates a new order book
func NewOrderBook(symbol string, logger *zap.Logger) *OrderBook {
	return &OrderBook{
		Symbol:        symbol,
		Bids:          NewOrderHeap(true),  // Max heap for bids (highest price first)
		Asks:          NewOrderHeap(false), // Min heap for asks (lowest price first)
		logger:        logger,
		orders:        make(map[string]*Order),
		ordersByPrice: make(map[float64][]*Order),
		lastUpdate:    time.Now(),
	}
}

// NewOrderHeap creates a new order heap
func NewOrderHeap(isMaxHeap bool) *OrderHeap {
	oh := &OrderHeap{
		Orders:    make([]*Order, 0),
		IsMaxHeap: isMaxHeap,
	}
	heap.Init(oh)
	return oh
}

// Len returns the number of orders in the heap
func (h *OrderHeap) Len() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.Orders)
}

// Less compares two orders based on price and time priority
func (h *OrderHeap) Less(i, j int) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	orderI := h.Orders[i]
	orderJ := h.Orders[j]
	
	if h.IsMaxHeap {
		// For bids: higher price has priority, then earlier time
		if orderI.Price != orderJ.Price {
			return orderI.Price > orderJ.Price
		}
	} else {
		// For asks: lower price has priority, then earlier time
		if orderI.Price != orderJ.Price {
			return orderI.Price < orderJ.Price
		}
	}
	
	// If prices are equal, earlier time has priority
	return orderI.CreatedAt.Before(orderJ.CreatedAt)
}

// Swap swaps two orders in the heap
func (h *OrderHeap) Swap(i, j int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.Orders[i], h.Orders[j] = h.Orders[j], h.Orders[i]
}

// Push adds an order to the heap
func (h *OrderHeap) Push(x interface{}) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.Orders = append(h.Orders, x.(*Order))
}

// Pop removes and returns the top order from the heap
func (h *OrderHeap) Pop() interface{} {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	old := h.Orders
	n := len(old)
	if n == 0 {
		return nil
	}
	
	order := old[n-1]
	h.Orders = old[0 : n-1]
	return order
}

// Peek returns the top order without removing it
func (h *OrderHeap) Peek() *Order {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	if len(h.Orders) == 0 {
		return nil
	}
	return h.Orders[0]
}

// Remove removes an order by ID
func (h *OrderHeap) Remove(orderID string) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	for i, order := range h.Orders {
		if order.ID == orderID {
			// Remove the order at index i
			h.Orders = append(h.Orders[:i], h.Orders[i+1:]...)
			heap.Init(h) // Re-heapify
			return true
		}
	}
	return false
}

// AddOrder adds an order to the order book
func (ob *OrderBook) AddOrder(order *Order) ([]*Trade, error) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()
	
	start := time.Now()
	trades := make([]*Trade, 0)
	
	// Validate order
	if order == nil {
		return nil, ErrInvalidOrder
	}
	
	// Set remaining quantity if not set
	if order.RemainingQuantity == 0 {
		order.RemainingQuantity = order.Quantity
	}
	
	// Store order
	ob.orders[order.ID] = order
	
	// Try to match the order
	if order.Side == OrderSideBuy {
		trades = ob.matchBuyOrder(order)
	} else {
		trades = ob.matchSellOrder(order)
	}
	
	// Add remaining order to book if not fully filled
	if order.RemainingQuantity > 0 {
		if order.Side == OrderSideBuy {
			heap.Push(ob.Bids, order)
		} else {
			heap.Push(ob.Asks, order)
		}
	}
	
	// Update statistics
	ob.totalTrades += int64(len(trades))
	for _, trade := range trades {
		ob.totalVolume += trade.Quantity
		ob.lastTrade = trade
	}
	ob.lastUpdate = time.Now()
	
	ob.logger.Debug("Order added to book",
		zap.String("order_id", order.ID),
		zap.String("symbol", order.Symbol),
		zap.String("side", string(order.Side)),
		zap.Float64("quantity", order.Quantity),
		zap.Float64("price", order.Price),
		zap.Int("trades_generated", len(trades)),
		zap.Duration("processing_time", time.Since(start)))
	
	return trades, nil
}

// matchBuyOrder matches a buy order against the ask side
func (ob *OrderBook) matchBuyOrder(buyOrder *Order) []*Trade {
	trades := make([]*Trade, 0)
	
	for buyOrder.RemainingQuantity > 0 && ob.Asks.Len() > 0 {
		bestAsk := ob.Asks.Peek()
		if bestAsk == nil || bestAsk.Price > buyOrder.Price {
			break // No more matching orders
		}
		
		// Remove the best ask
		sellOrder := heap.Pop(ob.Asks).(*Order)
		
		// Calculate trade quantity
		tradeQuantity := min(buyOrder.RemainingQuantity, sellOrder.RemainingQuantity)
		
		// Create trade
		trade := NewTrade(
			buyOrder.Symbol,
			sellOrder.Price, // Use maker price
			tradeQuantity,
			buyOrder.ID,
			sellOrder.ID,
		)
		trade.TakerSide = OrderSideBuy
		trade.IsMaker = false
		
		trades = append(trades, trade)
		
		// Update order quantities
		buyOrder.RemainingQuantity -= tradeQuantity
		sellOrder.RemainingQuantity -= tradeQuantity
		
		// Update order statuses
		if buyOrder.RemainingQuantity == 0 {
			buyOrder.Status = OrderStatusFilled
		} else {
			buyOrder.Status = OrderStatusPartiallyFilled
		}
		
		if sellOrder.RemainingQuantity == 0 {
			sellOrder.Status = OrderStatusFilled
		} else {
			sellOrder.Status = OrderStatusPartiallyFilled
			// Put the partially filled order back
			heap.Push(ob.Asks, sellOrder)
		}
	}
	
	return trades
}

// matchSellOrder matches a sell order against the bid side
func (ob *OrderBook) matchSellOrder(sellOrder *Order) []*Trade {
	trades := make([]*Trade, 0)
	
	for sellOrder.RemainingQuantity > 0 && ob.Bids.Len() > 0 {
		bestBid := ob.Bids.Peek()
		if bestBid == nil || bestBid.Price < sellOrder.Price {
			break // No more matching orders
		}
		
		// Remove the best bid
		buyOrder := heap.Pop(ob.Bids).(*Order)
		
		// Calculate trade quantity
		tradeQuantity := min(sellOrder.RemainingQuantity, buyOrder.RemainingQuantity)
		
		// Create trade
		trade := NewTrade(
			sellOrder.Symbol,
			buyOrder.Price, // Use maker price
			tradeQuantity,
			buyOrder.ID,
			sellOrder.ID,
		)
		trade.TakerSide = OrderSideSell
		trade.IsMaker = false
		
		trades = append(trades, trade)
		
		// Update order quantities
		sellOrder.RemainingQuantity -= tradeQuantity
		buyOrder.RemainingQuantity -= tradeQuantity
		
		// Update order statuses
		if sellOrder.RemainingQuantity == 0 {
			sellOrder.Status = OrderStatusFilled
		} else {
			sellOrder.Status = OrderStatusPartiallyFilled
		}
		
		if buyOrder.RemainingQuantity == 0 {
			buyOrder.Status = OrderStatusFilled
		} else {
			buyOrder.Status = OrderStatusPartiallyFilled
			// Put the partially filled order back
			heap.Push(ob.Bids, buyOrder)
		}
	}
	
	return trades
}

// CancelOrder cancels an order from the book
func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()
	
	// Find and remove the order
	order, exists := ob.orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}
	
	// Remove from appropriate heap
	var removed bool
	if order.Side == OrderSideBuy {
		removed = ob.Bids.Remove(orderID)
	} else {
		removed = ob.Asks.Remove(orderID)
	}
	
	if !removed {
		return ErrOrderNotFound
	}
	
	// Update order status
	order.Status = OrderStatusCanceled
	delete(ob.orders, orderID)
	
	ob.logger.Debug("Order cancelled",
		zap.String("order_id", orderID),
		zap.String("symbol", ob.Symbol))
	
	return nil
}

// GetBestBid returns the best bid price
func (ob *OrderBook) GetBestBid() float64 {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()
	
	if ob.Bids.Len() == 0 {
		return 0
	}
	
	bestBid := ob.Bids.Peek()
	if bestBid == nil {
		return 0
	}
	
	return bestBid.Price
}

// GetBestAsk returns the best ask price
func (ob *OrderBook) GetBestAsk() float64 {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()
	
	if ob.Asks.Len() == 0 {
		return 0
	}
	
	bestAsk := ob.Asks.Peek()
	if bestAsk == nil {
		return 0
	}
	
	return bestAsk.Price
}

// GetSpread returns the bid-ask spread
func (ob *OrderBook) GetSpread() float64 {
	bestBid := ob.GetBestBid()
	bestAsk := ob.GetBestAsk()
	
	if bestBid == 0 || bestAsk == 0 {
		return 0
	}
	
	return bestAsk - bestBid
}

// GetMidPrice returns the mid price
func (ob *OrderBook) GetMidPrice() float64 {
	bestBid := ob.GetBestBid()
	bestAsk := ob.GetBestAsk()
	
	if bestBid == 0 || bestAsk == 0 {
		return 0
	}
	
	return (bestBid + bestAsk) / 2
}

// GetDepth returns the market depth for a given number of levels
func (ob *OrderBook) GetDepth(levels int) ([]OrderBookLevel, []OrderBookLevel) {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()
	
	bidLevels := ob.getHeapLevels(ob.Bids, levels)
	askLevels := ob.getHeapLevels(ob.Asks, levels)
	
	return bidLevels, askLevels
}

// getHeapLevels extracts price levels from a heap
func (ob *OrderBook) getHeapLevels(heap *OrderHeap, maxLevels int) []OrderBookLevel {
	heap.mutex.RLock()
	defer heap.mutex.RUnlock()
	
	levelMap := make(map[float64]*OrderBookLevel)
	
	// Aggregate orders by price level
	for _, order := range heap.Orders {
		if level, exists := levelMap[order.Price]; exists {
			level.Quantity += order.RemainingQuantity
			level.Count++
		} else {
			levelMap[order.Price] = &OrderBookLevel{
				Price:    order.Price,
				Quantity: order.RemainingQuantity,
				Count:    1,
			}
		}
	}
	
	// Convert to slice
	levels := make([]OrderBookLevel, 0, len(levelMap))
	for _, level := range levelMap {
		levels = append(levels, *level)
	}
	
	// Sort levels by price
	for i := 0; i < len(levels)-1; i++ {
		for j := i + 1; j < len(levels); j++ {
			if heap.IsMaxHeap {
				// For bids, sort descending (highest first)
				if levels[i].Price < levels[j].Price {
					levels[i], levels[j] = levels[j], levels[i]
				}
			} else {
				// For asks, sort ascending (lowest first)
				if levels[i].Price > levels[j].Price {
					levels[i], levels[j] = levels[j], levels[i]
				}
			}
		}
	}
	
	// Limit to maxLevels
	if len(levels) > maxLevels {
		levels = levels[:maxLevels]
	}
	
	return levels
}

// GetSnapshot returns a complete snapshot of the order book
func (ob *OrderBook) GetSnapshot() *OrderBookSnapshot {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()
	
	snapshot := NewOrderBookSnapshot(ob.Symbol)
	
	// Get top 10 levels for each side
	bidLevels, askLevels := ob.GetDepth(10)
	
	for _, level := range bidLevels {
		snapshot.AddBidLevel(level.Price, level.Quantity, level.Count)
	}
	
	for _, level := range askLevels {
		snapshot.AddAskLevel(level.Price, level.Quantity, level.Count)
	}
	
	if ob.lastTrade != nil {
		snapshot.LastTrade = ob.lastTrade
	}
	
	return snapshot
}

// GetStats returns order book statistics
func (ob *OrderBook) GetStats() map[string]interface{} {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()
	
	return map[string]interface{}{
		"symbol":       ob.Symbol,
		"bid_count":    ob.Bids.Len(),
		"ask_count":    ob.Asks.Len(),
		"total_orders": len(ob.orders),
		"total_trades": ob.totalTrades,
		"total_volume": ob.totalVolume,
		"best_bid":     ob.GetBestBid(),
		"best_ask":     ob.GetBestAsk(),
		"spread":       ob.GetSpread(),
		"mid_price":    ob.GetMidPrice(),
		"last_update":  ob.lastUpdate,
	}
}

// Helper function to find minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
