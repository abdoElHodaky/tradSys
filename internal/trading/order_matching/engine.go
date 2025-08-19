package order_matching

import (
	"container/heap"
	"sync"
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
	// ID is the unique identifier for the order
	ID string
	// Symbol is the trading symbol
	Symbol string
	// Side is the side of the order (buy or sell)
	Side OrderSide
	// Type is the type of the order
	Type OrderType
	// Price is the price of the order
	Price float64
	// Quantity is the quantity of the order
	Quantity float64
	// FilledQuantity is the filled quantity of the order
	FilledQuantity float64
	// Status is the status of the order
	Status OrderStatus
	// CreatedAt is the time the order was created
	CreatedAt time.Time
	// UpdatedAt is the time the order was last updated
	UpdatedAt time.Time
	// ClientOrderID is the client order ID
	ClientOrderID string
	// UserID is the user ID
	UserID string
	// StopPrice is the stop price for stop orders
	StopPrice float64
	// TimeInForce is the time in force for the order
	TimeInForce string
	// Index is the index in the heap
	Index int
}

// Trade represents a trade
type Trade struct {
	// ID is the unique identifier for the trade
	ID string
	// Symbol is the trading symbol
	Symbol string
	// Price is the price of the trade
	Price float64
	// Quantity is the quantity of the trade
	Quantity float64
	// BuyOrderID is the buy order ID
	BuyOrderID string
	// SellOrderID is the sell order ID
	SellOrderID string
	// Timestamp is the time the trade was executed
	Timestamp time.Time
	// TakerSide is the side of the taker
	TakerSide OrderSide
	// MakerSide is the side of the maker
	MakerSide OrderSide
	// TakerFee is the fee for the taker
	TakerFee float64
	// MakerFee is the fee for the maker
	MakerFee float64
}

// OrderBook represents an order book for a symbol
type OrderBook struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the buy orders
	Bids *OrderHeap
	// Asks is the sell orders
	Asks *OrderHeap
	// Orders is a map of order ID to order
	Orders map[string]*Order
	// StopBids is the stop buy orders
	StopBids *OrderHeap
	// StopAsks is the stop sell orders
	StopAsks *OrderHeap
	// LastPrice is the last traded price
	LastPrice float64
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
}

// OrderHeap is a heap of orders
type OrderHeap struct {
	// Orders is the slice of orders
	Orders []*Order
	// Side is the side of the orders
	Side OrderSide
}

// Len returns the length of the heap
func (h OrderHeap) Len() int { return len(h.Orders) }

// Less returns whether the order at index i is less than the order at index j
func (h OrderHeap) Less(i, j int) bool {
	if h.Side == OrderSideBuy {
		// For buy orders, higher prices have higher priority
		if h.Orders[i].Price == h.Orders[j].Price {
			// If prices are equal, older orders have higher priority
			return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
		}
		return h.Orders[i].Price > h.Orders[j].Price
	}
	// For sell orders, lower prices have higher priority
	if h.Orders[i].Price == h.Orders[j].Price {
		// If prices are equal, older orders have higher priority
		return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
	}
	return h.Orders[i].Price < h.Orders[j].Price
}

// Swap swaps the orders at indices i and j
func (h OrderHeap) Swap(i, j int) {
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

// NewOrderBook creates a new order book
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

// processOrder processes an order and returns any trades that were executed
func (ob *OrderBook) processOrder(order *Order) ([]*Trade, error) {
	trades := make([]*Trade, 0)

	// Handle market orders
	if order.Type == OrderTypeMarket {
		if order.Side == OrderSideBuy {
			// Process market buy order
			for ob.Asks.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestAsk := ob.Asks.Peek()
				trade := ob.matchOrders(order, bestAsk)
				trades = append(trades, trade)

				// Update order status
				if bestAsk.Status == OrderStatusFilled {
					heap.Pop(ob.Asks)
				}
			}
		} else {
			// Process market sell order
			for ob.Bids.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestBid := ob.Bids.Peek()
				trade := ob.matchOrders(order, bestBid)
				trades = append(trades, trade)

				// Update order status
				if bestBid.Status == OrderStatusFilled {
					heap.Pop(ob.Bids)
				}
			}
		}

		// If market order is not fully filled, cancel the remaining quantity
		if order.Quantity > order.FilledQuantity {
			order.Status = OrderStatusPartiallyFilled
			ob.logger.Warn("Market order not fully filled",
				zap.String("order_id", order.ID),
				zap.Float64("quantity", order.Quantity),
				zap.Float64("filled_quantity", order.FilledQuantity))
		} else {
			order.Status = OrderStatusFilled
		}
	} else if order.Type == OrderTypeLimit {
		// Handle limit orders
		if order.Side == OrderSideBuy {
			// Process limit buy order
			for ob.Asks.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestAsk := ob.Asks.Peek()
				// Check if the best ask price is less than or equal to the buy price
				if bestAsk.Price <= order.Price {
					trade := ob.matchOrders(order, bestAsk)
					trades = append(trades, trade)

					// Update order status
					if bestAsk.Status == OrderStatusFilled {
						heap.Pop(ob.Asks)
					}
				} else {
					break
				}
			}

			// If limit order is not fully filled, add it to the order book
			if order.Quantity > order.FilledQuantity {
				if order.FilledQuantity > 0 {
					order.Status = OrderStatusPartiallyFilled
				}
				heap.Push(ob.Bids, order)
			} else {
				order.Status = OrderStatusFilled
			}
		} else {
			// Process limit sell order
			for ob.Bids.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestBid := ob.Bids.Peek()
				// Check if the best bid price is greater than or equal to the sell price
				if bestBid.Price >= order.Price {
					trade := ob.matchOrders(order, bestBid)
					trades = append(trades, trade)

					// Update order status
					if bestBid.Status == OrderStatusFilled {
						heap.Pop(ob.Bids)
					}
				} else {
					break
				}
			}

			// If limit order is not fully filled, add it to the order book
			if order.Quantity > order.FilledQuantity {
				if order.FilledQuantity > 0 {
					order.Status = OrderStatusPartiallyFilled
				}
				heap.Push(ob.Asks, order)
			} else {
				order.Status = OrderStatusFilled
			}
		}
	}

	// Update last price if trades were executed
	if len(trades) > 0 {
		ob.LastPrice = trades[len(trades)-1].Price
		// Check stop orders
		ob.checkStopOrders()
	}

	return trades, nil
}

// matchOrders matches two orders and creates a trade
func (ob *OrderBook) matchOrders(taker *Order, maker *Order) *Trade {
	// Calculate the trade quantity
	tradeQuantity := taker.Quantity - taker.FilledQuantity
	remainingMakerQuantity := maker.Quantity - maker.FilledQuantity
	if tradeQuantity > remainingMakerQuantity {
		tradeQuantity = remainingMakerQuantity
	}

	// Calculate the trade price (maker's price)
	tradePrice := maker.Price

	// Update filled quantities
	taker.FilledQuantity += tradeQuantity
	maker.FilledQuantity += tradeQuantity

	// Update order statuses
	if maker.FilledQuantity >= maker.Quantity {
		maker.Status = OrderStatusFilled
	} else {
		maker.Status = OrderStatusPartiallyFilled
	}

	if taker.FilledQuantity >= taker.Quantity {
		taker.Status = OrderStatusFilled
	} else {
		taker.Status = OrderStatusPartiallyFilled
	}

	// Update timestamps
	now := time.Now()
	taker.UpdatedAt = now
	maker.UpdatedAt = now

	// Create trade
	trade := &Trade{
		ID:         uuid.New().String(),
		Symbol:     ob.Symbol,
		Price:      tradePrice,
		Quantity:   tradeQuantity,
		BuyOrderID: "",
		SellOrderID: "",
		Timestamp:  now,
		TakerSide:  taker.Side,
		MakerSide:  maker.Side,
		TakerFee:   0, // Fees would be calculated based on fee schedule
		MakerFee:   0, // Fees would be calculated based on fee schedule
	}

	// Set buy and sell order IDs
	if taker.Side == OrderSideBuy {
		trade.BuyOrderID = taker.ID
		trade.SellOrderID = maker.ID
	} else {
		trade.BuyOrderID = maker.ID
		trade.SellOrderID = taker.ID
	}

	return trade
}

// checkStopOrders checks if any stop orders should be triggered
func (ob *OrderBook) checkStopOrders() {
	// Check stop buy orders
	triggeredStopBuys := make([]*Order, 0)
	for ob.StopBids.Len() > 0 {
		stopBuy := ob.StopBids.Peek()
		if stopBuy.StopPrice <= ob.LastPrice {
			// Stop price triggered
			heap.Pop(ob.StopBids)
			triggeredStopBuys = append(triggeredStopBuys, stopBuy)
		} else {
			break
		}
	}

	// Check stop sell orders
	triggeredStopSells := make([]*Order, 0)
	for ob.StopAsks.Len() > 0 {
		stopSell := ob.StopAsks.Peek()
		if stopSell.StopPrice >= ob.LastPrice {
			// Stop price triggered
			heap.Pop(ob.StopAsks)
			triggeredStopSells = append(triggeredStopSells, stopSell)
		} else {
			break
		}
	}

	// Process triggered stop orders
	for _, stopBuy := range triggeredStopBuys {
		if stopBuy.Type == OrderTypeStopLimit {
			stopBuy.Type = OrderTypeLimit
		} else {
			stopBuy.Type = OrderTypeMarket
		}
		ob.processOrder(stopBuy)
	}

	for _, stopSell := range triggeredStopSells {
		if stopSell.Type == OrderTypeStopLimit {
			stopSell.Type = OrderTypeLimit
		} else {
			stopSell.Type = OrderTypeMarket
		}
		ob.processOrder(stopSell)
	}
}

// CancelOrder cancels an order
func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}

	if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
		return ErrInvalidOrderStatus
	}

	// Update order status
	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()

	// Remove from appropriate heap
	if order.Type == OrderTypeLimit {
		if order.Side == OrderSideBuy {
			// Find and remove from bids
			for i, o := range ob.Bids.Orders {
				if o.ID == orderID {
					heap.Remove(ob.Bids, i)
					break
				}
			}
		} else {
			// Find and remove from asks
			for i, o := range ob.Asks.Orders {
				if o.ID == orderID {
					heap.Remove(ob.Asks, i)
					break
				}
			}
		}
	} else if order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket {
		if order.Side == OrderSideBuy {
			// Find and remove from stop bids
			for i, o := range ob.StopBids.Orders {
				if o.ID == orderID {
					heap.Remove(ob.StopBids, i)
					break
				}
			}
		} else {
			// Find and remove from stop asks
			for i, o := range ob.StopAsks.Orders {
				if o.ID == orderID {
					heap.Remove(ob.StopAsks, i)
					break
				}
			}
		}
	}

	return nil
}

// GetOrder gets an order by ID
func (ob *OrderBook) GetOrder(orderID string) (*Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return nil, ErrOrderNotFound
	}

	return order, nil
}

// GetOrderBook gets the order book
func (ob *OrderBook) GetOrderBook(depth int) ([][]float64, [][]float64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	// Create a copy of the bids and asks
	bids := make([]*Order, len(ob.Bids.Orders))
	asks := make([]*Order, len(ob.Asks.Orders))
	copy(bids, ob.Bids.Orders)
	copy(asks, ob.Asks.Orders)

	// Sort bids and asks
	bidPrices := make(map[float64]float64)
	askPrices := make(map[float64]float64)

	for _, bid := range bids {
		bidPrices[bid.Price] += bid.Quantity - bid.FilledQuantity
	}

	for _, ask := range asks {
		askPrices[ask.Price] += ask.Quantity - ask.FilledQuantity
	}

	// Convert to arrays
	bidArray := make([][]float64, 0, len(bidPrices))
	askArray := make([][]float64, 0, len(askPrices))

	for price, quantity := range bidPrices {
		bidArray = append(bidArray, []float64{price, quantity})
	}

	for price, quantity := range askPrices {
		askArray = append(askArray, []float64{price, quantity})
	}

	// Sort bids in descending order
	for i := 0; i < len(bidArray); i++ {
		for j := i + 1; j < len(bidArray); j++ {
			if bidArray[i][0] < bidArray[j][0] {
				bidArray[i], bidArray[j] = bidArray[j], bidArray[i]
			}
		}
	}

	// Sort asks in ascending order
	for i := 0; i < len(askArray); i++ {
		for j := i + 1; j < len(askArray); j++ {
			if askArray[i][0] > askArray[j][0] {
				askArray[i], askArray[j] = askArray[j], askArray[i]
			}
		}
	}

	// Limit to depth
	if depth > 0 {
		if len(bidArray) > depth {
			bidArray = bidArray[:depth]
		}
		if len(askArray) > depth {
			askArray = askArray[:depth]
		}
	}

	return bidArray, askArray
}

// Engine represents an order matching engine
type Engine struct {
	// OrderBooks is a map of symbol to order book
	OrderBooks map[string]*OrderBook
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Trade channel
	TradeChannel chan *Trade
}

// NewEngine creates a new order matching engine
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{
		OrderBooks:   make(map[string]*OrderBook),
		logger:       logger,
		TradeChannel: make(chan *Trade, 1000),
	}
}

// GetOrderBook gets an order book for a symbol
func (e *Engine) GetOrderBook(symbol string) *OrderBook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	orderBook, exists := e.OrderBooks[symbol]
	if !exists {
		return nil
	}

	return orderBook
}

// CreateOrderBook creates an order book for a symbol
func (e *Engine) CreateOrderBook(symbol string) *OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()

	orderBook, exists := e.OrderBooks[symbol]
	if exists {
		return orderBook
	}

	orderBook = NewOrderBook(symbol, e.logger)
	e.OrderBooks[symbol] = orderBook

	return orderBook
}

// PlaceOrder places an order
func (e *Engine) PlaceOrder(order *Order) ([]*Trade, error) {
	e.mu.RLock()
	orderBook, exists := e.OrderBooks[order.Symbol]
	e.mu.RUnlock()

	if !exists {
		e.mu.Lock()
		orderBook = NewOrderBook(order.Symbol, e.logger)
		e.OrderBooks[order.Symbol] = orderBook
		e.mu.Unlock()
	}

	trades, err := orderBook.AddOrder(order)
	if err != nil {
		return nil, err
	}

	// Send trades to trade channel
	for _, trade := range trades {
		select {
		case e.TradeChannel <- trade:
		default:
			e.logger.Warn("Trade channel full, dropping trade",
				zap.String("trade_id", trade.ID),
				zap.String("symbol", trade.Symbol),
				zap.Float64("price", trade.Price),
				zap.Float64("quantity", trade.Quantity))
		}
	}

	return trades, nil
}

// CancelOrder cancels an order
func (e *Engine) CancelOrder(symbol, orderID string) error {
	e.mu.RLock()
	orderBook, exists := e.OrderBooks[symbol]
	e.mu.RUnlock()

	if !exists {
		return ErrSymbolNotFound
	}

	return orderBook.CancelOrder(orderID)
}

// GetOrder gets an order
func (e *Engine) GetOrder(symbol, orderID string) (*Order, error) {
	e.mu.RLock()
	orderBook, exists := e.OrderBooks[symbol]
	e.mu.RUnlock()

	if !exists {
		return nil, ErrSymbolNotFound
	}

	return orderBook.GetOrder(orderID)
}

// GetMarketData gets market data for a symbol
func (e *Engine) GetMarketData(symbol string, depth int) ([][]float64, [][]float64, float64, error) {
	e.mu.RLock()
	orderBook, exists := e.OrderBooks[symbol]
	e.mu.RUnlock()

	if !exists {
		return nil, nil, 0, ErrSymbolNotFound
	}

	bids, asks := orderBook.GetOrderBook(depth)
	return bids, asks, orderBook.LastPrice, nil
}

// Errors
var (
	ErrOrderNotFound     = NewError("order not found")
	ErrInvalidOrderStatus = NewError("invalid order status")
	ErrSymbolNotFound    = NewError("symbol not found")
)

// Error represents an error
type Error struct {
	Message string
}

// NewError creates a new error
func NewError(message string) *Error {
	return &Error{
		Message: message,
	}
}

// Error returns the error message
func (e *Error) Error() string {
	return e.Message
}

