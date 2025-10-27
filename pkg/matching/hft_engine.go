package matching

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/abdoElHodaky/tradSys/pkg/common/pool"
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
	AvgLatencyNs      uint64
	MaxLatencyNs      uint64
	MinLatencyNs      uint64
	TotalVolumeTraded uint64
	ActiveOrders      uint64
	CancelledOrders   uint64
	RejectedOrders    uint64
	LastUpdateTime    time.Time
}

// HFTOrderBook represents a high-frequency trading optimized order book
type HFTOrderBook struct {
	Symbol string

	// Lock-free order storage using atomic operations
	buyOrders  unsafe.Pointer // *OrderLevel
	sellOrders unsafe.Pointer // *OrderLevel

	// Fast lookup maps for order management
	orderMap sync.Map // map[string]*HFTOrder

	// Performance counters
	totalOrders   uint64
	totalTrades   uint64
	totalVolume   uint64
	lastTradeTime time.Time

	// Spread tracking
	bestBid  uint64 // atomic
	bestAsk  uint64 // atomic
	spread   uint64 // atomic

	// Lock for critical sections (minimal usage)
	mu sync.RWMutex
}

// OrderLevel represents a price level in the order book
type OrderLevel struct {
	Price    uint64 // Fixed-point price representation
	Quantity uint64
	Orders   unsafe.Pointer // *HFTOrder (linked list)
	Next     unsafe.Pointer // *OrderLevel
}

// HFTOrder represents a trading order optimized for HFT
type HFTOrder struct {
	ID        string
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Price     uint64 // Fixed-point representation
	Quantity  uint64
	Filled    uint64
	Status    OrderStatus
	Timestamp time.Time
	UserID    string

	// Linked list pointers for order book
	Next *HFTOrder
	Prev *HFTOrder

	// Performance tracking
	LatencyNs uint64
}

// Use types from engine.go to avoid duplication

// Use OrderStatus constants from engine.go

// Use Trade from engine.go

// NewHFTEngine creates a new high-frequency trading engine
func NewHFTEngine(logger *zap.Logger, workerCount int) *HFTEngine {
	ctx, cancel := context.WithCancel(context.Background())
	
	engine := &HFTEngine{
		orderBooks:    unsafe.Pointer(&map[string]*HFTOrderBook{}),
		TradeChannel:  make(chan *Trade, 10000), // High-capacity buffer
		fastOrderPool: pool.NewFastOrderPool(),
		tradePool:     pool.NewTradePool(1000),
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		workerPool:    make(chan struct{}, workerCount),
		stats:         &EngineStats{LastUpdateTime: time.Now()},
	}

	// Initialize worker pool
	for i := 0; i < workerCount; i++ {
		engine.workerPool <- struct{}{}
	}

	return engine
}

// Start starts the HFT engine
func (e *HFTEngine) Start() error {
	e.logger.Info("Starting HFT matching engine",
		zap.Int("worker_count", cap(e.workerPool)),
		zap.Int("trade_channel_buffer", cap(e.TradeChannel)))

	// Start performance monitoring
	go e.performanceMonitor()

	// Start trade processing
	go e.tradeProcessor()

	return nil
}

// Stop stops the HFT engine gracefully
func (e *HFTEngine) Stop() error {
	e.logger.Info("Stopping HFT matching engine")
	e.cancel()
	
	// Close trade channel
	close(e.TradeChannel)
	
	return nil
}

// AddOrder adds an order to the matching engine
func (e *HFTEngine) AddOrder(order *HFTOrder) error {
	startTime := time.Now()
	
	// Get or create order book
	orderBook := e.getOrCreateOrderBook(order.Symbol)
	
	// Process order
	trades := e.processOrder(orderBook, order)
	
	// Send trades to channel
	for _, trade := range trades {
		select {
		case e.TradeChannel <- trade:
		default:
			e.logger.Warn("Trade channel full, dropping trade", zap.String("trade_id", trade.ID))
		}
	}
	
	// Update performance metrics
	latency := time.Since(startTime).Nanoseconds()
	atomic.AddUint64(&e.ordersProcessed, 1)
	atomic.AddUint64(&e.avgLatency, uint64(latency))
	
	order.LatencyNs = uint64(latency)
	
	return nil
}

// CancelOrder cancels an order
func (e *HFTEngine) CancelOrder(orderID string, symbol string) error {
	orderBook := e.getOrderBook(symbol)
	if orderBook == nil {
		return ErrOrderBookNotFound
	}
	
	// Find and cancel order
	if orderInterface, ok := orderBook.orderMap.Load(orderID); ok {
		order := orderInterface.(*HFTOrder)
		order.Status = OrderStatusCancelled
		orderBook.orderMap.Delete(orderID)
		
		// Remove from order book levels
		e.removeOrderFromBook(orderBook, order)
		
		atomic.AddUint64(&e.stats.CancelledOrders, 1)
		return nil
	}
	
	return ErrOrderNotFound
}

// GetOrderBook returns the order book for a symbol
func (e *HFTEngine) GetOrderBook(symbol string) *HFTOrderBook {
	return e.getOrderBook(symbol)
}

// GetStats returns engine statistics
func (e *HFTEngine) GetStats() *EngineStats {
	stats := &EngineStats{
		OrdersProcessed:   atomic.LoadUint64(&e.ordersProcessed),
		TradesExecuted:    atomic.LoadUint64(&e.tradesExecuted),
		AvgLatencyNs:      atomic.LoadUint64(&e.avgLatency),
		TotalVolumeTraded: atomic.LoadUint64(&e.stats.TotalVolumeTraded),
		ActiveOrders:      atomic.LoadUint64(&e.stats.ActiveOrders),
		CancelledOrders:   atomic.LoadUint64(&e.stats.CancelledOrders),
		RejectedOrders:    atomic.LoadUint64(&e.stats.RejectedOrders),
		LastUpdateTime:    time.Now(),
	}
	
	if stats.OrdersProcessed > 0 {
		stats.AvgLatencyNs = stats.AvgLatencyNs / stats.OrdersProcessed
	}
	
	return stats
}

// getOrCreateOrderBook gets or creates an order book for a symbol
func (e *HFTEngine) getOrCreateOrderBook(symbol string) *HFTOrderBook {
	orderBooksMap := (*map[string]*HFTOrderBook)(atomic.LoadPointer(&e.orderBooks))
	
	if orderBook, exists := (*orderBooksMap)[symbol]; exists {
		return orderBook
	}
	
	// Create new order book
	newOrderBook := &HFTOrderBook{
		Symbol:        symbol,
		lastTradeTime: time.Now(),
	}
	
	// Update map atomically
	newMap := make(map[string]*HFTOrderBook)
	for k, v := range *orderBooksMap {
		newMap[k] = v
	}
	newMap[symbol] = newOrderBook
	
	atomic.StorePointer(&e.orderBooks, unsafe.Pointer(&newMap))
	
	e.logger.Info("Created new order book", zap.String("symbol", symbol))
	return newOrderBook
}

// getOrderBook gets an order book for a symbol
func (e *HFTEngine) getOrderBook(symbol string) *HFTOrderBook {
	orderBooksMap := (*map[string]*HFTOrderBook)(atomic.LoadPointer(&e.orderBooks))
	return (*orderBooksMap)[symbol]
}

// processOrder processes an order and returns resulting trades
func (e *HFTEngine) processOrder(orderBook *HFTOrderBook, order *HFTOrder) []*Trade {
	var trades []*Trade
	
	switch order.Type {
	case OrderTypeMarket:
		trades = e.processMarketOrder(orderBook, order)
	case OrderTypeLimit:
		trades = e.processLimitOrder(orderBook, order)
	case OrderTypeStopLimit, OrderTypeStopMarket:
		// Handle stop orders (simplified for now)
		trades = e.processStopOrder(orderBook, order)
	}
	
	return trades
}

// processMarketOrder processes a market order
func (e *HFTEngine) processMarketOrder(orderBook *HFTOrderBook, order *HFTOrder) []*Trade {
	var trades []*Trade
	remainingQty := order.Quantity
	
	if order.Side == OrderSideBuy {
		// Match against sell orders (asks)
		sellOrders := (*OrderLevel)(atomic.LoadPointer(&orderBook.sellOrders))
		for sellOrders != nil && remainingQty > 0 {
			trade := e.executeTrade(orderBook, order, sellOrders, &remainingQty)
			if trade != nil {
				trades = append(trades, trade)
			}
			sellOrders = (*OrderLevel)(atomic.LoadPointer(&sellOrders.Next))
		}
	} else {
		// Match against buy orders (bids)
		buyOrders := (*OrderLevel)(atomic.LoadPointer(&orderBook.buyOrders))
		for buyOrders != nil && remainingQty > 0 {
			trade := e.executeTrade(orderBook, order, buyOrders, &remainingQty)
			if trade != nil {
				trades = append(trades, trade)
			}
			buyOrders = (*OrderLevel)(atomic.LoadPointer(&buyOrders.Next))
		}
	}
	
	// Update order status
	if remainingQty == 0 {
		order.Status = OrderStatusFilled
	} else if remainingQty < order.Quantity {
		order.Status = OrderStatusPartiallyFilled
	} else {
		order.Status = OrderStatusRejected
		atomic.AddUint64(&e.stats.RejectedOrders, 1)
	}
	
	order.Filled = order.Quantity - remainingQty
	return trades
}

// processLimitOrder processes a limit order
func (e *HFTEngine) processLimitOrder(orderBook *HFTOrderBook, order *HFTOrder) []*Trade {
	var trades []*Trade
	remainingQty := order.Quantity
	
	// First try to match against existing orders
	if order.Side == OrderSideBuy {
		sellOrders := (*OrderLevel)(atomic.LoadPointer(&orderBook.sellOrders))
		for sellOrders != nil && remainingQty > 0 && sellOrders.Price <= order.Price {
			trade := e.executeTrade(orderBook, order, sellOrders, &remainingQty)
			if trade != nil {
				trades = append(trades, trade)
			}
			sellOrders = (*OrderLevel)(atomic.LoadPointer(&sellOrders.Next))
		}
	} else {
		buyOrders := (*OrderLevel)(atomic.LoadPointer(&orderBook.buyOrders))
		for buyOrders != nil && remainingQty > 0 && buyOrders.Price >= order.Price {
			trade := e.executeTrade(orderBook, order, buyOrders, &remainingQty)
			if trade != nil {
				trades = append(trades, trade)
			}
			buyOrders = (*OrderLevel)(atomic.LoadPointer(&buyOrders.Next))
		}
	}
	
	// If there's remaining quantity, add to order book
	if remainingQty > 0 {
		order.Quantity = remainingQty
		e.addOrderToBook(orderBook, order)
		order.Status = OrderStatusNew
		atomic.AddUint64(&e.stats.ActiveOrders, 1)
	} else {
		order.Status = OrderStatusFilled
	}
	
	order.Filled = order.Quantity - remainingQty
	return trades
}

// processStopOrder processes a stop order (simplified implementation)
func (e *HFTEngine) processStopOrder(orderBook *HFTOrderBook, order *HFTOrder) []*Trade {
	// For now, treat as limit order
	// In production, would implement proper stop logic
	return e.processLimitOrder(orderBook, order)
}

// executeTrade executes a trade between two orders
func (e *HFTEngine) executeTrade(orderBook *HFTOrderBook, incomingOrder *HFTOrder, level *OrderLevel, remainingQty *uint64) *Trade {
	if level == nil {
		return nil
	}
	
	// Get first order from level
	levelOrder := (*HFTOrder)(atomic.LoadPointer(&level.Orders))
	if levelOrder == nil {
		return nil
	}
	
	// Calculate trade quantity
	tradeQty := minUint64(*remainingQty, levelOrder.Quantity-levelOrder.Filled)
	if tradeQty == 0 {
		return nil
	}
	
	// Create trade
	trade := &Trade{
		ID:        uuid.New().String(),
		Symbol:    orderBook.Symbol,
		Price:     float64(level.Price),
		Quantity:  float64(tradeQty),
		Timestamp: time.Now(),
	}
	
	if incomingOrder.Side == OrderSideBuy {
		trade.BuyOrderID = incomingOrder.ID
		trade.SellOrderID = levelOrder.ID
	} else {
		trade.BuyOrderID = levelOrder.ID
		trade.SellOrderID = incomingOrder.ID
	}
	
	// Update orders
	levelOrder.Filled += tradeQty
	*remainingQty -= tradeQty
	
	// Update statistics
	atomic.AddUint64(&e.tradesExecuted, 1)
	atomic.AddUint64(&e.stats.TotalVolumeTraded, tradeQty)
	
	// Update order book spread
	e.updateSpread(orderBook)
	
	// If level order is fully filled, remove it
	if levelOrder.Filled >= levelOrder.Quantity {
		levelOrder.Status = OrderStatusFilled
		e.removeOrderFromLevel(level, levelOrder)
		atomic.AddUint64(&e.stats.ActiveOrders, ^uint64(0)) // Decrement
	}
	
	return trade
}

// addOrderToBook adds an order to the order book
func (e *HFTEngine) addOrderToBook(orderBook *HFTOrderBook, order *HFTOrder) {
	// Store order in map for fast lookup
	orderBook.orderMap.Store(order.ID, order)
	
	// Add to appropriate side of the book
	if order.Side == OrderSideBuy {
		e.addOrderToSide(&orderBook.buyOrders, order, true)
	} else {
		e.addOrderToSide(&orderBook.sellOrders, order, false)
	}
}

// addOrderToSide adds an order to a specific side of the order book
func (e *HFTEngine) addOrderToSide(sidePtr *unsafe.Pointer, order *HFTOrder, isBuy bool) {
	// Simplified implementation - in production would use more sophisticated data structures
	// This is a basic linked list insertion
	
	newLevel := &OrderLevel{
		Price:    order.Price,
		Quantity: order.Quantity,
		Orders:   unsafe.Pointer(order),
	}
	
	// Insert at appropriate position (price-time priority)
	currentLevel := (*OrderLevel)(atomic.LoadPointer(sidePtr))
	if currentLevel == nil {
		atomic.StorePointer(sidePtr, unsafe.Pointer(newLevel))
		return
	}
	
	// Find insertion point
	var prevLevel *OrderLevel
	for currentLevel != nil {
		if (isBuy && newLevel.Price > currentLevel.Price) || (!isBuy && newLevel.Price < currentLevel.Price) {
			break
		}
		if newLevel.Price == currentLevel.Price {
			// Add to existing level
			e.addOrderToLevel(currentLevel, order)
			return
		}
		prevLevel = currentLevel
		currentLevel = (*OrderLevel)(atomic.LoadPointer(&currentLevel.Next))
	}
	
	// Insert new level
	if prevLevel == nil {
		atomic.StorePointer(&newLevel.Next, unsafe.Pointer(currentLevel))
		atomic.StorePointer(sidePtr, unsafe.Pointer(newLevel))
	} else {
		atomic.StorePointer(&newLevel.Next, unsafe.Pointer(currentLevel))
		atomic.StorePointer(&prevLevel.Next, unsafe.Pointer(newLevel))
	}
}

// addOrderToLevel adds an order to an existing price level
func (e *HFTEngine) addOrderToLevel(level *OrderLevel, order *HFTOrder) {
	// Add order to end of level's order list
	currentOrder := (*HFTOrder)(atomic.LoadPointer(&level.Orders))
	if currentOrder == nil {
		atomic.StorePointer(&level.Orders, unsafe.Pointer(order))
		return
	}
	
	// Find end of list
	for currentOrder.Next != nil {
		currentOrder = currentOrder.Next
	}
	
	currentOrder.Next = order
	order.Prev = currentOrder
	level.Quantity += order.Quantity
}

// removeOrderFromBook removes an order from the order book
func (e *HFTEngine) removeOrderFromBook(orderBook *HFTOrderBook, order *HFTOrder) {
	orderBook.orderMap.Delete(order.ID)
	
	// Remove from appropriate side
	if order.Side == OrderSideBuy {
		e.removeOrderFromSide(&orderBook.buyOrders, order)
	} else {
		e.removeOrderFromSide(&orderBook.sellOrders, order)
	}
}

// removeOrderFromSide removes an order from a specific side of the order book
func (e *HFTEngine) removeOrderFromSide(sidePtr *unsafe.Pointer, order *HFTOrder) {
	currentLevel := (*OrderLevel)(atomic.LoadPointer(sidePtr))
	var prevLevel *OrderLevel
	
	for currentLevel != nil {
		if currentLevel.Price == order.Price {
			e.removeOrderFromLevel(currentLevel, order)
			
			// If level is empty, remove it
			if (*HFTOrder)(atomic.LoadPointer(&currentLevel.Orders)) == nil {
				if prevLevel == nil {
					atomic.StorePointer(sidePtr, atomic.LoadPointer(&currentLevel.Next))
				} else {
					atomic.StorePointer(&prevLevel.Next, atomic.LoadPointer(&currentLevel.Next))
				}
			}
			return
		}
		prevLevel = currentLevel
		currentLevel = (*OrderLevel)(atomic.LoadPointer(&currentLevel.Next))
	}
}

// removeOrderFromLevel removes an order from a price level
func (e *HFTEngine) removeOrderFromLevel(level *OrderLevel, order *HFTOrder) {
	currentOrder := (*HFTOrder)(atomic.LoadPointer(&level.Orders))
	
	if currentOrder == order {
		atomic.StorePointer(&level.Orders, unsafe.Pointer(order.Next))
		if order.Next != nil {
			order.Next.Prev = nil
		}
		level.Quantity -= order.Quantity
		return
	}
	
	for currentOrder != nil {
		if currentOrder == order {
			if order.Prev != nil {
				order.Prev.Next = order.Next
			}
			if order.Next != nil {
				order.Next.Prev = order.Prev
			}
			level.Quantity -= order.Quantity
			return
		}
		currentOrder = currentOrder.Next
	}
}

// updateSpread updates the bid-ask spread
func (e *HFTEngine) updateSpread(orderBook *HFTOrderBook) {
	buyOrders := (*OrderLevel)(atomic.LoadPointer(&orderBook.buyOrders))
	sellOrders := (*OrderLevel)(atomic.LoadPointer(&orderBook.sellOrders))
	
	var bestBid, bestAsk uint64
	
	if buyOrders != nil {
		bestBid = buyOrders.Price
	}
	if sellOrders != nil {
		bestAsk = sellOrders.Price
	}
	
	atomic.StoreUint64(&orderBook.bestBid, bestBid)
	atomic.StoreUint64(&orderBook.bestAsk, bestAsk)
	
	if bestBid > 0 && bestAsk > 0 {
		spread := bestAsk - bestBid
		atomic.StoreUint64(&orderBook.spread, spread)
	}
}

// performanceMonitor monitors engine performance
func (e *HFTEngine) performanceMonitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			stats := e.GetStats()
			e.logger.Debug("HFT Engine Performance",
				zap.Uint64("orders_processed", stats.OrdersProcessed),
				zap.Uint64("trades_executed", stats.TradesExecuted),
				zap.Uint64("avg_latency_ns", stats.AvgLatencyNs),
				zap.Uint64("active_orders", stats.ActiveOrders),
				zap.Uint64("total_volume", stats.TotalVolumeTraded))
		}
	}
}

// tradeProcessor processes completed trades
func (e *HFTEngine) tradeProcessor() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case trade := <-e.TradeChannel:
			if trade == nil {
				return
			}
			
			// Process trade (send to downstream systems, update databases, etc.)
			e.logger.Debug("Trade executed",
				zap.String("trade_id", trade.ID),
				zap.String("symbol", trade.Symbol),
				zap.Float64("price", trade.Price),
				zap.Float64("quantity", trade.Quantity))

		}
	}
}

// Helper function
// Use min function from engine.go

// minUint64 returns the minimum of two uint64 values
func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Error definitions
var (
	ErrOrderBookNotFound = errors.New("order book not found")
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidOrder      = errors.New("invalid order")
)
