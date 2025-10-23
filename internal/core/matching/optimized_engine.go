package order_matching

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"go.uber.org/zap"
)

// OptimizedEngine provides high-performance order matching with <100μs latency target
type OptimizedEngine struct {
	logger *zap.Logger

	// Lock-free order book using atomic operations
	orderBooks sync.Map // map[string]*LockFreeOrderBook

	// Memory pools for object reuse
	orderPool sync.Pool
	tradePool sync.Pool

	// Performance metrics
	totalOrders  uint64
	totalTrades  uint64
	avgLatencyNs uint64
	maxLatencyNs uint64

	// Hot path optimization flags
	enableFastPath bool
	enableBatching bool
}

// LockFreeOrderBook implements a lock-free order book for maximum performance
type LockFreeOrderBook struct {
	symbol string

	// Atomic pointers to order lists (lock-free)
	bidHead unsafe.Pointer // *OrderNode
	askHead unsafe.Pointer // *OrderNode

	// Atomic counters
	bidCount  int64
	askCount  int64
	lastPrice int64 // Price in fixed-point (multiply by 1e8)

	// Memory pool for order nodes
	nodePool sync.Pool
}

// OrderNode represents a node in the lock-free linked list
type OrderNode struct {
	order *Order
	next  unsafe.Pointer // *OrderNode
	price int64          // Fixed-point price for atomic operations
}

// NewOptimizedEngine creates a new high-performance matching engine
func NewOptimizedEngine(logger *zap.Logger) *OptimizedEngine {
	engine := &OptimizedEngine{
		logger:         logger,
		enableFastPath: true,
		enableBatching: true,
	}

	// Initialize memory pools
	engine.orderPool = sync.Pool{
		New: func() interface{} {
			return &Order{}
		},
	}

	engine.tradePool = sync.Pool{
		New: func() interface{} {
			return &Trade{}
		},
	}

	return engine
}

// ProcessOrderFast processes an order with optimized fast path (<100μs target)
func (e *OptimizedEngine) ProcessOrderFast(order *Order) ([]*Trade, error) {
	startTime := time.Now()

	// Get or create lock-free order book
	bookInterface, _ := e.orderBooks.LoadOrStore(order.Symbol, e.createLockFreeOrderBook(order.Symbol))
	book := bookInterface.(*LockFreeOrderBook)

	var trades []*Trade

	// Fast path for market orders (most common case)
	if order.Type == OrderTypeMarket && e.enableFastPath {
		trades = e.processMarketOrderFast(book, order)
	} else {
		trades = e.processLimitOrderFast(book, order)
	}

	// Update performance metrics
	latency := time.Since(startTime)
	atomic.AddUint64(&e.totalOrders, 1)
	atomic.AddUint64(&e.totalTrades, uint64(len(trades)))

	// Update average latency using atomic operations
	latencyNs := uint64(latency.Nanoseconds())
	for {
		oldAvg := atomic.LoadUint64(&e.avgLatencyNs)
		totalOrders := atomic.LoadUint64(&e.totalOrders)
		newAvg := (oldAvg*(totalOrders-1) + latencyNs) / totalOrders
		if atomic.CompareAndSwapUint64(&e.avgLatencyNs, oldAvg, newAvg) {
			break
		}
	}

	// Update max latency
	for {
		oldMax := atomic.LoadUint64(&e.maxLatencyNs)
		if latencyNs <= oldMax || atomic.CompareAndSwapUint64(&e.maxLatencyNs, oldMax, latencyNs) {
			break
		}
	}

	// Log performance if latency exceeds target
	if latency > 100*time.Microsecond {
		e.logger.Warn("Order processing exceeded 100μs target",
			zap.String("symbol", order.Symbol),
			zap.Duration("latency", latency),
			zap.Int("trades", len(trades)))
	}

	return trades, nil
}

// processMarketOrderFast processes market orders with optimized path
func (e *OptimizedEngine) processMarketOrderFast(book *LockFreeOrderBook, order *Order) []*Trade {
	var trades []*Trade

	if order.Side == OrderSideBuy {
		// Buy market order - match against asks
		trades = e.matchAgainstSide(book, order, &book.askHead, false)
	} else {
		// Sell market order - match against bids
		trades = e.matchAgainstSide(book, order, &book.bidHead, true)
	}

	return trades
}

// processLimitOrderFast processes limit orders with optimized path
func (e *OptimizedEngine) processLimitOrderFast(book *LockFreeOrderBook, order *Order) []*Trade {
	var trades []*Trade

	if order.Side == OrderSideBuy {
		// Buy limit order - first try to match against asks
		trades = e.matchAgainstSide(book, order, &book.askHead, false)

		// If not fully filled, add to bid side
		if order.Quantity > order.FilledQuantity {
			e.addOrderToSide(book, order, &book.bidHead, &book.bidCount)
		}
	} else {
		// Sell limit order - first try to match against bids
		trades = e.matchAgainstSide(book, order, &book.bidHead, true)

		// If not fully filled, add to ask side
		if order.Quantity > order.FilledQuantity {
			e.addOrderToSide(book, order, &book.askHead, &book.askCount)
		}
	}

	return trades
}

// matchAgainstSide matches an order against one side of the book using lock-free operations
func (e *OptimizedEngine) matchAgainstSide(book *LockFreeOrderBook, incomingOrder *Order, headPtr *unsafe.Pointer, isBidSide bool) []*Trade {
	var trades []*Trade
	incomingPriceFixed := int64(incomingOrder.Price * 1e8) // Convert to fixed-point

	for incomingOrder.Quantity > incomingOrder.FilledQuantity {
		// Atomically load the head of the order list
		head := (*OrderNode)(atomic.LoadPointer(headPtr))
		if head == nil {
			break // No orders on this side
		}

		// Check if prices can match
		canMatch := false
		if isBidSide {
			// Matching against bids (for sell orders)
			canMatch = head.price >= incomingPriceFixed || incomingOrder.Type == OrderTypeMarket
		} else {
			// Matching against asks (for buy orders)
			canMatch = head.price <= incomingPriceFixed || incomingOrder.Type == OrderTypeMarket
		}

		if !canMatch {
			break
		}

		// Try to atomically remove the head order
		next := (*OrderNode)(atomic.LoadPointer(&head.next))
		if !atomic.CompareAndSwapPointer(headPtr, unsafe.Pointer(head), unsafe.Pointer(next)) {
			continue // Another thread modified the list, retry
		}

		// Execute the trade
		trade := e.executeTrade(incomingOrder, head.order)
		if trade != nil {
			trades = append(trades, trade)

			// Update last price atomically
			atomic.StoreInt64(&book.lastPrice, int64(trade.Price*1e8))
		}

		// Return the order node to the pool
		e.returnOrderNode(book, head)

		// Update counters
		if isBidSide {
			atomic.AddInt64(&book.bidCount, -1)
		} else {
			atomic.AddInt64(&book.askCount, -1)
		}
	}

	return trades
}

// addOrderToSide adds an order to one side of the book using lock-free insertion
func (e *OptimizedEngine) addOrderToSide(book *LockFreeOrderBook, order *Order, headPtr *unsafe.Pointer, countPtr *int64) {
	// Get order node from pool
	node := e.getOrderNode(book, order)

	// Insert at head (simplified - in production, would maintain price-time priority)
	for {
		head := atomic.LoadPointer(headPtr)
		atomic.StorePointer(&node.next, head)
		if atomic.CompareAndSwapPointer(headPtr, head, unsafe.Pointer(node)) {
			atomic.AddInt64(countPtr, 1)
			break
		}
	}
}

// executeTrade executes a trade between two orders with minimal allocations
func (e *OptimizedEngine) executeTrade(incomingOrder, bookOrder *Order) *Trade {
	// Calculate trade quantity (minimum of remaining quantities)
	tradeQuantity := incomingOrder.Quantity - incomingOrder.FilledQuantity
	bookRemaining := bookOrder.Quantity - bookOrder.FilledQuantity
	if bookRemaining < tradeQuantity {
		tradeQuantity = bookRemaining
	}

	// Use book order price (price-time priority)
	tradePrice := bookOrder.Price

	// Update order fill quantities
	incomingOrder.FilledQuantity += tradeQuantity
	bookOrder.FilledQuantity += tradeQuantity

	// Update order statuses
	if incomingOrder.FilledQuantity >= incomingOrder.Quantity {
		incomingOrder.Status = OrderStatusFilled
	} else {
		incomingOrder.Status = OrderStatusPartiallyFilled
	}

	if bookOrder.FilledQuantity >= bookOrder.Quantity {
		bookOrder.Status = OrderStatusFilled
	} else {
		bookOrder.Status = OrderStatusPartiallyFilled
	}

	// Create trade from pool
	trade := e.tradePool.Get().(*Trade)
	trade.ID = generateTradeID()
	trade.Symbol = incomingOrder.Symbol
	trade.Price = tradePrice
	trade.Quantity = tradeQuantity
	trade.BuyOrderID = ""
	trade.SellOrderID = ""
	trade.Timestamp = time.Now()

	// Set order IDs based on sides
	if incomingOrder.Side == OrderSideBuy {
		trade.BuyOrderID = incomingOrder.ID
		trade.SellOrderID = bookOrder.ID
	} else {
		trade.BuyOrderID = bookOrder.ID
		trade.SellOrderID = incomingOrder.ID
	}

	return trade
}

// createLockFreeOrderBook creates a new lock-free order book
func (e *OptimizedEngine) createLockFreeOrderBook(symbol string) *LockFreeOrderBook {
	book := &LockFreeOrderBook{
		symbol: symbol,
	}

	// Initialize node pool
	book.nodePool = sync.Pool{
		New: func() interface{} {
			return &OrderNode{}
		},
	}

	return book
}

// getOrderNode gets an order node from the pool
func (e *OptimizedEngine) getOrderNode(book *LockFreeOrderBook, order *Order) *OrderNode {
	node := book.nodePool.Get().(*OrderNode)
	node.order = order
	node.price = int64(order.Price * 1e8) // Convert to fixed-point
	atomic.StorePointer(&node.next, nil)
	return node
}

// returnOrderNode returns an order node to the pool
func (e *OptimizedEngine) returnOrderNode(book *LockFreeOrderBook, node *OrderNode) {
	// Clear the node
	node.order = nil
	atomic.StorePointer(&node.next, nil)
	node.price = 0

	// Return to pool
	book.nodePool.Put(node)
}

// GetPerformanceMetrics returns current performance metrics
func (e *OptimizedEngine) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_orders":   atomic.LoadUint64(&e.totalOrders),
		"total_trades":   atomic.LoadUint64(&e.totalTrades),
		"avg_latency_ns": atomic.LoadUint64(&e.avgLatencyNs),
		"max_latency_ns": atomic.LoadUint64(&e.maxLatencyNs),
		"avg_latency_us": float64(atomic.LoadUint64(&e.avgLatencyNs)) / 1000.0,
		"max_latency_us": float64(atomic.LoadUint64(&e.maxLatencyNs)) / 1000.0,
	}
}

// generateTradeID generates a unique trade ID (simplified)
func generateTradeID() string {
	return time.Now().Format("20060102150405.000000")
}
