// 🎯 **Standard Engine Core Service**
// Generated using TradSys Code Splitting Standards
//
// This file contains the main service struct, constructor, and core API methods
// for the Standard Order Matching Engine component. It follows the established patterns for
// service initialization, lifecycle management, and primary business operations.
//
// Performance Requirements: Standard latency, heap-based order book management
// File size limit: 350 lines

package order_matching

import (
	"container/heap"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NewEngine creates a new standard order matching engine
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{
		OrderBooks:   make(map[string]*OrderBook),
		TradeChannel: make(chan *Trade, DefaultTradeChannelBuffer),
		logger:       logger,
	}
}

// NewEngineWithConfig creates a new engine with custom configuration
func NewEngineWithConfig(config *EngineConfig, logger *zap.Logger) *Engine {
	return &Engine{
		OrderBooks:   make(map[string]*OrderBook),
		TradeChannel: make(chan *Trade, config.TradeChannelBuffer),
		logger:       logger,
	}
}

// NewOrderBook creates a new order book for a symbol
func NewOrderBook(symbol string, logger *zap.Logger) *OrderBook {
	bids := &OrderHeap{
		Orders: make([]*Order, 0),
		Side:   OrderSideBuy,
	}
	asks := &OrderHeap{
		Orders: make([]*Order, 0),
		Side:   OrderSideSell,
	}
	stopBids := &OrderHeap{
		Orders: make([]*Order, 0),
		Side:   OrderSideBuy,
	}
	stopAsks := &OrderHeap{
		Orders: make([]*Order, 0),
		Side:   OrderSideSell,
	}
	heap.Init(bids)
	heap.Init(asks)
	heap.Init(stopBids)
	heap.Init(stopAsks)

	return &OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Orders:    make(map[string]*Order),
		StopBids:  stopBids,
		StopAsks:  stopAsks,
		LastPrice: 0,
		logger:    logger,
	}
}

// PlaceOrder places an order in the engine
func (e *Engine) PlaceOrder(order *Order) ([]*Trade, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get or create order book
	orderBook, exists := e.OrderBooks[order.Symbol]
	if !exists {
		orderBook = NewOrderBook(order.Symbol, e.logger)
		e.OrderBooks[order.Symbol] = orderBook
	}

	// Add order to the order book
	trades, err := orderBook.AddOrder(order)
	if err != nil {
		return nil, err
	}

	// Send trades to channel
	for _, trade := range trades {
		select {
		case e.TradeChannel <- trade:
		default:
			e.logger.Warn("Trade channel full, dropping trade",
				zap.String("trade_id", trade.ID),
				zap.String("symbol", trade.Symbol))
		}
	}

	return trades, nil
}

// CancelOrder cancels an existing order
func (e *Engine) CancelOrder(orderID, symbol string) error {
	e.mu.RLock()
	orderBook, exists := e.OrderBooks[symbol]
	e.mu.RUnlock()

	if !exists {
		return ErrOrderBookNotFound
	}

	return orderBook.CancelOrder(orderID)
}

// GetOrderBook returns the order book for a symbol
func (e *Engine) GetOrderBook(symbol string) *OrderBook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.OrderBooks[symbol]
}

// GetOrderBookSnapshot returns a snapshot of the order book
func (e *Engine) GetOrderBookSnapshot(symbol string, depth int) (*OrderBookSnapshot, error) {
	e.mu.RLock()
	orderBook, exists := e.OrderBooks[symbol]
	e.mu.RUnlock()

	if !exists {
		return nil, ErrOrderBookNotFound
	}

	return orderBook.GetSnapshot(depth), nil
}

// GetStats returns engine statistics
func (e *Engine) GetStats() *EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &EngineStats{
		ActiveSymbols:  len(e.OrderBooks),
		LastUpdateTime: time.Now(),
	}

	var totalOrders, totalTrades int64
	var totalSpread float64
	var spreadCount int

	for _, orderBook := range e.OrderBooks {
		orderBook.mu.RLock()
		totalOrders += int64(len(orderBook.Orders))

		// Calculate spread if we have both bids and asks
		if orderBook.Bids.Len() > 0 && orderBook.Asks.Len() > 0 {
			bestBid := orderBook.Bids.Peek()
			bestAsk := orderBook.Asks.Peek()
			if bestBid != nil && bestAsk != nil {
				spread := bestAsk.Price - bestBid.Price
				totalSpread += spread
				spreadCount++
			}
		}
		orderBook.mu.RUnlock()
	}

	stats.TotalOrders = totalOrders
	stats.TotalTrades = totalTrades
	if spreadCount > 0 {
		stats.AverageSpread = totalSpread / float64(spreadCount)
	}

	return stats
}

// AddOrder adds an order to the order book
func (ob *OrderBook) AddOrder(order *Order) ([]*Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Generate an ID if not provided
	if order.ID == "" {
		order.ID = uuid.New().String()
	}

	// Set created time if not provided
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now()
	}

	// Set updated time
	order.UpdatedAt = time.Now()

	// Set status to new
	order.Status = OrderStatusNew

	// Add to orders map
	ob.Orders[order.ID] = order

	// Handle stop orders
	if order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket {
		if order.Side == OrderSideBuy {
			if ob.LastPrice > 0 && order.StopPrice <= ob.LastPrice {
				// Stop price triggered, convert to limit/market order
				if order.Type == OrderTypeStopLimit {
					order.Type = OrderTypeLimit
				} else {
					order.Type = OrderTypeMarket
				}
			} else {
				// Add to stop bids
				heap.Push(ob.StopBids, order)
				return nil, nil
			}
		} else {
			if ob.LastPrice > 0 && order.StopPrice >= ob.LastPrice {
				// Stop price triggered, convert to limit/market order
				if order.Type == OrderTypeStopLimit {
					order.Type = OrderTypeLimit
				} else {
					order.Type = OrderTypeMarket
				}
			} else {
				// Add to stop asks
				heap.Push(ob.StopAsks, order)
				return nil, nil
			}
		}
	}

	// Process the order
	return ob.processOrder(order)
}

// CancelOrder cancels an existing order
func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}

	// Remove from orders map
	delete(ob.Orders, orderID)

	// Remove from appropriate heap
	if order.Status == OrderStatusNew {
		if order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket {
			// Remove from stop orders
			if order.Side == OrderSideBuy {
				ob.removeFromHeap(ob.StopBids, orderID)
			} else {
				ob.removeFromHeap(ob.StopAsks, orderID)
			}
		} else {
			// Remove from regular orders
			if order.Side == OrderSideBuy {
				ob.removeFromHeap(ob.Bids, orderID)
			} else {
				ob.removeFromHeap(ob.Asks, orderID)
			}
		}
	}

	// Update order status
	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()

	return nil
}

// GetSnapshot creates a snapshot of the order book
func (ob *OrderBook) GetSnapshot(depth int) *OrderBookSnapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	snapshot := &OrderBookSnapshot{
		Symbol:    ob.Symbol,
		Timestamp: time.Now(),
		LastPrice: ob.LastPrice,
		Bids:      make([]PriceLevel, 0),
		Asks:      make([]PriceLevel, 0),
	}

	// Get bid levels
	bidPrices := make(map[float64]*PriceLevel)
	for _, order := range ob.Bids.Orders {
		if level, exists := bidPrices[order.Price]; exists {
			level.Quantity += order.Quantity - order.FilledQuantity
			level.Orders++
		} else {
			bidPrices[order.Price] = &PriceLevel{
				Price:    order.Price,
				Quantity: order.Quantity - order.FilledQuantity,
				Orders:   1,
			}
		}
	}

	// Convert to slice and sort (highest price first for bids)
	for _, level := range bidPrices {
		snapshot.Bids = append(snapshot.Bids, *level)
	}

	// Get ask levels
	askPrices := make(map[float64]*PriceLevel)
	for _, order := range ob.Asks.Orders {
		if level, exists := askPrices[order.Price]; exists {
			level.Quantity += order.Quantity - order.FilledQuantity
			level.Orders++
		} else {
			askPrices[order.Price] = &PriceLevel{
				Price:    order.Price,
				Quantity: order.Quantity - order.FilledQuantity,
				Orders:   1,
			}
		}
	}

	// Convert to slice and sort (lowest price first for asks)
	for _, level := range askPrices {
		snapshot.Asks = append(snapshot.Asks, *level)
	}

	// Limit depth
	if len(snapshot.Bids) > depth {
		snapshot.Bids = snapshot.Bids[:depth]
	}
	if len(snapshot.Asks) > depth {
		snapshot.Asks = snapshot.Asks[:depth]
	}

	return snapshot
}

// GetOrderBookState returns the current state of the order book
func (ob *OrderBook) GetOrderBookState() *OrderBookState {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	state := &OrderBookState{
		Symbol:         ob.Symbol,
		BidCount:       ob.Bids.Len(),
		AskCount:       ob.Asks.Len(),
		StopBidCount:   ob.StopBids.Len(),
		StopAskCount:   ob.StopAsks.Len(),
		LastPrice:      ob.LastPrice,
		LastUpdateTime: time.Now(),
	}

	// Get best bid and ask
	if ob.Bids.Len() > 0 {
		state.BestBid = ob.Bids.Peek().Price
	}
	if ob.Asks.Len() > 0 {
		state.BestAsk = ob.Asks.Peek().Price
	}

	// Calculate spread
	if state.BestBid > 0 && state.BestAsk > 0 {
		state.Spread = state.BestAsk - state.BestBid
	}

	return state
}

// removeFromHeap removes an order from a heap by order ID
func (ob *OrderBook) removeFromHeap(h *OrderHeap, orderID string) {
	for i, order := range h.Orders {
		if order.ID == orderID {
			heap.Remove(h, i)
			break
		}
	}
}
