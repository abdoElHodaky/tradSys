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
type OrderType int32

const (
	// OrderTypeMarket represents a market order
	OrderTypeMarket OrderType = 0
	// OrderTypeLimit represents a limit order
	OrderTypeLimit OrderType = 1
	// OrderTypeStop represents a stop order
	OrderTypeStop OrderType = 2
	// OrderTypeStopLimit represents a stop limit order
	OrderTypeStopLimit OrderType = 3
	// OrderTypeTrailing represents a trailing order
	OrderTypeTrailing OrderType = 4
	// OrderTypeIOC represents an immediate-or-cancel order
	OrderTypeIOC OrderType = 5
	// OrderTypeFOK represents a fill-or-kill order
	OrderTypeFOK OrderType = 6
	// OrderTypeConditional represents a conditional order
	OrderTypeConditional OrderType = 7
)

// OrderSide represents the side of an order
type OrderSide int32

const (
	// OrderSideBuy represents a buy order
	OrderSideBuy OrderSide = 0
	// OrderSideSell represents a sell order
	OrderSideSell OrderSide = 1
)

// OrderStatus represents the status of an order
type OrderStatus int32

const (
	// OrderStatusNew represents a new order
	OrderStatusNew OrderStatus = 0
	// OrderStatusPartiallyFilled represents a partially filled order
	OrderStatusPartiallyFilled OrderStatus = 1
	// OrderStatusFilled represents a filled order
	OrderStatusFilled OrderStatus = 2
	// OrderStatusCancelled represents a cancelled order
	OrderStatusCancelled OrderStatus = 3
	// OrderStatusRejected represents a rejected order
	OrderStatusRejected OrderStatus = 4
	// OrderStatusExpired represents an expired order
	OrderStatusExpired OrderStatus = 5
	// OrderStatusPending represents a pending order
	OrderStatusPending OrderStatus = 6
	// OrderStatusProcessing represents a processing order
	OrderStatusProcessing OrderStatus = 7
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
type OrderHeap []*Order

// Len returns the length of the heap
func (h OrderHeap) Len() int { return len(h) }

// Less returns whether the order at index i should be before the order at index j
func (h OrderHeap) Less(i, j int) bool {
	// For buy orders, higher prices come first
	if h[i].Side == OrderSideBuy {
		if h[i].Price == h[j].Price {
			return h[i].Timestamp.Before(h[j].Timestamp)
		}
		return h[i].Price > h[j].Price
	}

	// For sell orders, lower prices come first
	if h[i].Price == h[j].Price {
		return h[i].Timestamp.Before(h[j].Timestamp)
	}
	return h[i].Price < h[j].Price
}

// Swap swaps the orders at indices i and j
func (h OrderHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

// Push adds an order to the heap
func (h *OrderHeap) Push(x interface{}) {
	n := len(*h)
	order := x.(*Order)
	order.Index = n
	*h = append(*h, order)
}

// Pop removes and returns the top order from the heap
func (h *OrderHeap) Pop() interface{} {
	old := *h
	n := len(old)
	order := old[n-1]
	old[n-1] = nil  // avoid memory leak
	order.Index = -1
	*h = old[0 : n-1]
	return order
}

// OrderBook represents an order book for a symbol
type OrderBook struct {
	// Symbol
	Symbol string

	// Orders
	BuyOrders     *OrderHeap
	SellOrders    *OrderHeap
	BuyStopOrders *OrderHeap
	SellStopOrders *OrderHeap

	// Order map for quick lookup
	OrderMap map[string]*Order

	// Mutex for thread safety
	mu sync.RWMutex

	// Last price
	LastPrice float64

	// Statistics
	TradeCount uint64
	Volume     float64
}

// NewOrderBook creates a new order book
func NewOrderBook(symbol string) *OrderBook {
	buyOrders := make(OrderHeap, 0)
	sellOrders := make(OrderHeap, 0)
	buyStopOrders := make(OrderHeap, 0)
	sellStopOrders := make(OrderHeap, 0)

	return &OrderBook{
		Symbol:        symbol,
		BuyOrders:     &buyOrders,
		SellOrders:    &sellOrders,
		BuyStopOrders: &buyStopOrders,
		SellStopOrders: &sellStopOrders,
		OrderMap:      make(map[string]*Order),
		LastPrice:     0,
		TradeCount:    0,
		Volume:        0,
	}
}

// OrderMatchingEngineConfig contains configuration for the order matching engine
type OrderMatchingEngineConfig struct {
	// CleanupInterval is the interval at which to clean up the order book
	CleanupInterval time.Duration

	// MaxOrdersPerBook is the maximum number of orders per book
	MaxOrdersPerBook int

	// EnableMetrics enables metrics collection
	EnableMetrics bool
}

// DefaultOrderMatchingEngineConfig returns the default order matching engine configuration
func DefaultOrderMatchingEngineConfig() OrderMatchingEngineConfig {
	return OrderMatchingEngineConfig{
		CleanupInterval:  1 * time.Hour,
		MaxOrdersPerBook: 10000,
		EnableMetrics:    true,
	}
}

// OrderMatchingEngine is an order matching engine
type OrderMatchingEngine struct {
	// Configuration
	config OrderMatchingEngineConfig

	// Order books
	OrderBooks map[string]*OrderBook

	// Mutex for thread safety
	mu sync.RWMutex

	// Statistics
	ordersProcessed uint64
	tradesExecuted  uint64

	// Cleanup
	lastCleanup     time.Time
	cleanupInterval time.Duration

	// Logger
	logger *zap.Logger
}

// NewOrderMatchingEngine creates a new order matching engine
func NewOrderMatchingEngine(logger *zap.Logger) *OrderMatchingEngine {
	return NewOrderMatchingEngineWithConfig(DefaultOrderMatchingEngineConfig(), logger)
}

// NewOrderMatchingEngineWithConfig creates a new order matching engine with the given configuration
func NewOrderMatchingEngineWithConfig(config OrderMatchingEngineConfig, logger *zap.Logger) *OrderMatchingEngine {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &OrderMatchingEngine{
		config:          config,
		OrderBooks:      make(map[string]*OrderBook),
		lastCleanup:     time.Now(),
		cleanupInterval: config.CleanupInterval,
		logger:          logger,
	}
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
	orderBook := NewOrderBook(symbol)
	e.OrderBooks[symbol] = orderBook

	e.logger.Info("Created order book",
		zap.String("symbol", symbol),
	)

	return orderBook
}

// GetOrderBook gets an order book for a symbol
func (e *OrderMatchingEngine) GetOrderBook(symbol string) (*OrderBook, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if the order book exists
	orderBook, exists := e.OrderBooks[symbol]
	if !exists {
		return nil, fmt.Errorf("order book for symbol %s does not exist", symbol)
	}

	return orderBook, nil
}

// PlaceOrder places an order in the order book
func (e *OrderMatchingEngine) PlaceOrder(order *Order) error {
	// Get the order book
	orderBook, err := e.GetOrderBook(order.Symbol)
	if err != nil {
		return err
	}

	// Lock the order book
	orderBook.mu.Lock()
	defer orderBook.mu.Unlock()

	// Check if the order book is full
	if len(orderBook.OrderMap) >= e.config.MaxOrdersPerBook {
		return fmt.Errorf("order book for symbol %s is full", order.Symbol)
	}

	// Add the order to the order map
	orderBook.OrderMap[order.ID] = order

	// Handle stop orders
	if order.Type == OrderTypeStopLimit || order.Type == OrderTypeStop {
		return e.placeStopOrder(orderBook, order)
	}

	// Handle market and limit orders
	return e.placeMarketOrLimitOrder(orderBook, order)
}

// placeStopOrder places a stop order in the order book
func (e *OrderMatchingEngine) placeStopOrder(orderBook *OrderBook, order *Order) error {
	// Add the order to the appropriate stop order heap
	if order.Side == OrderSideBuy {
		heap.Push(orderBook.BuyStopOrders, order)
	} else {
		heap.Push(orderBook.SellStopOrders, order)
	}

	e.logger.Debug("Placed stop order",
		zap.String("orderID", order.ID),
		zap.String("symbol", order.Symbol),
		zap.String("side", string(order.Side)),
		zap.String("type", string(order.Type)),
		zap.Float64("price", order.Price),
		zap.Float64("stopPrice", order.StopPrice),
		zap.Float64("size", order.Size),
	)

	// Check if the stop price is triggered
	if orderBook.LastPrice > 0 {
		e.checkStopOrders(orderBook)
	}

	return nil
}

// placeMarketOrLimitOrder places a market or limit order in the order book
func (e *OrderMatchingEngine) placeMarketOrLimitOrder(orderBook *OrderBook, order *Order) error {
	// Update statistics
	atomic.AddUint64(&e.ordersProcessed, 1)

	// Match the order
	trades, err := e.matchOrder(orderBook, order)
	if err != nil {
		return err
	}

	// If the order is a market order and it's not fully filled, reject it
	if order.Type == OrderTypeMarket && order.Status != OrderStatusFilled {
		order.Status = OrderStatusRejected
		return fmt.Errorf("market order could not be fully filled")
	}

	// If the order is not fully filled, add it to the order book
	if order.Status != OrderStatusFilled {
		if order.Side == OrderSideBuy {
			heap.Push(orderBook.BuyOrders, order)
		} else {
			heap.Push(orderBook.SellOrders, order)
		}
	}

	e.logger.Debug("Placed order",
		zap.String("orderID", order.ID),
		zap.String("symbol", order.Symbol),
		zap.String("side", string(order.Side)),
		zap.String("type", string(order.Type)),
		zap.Float64("price", order.Price),
		zap.Float64("size", order.Size),
		zap.Float64("remainingSize", order.RemainingSize),
		zap.String("status", string(order.Status)),
		zap.Int("trades", len(trades)),
	)

	return nil
}

// matchOrder matches an order against the order book
func (e *OrderMatchingEngine) matchOrder(orderBook *OrderBook, order *Order) ([]Trade, error) {
	trades := make([]Trade, 0)

	// Get the opposite order heap
	var oppositeOrders *OrderHeap
	if order.Side == OrderSideBuy {
		oppositeOrders = orderBook.SellOrders
	} else {
		oppositeOrders = orderBook.BuyOrders
	}

	// Match the order
	for oppositeOrders.Len() > 0 {
		// Get the top order
		topOrder := (*oppositeOrders)[0]

		// Check if the orders can be matched
		if !e.canMatch(order, topOrder) {
			break
		}

		// Match the orders
		trade, err := e.executeMatch(orderBook, order, topOrder)
		if err != nil {
			return trades, err
		}

		// Add the trade to the list
		trades = append(trades, trade)

		// Update the last price
		orderBook.LastPrice = trade.Price

		// Check if the top order is fully filled
		if topOrder.Status == OrderStatusFilled {
			// Remove the top order from the heap
			heap.Pop(oppositeOrders)
		}

		// Check if the order is fully filled
		if order.Status == OrderStatusFilled {
			break
		}
	}

	// Check if any stop orders are triggered
	if len(trades) > 0 {
		e.checkStopOrders(orderBook)
	}

	return trades, nil
}

// canMatch checks if two orders can be matched
func (e *OrderMatchingEngine) canMatch(order, oppositeOrder *Order) bool {
	// Check if the orders are on opposite sides
	if order.Side == oppositeOrder.Side {
		return false
	}

	// Check if the prices match
	if order.Side == OrderSideBuy {
		// For market orders, match at any price
		if order.Type == OrderTypeMarket {
			return true
		}
		// For limit orders, buy price must be >= sell price
		return order.Price >= oppositeOrder.Price
	} else {
		// For market orders, match at any price
		if order.Type == OrderTypeMarket {
			return true
		}
		// For limit orders, sell price must be <= buy price
		return order.Price <= oppositeOrder.Price
	}
}

// executeMatch executes a match between two orders
func (e *OrderMatchingEngine) executeMatch(orderBook *OrderBook, order, oppositeOrder *Order) (Trade, error) {
	// Calculate the match size
	matchSize := order.RemainingSize
	if oppositeOrder.RemainingSize < matchSize {
		matchSize = oppositeOrder.RemainingSize
	}

	// Calculate the match price (use the price of the order that was in the book first)
	matchPrice := oppositeOrder.Price

	// Update the remaining sizes
	order.RemainingSize -= matchSize
	oppositeOrder.RemainingSize -= matchSize

	// Update the order statuses
	if order.RemainingSize == 0 {
		order.Status = OrderStatusFilled
	} else {
		order.Status = OrderStatusPartiallyFilled
	}

	if oppositeOrder.RemainingSize == 0 {
		oppositeOrder.Status = OrderStatusFilled
	} else {
		oppositeOrder.Status = OrderStatusPartiallyFilled
	}

	// Create the trade
	trade := Trade{
		ID:             uuid.New().String(),
		Symbol:         order.Symbol,
		Price:          matchPrice,
		Size:           matchSize,
		Timestamp:      time.Now(),
		BuyOrderID:     "",
		SellOrderID:    "",
		BuyUserID:      "",
		SellUserID:     "",
		MakerOrderID:   oppositeOrder.ID,
		TakerOrderID:   order.ID,
		MakerUserID:    oppositeOrder.UserID,
		TakerUserID:    order.UserID,
		MakerFee:       0,
		TakerFee:       0,
		MakerFeeCurrency: "",
		TakerFeeCurrency: "",
	}

	// Set the buy and sell order IDs
	if order.Side == OrderSideBuy {
		trade.BuyOrderID = order.ID
		trade.SellOrderID = oppositeOrder.ID
		trade.BuyUserID = order.UserID
		trade.SellUserID = oppositeOrder.UserID
	} else {
		trade.BuyOrderID = oppositeOrder.ID
		trade.SellOrderID = order.ID
		trade.BuyUserID = oppositeOrder.UserID
		trade.SellUserID = order.UserID
	}

	// Update statistics
	atomic.AddUint64(&e.tradesExecuted, 1)
	atomic.AddUint64(&orderBook.TradeCount, 1)
	orderBook.Volume += matchSize

	e.logger.Debug("Executed trade",
		zap.String("tradeID", trade.ID),
		zap.String("symbol", trade.Symbol),
		zap.Float64("price", trade.Price),
		zap.Float64("size", trade.Size),
		zap.String("buyOrderID", trade.BuyOrderID),
		zap.String("sellOrderID", trade.SellOrderID),
	)

	return trade, nil
}

// checkStopOrders checks if any stop orders are triggered
func (e *OrderMatchingEngine) checkStopOrders(orderBook *OrderBook) {
	// Check buy stop orders
	for orderBook.BuyStopOrders.Len() > 0 {
		// Get the top order
		topOrder := (*orderBook.BuyStopOrders)[0]

		// Check if the stop price is triggered
		if orderBook.LastPrice >= topOrder.StopPrice {
			// Remove the order from the stop order heap
			heap.Pop(orderBook.BuyStopOrders)

			// Convert to a market or limit order
			if topOrder.Type == OrderTypeStop {
				topOrder.Type = OrderTypeMarket
			} else {
				topOrder.Type = OrderTypeLimit
			}

			// Place the order
			e.placeMarketOrLimitOrder(orderBook, topOrder)
		} else {
			// Stop orders are sorted by price, so if the top order is not triggered, none are
			break
		}
	}

	// Check sell stop orders
	for orderBook.SellStopOrders.Len() > 0 {
		// Get the top order
		topOrder := (*orderBook.SellStopOrders)[0]

		// Check if the stop price is triggered
		if orderBook.LastPrice <= topOrder.StopPrice {
			// Remove the order from the stop order heap
			heap.Pop(orderBook.SellStopOrders)

			// Convert to a market or limit order
			if topOrder.Type == OrderTypeStop {
				topOrder.Type = OrderTypeMarket
			} else {
				topOrder.Type = OrderTypeLimit
			}

			// Place the order
			e.placeMarketOrLimitOrder(orderBook, topOrder)
		} else {
			// Stop orders are sorted by price, so if the top order is not triggered, none are
			break
		}
	}
}

// CancelOrder cancels an order
func (e *OrderMatchingEngine) CancelOrder(symbol, orderID string) error {
	// Get the order book
	orderBook, err := e.GetOrderBook(symbol)
	if err != nil {
		return err
	}

	// Lock the order book
	orderBook.mu.Lock()
	defer orderBook.mu.Unlock()

	// Get the order
	order, exists := orderBook.OrderMap[orderID]
	if !exists {
		return fmt.Errorf("order %s does not exist", orderID)
	}

	// Check if the order is already filled or cancelled
	if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
		return fmt.Errorf("order %s is already %s", orderID, order.Status)
	}

	// Mark the order as cancelled
	order.Status = OrderStatusCancelled

	e.logger.Debug("Cancelled order",
		zap.String("orderID", order.ID),
		zap.String("symbol", order.Symbol),
		zap.String("side", string(order.Side)),
		zap.String("type", string(order.Type)),
	)

	return nil
}

// GetOrder gets an order
func (e *OrderMatchingEngine) GetOrder(symbol, orderID string) (*Order, error) {
	// Get the order book
	orderBook, err := e.GetOrderBook(symbol)
	if err != nil {
		return nil, err
	}

	// Lock the order book
	orderBook.mu.RLock()
	defer orderBook.mu.RUnlock()

	// Get the order
	order, exists := orderBook.OrderMap[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s does not exist", orderID)
	}

	return order, nil
}

// GetOrderBookSnapshot gets a snapshot of the order book
func (e *OrderMatchingEngine) GetOrderBookSnapshot(symbol string, depth int) (*OrderBookSnapshot, error) {
	// Get the order book
	orderBook, err := e.GetOrderBook(symbol)
	if err != nil {
		return nil, err
	}

	// Lock the order book
	orderBook.mu.RLock()
	defer orderBook.mu.RUnlock()

	// Create the snapshot
	snapshot := &OrderBookSnapshot{
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids:      make([]OrderBookLevel, 0, depth),
		Asks:      make([]OrderBookLevel, 0, depth),
		LastPrice: orderBook.LastPrice,
		Volume:    orderBook.Volume,
	}

	// Add the bids
	bids := make(OrderHeap, orderBook.BuyOrders.Len())
	copy(bids, *orderBook.BuyOrders)
	heap.Init(&bids)

	for i := 0; i < depth && bids.Len() > 0; i++ {
		order := heap.Pop(&bids).(*Order)
		level := OrderBookLevel{
			Price: order.Price,
			Size:  order.RemainingSize,
		}
		snapshot.Bids = append(snapshot.Bids, level)
	}

	// Add the asks
	asks := make(OrderHeap, orderBook.SellOrders.Len())
	copy(asks, *orderBook.SellOrders)
	heap.Init(&asks)

	for i := 0; i < depth && asks.Len() > 0; i++ {
		order := heap.Pop(&asks).(*Order)
		level := OrderBookLevel{
			Price: order.Price,
			Size:  order.RemainingSize,
		}
		snapshot.Asks = append(snapshot.Asks, level)
	}

	return snapshot, nil
}

// Trade represents a trade between two orders
type Trade struct {
	// Trade ID
	ID string

	// Trade details
	Symbol    string
	Price     float64
	Size      float64
	Timestamp time.Time

	// Order IDs
	BuyOrderID  string
	SellOrderID string

	// User IDs
	BuyUserID  string
	SellUserID string

	// Maker and taker
	MakerOrderID string
	TakerOrderID string
	MakerUserID  string
	TakerUserID  string

	// Fees
	MakerFee       float64
	TakerFee       float64
	MakerFeeCurrency string
	TakerFeeCurrency string
}

// OrderBookLevel represents a level in the order book
type OrderBookLevel struct {
	Price float64
	Size  float64
}

// OrderBookSnapshot represents a snapshot of the order book
type OrderBookSnapshot struct {
	Symbol    string
	Timestamp time.Time
	Bids      []OrderBookLevel
	Asks      []OrderBookLevel
	LastPrice float64
	Volume    float64
}

// GetStats gets statistics about the order matching engine
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
	// Create a new heap
	newHeap := make(OrderHeap, 0, h.Len())

	// Add non-nil entries to the new heap
	for _, order := range *h {
		if order != nil && order.Status != OrderStatusFilled && order.Status != OrderStatusCancelled {
			newHeap = append(newHeap, order)
		}
	}

	// Replace the old heap with the new one
	*h = newHeap

	// Heapify the new heap
	heap.Init(h)
}
