package matching

import (
	"container/heap"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Heap interface implementation for OrderHeap

// Len returns the length of the heap
func (h OrderHeap) Len() int { return len(h.Orders) }

// Less returns whether the order at index i is less than the order at index j
func (h OrderHeap) Less(i, j int) bool {
	if h.Side == OrderSideBuy {
		// For buy orders, higher prices have higher priority
		if h.Orders[i].Price == h.Orders[j].Price {
			// If prices are equal, earlier orders have higher priority
			return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
		}
		return h.Orders[i].Price > h.Orders[j].Price
	} else {
		// For sell orders, lower prices have higher priority
		if h.Orders[i].Price == h.Orders[j].Price {
			// If prices are equal, earlier orders have higher priority
			return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
		}
		return h.Orders[i].Price < h.Orders[j].Price
	}
}

// Swap swaps the orders at indices i and j
func (h OrderHeap) Swap(i, j int) {
	h.Orders[i], h.Orders[j] = h.Orders[j], h.Orders[i]
}

// Push adds an order to the heap
func (h *OrderHeap) Push(x interface{}) {
	h.Orders = append(h.Orders, x.(*Order))
}

// Pop removes and returns the top order from the heap
func (h *OrderHeap) Pop() interface{} {
	old := h.Orders
	n := len(old)
	order := old[n-1]
	h.Orders = old[0 : n-1]
	return order
}

// Peek returns the top order without removing it
func (h *OrderHeap) Peek() *Order {
	if len(h.Orders) == 0 {
		return nil
	}
	return h.Orders[0]
}

// NewOrderBook creates a new order book for a symbol
func NewOrderBook(symbol string, logger *zap.Logger) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideBuy,
		},
		Asks: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideSell,
		},
		Orders: make(map[string]*Order),
		StopBids: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideBuy,
		},
		StopAsks: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideSell,
		},
		logger: logger,
	}
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine(logger *zap.Logger) *MatchingEngine {
	return &MatchingEngine{
		OrderBooks: make(map[string]*OrderBook),
		Trades:     make([]*Trade, 0),
		logger:     logger,
		TradeChan:  make(chan *Trade, 1000),
		OrderChan:  make(chan *Order, 1000),
		CancelChan: make(chan string, 1000),
		StopChan:   make(chan struct{}),
		Metrics:    NewEngineMetrics(),
	}
}

// NewEngineMetrics creates new engine metrics
func NewEngineMetrics() *EngineMetrics {
	return &EngineMetrics{
		MinLatency: time.Hour, // Initialize with a high value
	}
}

// Start starts the matching engine
func (me *MatchingEngine) Start() error {
	me.mu.Lock()
	defer me.mu.Unlock()

	if me.Running {
		return fmt.Errorf("matching engine is already running")
	}

	me.Running = true
	me.logger.Info("Starting matching engine")

	// Start processing goroutines
	go me.processOrders()
	go me.processCancellations()

	return nil
}

// Stop stops the matching engine
func (me *MatchingEngine) Stop() error {
	me.mu.Lock()
	defer me.mu.Unlock()

	if !me.Running {
		return fmt.Errorf("matching engine is not running")
	}

	me.Running = false
	close(me.StopChan)
	me.logger.Info("Stopping matching engine")

	return nil
}

// AddOrder adds an order to the matching engine
func (me *MatchingEngine) AddOrder(order *Order) ([]*Trade, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}

	// Validate order
	if err := me.validateOrder(order); err != nil {
		return nil, err
	}

	me.mu.Lock()
	defer me.mu.Unlock()

	// Get or create order book
	orderBook, exists := me.OrderBooks[order.Symbol]
	if !exists {
		orderBook = NewOrderBook(order.Symbol, me.logger)
		me.OrderBooks[order.Symbol] = orderBook
	}

	// Process the order
	trades := orderBook.processOrder(order)

	// Update metrics
	atomic.AddInt64(&me.Metrics.TotalOrders, 1)
	if len(trades) > 0 {
		atomic.AddInt64(&me.Metrics.TotalTrades, int64(len(trades)))
		for _, trade := range trades {
			me.Metrics.mu.Lock()
			me.Metrics.TotalVolume += trade.Quantity * trade.Price
			me.Metrics.LastTradeTime = trade.Timestamp
			me.Metrics.mu.Unlock()
		}
	}

	// Store trades
	me.Trades = append(me.Trades, trades...)

	return trades, nil
}

// CancelOrder cancels an order
func (me *MatchingEngine) CancelOrder(orderID string) (*CancelResult, error) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Find the order in all order books
	for _, orderBook := range me.OrderBooks {
		if result := orderBook.cancelOrder(orderID); result.Success {
			return result, nil
		}
	}

	return &CancelResult{
		Success: false,
		OrderID: orderID,
		Error:   "order not found",
	}, nil
}

// GetOrderBook returns the order book for a symbol
func (me *MatchingEngine) GetOrderBook(symbol string) (*OrderBook, error) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	orderBook, exists := me.OrderBooks[symbol]
	if !exists {
		return nil, fmt.Errorf("order book not found for symbol: %s", symbol)
	}

	return orderBook, nil
}

// validateOrder validates an order
func (me *MatchingEngine) validateOrder(order *Order) error {
	if order.Symbol == "" {
		return fmt.Errorf("order symbol cannot be empty")
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity must be positive")
	}
	if order.Side != OrderSideBuy && order.Side != OrderSideSell {
		return fmt.Errorf("invalid order side: %s", order.Side)
	}
	if order.Type == OrderTypeLimit && order.Price <= 0 {
		return fmt.Errorf("limit order price must be positive")
	}
	return nil
}

// processOrders processes orders from the order channel
func (me *MatchingEngine) processOrders() {
	for {
		select {
		case order := <-me.OrderChan:
			if order != nil {
				me.AddOrder(order)
			}
		case <-me.StopChan:
			return
		}
	}
}

// processCancellations processes cancellations from the cancel channel
func (me *MatchingEngine) processCancellations() {
	for {
		select {
		case orderID := <-me.CancelChan:
			if orderID != "" {
				me.CancelOrder(orderID)
			}
		case <-me.StopChan:
			return
		}
	}
}

// OrderBook methods

// processOrder processes an order in the order book
func (ob *OrderBook) processOrder(order *Order) []*Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Store the order
	ob.Orders[order.ID] = order

	var trades []*Trade

	switch order.Type {
	case OrderTypeMarket:
		trades = ob.processMarketOrder(order)
	case OrderTypeLimit:
		trades = ob.processLimitOrder(order)
	case OrderTypeStop, OrderTypeStopLimit, OrderTypeStopMarket:
		trades = ob.processStopOrder(order)
	}

	return trades
}

// processMarketOrder processes a market order
func (ob *OrderBook) processMarketOrder(order *Order) []*Trade {
	var trades []*Trade
	remainingQuantity := order.Quantity

	if order.Side == OrderSideBuy {
		// Match against asks (sell orders)
		for ob.Asks.Len() > 0 && remainingQuantity > 0 {
			bestAsk := ob.Asks.Peek()
			if bestAsk == nil {
				break
			}

			trade := ob.executeTrade(order, bestAsk, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestAsk.RemainingQuantity() <= 0 {
				heap.Pop(ob.Asks)
				bestAsk.Status = OrderStatusFilled
			}
		}
	} else {
		// Match against bids (buy orders)
		for ob.Bids.Len() > 0 && remainingQuantity > 0 {
			bestBid := ob.Bids.Peek()
			if bestBid == nil {
				break
			}

			trade := ob.executeTrade(order, bestBid, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestBid.RemainingQuantity() <= 0 {
				heap.Pop(ob.Bids)
				bestBid.Status = OrderStatusFilled
			}
		}
	}

	// Update order status
	if remainingQuantity <= 0 {
		order.Status = OrderStatusFilled
	} else if remainingQuantity < order.Quantity {
		order.Status = OrderStatusPartiallyFilled
	} else {
		order.Status = OrderStatusRejected
	}

	return trades
}

// processLimitOrder processes a limit order
func (ob *OrderBook) processLimitOrder(order *Order) []*Trade {
	var trades []*Trade
	remainingQuantity := order.Quantity

	if order.Side == OrderSideBuy {
		// Try to match against asks
		for ob.Asks.Len() > 0 && remainingQuantity > 0 {
			bestAsk := ob.Asks.Peek()
			if bestAsk == nil || bestAsk.Price > order.Price {
				break
			}

			trade := ob.executeTrade(order, bestAsk, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestAsk.RemainingQuantity() <= 0 {
				heap.Pop(ob.Asks)
				bestAsk.Status = OrderStatusFilled
			}
		}

		// If there's remaining quantity, add to bids
		if remainingQuantity > 0 {
			order.Quantity = remainingQuantity
			heap.Push(ob.Bids, order)
			order.Status = OrderStatusNew
		}
	} else {
		// Try to match against bids
		for ob.Bids.Len() > 0 && remainingQuantity > 0 {
			bestBid := ob.Bids.Peek()
			if bestBid == nil || bestBid.Price < order.Price {
				break
			}

			trade := ob.executeTrade(order, bestBid, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestBid.RemainingQuantity() <= 0 {
				heap.Pop(ob.Bids)
				bestBid.Status = OrderStatusFilled
			}
		}

		// If there's remaining quantity, add to asks
		if remainingQuantity > 0 {
			order.Quantity = remainingQuantity
			heap.Push(ob.Asks, order)
			order.Status = OrderStatusNew
		}
	}

	// Update order status
	if remainingQuantity <= 0 {
		order.Status = OrderStatusFilled
	} else if remainingQuantity < order.Quantity {
		order.Status = OrderStatusPartiallyFilled
	}

	return trades
}

// processStopOrder processes a stop order
func (ob *OrderBook) processStopOrder(order *Order) []*Trade {
	// For now, just add to stop order heaps
	// In a real implementation, these would be triggered when price conditions are met
	if order.Side == OrderSideBuy {
		heap.Push(ob.StopBids, order)
	} else {
		heap.Push(ob.StopAsks, order)
	}
	order.Status = OrderStatusNew
	return nil
}

// executeTrade executes a trade between two orders
func (ob *OrderBook) executeTrade(takerOrder, makerOrder *Order, remainingQuantity *float64) *Trade {
	tradeQuantity := min(*remainingQuantity, makerOrder.RemainingQuantity())
	if tradeQuantity <= 0 {
		return nil
	}

	trade := &Trade{
		ID:          uuid.New().String(),
		Symbol:      ob.Symbol,
		Price:       makerOrder.Price,
		Quantity:    tradeQuantity,
		Timestamp:   time.Now(),
		TakerSide:   takerOrder.Side,
		MakerSide:   makerOrder.Side,
		TakerFee:    tradeQuantity * makerOrder.Price * 0.001, // 0.1% fee
		MakerFee:    tradeQuantity * makerOrder.Price * 0.0005, // 0.05% fee
	}

	if takerOrder.Side == OrderSideBuy {
		trade.BuyOrderID = takerOrder.ID
		trade.SellOrderID = makerOrder.ID
	} else {
		trade.BuyOrderID = makerOrder.ID
		trade.SellOrderID = takerOrder.ID
	}

	// Update order quantities
	*remainingQuantity -= tradeQuantity
	makerOrder.FilledQuantity += tradeQuantity

	// Update last price
	ob.LastPrice = trade.Price

	ob.logger.Debug("Trade executed",
		zap.String("trade_id", trade.ID),
		zap.String("symbol", trade.Symbol),
		zap.Float64("price", trade.Price),
		zap.Float64("quantity", trade.Quantity))

	return trade
}

// cancelOrder cancels an order in the order book
func (ob *OrderBook) cancelOrder(orderID string) *CancelResult {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return &CancelResult{
			Success: false,
			OrderID: orderID,
			Error:   "order not found",
		}
	}

	// Remove from appropriate heap
	if order.Side == OrderSideBuy {
		ob.removeFromHeap(ob.Bids, orderID)
	} else {
		ob.removeFromHeap(ob.Asks, orderID)
	}

	// Update order status
	order.Status = OrderStatusCanceled
	delete(ob.Orders, orderID)

	return &CancelResult{
		Success:        true,
		OrderID:        orderID,
		CancelledOrder: order,
	}
}

// removeFromHeap removes an order from a heap by ID
func (ob *OrderBook) removeFromHeap(h *OrderHeap, orderID string) {
	for i, order := range h.Orders {
		if order.ID == orderID {
			// Remove the order from the slice
			h.Orders = append(h.Orders[:i], h.Orders[i+1:]...)
			// Re-heapify
			heap.Init(h)
			break
		}
	}
}

// Helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
