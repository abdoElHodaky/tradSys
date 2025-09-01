package order_matching

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OrderType represents the type of order
type OrderType string

const (
	// OrderTypeLimit represents a limit order
	OrderTypeLimit OrderType = "limit"
	// OrderTypeMarket represents a market order
	OrderTypeMarket OrderType = "market"
	// OrderTypeStopLimit represents a stop limit order
	OrderTypeStopLimit OrderType = "stop_limit"
	// OrderTypeStopMarket represents a stop market order
	OrderTypeStopMarket OrderType = "stop_market"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	// OrderSideBuy represents a buy order
	OrderSideBuy OrderSide = "buy"
	// OrderSideSell represents a sell order
	OrderSideSell OrderSide = "sell"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	// OrderStatusNew represents a new order
	OrderStatusNew OrderStatus = "new"
	// OrderStatusPartiallyFilled represents a partially filled order
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	// OrderStatusFilled represents a filled order
	OrderStatusFilled OrderStatus = "filled"
	// OrderStatusCancelled represents a cancelled order
	OrderStatusCancelled OrderStatus = "cancelled"
	// OrderStatusRejected represents a rejected order
	OrderStatusRejected OrderStatus = "rejected"
)

// Order represents an order in the order book
type Order struct {
	// Order ID
	ID string

	// Order details
	Symbol    string
	Type      OrderType
	Side      OrderSide
	Price     float64
	Size      float64
	Timestamp time.Time
	Status    OrderStatus

	// Stop price for stop orders
	StopPrice float64

	// Remaining size
	RemainingSize float64

	// User ID
	UserID string

	// Index in the heap
	Index int

	// Metadata
	Metadata map[string]interface{}
}

// NewOrder creates a new order
func NewOrder(symbol string, orderType OrderType, side OrderSide, price, size float64, userID string) *Order {
	return &Order{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          orderType,
		Side:          side,
		Price:         price,
		Size:          size,
		RemainingSize: size,
		Timestamp:     time.Now(),
		Status:        OrderStatusNew,
		UserID:        userID,
		Index:         -1,
		Metadata:      make(map[string]interface{}),
	}
}

// NewStopOrder creates a new stop order
func NewStopOrder(symbol string, orderType OrderType, side OrderSide, price, stopPrice, size float64, userID string) *Order {
	order := NewOrder(symbol, orderType, side, price, size, userID)
	order.StopPrice = stopPrice
	return order
}

// OrderHeap is a heap of orders
type OrderHeap struct {
	Orders []*Order
	Less   func(i, j int) bool
}

// Len returns the length of the heap
func (h *OrderHeap) Len() int {
	return len(h.Orders)
}

// Less returns whether the order at index i is less than the order at index j
func (h *OrderHeap) Less(i, j int) bool {
	return h.Less(i, j)
}

// Swap swaps the orders at indices i and j
func (h *OrderHeap) Swap(i, j int) {
	h.Orders[i], h.Orders[j] = h.Orders[j], h.Orders[i]
	h.Orders[i].Index = i
	h.Orders[j].Index = j
}

// Push adds an order to the heap
func (h *OrderHeap) Push(x interface{}) {
	n := len(h.Orders)
	order := x.(*Order)
	order.Index = n
	h.Orders = append(h.Orders, order)
}

// Pop removes and returns the top order from the heap
func (h *OrderHeap) Pop() interface{} {
	old := h.Orders
	n := len(old)
	order := old[n-1]
	old[n-1] = nil  // avoid memory leak
	order.Index = -1 // for safety
	h.Orders = old[0 : n-1]
	return order
}

// Peek returns the top order from the heap without removing it
func (h *OrderHeap) Peek() *Order {
	if len(h.Orders) == 0 {
		return nil
	}
	return h.Orders[0]
}

// OrderBook represents an order book for a symbol
type OrderBook struct {
	// Symbol
	Symbol string

	// Buy and sell orders
	BuyOrders  *OrderHeap
	SellOrders *OrderHeap

	// Stop orders
	BuyStopOrders  *OrderHeap
	SellStopOrders *OrderHeap

	// Order map for quick lookup
	OrderMap map[string]*Order

	// Last price
	LastPrice float64

	// Mutex for thread safety
	mu sync.RWMutex

	// Logger
	logger *zap.Logger
}

// NewOrderBook creates a new order book
func NewOrderBook(symbol string, logger *zap.Logger) *OrderBook {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create buy orders heap (highest price first)
	buyOrders := &OrderHeap{
		Orders: make([]*Order, 0),
		Less: func(i, j int) bool {
			h := buyOrders.Orders
			// Higher price has higher priority
			if h[i].Price != h[j].Price {
				return h[i].Price > h[j].Price
			}
			// Earlier timestamp has higher priority
			return h[i].Timestamp.Before(h[j].Timestamp)
		},
	}

	// Create sell orders heap (lowest price first)
	sellOrders := &OrderHeap{
		Orders: make([]*Order, 0),
		Less: func(i, j int) bool {
			h := sellOrders.Orders
			// Lower price has higher priority
			if h[i].Price != h[j].Price {
				return h[i].Price < h[j].Price
			}
			// Earlier timestamp has higher priority
			return h[i].Timestamp.Before(h[j].Timestamp)
		},
	}

	// Create buy stop orders heap (lowest stop price first)
	buyStopOrders := &OrderHeap{
		Orders: make([]*Order, 0),
		Less: func(i, j int) bool {
			h := buyStopOrders.Orders
			// Lower stop price has higher priority
			if h[i].StopPrice != h[j].StopPrice {
				return h[i].StopPrice < h[j].StopPrice
			}
			// Earlier timestamp has higher priority
			return h[i].Timestamp.Before(h[j].Timestamp)
		},
	}

	// Create sell stop orders heap (highest stop price first)
	sellStopOrders := &OrderHeap{
		Orders: make([]*Order, 0),
		Less: func(i, j int) bool {
			h := sellStopOrders.Orders
			// Higher stop price has higher priority
			if h[i].StopPrice != h[j].StopPrice {
				return h[i].StopPrice > h[j].StopPrice
			}
			// Earlier timestamp has higher priority
			return h[i].Timestamp.Before(h[j].Timestamp)
		},
	}

	// Initialize the heaps
	heap.Init(buyOrders)
	heap.Init(sellOrders)
	heap.Init(buyStopOrders)
	heap.Init(sellStopOrders)

	return &OrderBook{
		Symbol:         symbol,
		BuyOrders:      buyOrders,
		SellOrders:     sellOrders,
		BuyStopOrders:  buyStopOrders,
		SellStopOrders: sellStopOrders,
		OrderMap:       make(map[string]*Order),
		LastPrice:      0,
		logger:         logger,
	}
}

// AddOrder adds an order to the order book
func (ob *OrderBook) AddOrder(order *Order) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Check if order already exists
	if _, exists := ob.OrderMap[order.ID]; exists {
		return fmt.Errorf("order %s already exists", order.ID)
	}

	// Add order to the map
	ob.OrderMap[order.ID] = order

	// Add order to the appropriate heap
	switch {
	case order.Type == OrderTypeLimit && order.Side == OrderSideBuy:
		heap.Push(ob.BuyOrders, order)
	case order.Type == OrderTypeLimit && order.Side == OrderSideSell:
		heap.Push(ob.SellOrders, order)
	case (order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket) && order.Side == OrderSideBuy:
		heap.Push(ob.BuyStopOrders, order)
	case (order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket) && order.Side == OrderSideSell:
		heap.Push(ob.SellStopOrders, order)
	case order.Type == OrderTypeMarket:
		// Market orders are executed immediately
		return ob.executeMarketOrder(order)
	default:
		return fmt.Errorf("invalid order type: %s", order.Type)
	}

	ob.logger.Debug("Added order to order book",
		zap.String("symbol", ob.Symbol),
		zap.String("orderID", order.ID),
		zap.String("type", string(order.Type)),
		zap.String("side", string(order.Side)),
		zap.Float64("price", order.Price),
		zap.Float64("size", order.Size),
	)

	// Check for matching orders
	if order.Type == OrderTypeLimit {
		ob.matchOrders()
	}

	// Check for triggered stop orders
	ob.checkStopOrders()

	return nil
}

// CancelOrder cancels an order in the order book
func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Find the order
	order, exists := ob.OrderMap[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	// Check if order can be cancelled
	if order.Status != OrderStatusNew && order.Status != OrderStatusPartiallyFilled {
		return fmt.Errorf("order %s cannot be cancelled: %s", orderID, order.Status)
	}

	// Remove order from the map
	delete(ob.OrderMap, orderID)

	// Remove order from the appropriate heap
	switch {
	case order.Type == OrderTypeLimit && order.Side == OrderSideBuy:
		if order.Index >= 0 && order.Index < len(ob.BuyOrders.Orders) {
			heap.Remove(ob.BuyOrders, order.Index)
		}
	case order.Type == OrderTypeLimit && order.Side == OrderSideSell:
		if order.Index >= 0 && order.Index < len(ob.SellOrders.Orders) {
			heap.Remove(ob.SellOrders, order.Index)
		}
	case (order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket) && order.Side == OrderSideBuy:
		if order.Index >= 0 && order.Index < len(ob.BuyStopOrders.Orders) {
			heap.Remove(ob.BuyStopOrders, order.Index)
		}
	case (order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket) && order.Side == OrderSideSell:
		if order.Index >= 0 && order.Index < len(ob.SellStopOrders.Orders) {
			heap.Remove(ob.SellStopOrders, order.Index)
		}
	}

	// Update order status
	order.Status = OrderStatusCancelled

	ob.logger.Debug("Cancelled order",
		zap.String("symbol", ob.Symbol),
		zap.String("orderID", order.ID),
		zap.String("type", string(order.Type)),
		zap.String("side", string(order.Side)),
		zap.Float64("price", order.Price),
		zap.Float64("size", order.Size),
	)

	return nil
}

// GetOrder gets an order from the order book
func (ob *OrderBook) GetOrder(orderID string) (*Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	// Find the order
	order, exists := ob.OrderMap[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	// Create a copy to avoid race conditions
	orderCopy := *order

	return &orderCopy, nil
}

// GetOrderBook gets the order book
func (ob *OrderBook) GetOrderBook(depth int) map[string]interface{} {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	// Create a copy of the order book
	result := make(map[string]interface{})
	result["symbol"] = ob.Symbol
	result["lastPrice"] = ob.LastPrice

	// Get buy orders
	buyOrders := make([]map[string]interface{}, 0, depth)
	for i := 0; i < len(ob.BuyOrders.Orders) && i < depth; i++ {
		order := ob.BuyOrders.Orders[i]
		buyOrders = append(buyOrders, map[string]interface{}{
			"price": order.Price,
			"size":  order.RemainingSize,
		})
	}
	result["buyOrders"] = buyOrders

	// Get sell orders
	sellOrders := make([]map[string]interface{}, 0, depth)
	for i := 0; i < len(ob.SellOrders.Orders) && i < depth; i++ {
		order := ob.SellOrders.Orders[i]
		sellOrders = append(sellOrders, map[string]interface{}{
			"price": order.Price,
			"size":  order.RemainingSize,
		})
	}
	result["sellOrders"] = sellOrders

	return result
}

// executeMarketOrder executes a market order
func (ob *OrderBook) executeMarketOrder(order *Order) error {
	// Check if there are matching orders
	var matchingOrders *OrderHeap
	if order.Side == OrderSideBuy {
		matchingOrders = ob.SellOrders
	} else {
		matchingOrders = ob.BuyOrders
	}

	// Execute the market order
	remainingSize := order.Size
	for remainingSize > 0 && matchingOrders.Len() > 0 {
		// Get the best matching order
		matchingOrder := matchingOrders.Peek()
		if matchingOrder == nil {
			break
		}

		// Calculate the matched size
		matchedSize := min(remainingSize, matchingOrder.RemainingSize)
		remainingSize -= matchedSize
		matchingOrder.RemainingSize -= matchedSize

		// Update the last price
		ob.LastPrice = matchingOrder.Price

		// Create a trade
		ob.createTrade(order, matchingOrder, matchedSize, matchingOrder.Price)

		// Remove the matching order if it's fully filled
		if matchingOrder.RemainingSize == 0 {
			heap.Pop(matchingOrders)
			matchingOrder.Status = OrderStatusFilled
			delete(ob.OrderMap, matchingOrder.ID)
		} else {
			matchingOrder.Status = OrderStatusPartiallyFilled
		}
	}

	// Update the order status
	if remainingSize == 0 {
		order.Status = OrderStatusFilled
	} else if remainingSize < order.Size {
		order.Status = OrderStatusPartiallyFilled
		order.RemainingSize = remainingSize
	} else {
		order.Status = OrderStatusRejected
		return fmt.Errorf("market order could not be fully executed")
	}

	return nil
}

// matchOrders matches buy and sell orders
func (ob *OrderBook) matchOrders() {
	// Match orders until there are no more matches
	for ob.BuyOrders.Len() > 0 && ob.SellOrders.Len() > 0 {
		// Get the best buy and sell orders
		bestBuy := ob.BuyOrders.Peek()
		bestSell := ob.SellOrders.Peek()

		// Check if the orders match
		if bestBuy.Price < bestSell.Price {
			break
		}

		// Calculate the matched size
		matchedSize := min(bestBuy.RemainingSize, bestSell.RemainingSize)

		// Update the last price
		ob.LastPrice = bestSell.Price

		// Create a trade
		ob.createTrade(bestBuy, bestSell, matchedSize, bestSell.Price)

		// Update the orders
		bestBuy.RemainingSize -= matchedSize
		bestSell.RemainingSize -= matchedSize

		// Remove the orders if they're fully filled
		if bestBuy.RemainingSize == 0 {
			heap.Pop(ob.BuyOrders)
			bestBuy.Status = OrderStatusFilled
			delete(ob.OrderMap, bestBuy.ID)
		} else {
			bestBuy.Status = OrderStatusPartiallyFilled
		}

		if bestSell.RemainingSize == 0 {
			heap.Pop(ob.SellOrders)
			bestSell.Status = OrderStatusFilled
			delete(ob.OrderMap, bestSell.ID)
		} else {
			bestSell.Status = OrderStatusPartiallyFilled
		}
	}
}

// checkStopOrders checks if any stop orders should be triggered
func (ob *OrderBook) checkStopOrders() {
	// Check buy stop orders
	for ob.BuyStopOrders.Len() > 0 {
		// Get the best buy stop order
		bestBuyStop := ob.BuyStopOrders.Peek()

		// Check if the stop price is reached
		if ob.LastPrice == 0 || ob.LastPrice < bestBuyStop.StopPrice {
			break
		}

		// Remove the stop order
		heap.Pop(ob.BuyStopOrders)
		delete(ob.OrderMap, bestBuyStop.ID)

		// Convert to a limit or market order
		var newOrder *Order
		if bestBuyStop.Type == OrderTypeStopLimit {
			newOrder = NewOrder(bestBuyStop.Symbol, OrderTypeLimit, bestBuyStop.Side, bestBuyStop.Price, bestBuyStop.Size, bestBuyStop.UserID)
		} else {
			newOrder = NewOrder(bestBuyStop.Symbol, OrderTypeMarket, bestBuyStop.Side, 0, bestBuyStop.Size, bestBuyStop.UserID)
		}

		// Add the new order
		ob.AddOrder(newOrder)
	}

	// Check sell stop orders
	for ob.SellStopOrders.Len() > 0 {
		// Get the best sell stop order
		bestSellStop := ob.SellStopOrders.Peek()

		// Check if the stop price is reached
		if ob.LastPrice == 0 || ob.LastPrice > bestSellStop.StopPrice {
			break
		}

		// Remove the stop order
		heap.Pop(ob.SellStopOrders)
		delete(ob.OrderMap, bestSellStop.ID)

		// Convert to a limit or market order
		var newOrder *Order
		if bestSellStop.Type == OrderTypeStopLimit {
			newOrder = NewOrder(bestSellStop.Symbol, OrderTypeLimit, bestSellStop.Side, bestSellStop.Price, bestSellStop.Size, bestSellStop.UserID)
		} else {
			newOrder = NewOrder(bestSellStop.Symbol, OrderTypeMarket, bestSellStop.Side, 0, bestSellStop.Size, bestSellStop.UserID)
		}

		// Add the new order
		ob.AddOrder(newOrder)
	}
}

// createTrade creates a trade between two orders
func (ob *OrderBook) createTrade(buyOrder, sellOrder *Order, size, price float64) {
	ob.logger.Info("Trade executed",
		zap.String("symbol", ob.Symbol),
		zap.String("buyOrderID", buyOrder.ID),
		zap.String("sellOrderID", sellOrder.ID),
		zap.Float64("price", price),
		zap.Float64("size", size),
	)
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// OrderMatchingEngine is an order matching engine
type OrderMatchingEngine struct {
	// Order books by symbol
	OrderBooks map[string]*OrderBook

	// Mutex for thread safety
	mu sync.RWMutex

	// Logger
	logger *zap.Logger

	// Statistics
	ordersProcessed uint64
	tradesExecuted  uint64
	
	// Memory management
	cleanupInterval time.Duration
	lastCleanup     time.Time
}

// NewOrderMatchingEngine creates a new order matching engine
func NewOrderMatchingEngine(logger *zap.Logger) *OrderMatchingEngine {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &OrderMatchingEngine{
		OrderBooks:      make(map[string]*OrderBook),
		logger:          logger,
		cleanupInterval: 1 * time.Hour,
		lastCleanup:     time.Now(),
	}
}

// GetOrderBook gets an order book for a symbol
func (e *OrderMatchingEngine) GetOrderBook(symbol string) *OrderBook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if the order book exists
	orderBook, exists := e.OrderBooks[symbol]
	if !exists {
		return nil
	}

	return orderBook
}

// CreateOrderBook creates an order book for a symbol
func (e *OrderMatchingEngine) CreateOrderBook(symbol string) *OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if the order book already exists
	if orderBook, exists := e.OrderBooks[symbol]; exists {
		return orderBook
	}

	// Create a new order book
	orderBook := NewOrderBook(symbol, e.logger)
	e.OrderBooks[symbol] = orderBook

	e.logger.Info("Created order book",
		zap.String("symbol", symbol),
	)

	return orderBook
}

// PlaceOrder places an order in the order matching engine
func (e *OrderMatchingEngine) PlaceOrder(order *Order) error {
	// Increment orders processed
	atomic.AddUint64(&e.ordersProcessed, 1)

	// Get the order book
	orderBook := e.GetOrderBook(order.Symbol)
	if orderBook == nil {
		// Create a new order book
		orderBook = e.CreateOrderBook(order.Symbol)
	}

	// Add the order to the order book
	return orderBook.AddOrder(order)
}

// CancelOrder cancels an order in the order matching engine
func (e *OrderMatchingEngine) CancelOrder(symbol, orderID string) error {
	// Get the order book
	orderBook := e.GetOrderBook(symbol)
	if orderBook == nil {
		return fmt.Errorf("order book for symbol %s not found", symbol)
	}

	// Cancel the order
	return orderBook.CancelOrder(orderID)
}

// GetOrder gets an order from the order matching engine
func (e *OrderMatchingEngine) GetOrder(symbol, orderID string) (*Order, error) {
	// Get the order book
	orderBook := e.GetOrderBook(symbol)
	if orderBook == nil {
		return nil, fmt.Errorf("order book for symbol %s not found", symbol)
	}

	// Get the order
	return orderBook.GetOrder(orderID)
}

// GetOrderBookSnapshot gets a snapshot of an order book
func (e *OrderMatchingEngine) GetOrderBookSnapshot(symbol string, depth int) (map[string]interface{}, error) {
	// Get the order book
	orderBook := e.GetOrderBook(symbol)
	if orderBook == nil {
		return nil, fmt.Errorf("order book for symbol %s not found", symbol)
	}

	// Get the order book snapshot
	return orderBook.GetOrderBook(depth), nil
}

// GetStats gets the order matching engine statistics
func (e *OrderMatchingEngine) GetStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["ordersProcessed"] = atomic.LoadUint64(&e.ordersProcessed)
	stats["tradesExecuted"] = atomic.LoadUint64(&e.tradesExecuted)
	stats["orderBookCount"] = len(e.OrderBooks)
	stats["lastCleanup"] = e.lastCleanup

	return stats
}

// Cleanup performs cleanup operations to prevent memory leaks
func (e *OrderMatchingEngine) Cleanup(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if cleanup is needed
	if time.Since(e.lastCleanup) < e.cleanupInterval {
		return
	}

	e.logger.Info("Starting order matching engine cleanup")

	// Cleanup each order book
	for symbol, orderBook := range e.OrderBooks {
		// Skip if context is cancelled
		if ctx.Err() != nil {
			e.logger.Warn("Cleanup cancelled",
				zap.Error(ctx.Err()),
			)
			return
		}

		// Lock the order book
		orderBook.mu.Lock()

		// Remove filled and cancelled orders from the map
		for orderID, order := range orderBook.OrderMap {
			if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
				delete(orderBook.OrderMap, orderID)
			}
		}

		// Rebuild the heaps to remove any nil entries
		e.rebuildHeap(orderBook.BuyOrders)
		e.rebuildHeap(orderBook.SellOrders)
		e.rebuildHeap(orderBook.BuyStopOrders)
		e.rebuildHeap(orderBook.SellStopOrders)

		// Unlock the order book
		orderBook.mu.Unlock()

		e.logger.Debug("Cleaned up order book",
			zap.String("symbol", symbol),
		)
	}

	// Update last cleanup time
	e.lastCleanup = time.Now()

	e.logger.Info("Completed order matching engine cleanup")
}

// rebuildHeap rebuilds a heap to remove nil entries
func (e *OrderMatchingEngine) rebuildHeap(h *OrderHeap) {
	// Create a new slice without nil entries
	orders := make([]*Order, 0, len(h.Orders))
	for _, order := range h.Orders {
		if order != nil {
			order.Index = -1 // Reset index
			orders = append(orders, order)
		}
	}

	// Replace the orders slice
	h.Orders = orders

	// Rebuild the heap
	heap.Init(h)
}

// SetCleanupInterval sets the cleanup interval
func (e *OrderMatchingEngine) SetCleanupInterval(interval time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cleanupInterval = interval
}

// StartPeriodicCleanup starts periodic cleanup
func (e *OrderMatchingEngine) StartPeriodicCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(e.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.Cleanup(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

