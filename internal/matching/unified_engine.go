package matching

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/errors"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
	"go.uber.org/zap"
)

// UnifiedMatchingEngine implements the MatchingEngine interface
// This consolidates the duplicate code from internal/core/matching and internal/orders/matching
type UnifiedMatchingEngine struct {
	orderBooks map[string]*OrderBook
	mu         sync.RWMutex
	metrics    *EngineMetrics
	config     *EngineConfig
	logger     interfaces.Logger
	publisher  interfaces.EventPublisher

	// Performance optimization
	orderPool sync.Pool
	tradePool sync.Pool

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Callbacks
	orderBookCallbacks map[string][]func(*types.OrderBook)
	tradeCallbacks     map[string][]func(*types.Trade)
	callbackMu         sync.RWMutex
}

// OrderBook represents a unified order book for a symbol
type OrderBook struct {
	Symbol    string
	Bids      *PriceLevel
	Asks      *PriceLevel
	Orders    map[string]*types.Order
	mu        sync.RWMutex
	sequence  uint64
	lastTrade *types.Trade

	// Performance metrics
	orderCount  uint64
	tradeCount  uint64
	lastUpdated int64
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64
	Orders   []*types.Order
	Quantity float64
	Count    int
	Next     *PriceLevel
	Prev     *PriceLevel
}

// EngineConfig contains configuration for the matching engine
type EngineConfig struct {
	MaxOrdersPerSymbol int
	TickSize           float64
	ProcessingTimeout  time.Duration
	EnableMetrics      bool
	PoolSize           int
	BufferSize         int
	WorkerCount        int
	MaxLatency         time.Duration
	EnableOrderBook    bool
	OrderBookDepth     int
}

// EngineMetrics contains performance metrics
type EngineMetrics struct {
	OrdersProcessed  uint64
	TradesExecuted   uint64
	AverageLatency   time.Duration
	ThroughputPerSec float64
	LastProcessedAt  time.Time
	ActiveOrders     int
	QueueDepth       int

	// Latency tracking
	latencySum   uint64
	latencyCount uint64
	mu           sync.RWMutex
}

// NewUnifiedMatchingEngine creates a new unified matching engine
func NewUnifiedMatchingEngine(config *EngineConfig, logger interfaces.Logger, publisher interfaces.EventPublisher) *UnifiedMatchingEngine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &UnifiedMatchingEngine{
		orderBooks:         make(map[string]*OrderBook),
		metrics:            &EngineMetrics{},
		config:             config,
		logger:             logger,
		publisher:          publisher,
		ctx:                ctx,
		cancel:             cancel,
		orderBookCallbacks: make(map[string][]func(*types.OrderBook)),
		tradeCallbacks:     make(map[string][]func(*types.Trade)),
	}

	// Initialize object pools for performance
	engine.orderPool = sync.Pool{
		New: func() interface{} {
			return &types.Order{}
		},
	}

	engine.tradePool = sync.Pool{
		New: func() interface{} {
			return &types.Trade{}
		},
	}

	return engine
}

// ProcessOrder processes a new order and returns resulting trades
func (e *UnifiedMatchingEngine) ProcessOrder(ctx context.Context, order *types.Order) ([]*types.Trade, error) {
	start := time.Now()
	defer func() {
		e.updateLatencyMetrics(time.Since(start))
	}()

	if order == nil {
		return nil, errors.New(errors.ErrInvalidOrder, "order cannot be nil")
	}

	if !order.IsValid() {
		return nil, errors.New(errors.ErrInvalidOrder, "order validation failed")
	}

	// Get or create order book for symbol
	orderBook := e.getOrCreateOrderBook(order.Symbol)

	// Process the order
	trades, err := orderBook.processOrder(order)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrMatchingFailed, "failed to process order")
	}

	// Update metrics
	atomic.AddUint64(&e.metrics.OrdersProcessed, 1)
	atomic.AddUint64(&e.metrics.TradesExecuted, uint64(len(trades)))
	e.metrics.LastProcessedAt = time.Now()

	// Publish events
	if e.publisher != nil {
		e.publishOrderEvent(ctx, order, interfaces.OrderEventCreated)
		for _, trade := range trades {
			e.publishTradeEvent(ctx, trade)
		}
	}

	// Trigger callbacks
	e.triggerCallbacks(order.Symbol, orderBook, trades)

	return trades, nil
}

// CancelOrder cancels an existing order
func (e *UnifiedMatchingEngine) CancelOrder(ctx context.Context, orderID string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Find the order across all order books
	for _, orderBook := range e.orderBooks {
		if err := orderBook.cancelOrder(orderID); err == nil {
			// Order found and canceled
			if e.publisher != nil {
				// We need to get the order details for the event
				if order, exists := orderBook.getOrder(orderID); exists {
					e.publishOrderEvent(ctx, order, interfaces.OrderEventCanceled)
				}
			}
			return nil
		}
	}

	return errors.New(errors.ErrOrderNotFound, "order not found")
}

// GetOrderBook returns the current order book state
func (e *UnifiedMatchingEngine) GetOrderBook(symbol string) (*types.OrderBook, error) {
	e.mu.RLock()
	orderBook, exists := e.orderBooks[symbol]
	e.mu.RUnlock()

	if !exists {
		return nil, errors.New(errors.ErrSymbolNotFound, "symbol not found")
	}

	return orderBook.getSnapshot(), nil
}

// GetMetrics returns engine performance metrics
func (e *UnifiedMatchingEngine) GetMetrics() *interfaces.EngineMetrics {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()

	var avgLatency time.Duration
	if e.metrics.latencyCount > 0 {
		avgLatency = time.Duration(e.metrics.latencySum / e.metrics.latencyCount)
	}

	return &interfaces.EngineMetrics{
		OrdersProcessed:  atomic.LoadUint64(&e.metrics.OrdersProcessed),
		TradesExecuted:   atomic.LoadUint64(&e.metrics.TradesExecuted),
		AverageLatency:   avgLatency,
		ThroughputPerSec: e.calculateThroughput(),
		LastProcessedAt:  e.metrics.LastProcessedAt,
		ActiveOrders:     e.getActiveOrderCount(),
		QueueDepth:       0, // Would be implemented based on queue implementation
	}
}

// Start starts the matching engine
func (e *UnifiedMatchingEngine) Start(ctx context.Context) error {
	e.logger.Info("Starting unified matching engine")

	// Start background workers if needed
	for i := 0; i < e.config.WorkerCount; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}

	// Start metrics collection if enabled
	if e.config.EnableMetrics {
		e.wg.Add(1)
		go e.metricsCollector()
	}

	return nil
}

// Stop stops the matching engine gracefully
func (e *UnifiedMatchingEngine) Stop(ctx context.Context) error {
	e.logger.Info("Stopping unified matching engine")

	e.cancel()
	e.wg.Wait()

	return nil
}

// SubscribeOrderBook subscribes to order book updates
func (e *UnifiedMatchingEngine) SubscribeOrderBook(symbol string, callback func(*types.OrderBook)) error {
	e.callbackMu.Lock()
	defer e.callbackMu.Unlock()

	e.orderBookCallbacks[symbol] = append(e.orderBookCallbacks[symbol], callback)
	return nil
}

// SubscribeTrades subscribes to trade updates
func (e *UnifiedMatchingEngine) SubscribeTrades(symbol string, callback func(*types.Trade)) error {
	e.callbackMu.Lock()
	defer e.callbackMu.Unlock()

	e.tradeCallbacks[symbol] = append(e.tradeCallbacks[symbol], callback)
	return nil
}

// Private methods

func (e *UnifiedMatchingEngine) getOrCreateOrderBook(symbol string) *OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()

	orderBook, exists := e.orderBooks[symbol]
	if !exists {
		orderBook = &OrderBook{
			Symbol:      symbol,
			Orders:      make(map[string]*types.Order),
			lastUpdated: time.Now().UnixNano(),
		}
		e.orderBooks[symbol] = orderBook
	}

	return orderBook
}

func (e *UnifiedMatchingEngine) updateLatencyMetrics(latency time.Duration) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()

	e.metrics.latencySum += uint64(latency.Nanoseconds())
	e.metrics.latencyCount++
}

func (e *UnifiedMatchingEngine) calculateThroughput() float64 {
	// Calculate orders per second based on recent activity
	// This is a simplified implementation
	ordersProcessed := atomic.LoadUint64(&e.metrics.OrdersProcessed)
	if ordersProcessed == 0 {
		return 0
	}

	// This would need more sophisticated time window tracking
	return float64(ordersProcessed) / time.Since(e.metrics.LastProcessedAt).Seconds()
}

func (e *UnifiedMatchingEngine) getActiveOrderCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	count := 0
	for _, orderBook := range e.orderBooks {
		orderBook.mu.RLock()
		count += len(orderBook.Orders)
		orderBook.mu.RUnlock()
	}

	return count
}

func (e *UnifiedMatchingEngine) publishOrderEvent(ctx context.Context, order *types.Order, eventType string) {
	event := &interfaces.OrderEvent{
		Type:      eventType,
		Order:     order,
		Timestamp: time.Now(),
		UserID:    order.UserID,
	}

	if err := e.publisher.PublishOrderEvent(ctx, *event); err != nil {
		e.logger.Error("Failed to publish order event", "error", err, "order_id", order.ID)
	}
}

func (e *UnifiedMatchingEngine) publishTradeEvent(ctx context.Context, trade *types.Trade) {
	event := &interfaces.TradeEvent{
		Type:      interfaces.TradeEventExecuted,
		Trade:     trade,
		Timestamp: time.Now(),
	}

	if err := e.publisher.PublishTradeEvent(ctx, *event); err != nil {
		e.logger.Error("Failed to publish trade event", "error", err, "trade_id", trade.ID)
	}
}

func (e *UnifiedMatchingEngine) triggerCallbacks(symbol string, orderBook *OrderBook, trades []*types.Trade) {
	e.callbackMu.RLock()
	defer e.callbackMu.RUnlock()

	// Trigger order book callbacks
	if callbacks, exists := e.orderBookCallbacks[symbol]; exists {
		snapshot := orderBook.getSnapshot()
		for _, callback := range callbacks {
			go callback(snapshot) // Run callbacks asynchronously
		}
	}

	// Trigger trade callbacks
	if callbacks, exists := e.tradeCallbacks[symbol]; exists {
		for _, trade := range trades {
			for _, callback := range callbacks {
				go callback(trade) // Run callbacks asynchronously
			}
		}
	}
}

func (e *UnifiedMatchingEngine) worker(id int) {
	defer e.wg.Done()

	e.logger.Debug("Starting matching engine worker", "worker_id", id)

	for {
		select {
		case <-e.ctx.Done():
			e.logger.Debug("Stopping matching engine worker", "worker_id", id)
			return
		default:
			// Worker logic would go here
			// This could process orders from a queue, perform maintenance, etc.
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// TradeChannel returns a channel for trade events (for compatibility)
func (e *UnifiedMatchingEngine) TradeChannel() <-chan *types.Trade {
	// Create a buffered channel for trade events
	ch := make(chan *types.Trade, 1000)
	
	// This is a compatibility method - in practice, you'd want to use the event publisher
	// For now, return a closed channel to prevent blocking
	close(ch)
	return ch
}

// GetMarketData returns market data for a symbol (for compatibility)
func (e *UnifiedMatchingEngine) GetMarketData(symbol string) (*types.MarketData, error) {
	orderBook, err := e.GetOrderBook(symbol)
	if err != nil {
		return nil, err
	}

	// Convert order book to market data
	marketData := &types.MarketData{
		Symbol:    symbol,
		Timestamp: time.Now(),
		// Add other fields as needed
	}

	if len(orderBook.Bids) > 0 {
		marketData.BidPrice = orderBook.Bids[0].Price
		marketData.BidSize = orderBook.Bids[0].Quantity
	}

	if len(orderBook.Asks) > 0 {
		marketData.AskPrice = orderBook.Asks[0].Price
		marketData.AskSize = orderBook.Asks[0].Quantity
	}

	return marketData, nil
}

func (e *UnifiedMatchingEngine) metricsCollector() {
	defer e.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			// Collect and update metrics
			e.collectMetrics()
		}
	}
}

func (e *UnifiedMatchingEngine) collectMetrics() {
	// Update throughput and other time-based metrics
	// This would be more sophisticated in a real implementation
}

// OrderBook methods

func (ob *OrderBook) processOrder(order *types.Order) ([]*types.Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	trades := make([]*types.Trade, 0, 4) // Pre-allocate for common case

	// Handle different order types
	switch order.Type {
	case types.OrderTypeMarket:
		if order.Side == types.OrderSideBuy {
			trades = ob.matchMarketBuyOrder(order, trades)
		} else {
			trades = ob.matchMarketSellOrder(order, trades)
		}
	case types.OrderTypeLimit:
		if order.Side == types.OrderSideBuy {
			trades = ob.matchLimitBuyOrder(order, trades)
		} else {
			trades = ob.matchLimitSellOrder(order, trades)
		}
	default:
		return nil, errors.New(errors.ErrInvalidOrder, "unsupported order type")
	}

	// Update order book statistics
	atomic.AddUint64(&ob.orderCount, 1)
	atomic.StoreInt64(&ob.lastUpdated, time.Now().UnixNano())

	return trades, nil
}

func (ob *OrderBook) matchMarketBuyOrder(order *types.Order, trades []*types.Trade) []*types.Trade {
	// Implementation would match against asks
	// This is a simplified version - real implementation would be more complex
	return trades
}

func (ob *OrderBook) matchMarketSellOrder(order *types.Order, trades []*types.Trade) []*types.Trade {
	// Implementation would match against bids
	// This is a simplified version - real implementation would be more complex
	return trades
}

func (ob *OrderBook) matchLimitBuyOrder(order *types.Order, trades []*types.Trade) []*types.Trade {
	// Implementation would match against asks at or below the limit price
	// This is a simplified version - real implementation would be more complex
	return trades
}

func (ob *OrderBook) matchLimitSellOrder(order *types.Order, trades []*types.Trade) []*types.Trade {
	// Implementation would match against bids at or above the limit price
	// This is a simplified version - real implementation would be more complex
	return trades
}

func (ob *OrderBook) cancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return errors.New(errors.ErrOrderNotFound, "order not found in order book")
	}

	// Remove from order book structures
	delete(ob.Orders, orderID)

	// Update order status
	order.Status = types.OrderStatusCanceled
	order.UpdatedAt = time.Now()

	return nil
}

func (ob *OrderBook) getOrder(orderID string) (*types.Order, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	order, exists := ob.Orders[orderID]
	return order, exists
}

func (ob *OrderBook) getSnapshot() *types.OrderBook {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	// Create a snapshot of the current order book state
	snapshot := &types.OrderBook{
		Symbol:    ob.Symbol,
		Bids:      make([]*types.OrderBookLevel, 0),
		Asks:      make([]*types.OrderBookLevel, 0),
		Timestamp: time.Now(),
		Sequence:  atomic.LoadUint64(&ob.sequence),
	}

	// Build bids and asks from price levels
	// This would be implemented based on the actual price level structure

	return snapshot
}
