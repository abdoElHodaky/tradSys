package matching

import (
	"context"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

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

// CancelOrder cancels an existing order
func (e *HFTEngine) CancelOrder(symbol, orderID string) error {
	orderBook := e.getOrderBook(symbol)
	if orderBook == nil {
		return ErrOrderBookNotFound
	}

	// Find and remove order
	orderInterface, exists := orderBook.orderMap.LoadAndDelete(orderID)
	if !exists {
		return ErrOrderNotFound
	}

	order := orderInterface.(*HFTOrder)
	order.Status = OrderStatusCancelled

	// Remove from order book levels
	e.removeOrderFromBook(orderBook, order)

	atomic.AddUint64(&e.stats.CancelledOrders, 1)
	atomic.AddUint64(&e.stats.ActiveOrders, ^uint64(0)) // Decrement

	return nil
}

// getOrCreateOrderBook gets or creates an order book for a symbol
func (e *HFTEngine) getOrCreateOrderBook(symbol string) *HFTOrderBook {
	orderBooks := (*map[string]*HFTOrderBook)(atomic.LoadPointer(&e.orderBooks))

	if orderBook, exists := (*orderBooks)[symbol]; exists {
		return orderBook
	}

	// Create new order book
	newOrderBook := &HFTOrderBook{
		Symbol:        symbol,
		lastTradeTime: time.Now(),
	}

	// Create new map with the additional order book
	newOrderBooks := make(map[string]*HFTOrderBook)
	for k, v := range *orderBooks {
		newOrderBooks[k] = v
	}
	newOrderBooks[symbol] = newOrderBook

	// Atomically update the pointer
	atomic.StorePointer(&e.orderBooks, unsafe.Pointer(&newOrderBooks))

	return newOrderBook
}

// getOrderBook gets an order book for a symbol
func (e *HFTEngine) getOrderBook(symbol string) *HFTOrderBook {
	orderBooks := (*map[string]*HFTOrderBook)(atomic.LoadPointer(&e.orderBooks))
	return (*orderBooks)[symbol]
}

// processOrder processes an incoming order against the order book
func (e *HFTEngine) processOrder(orderBook *HFTOrderBook, order *HFTOrder) []*Trade {
	var trades []*Trade

	// Try to match against opposite side
	if order.Side == OrderSideBuy {
		trades = e.matchAgainstSide(orderBook, order, &orderBook.sellOrders)
	} else {
		trades = e.matchAgainstSide(orderBook, order, &orderBook.buyOrders)
	}

	// If order is not fully filled, add to book
	if order.Filled < order.Quantity {
		order.Status = OrderStatusPartiallyFilled
		if order.Filled == 0 {
			order.Status = OrderStatusNew
		}
		e.addOrderToBook(orderBook, order)
		atomic.AddUint64(&e.stats.ActiveOrders, 1)
	} else {
		order.Status = OrderStatusFilled
	}

	return trades
}

// matchAgainstSide matches an order against one side of the order book
func (e *HFTEngine) matchAgainstSide(orderBook *HFTOrderBook, incomingOrder *HFTOrder, sidePtr *unsafe.Pointer) []*Trade {
	var trades []*Trade
	remainingQty := incomingOrder.Quantity - incomingOrder.Filled

	currentLevel := (*OrderLevel)(atomic.LoadPointer(sidePtr))

	for currentLevel != nil && remainingQty > 0 {
		// Check if price matches
		if !e.priceMatches(incomingOrder, currentLevel) {
			break
		}

		// Match against orders at this level
		levelTrades := e.matchAgainstLevel(orderBook, incomingOrder, currentLevel, &remainingQty)
		trades = append(trades, levelTrades...)

		// Move to next level
		currentLevel = (*OrderLevel)(atomic.LoadPointer(&currentLevel.Next))
	}

	incomingOrder.Filled = incomingOrder.Quantity - remainingQty
	return trades
}

// priceMatches checks if an incoming order can match against a price level
func (e *HFTEngine) priceMatches(order *HFTOrder, level *OrderLevel) bool {
	if order.Side == OrderSideBuy {
		return order.Price >= level.Price
	}
	return order.Price <= level.Price
}

// matchAgainstLevel matches an order against a specific price level
func (e *HFTEngine) matchAgainstLevel(orderBook *HFTOrderBook, incomingOrder *HFTOrder, level *OrderLevel, remainingQty *uint64) []*Trade {
	var trades []*Trade

	levelOrder := (*HFTOrder)(atomic.LoadPointer(&level.Orders))

	for levelOrder != nil && *remainingQty > 0 {
		trade := e.executeTrade(orderBook, incomingOrder, levelOrder, level, remainingQty)
		if trade != nil {
			trades = append(trades, trade)
		}
		levelOrder = levelOrder.Next
	}

	return trades
}

// executeTrade executes a trade between two orders
func (e *HFTEngine) executeTrade(orderBook *HFTOrderBook, incomingOrder *HFTOrder, levelOrder *HFTOrder, level *OrderLevel, remainingQty *uint64) *Trade {
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
	// Add order to the end of the linked list at this level
	currentOrder := (*HFTOrder)(atomic.LoadPointer(&level.Orders))
	if currentOrder == nil {
		atomic.StorePointer(&level.Orders, unsafe.Pointer(order))
		return
	}

	// Find the end of the list
	for currentOrder.Next != nil {
		currentOrder = currentOrder.Next
	}

	// Add to end
	currentOrder.Next = order
	order.Prev = currentOrder

	// Update level quantity
	atomic.AddUint64(&level.Quantity, order.Quantity)
}

// removeOrderFromBook removes an order from the order book
func (e *HFTEngine) removeOrderFromBook(orderBook *HFTOrderBook, order *HFTOrder) {
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

	for currentLevel != nil {
		if currentLevel.Price == order.Price {
			e.removeOrderFromLevel(currentLevel, order)
			return
		}
		currentLevel = (*OrderLevel)(atomic.LoadPointer(&currentLevel.Next))
	}
}

// removeOrderFromLevel removes an order from a specific price level
func (e *HFTEngine) removeOrderFromLevel(level *OrderLevel, order *HFTOrder) {
	currentOrder := (*HFTOrder)(atomic.LoadPointer(&level.Orders))

	// If it's the first order in the level
	if currentOrder == order {
		atomic.StorePointer(&level.Orders, unsafe.Pointer(order.Next))
		if order.Next != nil {
			order.Next.Prev = nil
		}
		return
	}

	// Find and remove the order
	for currentOrder != nil {
		if currentOrder == order {
			if order.Prev != nil {
				order.Prev.Next = order.Next
			}
			if order.Next != nil {
				order.Next.Prev = order.Prev
			}
			return
		}
		currentOrder = currentOrder.Next
	}
}

// updateSpread updates the bid-ask spread for an order book
func (e *HFTEngine) updateSpread(orderBook *HFTOrderBook) {
	// Get best bid
	buyLevel := (*OrderLevel)(atomic.LoadPointer(&orderBook.buyOrders))
	var bestBid uint64
	if buyLevel != nil {
		bestBid = buyLevel.Price
	}

	// Get best ask
	sellLevel := (*OrderLevel)(atomic.LoadPointer(&orderBook.sellOrders))
	var bestAsk uint64
	if sellLevel != nil {
		bestAsk = sellLevel.Price
	}

	// Update atomically
	atomic.StoreUint64(&orderBook.bestBid, bestBid)
	atomic.StoreUint64(&orderBook.bestAsk, bestAsk)

	if bestBid > 0 && bestAsk > 0 {
		spread := bestAsk - bestBid
		atomic.StoreUint64(&orderBook.spread, spread)
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

// minUint64 returns the minimum of two uint64 values
func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
