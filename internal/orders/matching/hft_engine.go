package order_matching

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HFTEngine represents a high-frequency trading optimized order matching engine
type HFTEngine struct {
	// OrderBooks is a map of symbol to order book (lock-free access)
	orderBooks unsafe.Pointer // map[string]*HFTOrderBook

	// Trade channel with buffering for high throughput
	TradeChannel chan *Trade

	// Order pools for zero-allocation order processing
	fastOrderPool *pool.FastOrderPool
	tradePool     *pool.TradePool

	// Performance metrics
	ordersProcessed uint64
	tradesExecuted  uint64
	avgLatency      uint64 // nanoseconds

	// Logger
	logger *zap.Logger

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Worker pool for parallel processing
	workerPool chan struct{}

	// Lock-free statistics
	stats *EngineStats
}

// EngineStats represents engine performance statistics
type EngineStats struct {
	OrdersProcessed   uint64
	TradesExecuted    uint64
	AvgLatencyNanos   uint64
	MaxLatencyNanos   uint64
	MinLatencyNanos   uint64
	TotalLatencyNanos uint64
	LastUpdateTime    int64 // Unix nanoseconds
}

// HFTOrderBook represents a high-frequency trading optimized order book
type HFTOrderBook struct {
	Symbol string

	// Lock-free order book using atomic operations
	bids unsafe.Pointer // *PriceLevelTree
	asks unsafe.Pointer // *PriceLevelTree

	// Order lookup map with RWMutex for better read performance
	orders sync.Map // map[string]*Order

	// Last trade price (atomic)
	lastPrice uint64 // float64 as uint64 for atomic operations

	// Performance counters
	orderCount  uint64
	tradeCount  uint64
	lastUpdated int64 // Unix nanoseconds

	// Logger
	logger *zap.Logger
}

// PriceLevelTree represents a price level tree for efficient order book management
type PriceLevelTree struct {
	// Root node of the tree
	root *PriceLevelNode

	// Side of the tree (buy or sell)
	side OrderSide

	// Node count for quick size checks
	nodeCount uint32

	// RWMutex for concurrent access
	mu sync.RWMutex
}

// PriceLevelNode represents a node in the price level tree
type PriceLevelNode struct {
	// Price level
	price float64

	// Orders at this price level (FIFO queue)
	orders []*Order

	// Total quantity at this price level
	totalQuantity float64

	// Tree structure
	left   *PriceLevelNode
	right  *PriceLevelNode
	parent *PriceLevelNode
	height int

	// Order count at this level
	orderCount uint32
}

// Use FastOrder from pools package
type FastOrder = pool.FastOrder

// NewHFTEngine creates a new HFT-optimized order matching engine
func NewHFTEngine(logger *zap.Logger) *HFTEngine {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize order books map
	orderBooksMap := make(map[string]*HFTOrderBook)

	engine := &HFTEngine{
		TradeChannel:  make(chan *Trade, 10000), // Large buffer for high throughput
		fastOrderPool: pool.NewFastOrderPool(),  // Fast order pool for zero-allocation processing
		tradePool:     pool.NewTradePool(),
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		workerPool:    make(chan struct{}, runtime.NumCPU()*2), // 2x CPU cores
		stats: &EngineStats{
			MinLatencyNanos: ^uint64(0), // Max uint64 value
		},
	}

	// Store the map atomically
	atomic.StorePointer(&engine.orderBooks, unsafe.Pointer(&orderBooksMap))

	// Start performance monitoring
	go engine.monitorPerformance()

	// Start trade processor
	go engine.processTradesAsync()

	return engine
}

// PlaceOrderFast places an order with HFT optimizations
func (e *HFTEngine) PlaceOrderFast(order *Order) ([]*Trade, error) {
	startTime := time.Now()

	// Get or create order book
	orderBook := e.getOrCreateOrderBook(order.Symbol)

	// Convert to fast order for better performance
	fastOrder := e.fastOrderPool.Get()
	defer e.fastOrderPool.Put(fastOrder) // Return to pool when done

	// Copy order data to fast order
	fastOrder.Order.ID = order.ID
	fastOrder.Order.Symbol = order.Symbol
	fastOrder.Order.Side = string(order.Side)
	fastOrder.Order.Price = order.Price
	fastOrder.Order.Quantity = order.Quantity
	fastOrder.Order.Type = string(order.Type)
	fastOrder.PriceInt64 = int64(order.Price * 100000000) // 8 decimal places precision
	fastOrder.QuantityInt64 = int64(order.Quantity * 100000000)
	fastOrder.CreatedAtNano = startTime.UnixNano()
	fastOrder.UpdatedAtNano = startTime.UnixNano()

	// Process order
	trades, err := orderBook.processOrderFast(fastOrder)

	// Update performance metrics
	latency := uint64(time.Since(startTime).Nanoseconds())
	e.updateLatencyStats(latency)
	atomic.AddUint64(&e.ordersProcessed, 1)

	// Send trades to channel asynchronously
	if len(trades) > 0 {
		go func() {
			for _, trade := range trades {
				select {
				case e.TradeChannel <- trade:
					atomic.AddUint64(&e.tradesExecuted, 1)
				default:
					e.logger.Warn("Trade channel full, dropping trade",
						zap.String("trade_id", trade.ID),
						zap.String("symbol", trade.Symbol))
				}
			}
		}()
	}

	return trades, err
}

// getOrCreateOrderBook gets or creates an order book for a symbol
func (e *HFTEngine) getOrCreateOrderBook(symbol string) *HFTOrderBook {
	// Load current map
	orderBooksPtr := atomic.LoadPointer(&e.orderBooks)
	orderBooksMap := (*map[string]*HFTOrderBook)(orderBooksPtr)

	// Check if order book exists
	if orderBook, exists := (*orderBooksMap)[symbol]; exists {
		return orderBook
	}

	// Create new order book
	newOrderBook := &HFTOrderBook{
		Symbol:      symbol,
		logger:      e.logger,
		lastUpdated: time.Now().UnixNano(),
	}

	// Initialize price level trees
	bidsTree := &PriceLevelTree{side: OrderSideBuy}
	asksTree := &PriceLevelTree{side: OrderSideSell}
	atomic.StorePointer(&newOrderBook.bids, unsafe.Pointer(bidsTree))
	atomic.StorePointer(&newOrderBook.asks, unsafe.Pointer(asksTree))

	// Create new map with the new order book
	newOrderBooksMap := make(map[string]*HFTOrderBook)
	for k, v := range *orderBooksMap {
		newOrderBooksMap[k] = v
	}
	newOrderBooksMap[symbol] = newOrderBook

	// Atomically update the map
	atomic.StorePointer(&e.orderBooks, unsafe.Pointer(&newOrderBooksMap))

	return newOrderBook
}

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
	for order.Order.FilledQuantity < order.Order.Quantity && asksTree.root != nil {
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
	for order.Order.FilledQuantity < order.Order.Quantity && bidsTree.root != nil {
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
	for order.Order.FilledQuantity < order.Order.Quantity {
		bestAsk := asksTree.findBestPrice()
		if bestAsk == nil || bestAsk.price > order.Price {
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

	// Add remaining quantity to order book if not fully filled
	if order.Order.FilledQuantity < order.Order.Quantity {
		bidsPtr := atomic.LoadPointer(&ob.bids)
		bidsTree := (*PriceLevelTree)(bidsPtr)
		bidsTree.addOrder(&order.Order)
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
	for order.Order.FilledQuantity < order.Order.Quantity {
		bestBid := bidsTree.findBestPrice()
		if bestBid == nil || bestBid.price < order.Price {
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

	// Add remaining quantity to order book if not fully filled
	if order.Order.FilledQuantity < order.Order.Quantity {
		asksPtr := atomic.LoadPointer(&ob.asks)
		asksTree := (*PriceLevelTree)(asksPtr)
		asksTree.addOrder(&order.Order)
	}

	return trades
}

// executeTradeOptimized executes a trade with optimizations
func (ob *HFTOrderBook) executeTradeOptimized(taker *FastOrder, maker *Order) *Trade {
	// Calculate trade quantity (minimum of remaining quantities)
	takerRemaining := taker.Quantity - taker.FilledQuantity
	makerRemaining := maker.Quantity - maker.FilledQuantity
	tradeQuantity := takerRemaining
	if makerRemaining < tradeQuantity {
		tradeQuantity = makerRemaining
	}

	// Trade price is the maker's price (price-time priority)
	tradePrice := maker.Price

	// Update order quantities
	taker.FilledQuantity += tradeQuantity
	maker.FilledQuantity += tradeQuantity

	// Update order statuses
	if taker.FilledQuantity >= taker.Quantity {
		taker.Status = OrderStatusFilled
	} else {
		taker.Status = OrderStatusPartiallyFilled
	}

	if maker.FilledQuantity >= maker.Quantity {
		maker.Status = OrderStatusFilled
	} else {
		maker.Status = OrderStatusPartiallyFilled
	}

	// Update timestamps
	now := time.Now()
	taker.UpdatedAt = now
	maker.UpdatedAt = now
	taker.UpdatedAtNano = now.UnixNano()

	// Create trade with pre-allocated ID
	trade := &Trade{
		ID:        uuid.New().String(),
		Symbol:    ob.Symbol,
		Price:     tradePrice,
		Quantity:  tradeQuantity,
		Timestamp: now,
		TakerSide: taker.Side,
		MakerSide: maker.Side,
	}

	// Set order IDs
	if taker.Side == OrderSideBuy {
		trade.BuyOrderID = taker.ID
		trade.SellOrderID = maker.ID
	} else {
		trade.BuyOrderID = maker.ID
		trade.SellOrderID = taker.ID
	}

	return trade
}

// Price level tree operations

// findBestPrice finds the best price in the tree
func (tree *PriceLevelTree) findBestPrice() *PriceLevelNode {
	if tree.root == nil {
		return nil
	}

	if tree.side == OrderSideBuy {
		// For buy orders, find maximum price (rightmost node)
		node := tree.root
		for node.right != nil {
			node = node.right
		}
		return node
	} else {
		// For sell orders, find minimum price (leftmost node)
		node := tree.root
		for node.left != nil {
			node = node.left
		}
		return node
	}
}

// addOrder adds an order to the price level tree
func (tree *PriceLevelTree) addOrder(order *Order) {
	tree.mu.Lock()
	defer tree.mu.Unlock()

	if tree.root == nil {
		tree.root = &PriceLevelNode{
			price:         order.Price,
			orders:        []*Order{order},
			totalQuantity: order.Quantity,
			orderCount:    1,
		}
		atomic.AddUint32(&tree.nodeCount, 1)
		return
	}

	// Find or create price level
	node := tree.findOrCreatePriceLevel(order.Price)
	node.orders = append(node.orders, order)
	node.totalQuantity += order.Quantity
	node.orderCount++
}

// findOrCreatePriceLevel finds or creates a price level node
func (tree *PriceLevelTree) findOrCreatePriceLevel(price float64) *PriceLevelNode {
	node := tree.root

	for {
		if price == node.price {
			return node
		} else if price < node.price {
			if node.left == nil {
				node.left = &PriceLevelNode{
					price:         price,
					orders:        make([]*Order, 0, 4),
					totalQuantity: 0,
					parent:        node,
					orderCount:    0,
				}
				atomic.AddUint32(&tree.nodeCount, 1)
				return node.left
			}
			node = node.left
		} else {
			if node.right == nil {
				node.right = &PriceLevelNode{
					price:         price,
					orders:        make([]*Order, 0, 4),
					totalQuantity: 0,
					parent:        node,
					orderCount:    0,
				}
				atomic.AddUint32(&tree.nodeCount, 1)
				return node.right
			}
			node = node.right
		}
	}
}

// removeNode removes a node from the tree
func (tree *PriceLevelTree) removeNode(node *PriceLevelNode) {
	// Implementation of AVL tree node removal
	// This is a simplified version - full AVL implementation would include rebalancing

	if node.left == nil && node.right == nil {
		// Leaf node
		if node.parent != nil {
			if node.parent.left == node {
				node.parent.left = nil
			} else {
				node.parent.right = nil
			}
		} else {
			tree.root = nil
		}
	} else if node.left == nil {
		// Only right child
		if node.parent != nil {
			if node.parent.left == node {
				node.parent.left = node.right
			} else {
				node.parent.right = node.right
			}
			node.right.parent = node.parent
		} else {
			tree.root = node.right
			node.right.parent = nil
		}
	} else if node.right == nil {
		// Only left child
		if node.parent != nil {
			if node.parent.left == node {
				node.parent.left = node.left
			} else {
				node.parent.right = node.left
			}
			node.left.parent = node.parent
		} else {
			tree.root = node.left
			node.left.parent = nil
		}
	} else {
		// Two children - replace with inorder successor
		successor := node.right
		for successor.left != nil {
			successor = successor.left
		}

		// Copy successor data to current node
		node.price = successor.price
		node.orders = successor.orders
		node.totalQuantity = successor.totalQuantity
		node.orderCount = successor.orderCount

		// Remove successor (which has at most one child)
		tree.removeNode(successor)
		return // Don't decrement node count twice
	}

	atomic.AddUint32(&tree.nodeCount, ^uint32(0)) // Subtract 1
}

// Performance monitoring and statistics

// updateLatencyStats updates latency statistics atomically
func (e *HFTEngine) updateLatencyStats(latency uint64) {
	// Update average latency using exponential moving average
	currentAvg := atomic.LoadUint64(&e.stats.AvgLatencyNanos)
	newAvg := (currentAvg*9 + latency) / 10 // 90% weight to previous, 10% to current
	atomic.StoreUint64(&e.stats.AvgLatencyNanos, newAvg)

	// Update min latency
	for {
		currentMin := atomic.LoadUint64(&e.stats.MinLatencyNanos)
		if latency >= currentMin {
			break
		}
		if atomic.CompareAndSwapUint64(&e.stats.MinLatencyNanos, currentMin, latency) {
			break
		}
	}

	// Update max latency
	for {
		currentMax := atomic.LoadUint64(&e.stats.MaxLatencyNanos)
		if latency <= currentMax {
			break
		}
		if atomic.CompareAndSwapUint64(&e.stats.MaxLatencyNanos, currentMax, latency) {
			break
		}
	}

	// Update total latency
	atomic.AddUint64(&e.stats.TotalLatencyNanos, latency)
	atomic.StoreInt64(&e.stats.LastUpdateTime, time.Now().UnixNano())
}

// monitorPerformance monitors engine performance
func (e *HFTEngine) monitorPerformance() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			stats := e.GetStats()
			e.logger.Info("Engine performance stats",
				zap.Uint64("orders_processed", stats.OrdersProcessed),
				zap.Uint64("trades_executed", stats.TradesExecuted),
				zap.Uint64("avg_latency_nanos", stats.AvgLatencyNanos),
				zap.Uint64("min_latency_nanos", stats.MinLatencyNanos),
				zap.Uint64("max_latency_nanos", stats.MaxLatencyNanos),
				zap.Float64("avg_latency_micros", float64(stats.AvgLatencyNanos)/1000.0),
			)
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
				zap.String("taker_side", string(trade.TakerSide)),
			)
		}
	}
}

// GetStats returns current engine statistics
func (e *HFTEngine) GetStats() *EngineStats {
	return &EngineStats{
		OrdersProcessed:   atomic.LoadUint64(&e.ordersProcessed),
		TradesExecuted:    atomic.LoadUint64(&e.tradesExecuted),
		AvgLatencyNanos:   atomic.LoadUint64(&e.stats.AvgLatencyNanos),
		MaxLatencyNanos:   atomic.LoadUint64(&e.stats.MaxLatencyNanos),
		MinLatencyNanos:   atomic.LoadUint64(&e.stats.MinLatencyNanos),
		TotalLatencyNanos: atomic.LoadUint64(&e.stats.TotalLatencyNanos),
		LastUpdateTime:    atomic.LoadInt64(&e.stats.LastUpdateTime),
	}
}

// GetOrderBook returns an order book for a symbol
func (e *HFTEngine) GetOrderBook(symbol string) *HFTOrderBook {
	orderBooksPtr := atomic.LoadPointer(&e.orderBooks)
	orderBooksMap := (*map[string]*HFTOrderBook)(orderBooksPtr)

	if orderBook, exists := (*orderBooksMap)[symbol]; exists {
		return orderBook
	}

	return nil
}

// Stop gracefully stops the engine
func (e *HFTEngine) Stop() {
	e.cancel()
	close(e.TradeChannel)
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
