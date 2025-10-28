package matching

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/common/pool"
	"github.com/abdoElHodaky/tradSys/pkg/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AdvancedOrderMatchingEngine provides enhanced order matching with HFT optimizations
type AdvancedOrderMatchingEngine struct {
	orderBooks   sync.Map // map[string]*AdvancedOrderBook
	tradePool    *pool.ObjectPool
	orderPool    *pool.ObjectPool
	logger       *zap.Logger
	metrics      *MatchingMetrics
	config       *EngineConfig
	eventChannel chan *MatchingEvent
	stopChannel  chan struct{}
	isRunning    int32
}

// EngineConfig contains configuration for the matching engine
type EngineConfig struct {
	MaxOrdersPerSymbol     int           `json:"max_orders_per_symbol"`
	MaxTradesPerSecond     int           `json:"max_trades_per_second"`
	LatencyTarget          time.Duration `json:"latency_target"`
	EnablePriceImprovement bool          `json:"enable_price_improvement"`
	EnableIcebergOrders    bool          `json:"enable_iceberg_orders"`
	EnableHiddenOrders     bool          `json:"enable_hidden_orders"`
	TickSize               float64       `json:"tick_size"`
}

// MatchingMetrics tracks performance metrics
type MatchingMetrics struct {
	TotalTrades     int64         `json:"total_trades"`
	TotalVolume     float64       `json:"total_volume"`
	AverageLatency  time.Duration `json:"average_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	OrdersProcessed int64         `json:"orders_processed"`
	TradesPerSecond float64       `json:"trades_per_second"`
	LastUpdateTime  time.Time     `json:"last_update_time"`
}

// MatchingEvent represents events from the matching engine
type MatchingEvent struct {
	Type      MatchingEventType `json:"type"`
	Symbol    string            `json:"symbol"`
	Order     *types.Order      `json:"order,omitempty"`
	Trade     *Trade            `json:"trade,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// MatchingEventType defines types of matching events
type MatchingEventType string

const (
	EventOrderAdded    MatchingEventType = "order_added"
	EventOrderCanceled MatchingEventType = "order_canceled"
	EventOrderFilled   MatchingEventType = "order_filled"
	EventTradeExecuted MatchingEventType = "trade_executed"
	EventOrderRejected MatchingEventType = "order_rejected"
)

// AdvancedOrderBook extends the basic order book with advanced features
type AdvancedOrderBook struct {
	*OrderBook
	priceImprovement   *PriceImprovementEngine
	icebergManager     *IcebergOrderManager
	hiddenOrderPool    *HiddenOrderPool
	marketImpactCalc   *MarketImpactCalculator
	performanceTracker *PerformanceTracker
}

// PriceImprovementEngine handles price improvement logic
type PriceImprovementEngine struct {
	enabled          bool
	improvementTicks int
	minImprovement   float64
	maxImprovement   float64
	tickSize         float64
}

// IcebergOrderManager handles iceberg order logic
type IcebergOrderManager struct {
	enabled      bool
	activeOrders map[string]*IcebergOrder
	mu           sync.RWMutex
}

// IcebergOrder represents an iceberg order
type IcebergOrder struct {
	ParentOrder   *types.Order
	DisplaySize   float64
	TotalSize     float64
	RemainingSize float64
	RefreshSize   float64
	CurrentOrder  *types.Order
}

// HiddenOrderPool manages hidden orders
type HiddenOrderPool struct {
	enabled      bool
	hiddenOrders map[string]*types.Order
	mu           sync.RWMutex
}

// MarketImpactCalculator calculates market impact of orders
type MarketImpactCalculator struct {
	enabled          bool
	impactModel      string // "linear", "sqrt", "log"
	liquidityFactor  float64
	volatilityFactor float64
	historicalTrades []Trade
	mu               sync.RWMutex
}

// PerformanceTracker tracks order book performance
type PerformanceTracker struct {
	latencyHistogram  map[time.Duration]int64
	throughputCounter int64
	lastResetTime     time.Time
	mu                sync.RWMutex
}

// NewAdvancedOrderMatchingEngine creates a new advanced matching engine
func NewAdvancedOrderMatchingEngine(config *EngineConfig, logger *zap.Logger) *AdvancedOrderMatchingEngine {
	engine := &AdvancedOrderMatchingEngine{
		config:       config,
		logger:       logger,
		metrics:      &MatchingMetrics{LastUpdateTime: time.Now()},
		eventChannel: make(chan *MatchingEvent, 10000),
		stopChannel:  make(chan struct{}),
	}

	// Initialize object pools for performance
	engine.tradePool = pool.NewObjectPool(func() interface{} {
		return &Trade{}
	}, 1000)

	engine.orderPool = pool.NewObjectPool(func() interface{} {
		return &types.Order{}
	}, 1000)

	return engine
}

// Start starts the matching engine
func (e *AdvancedOrderMatchingEngine) Start() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return fmt.Errorf("engine is already running")
	}

	e.logger.Info("Starting advanced order matching engine",
		zap.Any("config", e.config))

	// Start event processing goroutine
	go e.processEvents()

	return nil
}

// Stop stops the matching engine
func (e *AdvancedOrderMatchingEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return fmt.Errorf("engine is not running")
	}

	close(e.stopChannel)
	e.logger.Info("Advanced order matching engine stopped")
	return nil
}

// GetOrCreateOrderBook gets or creates an order book for a symbol
func (e *AdvancedOrderMatchingEngine) GetOrCreateOrderBook(symbol string) *AdvancedOrderBook {
	if book, exists := e.orderBooks.Load(symbol); exists {
		return book.(*AdvancedOrderBook)
	}

	// Create new advanced order book
	basicBook := NewOrderBook(symbol, e.logger)
	advancedBook := &AdvancedOrderBook{
		OrderBook: basicBook,
		priceImprovement: &PriceImprovementEngine{
			enabled:        e.config.EnablePriceImprovement,
			tickSize:       e.config.TickSize,
			minImprovement: e.config.TickSize,
			maxImprovement: e.config.TickSize * 5,
		},
		icebergManager: &IcebergOrderManager{
			enabled:      e.config.EnableIcebergOrders,
			activeOrders: make(map[string]*IcebergOrder),
		},
		hiddenOrderPool: &HiddenOrderPool{
			enabled:      e.config.EnableHiddenOrders,
			hiddenOrders: make(map[string]*types.Order),
		},
		marketImpactCalc: &MarketImpactCalculator{
			enabled:          true,
			impactModel:      "sqrt",
			liquidityFactor:  0.1,
			volatilityFactor: 0.05,
			historicalTrades: make([]Trade, 0, 1000),
		},
		performanceTracker: &PerformanceTracker{
			latencyHistogram: make(map[time.Duration]int64),
			lastResetTime:    time.Now(),
		},
	}

	e.orderBooks.Store(symbol, advancedBook)
	return advancedBook
}

// AddOrder adds an order with advanced processing
func (e *AdvancedOrderMatchingEngine) AddOrder(order *types.Order) ([]*Trade, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.updateMetrics(latency)
	}()

	if atomic.LoadInt32(&e.isRunning) != 1 {
		return nil, fmt.Errorf("engine is not running")
	}

	// Validate order
	if err := e.validateOrder(order); err != nil {
		e.publishEvent(&MatchingEvent{
			Type:      EventOrderRejected,
			Symbol:    order.Symbol,
			Order:     order,
			Timestamp: time.Now(),
		})
		return nil, err
	}

	// Get order book
	book := e.GetOrCreateOrderBook(order.Symbol)

	// Handle special order types
	if order.IsIceberg() && e.config.EnableIcebergOrders {
		return e.handleIcebergOrder(book, order)
	}

	if order.IsHidden && e.config.EnableHiddenOrders {
		return e.handleHiddenOrder(book, order)
	}

	// Process regular order with enhancements
	trades, err := e.processAdvancedOrder(book, order)
	if err != nil {
		return nil, err
	}

	// Publish events
	e.publishEvent(&MatchingEvent{
		Type:      EventOrderAdded,
		Symbol:    order.Symbol,
		Order:     order,
		Timestamp: time.Now(),
	})

	for _, trade := range trades {
		e.publishEvent(&MatchingEvent{
			Type:      EventTradeExecuted,
			Symbol:    order.Symbol,
			Trade:     trade,
			Timestamp: time.Now(),
		})
	}

	atomic.AddInt64(&e.metrics.OrdersProcessed, 1)
	return trades, nil
}

// processAdvancedOrder processes an order with advanced features
func (e *AdvancedOrderMatchingEngine) processAdvancedOrder(book *AdvancedOrderBook, order *types.Order) ([]*Trade, error) {
	// Calculate market impact
	if book.marketImpactCalc.enabled {
		impact := e.calculateMarketImpact(book, order)
		order.EstimatedImpact = impact
	}

	// Apply price improvement if enabled
	if book.priceImprovement.enabled && order.Type == types.OrderTypeLimit {
		e.applyPriceImprovement(book, order)
	}

	// Process the order using the basic engine
	trades := book.AddOrder(order)

	// Update market impact calculator with new trades
	if book.marketImpactCalc.enabled {
		book.marketImpactCalc.mu.Lock()
		for _, trade := range trades {
			book.marketImpactCalc.historicalTrades = append(book.marketImpactCalc.historicalTrades, *trade)
			// Keep only last 1000 trades
			if len(book.marketImpactCalc.historicalTrades) > 1000 {
				book.marketImpactCalc.historicalTrades = book.marketImpactCalc.historicalTrades[1:]
			}
		}
		book.marketImpactCalc.mu.Unlock()
	}

	return trades, nil
}

// calculateMarketImpact calculates the market impact of an order
func (e *AdvancedOrderMatchingEngine) calculateMarketImpact(book *AdvancedOrderBook, order *types.Order) float64 {
	calc := book.marketImpactCalc
	calc.mu.RLock()
	defer calc.mu.RUnlock()

	if len(calc.historicalTrades) == 0 {
		return 0.0
	}

	// Calculate average trade size
	totalVolume := 0.0
	for _, trade := range calc.historicalTrades {
		totalVolume += trade.Quantity
	}
	avgTradeSize := totalVolume / float64(len(calc.historicalTrades))

	// Calculate impact based on model
	switch calc.impactModel {
	case "linear":
		return calc.liquidityFactor * (order.Quantity / avgTradeSize)
	case "sqrt":
		return calc.liquidityFactor * math.Sqrt(order.Quantity/avgTradeSize)
	case "log":
		return calc.liquidityFactor * math.Log(1+order.Quantity/avgTradeSize)
	default:
		return calc.liquidityFactor * math.Sqrt(order.Quantity/avgTradeSize)
	}
}

// applyPriceImprovement applies price improvement to limit orders
func (e *AdvancedOrderMatchingEngine) applyPriceImprovement(book *AdvancedOrderBook, order *types.Order) {
	improvement := book.priceImprovement

	if order.Side == types.OrderSideBuy {
		// For buy orders, improve by increasing the price slightly
		maxPrice := order.Price + improvement.maxImprovement
		if book.Asks.Len() > 0 {
			bestAsk := book.Asks.Peek()
			if bestAsk.Price < maxPrice {
				// Improve price to just below best ask
				improvedPrice := bestAsk.Price - improvement.tickSize
				if improvedPrice > order.Price {
					order.Price = improvedPrice
					order.IsPriceImproved = true
				}
			}
		}
	} else {
		// For sell orders, improve by decreasing the price slightly
		minPrice := order.Price - improvement.maxImprovement
		if book.Bids.Len() > 0 {
			bestBid := book.Bids.Peek()
			if bestBid.Price > minPrice {
				// Improve price to just above best bid
				improvedPrice := bestBid.Price + improvement.tickSize
				if improvedPrice < order.Price {
					order.Price = improvedPrice
					order.IsPriceImproved = true
				}
			}
		}
	}
}

// handleIcebergOrder handles iceberg order processing
func (e *AdvancedOrderMatchingEngine) handleIcebergOrder(book *AdvancedOrderBook, order *types.Order) ([]*Trade, error) {
	manager := book.icebergManager
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Create iceberg order
	icebergOrder := &IcebergOrder{
		ParentOrder:   order,
		DisplaySize:   order.DisplayQuantity,
		TotalSize:     order.Quantity,
		RemainingSize: order.Quantity,
		RefreshSize:   order.DisplayQuantity,
	}

	// Create first visible order
	visibleOrder := e.createVisibleOrder(icebergOrder)
	manager.activeOrders[order.ID] = icebergOrder

	// Process the visible order
	return e.processAdvancedOrder(book, visibleOrder)
}

// handleHiddenOrder handles hidden order processing
func (e *AdvancedOrderMatchingEngine) handleHiddenOrder(book *AdvancedOrderBook, order *types.Order) ([]*Trade, error) {
	pool := book.hiddenOrderPool
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Store in hidden pool
	pool.hiddenOrders[order.ID] = order

	// Hidden orders don't appear in the book but can be matched
	// This is a simplified implementation
	return nil, nil
}

// createVisibleOrder creates a visible order from an iceberg order
func (e *AdvancedOrderMatchingEngine) createVisibleOrder(iceberg *IcebergOrder) *types.Order {
	visibleOrder := *iceberg.ParentOrder // Copy the parent order
	visibleOrder.ID = uuid.New().String()
	visibleOrder.Quantity = math.Min(iceberg.DisplaySize, iceberg.RemainingSize)
	visibleOrder.ParentOrderID = iceberg.ParentOrder.ID
	visibleOrder.IsIcebergChild = true

	iceberg.CurrentOrder = &visibleOrder
	return &visibleOrder
}

// validateOrder validates an order before processing
func (e *AdvancedOrderMatchingEngine) validateOrder(order *types.Order) error {
	if order.Symbol == "" {
		return fmt.Errorf("order symbol cannot be empty")
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity must be positive")
	}
	if order.Type == types.OrderTypeLimit && order.Price <= 0 {
		return fmt.Errorf("limit order price must be positive")
	}
	if order.Side != types.OrderSideBuy && order.Side != types.OrderSideSell {
		return fmt.Errorf("invalid order side")
	}
	return nil
}

// updateMetrics updates engine performance metrics
func (e *AdvancedOrderMatchingEngine) updateMetrics(latency time.Duration) {
	// Update latency metrics
	if latency > e.metrics.MaxLatency {
		e.metrics.MaxLatency = latency
	}

	// Simple moving average for latency
	e.metrics.AverageLatency = (e.metrics.AverageLatency + latency) / 2

	// Update timestamp
	e.metrics.LastUpdateTime = time.Now()
}

// publishEvent publishes a matching event
func (e *AdvancedOrderMatchingEngine) publishEvent(event *MatchingEvent) {
	select {
	case e.eventChannel <- event:
	default:
		e.logger.Warn("Event channel full, dropping event",
			zap.String("event_type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// processEvents processes matching events
func (e *AdvancedOrderMatchingEngine) processEvents() {
	for {
		select {
		case event := <-e.eventChannel:
			e.handleEvent(event)
		case <-e.stopChannel:
			return
		}
	}
}

// handleEvent handles a matching event
func (e *AdvancedOrderMatchingEngine) handleEvent(event *MatchingEvent) {
	switch event.Type {
	case EventTradeExecuted:
		atomic.AddInt64(&e.metrics.TotalTrades, 1)
		if event.Trade != nil {
			e.metrics.TotalVolume += event.Trade.Quantity
		}
	case EventOrderAdded:
		// Handle order added event
	case EventOrderCanceled:
		// Handle order canceled event
	case EventOrderFilled:
		// Handle order filled event
	case EventOrderRejected:
		// Handle order rejected event
	}
}

// GetMetrics returns current engine metrics
func (e *AdvancedOrderMatchingEngine) GetMetrics() *MatchingMetrics {
	return e.metrics
}

// GetOrderBook returns the order book for a symbol
func (e *AdvancedOrderMatchingEngine) GetOrderBook(symbol string) *AdvancedOrderBook {
	if book, exists := e.orderBooks.Load(symbol); exists {
		return book.(*AdvancedOrderBook)
	}
	return nil
}
